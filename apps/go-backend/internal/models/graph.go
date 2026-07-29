package models

import (
	"context"
	"sort"
	"strconv"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultNodeLimit/maxNodeLimit cap how much of the graph one request can
// pull. NFR-11 wants "thousands of historical nodes" browsable, and the
// client renders them on the GPU, but an unbounded query against a growing
// corpus is still a denial-of-service waiting to happen.
const (
	defaultNodeLimit = 2000
	maxNodeLimit     = 10000
)

// Node/Edge/Cluster double as both the Neo4j-query-result shape and the JSON
// response shape — the same convention Article/User already use elsewhere in
// this codebase, rather than a parallel DTO set that would just mirror these
// fields a second time.
type Node struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	JournalistID     string   `json:"journalistId,omitempty"`
	JournalistName   string   `json:"journalistName,omitempty"`
	ReadershipVolume int64    `json:"readershipVolume"`
	CorruptionFactor float64  `json:"corruptionFactor"`
	ClusterID        *int64   `json:"clusterId,omitempty"`
	ClusterLabel     string   `json:"clusterLabel,omitempty"`
	Tags             []string `json:"tags"`
	IsRetracted      bool     `json:"isRetracted"`
	HasActiveAppeal  bool     `json:"hasActiveAppeal"`     // FR-9/FR-15: pulsing amber "under dispute" state
	CreatedAt        string   `json:"createdAt,omitempty"` // F-08: time-period bucketing, computed client-side from this
}

// Edge carries a Kind so the renderer can style the two relationship types
// differently. "sequence" is the directed SEQUENCE_OF lineage edge that FR-2
// is about; "topic" is an undirected co-tag link derived from the HAS_TAG
// relationships in Section 5.1. Lineage alone is a forest (an article has at
// most one parent), which lays out as disconnected strands — the topic edges
// are what give the graph its actual community structure for Louvain (F-07)
// to find.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

// Cluster is one Louvain community, summarised for the legend: a stable id, a
// human label (its dominant tag), and its size.
type Cluster struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
	Size  int    `json:"size"`
}

// ClampGraphLimit turns a raw ?limit= query value into a bounded int — a
// domain rule about what a reasonable graph request looks like, not HTTP
// parsing itself, so it lives here rather than in the controller.
func ClampGraphLimit(raw string) int {
	if raw == "" {
		return defaultNodeLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultNodeLimit
	}
	if n > maxNodeLimit {
		return maxNodeLimit
	}
	return n
}

// GraphResult bundles everything a graph request produces — kept in models/
// alongside Node/Edge/Cluster since views/graph.go's GraphView is a thin
// wrapper around exactly this.
type GraphResult struct {
	Nodes     []Node
	Edges     []Edge
	Clusters  []Cluster
	Truncated bool
}

