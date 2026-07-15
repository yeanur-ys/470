package graph

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Handler struct {
	Driver neo4j.DriverWithContext
}

func NewHandler(driver neo4j.DriverWithContext) *Handler {
	return &Handler{Driver: driver}
}

type Node struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	ReadershipVolume int64   `json:"readershipVolume"`
	CorruptionFactor float64 `json:"corruptionFactor"`
	ClusterID        *int64  `json:"clusterId,omitempty"`
	IsRetracted      bool    `json:"isRetracted"`
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
// straight from Neo4j — Postgres is never queried here.
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
			       a.isRetracted AS isRetracted, parent.id AS parentId
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

	records, _ := result.([]*neo4j.Record)
	resp := graphResponse{Nodes: []Node{}, Edges: []Edge{}}

	for _, rec := range records {
		id, _ := rec.Get("id")
		title, _ := rec.Get("title")
		readership, _ := rec.Get("readershipVolume")
		corruption, _ := rec.Get("corruptionFactor")
		isRetracted, _ := rec.Get("isRetracted")
		parentID, hasParent := rec.Get("parentId")

		node := Node{
			ID:               toString(id),
			Title:            toString(title),
			ReadershipVolume: toInt64(readership),
			CorruptionFactor: toFloat64(corruption),
			IsRetracted:      isRetracted == true,
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
