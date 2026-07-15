package db

import (
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func NewNeo4jDriver(uri, username, password string) (neo4j.DriverWithContext, error) {
	return neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
}
