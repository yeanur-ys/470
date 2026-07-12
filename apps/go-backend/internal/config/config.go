package config

import "os"

type Config struct {
	Port         string
	DatabaseURL  string
	RedisURL     string
	Neo4jURI     string
	Neo4jUser    string
	Neo4jPass    string
	KafkaBrokers string
	JWTSecret    string
}

func Load() Config {
	return Config{
		Port:         getenv("PORT", "8080"),
		DatabaseURL:  getenv("DATABASE_URL", "postgres://ngj:ngj_dev_password@localhost:5432/nextgenjournalism"),
		RedisURL:     getenv("REDIS_URL", "redis://localhost:6379"),
		Neo4jURI:     getenv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:    getenv("NEO4J_USER", "neo4j"),
		Neo4jPass:    getenv("NEO4J_PASSWORD", "ngj_dev_password"),
		KafkaBrokers: getenv("KAFKA_BROKERS", "localhost:29092"),
		JWTSecret:    getenv("JWT_SECRET", "dev-secret-change-me"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
