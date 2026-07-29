package views

import "github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/models"

// GraphView is the JSON response envelope for both the single-journalist and
// platform-wide graph endpoints — a thin wrapper around models.GraphResult,
// since Node/Edge/Cluster themselves are already response-shaped.
type GraphView struct {
	Nodes     []models.Node    `json:"nodes"`
	Edges     []models.Edge    `json:"edges"`
	Clusters  []models.Cluster `json:"clusters"`
	Truncated bool             `json:"truncated"`
}

func NewGraphView(r models.GraphResult) GraphView {
	return GraphView{Nodes: r.Nodes, Edges: r.Edges, Clusters: r.Clusters, Truncated: r.Truncated}
}
