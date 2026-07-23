package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

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
			// Debezium's Postgres connector encodes a `timestamptz` column as
			// io.debezium.time.ZonedTimestamp, which the default JSON
			// converter renders as an ISO-8601 string (e.g.
			// "2024-01-15T10:30:00Z") — not epoch millis/micros, which is
			// what timestamp-without-timezone columns get instead.
			CreatedAt string `json:"created_at"`
		} `json:"after"`
	} `json:"payload"`
}

// claimEnvelope mirrors the subset of the public.claims CDC payload the graph
// needs: a claim's category tag becomes a (:Article)-[:HAS_TAG]->(:Tag)
// relationship. Section 5.1 of the SRS lists HAS_TAG alongside SEQUENCE_OF as
// the relationships Neo4j governs, but nothing in the pipeline ever created
// one — the Cypher for it sat unused in packages/database/neo4j/queries.ts.
// Without these the only edges in the graph are lineage edges, which form a
// forest with essentially no community structure for Louvain (F-07) to find.
type claimEnvelope struct {
	Payload struct {
		After *struct {
			ID        string `json:"id"`
			ArticleID string `json:"article_id"`
			Tag       string `json:"tag"`
		} `json:"after"`
	} `json:"payload"`
}

// RunArticleSync consumes "ngj.public.articles" and mirrors each change into
// Neo4j, keeping the graph self-sufficient for reads: the graph API never has
// to fan out to Postgres to render a node.
func RunArticleSync(ctx context.Context, brokers []string, driver neo4j.DriverWithContext) {
	runSync(ctx, brokers, "ngj.public.articles", "ngj-graph-sync", func(value []byte) {
		var envelope debeziumEnvelope
		if err := json.Unmarshal(value, &envelope); err != nil {
			log.Printf("skipping malformed article CDC message: %v", err)
			return
		}
		after := envelope.Payload.After
		if after == nil {
			return // delete event; retractions are tombstone updates, not deletes
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
			createdAt:        after.CreatedAt,
		}); err != nil {
			log.Printf("neo4j sync error for article %s: %v", after.ID, err)
		}
	})
}

// RunClaimSync consumes "ngj.public.claims" and mirrors each claim's category
// tag into Neo4j as a HAS_TAG relationship.
func RunClaimSync(ctx context.Context, brokers []string, driver neo4j.DriverWithContext) {
	runSync(ctx, brokers, "ngj.public.claims", "ngj-tag-sync", func(value []byte) {
		var envelope claimEnvelope
		if err := json.Unmarshal(value, &envelope); err != nil {
			log.Printf("skipping malformed claim CDC message: %v", err)
			return
		}
		after := envelope.Payload.After
		if after == nil || after.Tag == "" || after.ArticleID == "" {
			return
		}
		if err := upsertArticleTag(ctx, driver, after.ArticleID, after.Tag); err != nil {
			log.Printf("neo4j tag sync error for article %s: %v", after.ArticleID, err)
		}
	})
}

// runSync is the shared consume loop. It retries with capped exponential
// backoff rather than returning on the first read error — the previous
// implementation gave up permanently the moment Kafka wasn't reachable, which
// on a cold `docker compose up` is all but guaranteed (the backend wins the
// race against Kafka's startup). The result was a single logged
// "kafka read error" followed by silence, with nothing ever reaching Neo4j
// again for the lifetime of the process. NFR-8 asks for exactly the opposite:
// a fault-tolerant pipeline that replays rather than drops.
func runSync(ctx context.Context, brokers []string, topic, groupID string, handle func([]byte)) {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
	})
	defer reader.Close()

	const maxBackoff = 30 * time.Second
	backoff := time.Second

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // shutting down
			}
			log.Printf("kafka read error on %s (retrying in %s): %v", topic, backoff, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < maxBackoff {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
		handle(msg.Value)
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
	createdAt        string
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
			    a.isRetracted = $isRetracted,
			    a.createdAt = coalesce($createdAt, a.createdAt)
			WITH a
			MERGE (j:Journalist {id: $journalistId})
			MERGE (j)-[:AUTHORED]->(a)
		`, map[string]any{
			"id": n.id, "title": n.title, "readership": n.readershipVolume,
			"corruptionFactor": n.corruptionFactor, "isRetracted": n.isRetracted,
			"journalistId": n.journalistID, "createdAt": nullIfEmpty(n.createdAt),
		}); err != nil {
			return nil, err
		}

		// Reconcile the lineage edge rather than only ever adding one. An
		// article has at most one parent in Postgres, so it must have at most
		// one outgoing SEQUENCE_OF here. MERGE alone is add-only: if an
		// article's parent_article_id is changed or cleared, the previous
		// edge survives forever and the graph accumulates lineage that no
		// longer exists. Deleting the child's existing outgoing edge first
		// makes this an idempotent mirror of the current row instead.
		if _, err := tx.Run(ctx, `
			MATCH (child:Article {id: $childId})-[r:SEQUENCE_OF]->()
			DELETE r
		`, map[string]any{"childId": n.id}); err != nil {
			return nil, err
		}

		if n.parentID != nil {
			// MERGE the parent rather than MATCH it: CDC gives no ordering
			// guarantee between a child and its parent, so a child can arrive
			// first. MATCH would silently drop the lineage edge in that case
			// (FR-2/F-02), and nothing would ever recreate it. The stub node
			// gets filled in when the parent's own event lands.
			if _, err := tx.Run(ctx, `
				MATCH (child:Article {id: $childId})
				MERGE (parent:Article {id: $parentId})
				MERGE (child)-[:SEQUENCE_OF]->(parent)
			`, map[string]any{"childId": n.id, "parentId": *n.parentID}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

func upsertArticleTag(ctx context.Context, driver neo4j.DriverWithContext, articleID, tag string) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return nil, runAndDiscard(ctx, tx, `
			MERGE (a:Article {id: $articleId})
			MERGE (t:Tag {name: $tag})
			MERGE (a)-[:HAS_TAG]->(t)
		`, map[string]any{"articleId": articleID, "tag": tag})
	})
	return err
}

func runAndDiscard(ctx context.Context, tx neo4j.ManagedTransaction, cypher string, params map[string]any) error {
	_, err := tx.Run(ctx, cypher, params)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
