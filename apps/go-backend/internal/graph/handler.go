package graph

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Handler struct {
	Driver neo4j.DriverWithContext
	DB     *pgxpool.Pool
}

func NewHandler(driver neo4j.DriverWithContext, db *pgxpool.Pool) *Handler {
	return &Handler{Driver: driver, DB: db}
}

type Node struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	ReadershipVolume int64   `json:"readershipVolume"`
	CorruptionFactor float64 `json:"corruptionFactor"`
	ClusterID        *int64  `json:"clusterId,omitempty"`
	IsRetracted      bool    `json:"isRetracted"`
	HasActiveAppeal  bool    `json:"hasActiveAppeal"`     // FR-9/FR-15: pulsing amber "under dispute" state
	CreatedAt        string  `json:"createdAt,omitempty"` // F-08: time-period bucketing, computed client-side from this
}

type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type graphResponse struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// JournalistGraph implements the reader-facing side of Section 2.2 ("WebGL
// Epistemic Graphs") and FR-10/FR-12: every field the frontend needs to color
// and size nodes (corruptionFactor, readershipVolume) and to semantic-zoom
// cluster them (clusterId, written by the Python Louvain worker) comes
// straight from Neo4j. The one exception is hasActiveAppeal (FR-9): appeals
// are a Postgres-only concept (there's no reason to duplicate them into the
// graph via CDC just for this one boolean), so it's looked up there and
// merged in before responding.
func (h *Handler) JournalistGraph(w http.ResponseWriter, r *http.Request) {
	journalistID := r.PathValue("journalistId")
	ctx := context.Background()

	session := h.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, `
			MATCH (j:Journalist {id: $journalistId})-[:AUTHORED]->(a:Article)
			OPTIONAL MATCH (a)-[:SEQUENCE_OF]->(parent:Article)
			RETURN a.id AS id, a.title AS title, a.readershipVolume AS readershipVolume,
			       a.corruptionFactor AS corruptionFactor, a.clusterId AS clusterId,
			       a.isRetracted AS isRetracted, a.createdAt AS createdAt, parent.id AS parentId
			ORDER BY a.readershipVolume DESC
		`, map[string]any{"journalistId": journalistID})
		if err != nil {
			return nil, err
		}
		return records.Collect(ctx)
	})
	if err != nil {
		http.Error(w, "failed to load graph", http.StatusInternalServerError)
		return
	}

	activeAppeals := make(map[string]bool)
	if h.DB != nil {
		rows, err := h.DB.Query(ctx, `
			SELECT ap.article_id
			FROM appeals ap
			JOIN articles a ON a.id = ap.article_id
			WHERE a.journalist_id = $1 AND ap.status = 'active'
		`, journalistID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var articleID string
				if err := rows.Scan(&articleID); err == nil {
					activeAppeals[articleID] = true
				}
			}
		}
	}

	records, _ := result.([]*neo4j.Record)
	resp := graphResponse{Nodes: []Node{}, Edges: []Edge{}}

	for _, rec := range records {
		id, _ := rec.Get("id")
		title, _ := rec.Get("title")
		readership, _ := rec.Get("readershipVolume")
		corruption, _ := rec.Get("corruptionFactor")
		isRetracted, _ := rec.Get("isRetracted")
		createdAt, _ := rec.Get("createdAt")
		parentID, hasParent := rec.Get("parentId")

		nodeID := toString(id)
		node := Node{
			ID:               nodeID,
			Title:            toString(title),
			ReadershipVolume: toInt64(readership),
			CorruptionFactor: toFloat64(corruption),
			IsRetracted:      isRetracted == true,
			HasActiveAppeal:  activeAppeals[nodeID],
			CreatedAt:        toString(createdAt),
		}
		if clusterID, ok := rec.Get("clusterId"); ok && clusterID != nil {
			v := toInt64(clusterID)
			node.ClusterID = &v
		}
		resp.Nodes = append(resp.Nodes, node)

		if hasParent && parentID != nil {
			resp.Edges = append(resp.Edges, Edge{Source: node.ID, Target: toString(parentID)})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func toString(v any) string {
	s, _ := v.(string)
	return s
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