// FetchGraph implements FR-01 (one journalist's graph, when journalistID is
// set) and the platform-wide epistemic graph / NFR-11 (when journalistID is
// ""). Nodes are sized by readership (FR-12), edges carry both lineage
// (FR-2) and co-tag structure, and the result is decorated with the two
// things Neo4j doesn't hold: active appeals and journalist display names
// (both Postgres-only).
func FetchGraph(ctx context.Context, driver neo4j.DriverWithContext, db *pgxpool.Pool, journalistID string, limit int) (GraphResult, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	// Scoped to one journalist, or the whole corpus when journalistId is "".
	// Written as one query with a conditional predicate rather than two
	// near-identical queries so the node shape can't drift between the two
	// views.
	const cypher = `
		MATCH (j:Journalist)-[:AUTHORED]->(a:Article)
		WHERE $journalistId = '' OR j.id = $journalistId
		WITH j, a
		ORDER BY a.readershipVolume DESC
		LIMIT $limit
		OPTIONAL MATCH (a)-[:HAS_TAG]->(t:Tag)
		RETURN a.id AS id,
		       a.title AS title,
		       j.id AS journalistId,
		       a.readershipVolume AS readershipVolume,
		       a.corruptionFactor AS corruptionFactor,
		       a.clusterId AS clusterId,
		       a.isRetracted AS isRetracted,
		       a.createdAt AS createdAt,
		       collect(DISTINCT t.name) AS tags
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, cypher, map[string]any{
			"journalistId": journalistID,
			"limit":        limit,
		})
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})
	if err != nil {
		return GraphResult{}, err
	}

	records, _ := result.([]*neo4j.Record)
	res := GraphResult{Nodes: []Node{}, Edges: []Edge{}, Clusters: []Cluster{}}
	included := make(map[string]bool, len(records))

	for _, rec := range records {
		node := Node{
			ID:               recString(rec, "id"),
			Title:            recString(rec, "title"),
			JournalistID:     recString(rec, "journalistId"),
			ReadershipVolume: recInt64(rec, "readershipVolume"),
			CorruptionFactor: recFloat64(rec, "corruptionFactor"),
			IsRetracted:      recBool(rec, "isRetracted"),
			CreatedAt:        recString(rec, "createdAt"),
			Tags:             recStrings(rec, "tags"),
		}
		if clusterID, ok := rec.Get("clusterId"); ok && clusterID != nil {
			v := toInt64(clusterID)
			node.ClusterID = &v
		}
		res.Nodes = append(res.Nodes, node)
		included[node.ID] = true
	}
	res.Truncated = len(records) >= limit

	// Edges are fetched separately and filtered to the node set above, so a
	// truncated graph can never contain an edge pointing at a node the client
	// didn't receive (Sigma throws on a dangling edge target).
	edges, err := fetchGraphEdges(ctx, session, journalistID)
	if err != nil {
		return GraphResult{}, err
	}
	for _, e := range edges {
		if included[e.Source] && included[e.Target] {
			res.Edges = append(res.Edges, e)
		}
	}

	decorateGraph(ctx, db, &res, journalistID)

	return res, nil
}

func fetchGraphEdges(ctx context.Context, session neo4j.SessionWithContext, journalistID string) ([]Edge, error) {
	// Lineage edges (FR-2) and co-tag edges in one round trip. The co-tag half
	// uses id(a) < id(b) to emit each undirected pair exactly once instead of
	// twice, and caps the per-tag fan-out: a tag applied to 500 articles would
	// otherwise generate ~125,000 edges on its own and drown the layout.
	const cypher = `
		MATCH (j:Journalist)-[:AUTHORED]->(a:Article)-[:SEQUENCE_OF]->(b:Article)
		WHERE $journalistId = '' OR j.id = $journalistId
		RETURN a.id AS source, b.id AS target, 'sequence' AS kind

		UNION

		MATCH (t:Tag)<-[:HAS_TAG]-(a:Article)
		WHERE $journalistId = '' OR (:Journalist {id: $journalistId})-[:AUTHORED]->(a)
		WITH t, collect(a)[..40] AS arts
		UNWIND arts AS a
		UNWIND arts AS b
		WITH a, b WHERE elementId(a) < elementId(b)
		RETURN a.id AS source, b.id AS target, 'topic' AS kind
	`

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, cypher, map[string]any{"journalistId": journalistID})
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})
	if err != nil {
		return nil, err
	}

	records, _ := result.([]*neo4j.Record)
	edges := make([]Edge, 0, len(records))
	for _, rec := range records {
		edges = append(edges, Edge{
			Source: recString(rec, "source"),
			Target: recString(rec, "target"),
			Kind:   recString(rec, "kind"),
		})
	}
	return edges, nil
}

// decorateGraph merges in the two things Neo4j doesn't hold: active appeals
// and journalist display names (both Postgres-only), and derives the cluster
// legend. Appeals are deliberately not mirrored into the graph via CDC —
// it's one boolean that changes on a different cadence to everything else.
func decorateGraph(ctx context.Context, db *pgxpool.Pool, res *GraphResult, journalistID string) {
	if db != nil && len(res.Nodes) > 0 {
		ids := make([]string, 0, len(res.Nodes))
		for _, n := range res.Nodes {
			ids = append(ids, n.ID)
		}

		activeAppeals := make(map[string]bool)
		if rows, err := db.Query(ctx,
			`SELECT article_id FROM appeals WHERE status = 'active' AND article_id = ANY($1)`, ids,
		); err == nil {
			for rows.Next() {
				var articleID string
				if rows.Scan(&articleID) == nil {
					activeAppeals[articleID] = true
				}
			}
			rows.Close()
		}

		names := make(map[string]string)
		if rows, err := db.Query(ctx,
			`SELECT id, display_name FROM users WHERE id = ANY(
			   SELECT DISTINCT journalist_id FROM articles WHERE id = ANY($1))`, ids,
		); err == nil {
			for rows.Next() {
				var id, name string
				if rows.Scan(&id, &name) == nil {
					names[id] = name
				}
			}
			rows.Close()
		}

		for i := range res.Nodes {
			res.Nodes[i].HasActiveAppeal = activeAppeals[res.Nodes[i].ID]
			res.Nodes[i].JournalistName = names[res.Nodes[i].JournalistID]
		}
	}

	res.Clusters = summariseClusters(res.Nodes)

	labels := make(map[int64]string, len(res.Clusters))
	for _, c := range res.Clusters {
		labels[c.ID] = c.Label
	}
	for i := range res.Nodes {
		if res.Nodes[i].ClusterID != nil {
			res.Nodes[i].ClusterLabel = labels[*res.Nodes[i].ClusterID]
		}
	}
}

// summariseClusters names each Louvain community after the tag that appears
// most often inside it, which is what turns an anonymous "Cluster 7" into the
// readable topic label the reference visualisation puts on the canvas. Ties
// break alphabetically so a label doesn't flicker between renders of the same
// data.
func summariseClusters(nodes []Node) []Cluster {
	type acc struct {
		size int
		tags map[string]int
	}
	byCluster := map[int64]*acc{}

	for _, n := range nodes {
		if n.ClusterID == nil {
			continue
		}
		a, ok := byCluster[*n.ClusterID]
		if !ok {
			a = &acc{tags: map[string]int{}}
			byCluster[*n.ClusterID] = a
		}
		a.size++
		for _, t := range n.Tags {
			a.tags[t]++
		}
	}

	clusters := make([]Cluster, 0, len(byCluster))
	for id, a := range byCluster {
		clusters = append(clusters, Cluster{ID: id, Label: dominantTag(a.tags, id), Size: a.size})
	}
	// Largest first: the legend and the on-canvas labels both want the
	// significant communities at the top.
	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Size != clusters[j].Size {
			return clusters[i].Size > clusters[j].Size
		}
		return clusters[i].ID < clusters[j].ID
	})
	return clusters
}

func dominantTag(tags map[string]int, clusterID int64) string {
	best, bestCount := "", 0
	for tag, count := range tags {
		if count > bestCount || (count == bestCount && tag < best) {
			best, bestCount = tag, count
		}
	}
	if best == "" {
		return "Cluster " + strconv.FormatInt(clusterID, 10)
	}
	return best
}

func recString(rec *neo4j.Record, key string) string {
	v, _ := rec.Get(key)
	s, _ := v.(string)
	return s
}

func recBool(rec *neo4j.Record, key string) bool {
	v, _ := rec.Get(key)
	b, _ := v.(bool)
	return b
}

func recInt64(rec *neo4j.Record, key string) int64 {
	v, _ := rec.Get(key)
	return toInt64(v)
}

func recFloat64(rec *neo4j.Record, key string) float64 {
	v, _ := rec.Get(key)
	return toFloat64(v)
}

func recStrings(rec *neo4j.Record, key string) []string {
	v, _ := rec.Get(key)
	raw, ok := v.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	default:
		return 0
	}
}
