from __future__ import annotations

from neo4j import GraphDatabase, Driver


def get_driver(uri: str, user: str, password: str) -> Driver:
    return GraphDatabase.driver(uri, auth=(user, password))


def fetch_edges(driver: Driver) -> list[tuple[str, str]]:
    """Pulls the article lineage graph (:Article)-[:SEQUENCE_OF]->(:Article)
    that node.fragment.glsl / sigma-config.ts render client-side."""
    query = "MATCH (a:Article)-[:SEQUENCE_OF]->(b:Article) RETURN a.id AS source, b.id AS target"
    with driver.session() as session:
        result = session.run(query)
        return [(record["source"], record["target"]) for record in result]


def write_clusters(driver: Driver, clusters: dict[str, int]) -> None:
    """Writes each node's Louvain community back onto the graph as
    `clusterId`, which useSemanticZoom.ts groups low-priority nodes by."""
    query = "UNWIND $rows AS row MATCH (a:Article {id: row.id}) SET a.clusterId = row.clusterId"
    rows = [{"id": node_id, "clusterId": cluster_id} for node_id, cluster_id in clusters.items()]
    if not rows:
        return
    with driver.session() as session:
        session.run(query, rows=rows)
