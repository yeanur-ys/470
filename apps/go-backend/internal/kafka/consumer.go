package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	kafkago "github.com/segmentio/kafka-go"
)

// debeziumEnvelope mirrors the subset of Debezium's Postgres connector payload
// (io.debezium.connector.postgresql.PostgresConnector) that the graph sync cares about.
type debeziumEnvelope struct {
	Payload struct {
		After *struct {
			ID               string  `json:"id"`
			JournalistID     string  `json:"journalist_id"`
			ParentArticleID  *string `json:"parent_article_id"`
			Title            string  `json:"title"`
			ReadershipVolume int64   `json:"readership_volume"`
			IsRetracted      bool    `json:"is_retracted"`
		} `json:"after"`
	} `json:"payload"`
}

// RunArticleSync blocks, consuming the "ngj.public.articles" topic produced by
// the Debezium connector (infra/debezium/register-postgres.json) and mirroring
// each change into Neo4j (packages/database/neo4j/schema.cypher node/edge layout).
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

		if err := upsertArticleNode(ctx, driver, after.ID, after.JournalistID, after.ParentArticleID, after.Title, after.ReadershipVolume, after.IsRetracted); err != nil {
			log.Printf("neo4j sync error for article %s: %v", after.ID, err)
		}
	}
}

func upsertArticleNode(ctx context.Context, driver neo4j.DriverWithContext, id, journalistID string, parentID *string, title string, readership int64, isRetracted bool) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		if _, err := tx.Run(ctx, `
			MERGE (a:Article {id: $id})
			SET a.title = $title, a.readershipVolume = $readership, a.isRetracted = $isRetracted
			WITH a
			MERGE (j:Journalist {id: $journalistId})
			MERGE (j)-[:AUTHORED]->(a)
		`, map[string]any{
			"id": id, "title": title, "readership": readership,
			"isRetracted": isRetracted, "journalistId": journalistID,
		}); err != nil {
			return nil, err
		}

		if parentID != nil {
			if _, err := tx.Run(ctx, `
				MATCH (child:Article {id: $childId})
				MATCH (parent:Article {id: $parentId})
				MERGE (child)-[:SEQUENCE_OF]->(parent)
			`, map[string]any{"childId": id, "parentId": *parentID}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}
