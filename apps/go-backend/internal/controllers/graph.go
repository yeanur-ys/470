package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/models"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/views"
)

type GraphController struct {
	Driver neo4j.DriverWithContext
	DB     *pgxpool.Pool
}

func NewGraphController(driver neo4j.DriverWithContext, db *pgxpool.Pool) *GraphController {
	return &GraphController{Driver: driver, DB: db}
}

// JournalistGraph implements FR-01: one journalist's own directed article
// graph, for their public profile page.
func (c *GraphController) JournalistGraph(w http.ResponseWriter, r *http.Request) {
	journalistID := r.PathValue("journalistId")
	c.respondGraph(w, r, journalistID)
}

// GlobalGraph implements the platform-wide epistemic graph (Section 2.2,
// "WebGL Epistemic Graphs" / NFR-11).
func (c *GraphController) GlobalGraph(w http.ResponseWriter, r *http.Request) {
	c.respondGraph(w, r, "")
}

func (c *GraphController) respondGraph(w http.ResponseWriter, r *http.Request, journalistID string) {
	limit := models.ClampGraphLimit(r.URL.Query().Get("limit"))

	result, err := models.FetchGraph(r.Context(), c.Driver, c.DB, journalistID, limit)
	if err != nil {
		http.Error(w, "failed to load graph", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(views.NewGraphView(result))
}
