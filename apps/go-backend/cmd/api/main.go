package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/auth"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/config"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/db"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/kafka"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/redisstore"
	"github.com/yeanur-ys/nextGENjournalism/apps/go-backend/internal/server"
)

func main() {
	cfg := config.Load()

	pool, err := db.NewPostgresPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres connection failed: %v", err)
	}
	defer pool.Close()

	neo4jDriver, err := db.NewNeo4jDriver(cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPass)
	if err != nil {
		log.Fatalf("neo4j driver init failed: %v", err)
	}
	defer neo4jDriver.Close(context.Background())

	redisClient, err := redisstore.NewClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	defer redisClient.Close()

	// CDC-sync: Debezium -> Kafka -> Neo4j (Section 5.2, "Data Synchronization Layer").
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	go kafka.RunArticleSync(context.Background(), brokers, neo4jDriver)

	tokens := auth.NewTokenService(cfg.JWTSecret)
	handler := server.NewRouter(server.Deps{
		DB:     pool,
		Redis:  redisClient,
		Neo4j:  neo4jDriver,
		Tokens: tokens,
	})

	log.Printf("go-backend listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatal(err)
	}
}
