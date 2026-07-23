package config

import (
	"log"
	"os"
	"strings"
)

// DevJWTSecret is the fallback signing key used when JWT_SECRET is unset. It
// is deliberately named so that it is obvious in a config dump that no real
// secret is in play — anyone who reads this repo can forge a token signed
// with it, so Validate() refuses to let it run outside development.
const DevJWTSecret = "dev-secret-change-me"

type Config struct {
	Port             string
	DatabaseURL      string
	RedisURL         string
	Neo4jURI         string
	Neo4jUser        string
	Neo4jPass        string
	KafkaBrokers     string
	JWTSecret        string
	CORSAllowOrigins string
	Env              string
}

// Validate fails fast on configurations that are unsafe to serve real traffic
// with. Previously the API would start happily with the published development
// signing key and a wildcard CORS policy, and nothing anywhere said so.
func (c Config) Validate() error {
	if !c.IsProduction() {
		if c.JWTSecret == DevJWTSecret {
			log.Println("WARNING: using the built-in development JWT secret — anyone with this repo can forge a session token. Set JWT_SECRET before exposing this beyond localhost.")
		}
		return nil
	}

	var problems []string
	if c.JWTSecret == DevJWTSecret {
		problems = append(problems, "JWT_SECRET is still the published development default")
	}
	if len(c.JWTSecret) < 32 {
		problems = append(problems, "JWT_SECRET must be at least 32 characters")
	}
	if c.CORSAllowOrigins == "*" {
		problems = append(problems, `CORS_ALLOW_ORIGINS is "*", which allows any site to call this API with a user's credentials`)
	}
	if len(problems) > 0 {
		return errUnsafeConfig(problems)
	}
	return nil
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Env, "production")
}

func Load() Config {
	return Config{
		Port:             getenv("PORT", "8080"),
		DatabaseURL:      getenv("DATABASE_URL", "postgres://ngj:ngj_dev_password@localhost:5432/nextgenjournalism"),
		RedisURL:         getenv("REDIS_URL", "redis://localhost:6379"),
		Neo4jURI:         getenv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:        getenv("NEO4J_USER", "neo4j"),
		Neo4jPass:        getenv("NEO4J_PASSWORD", "ngj_dev_password"),
		KafkaBrokers:     getenv("KAFKA_BROKERS", "localhost:29092"),
		JWTSecret:        getenv("JWT_SECRET", DevJWTSecret),
		CORSAllowOrigins: getenv("CORS_ALLOW_ORIGINS", "http://localhost:3010"),
		Env:              getenv("APP_ENV", "development"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type unsafeConfigError struct{ problems []string }

func errUnsafeConfig(problems []string) error { return &unsafeConfigError{problems: problems} }

func (e *unsafeConfigError) Error() string {
	return "unsafe configuration for APP_ENV=production: " + strings.Join(e.problems, "; ")
}
