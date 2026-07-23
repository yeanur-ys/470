from __future__ import annotations

from neo4j import GraphDatabase, Driver

# Caps the co-tag fan-out. A tag applied to N articles implies N*(N-1)/2 topic
# edges; at N=500 that is ~125,000 edges from a single tag, which dominates the
# modularity calculation and slows every poll. Sampling the 40 highest-read
# articles per tag keeps the signal (these communities are what readers
# actually browse) without the quadratic blow-up. The same cap is applied on
# the Go side in internal/graph/handler.go so the clustering the worker
# computes matches the edges the client is shown.
MAX_ARTICLES_PER_TAG = 40


def get_driver(uri: str, user: str, password: str) -> Driver:
    return GraphDatabase.driver(uri, auth=(user, password))


def fetch_all_article_ids(driver: Driver) -> list[str]:
    """Every article node, including ones with no relationships at all.

    Needed because Louvain must be told about isolated nodes explicitly —
    building the graph from edges alone silently omits them, and they then
    never receive a clusterId.
    """
    query = "MATCH (a:Article) RETURN a.id AS id"
    with driver.session() as session:
        return [record["id"] for record in session.run(query) if record["id"]]


def fetch_edges(driver: Driver) -> list[tuple[str, str]]:
    """The article lineage graph, (:Article)-[:SEQUENCE_OF]->(:Article)."""
    query = """
        MATCH (a:Article)-[:SEQUENCE_OF]->(b:Article)
        RETURN a.id AS source, b.id AS target
    """
    with driver.session() as session:
        return [(record["source"], record["target"]) for record in session.run(query)]


def fetch_topic_edges(driver: Driver) -> list[tuple[str, str]]:
    """Co-tag edges: pairs of articles sharing a HAS_TAG category.

    elementId ordering emits each undirected pair once rather than twice.
    """
    query = """
        MATCH (t:Tag)<-[:HAS_TAG]-(a:Article)
        WITH t, a ORDER BY a.readershipVolume DESC
        WITH t, collect(a)[..$cap] AS arts
        UNWIND arts AS a
        UNWIND arts AS b
        WITH a, b WHERE elementId(a) < elementId(b)
        RETURN a.id AS source, b.id AS target
    """
    with driver.session() as session:
        result = session.run(query, cap=MAX_ARTICLES_PER_TAG)
        return [(record["source"], record["target"]) for record in result]


def write_clusters(driver: Driver, clusters: dict[str, int]) -> None:
    """Writes each node's Louvain community back as `clusterId`, which
    useSemanticZoom.ts groups low-priority nodes by."""
    query = """
        UNWIND $rows AS row
        MATCH (a:Article {id: row.id})
        SET a.clusterId = row.clusterId
    """
    rows = [{"id": node_id, "clusterId": cluster_id} for node_id, cluster_id in clusters.items()]
    if not rows:
        return
    with driver.session() as session:
        # Chunked so a large corpus doesn't build one enormous parameter list.
        for start in range(0, len(rows), 1000):
            session.run(query, rows=rows[start : start + 1000])
