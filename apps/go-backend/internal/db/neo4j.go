package db

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func NewNeo4jDriver(uri, username, password string) (neo4j.DriverWithContext, error) {
	return neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
}

// neo4jConstraints mirrors packages/database/neo4j/schema.cypher.
//
// That file already existed and declared exactly the right constraints — but
// nothing in the system ever executed it, so a live database had none at all.
// This matters beyond tidiness: Neo4j's MERGE is only atomic with respect to a
// uniqueness constraint. Without one, two concurrent transactions can both
// fail to find a node and both create it. The CDC pipeline runs two consumers
// (articles and claims) that each MERGE (:Article {id}), and this raced in
// practice — a 900-article sync produced 901 nodes, one article duplicated.
//
// Applying them at startup means the guarantee travels with the application
// rather than depending on someone remembering to pipe a .cypher file into
// cypher-shell. Every statement is IF NOT EXISTS, so this is idempotent and
// safe to run on every boot.
var neo4jConstraints = []string{
	`CREATE CONSTRAINT article_id_unique IF NOT EXISTS
	   FOR (a:Article) REQUIRE a.id IS UNIQUE`,
	`CREATE CONSTRAINT journalist_id_unique IF NOT EXISTS
	   FOR (j:Journalist) REQUIRE j.id IS UNIQUE`,
	`CREATE CONSTRAINT tag_name_unique IF NOT EXISTS
	   FOR (t:Tag) REQUIRE t.name IS UNIQUE`,
	`CREATE INDEX article_corruption_idx IF NOT EXISTS
	   FOR (a:Article) ON (a.corruptionFactor)`,
	`CREATE INDEX article_readership_idx IF NOT EXISTS
	   FOR (a:Article) ON (a.readershipVolume)`,
	`CREATE INDEX article_created_at_idx IF NOT EXISTS
	   FOR (a:Article) ON (a.createdAt)`,
	`CREATE INDEX article_cluster_idx IF NOT EXISTS
	   FOR (a:Article) ON (a.clusterId)`,
}

// EnsureNeo4jConstraints applies the graph schema. Each statement runs in its
// own session because Neo4j refuses schema and data operations in the same
// transaction.
func EnsureNeo4jConstraints(ctx context.Context, driver neo4j.DriverWithContext) error {
	for _, stmt := range neo4jConstraints {
		session := driver.NewSession(ctx, neo4j.SessionConfig{})
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx, stmt, nil)
			return nil, err
		})
		session.Close(ctx)
		if err != nil {
			return fmt.Errorf("applying graph schema: %w", err)
		}
	}
	return nil
}
