package graph

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// defaultNodeLimit caps how much of the graph one request can pull. NFR-11
// wants "thousands of historical nodes" browsable, and the client renders
// them on the GPU, but an unbounded query against a growing corpus is still a
// denial-of-service waiting to happen.
const (
	defaultNodeLimit = 2000
	maxNodeLimit     = 10000
)

type Handler struct {
	Driver neo4j.DriverWithContext
	DB     *pgxpool.Pool
}

func NewHandler(driver neo4j.DriverWithContext, db *pgxpool.Pool) *Handler {
	return &Handler{Driver: driver, DB: db}
}

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

type graphResponse struct {
	Nodes    []Node    `json:"nodes"`
	Edges    []Edge    `json:"edges"`
	Clusters []Cluster `json:"clusters"`
	Truncated bool     `json:"truncated"`
}

// JournalistGraph implements FR-01: one journalist's own directed article
// graph, for their public profile page.
func (h *Handler) JournalistGraph(w http.ResponseWriter, r *http.Request) {
	journalistID := r.PathValue("journalistId")
	h.respondGraph(w, r, journalistID)
}

// GlobalGraph implements the platform-wide epistemic graph (Section 2.2,
// "WebGL Epistemic Graphs" / NFR-11): every article by every journalist, with
// both lineage and topic edges. This is the view that actually exercises
// Louvain clustering and semantic zooming, since a single journalist rarely
// has enough nodes for either to matter.
func (h *Handler) GlobalGraph(w http.ResponseWriter, r *http.Request) {
	h.respondGraph(w, r, "")
}

func (h *Handler) respondGraph(w http.ResponseWriter, r *http.Request, journalistID string) {
	ctx := r.Context()
	limit := parseLimit(r.URL.Query().Get("limit"))

	session := h.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
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
		http.Error(w, "failed to load graph", http.StatusInternalServerError)
		return
	}

	records, _ := result.([]*neo4j.Record)
	resp := graphResponse{Nodes: []Node{}, Edges: []Edge{}, Clusters: []Cluster{}}
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
		resp.Nodes = append(resp.Nodes, node)
		included[node.ID] = true
	}
	resp.Truncated = len(records) >= limit

	// Edges are fetched separately and filtered to the node set above, so a
	// truncated graph can never contain an edge pointing at a node the client
	// didn't receive (Sigma throws on a dangling edge target).
	edges, err := h.fetchEdges(ctx, session, journalistID)
	if err != nil {
		http.Error(w, "failed to load graph edges", http.StatusInternalServerError)
		return
	}
	for _, e := range edges {
		if included[e.Source] && included[e.Target] {
			resp.Edges = append(resp.Edges, e)
		}
	}

	h.decorate(ctx, &resp, journalistID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) fetchEdges(ctx context.Context, session neo4j.SessionWithContext, journalistID string) ([]Edge, error) {
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

// decorate merges in the two things Neo4j doesn't hold: active appeals and
// journalist display names (both Postgres-only), and derives the cluster
// legend. Appeals are deliberately not mirrored into the graph via CDC — it's
// one boolean that changes on a different cadence to everything else.
func (h *Handler) decorate(ctx context.Context, resp *graphResponse, journalistID string) {
	if h.DB != nil && len(resp.Nodes) > 0 {
		ids := make([]string, 0, len(resp.Nodes))
		for _, n := range resp.Nodes {
			ids = append(ids, n.ID)
		}

		activeAppeals := make(map[string]bool)
		if rows, err := h.DB.Query(ctx,
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
		if rows, err := h.DB.Query(ctx,
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

		for i := range resp.Nodes {
			resp.Nodes[i].HasActiveAppeal = activeAppeals[resp.Nodes[i].ID]
			resp.Nodes[i].JournalistName = names[resp.Nodes[i].JournalistID]
		}
	}

	resp.Clusters = summariseClusters(resp.Nodes)

	labels := make(map[int64]string, len(resp.Clusters))
	for _, c := range resp.Clusters {
		labels[c.ID] = c.Label
	}
	for i := range resp.Nodes {
		if resp.Nodes[i].ClusterID != nil {
			resp.Nodes[i].ClusterLabel = labels[*resp.Nodes[i].ClusterID]
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

func parseLimit(raw string) int {
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
