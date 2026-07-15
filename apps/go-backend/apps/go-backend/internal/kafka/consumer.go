package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	kafkago "github.com/segmentio/kafka-go"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/ranking"
)

// debeziumEnvelope mirrors the subset of Debezium's Postgres connector payload
// (io.debezium.connector.postgresql.PostgresConnector) that the graph sync cares about.
type debeziumEnvelope struct {
	Payload struct {
		After *struct {
			ID                  string  `json:"id"`
			JournalistID        string  `json:"journalist_id"`
			ParentArticleID     *string `json:"parent_article_id"`
			Title               string  `json:"title"`
			ReadershipVolume    int64   `json:"readership_volume"`
			VerifiedClaims      int64   `json:"verified_claims"`
			SelfCorrectedClaims int64   `json:"self_corrected_claims"`
			FalseClaims         int64   `json:"false_claims"`
			IsRetracted         bool    `json:"is_retracted"`
		} `json:"after"`
	} `json:"payload"`
}

// RunArticleSync blocks, consuming the "ngj.public.articles" topic produced by
// the Debezium connector (infra/debezium/register-postgres.json) and mirroring
// each change into Neo4j (packages/database/neo4j/schema.cypher node/edge layout).
// This keeps Neo4j self-sufficient for reads: the graph API (internal/graph)
// never has to fan out to Postgres to render a node.
func RunArticleSync(ctx context.Context, brokers []string, driver neo4j.DriverWithContext) {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: brokers,
		Topic:   "ngj.public.articles",
		GroupID: "ngj-graph-sync",
	})
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("kafka read error: %v", err)
			return
		}

		var envelope debeziumEnvelope
		if err := json.Unmarshal(msg.Value, &envelope); err != nil {
			log.Printf("skipping malformed CDC message: %v", err)
			continue
		}
		after := envelope.Payload.After
		if after == nil {
			continue // delete event; retractions are handled as tombstone updates, not deletes
		}

		corruptionFactor := ranking.CorruptionFactor(
			float64(after.VerifiedClaims), float64(after.SelfCorrectedClaims), float64(after.FalseClaims),
		)

		if err := upsertArticleNode(ctx, driver, articleNode{
			id:               after.ID,
			journalistID:     after.JournalistID,
			parentID:         after.ParentArticleID,
			title:            after.Title,
			readershipVolume: after.ReadershipVolume,
			corruptionFactor: corruptionFactor,
			isRetracted:      after.IsRetracted,
		}); err != nil {
			log.Printf("neo4j sync error for article %s: %v", after.ID, err)
		}
	}
}

type articleNode struct {
	id               string
	journalistID     string
	parentID         *string
	title            string
	readershipVolume int64
	corruptionFactor float64
	isRetracted      bool
}

func upsertArticleNode(ctx context.Context, driver neo4j.DriverWithContext, n articleNode) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		if _, err := tx.Run(ctx, `
			MERGE (a:Article {id: $id})
			SET a.title = $title,
			    a.readershipVolume = $readership,
			    a.corruptionFactor = $corruptionFactor,
			    a.isRetracted = $isRetracted
			WITH a
			MERGE (j:Journalist {id: $journalistId})
			MERGE (j)-[:AUTHORED]->(a)
		`, map[string]any{
			"id": n.id, "title": n.title, "readership": n.readershipVolume,
			"corruptionFactor": n.corruptionFactor, "isRetracted": n.isRetracted,
			"journalistId": n.journalistID,
		}); err != nil {
			return nil, err
		}

		if n.parentID != nil {
			if _, err := tx.Run(ctx, `
				MATCH (child:Article {id: $childId})
				MATCH (parent:Article {id: $parentId})
				MERGE (child)-[:SEQUENCE_OF]->(parent)
			`, map[string]any{"childId": n.id, "parentId": *n.parentID}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}
