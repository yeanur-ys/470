module github.com/yeanur-ys/nextGENjournalism/apps/go-backend

go 1.22

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.6.0
	github.com/neo4j/neo4j-go-driver/v5 v5.23.0
	github.com/redis/go-redis/v9 v9.6.1
	github.com/segmentio/kafka-go v0.4.47
	golang.org/x/crypto v0.24.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)

replace gopkg.in/yaml.v3 => github.com/go-yaml/yaml v0.0.0-20220521103104-8f96da9f5d5e

replace golang.org/x/net => github.com/golang/net v0.17.0

replace golang.org/x/crypto => github.com/golang/crypto v0.24.0

replace golang.org/x/text => github.com/golang/text v0.14.0

replace golang.org/x/sys => github.com/golang/sys v0.17.0

replace golang.org/x/sync => github.com/golang/sync v0.7.0

replace golang.org/x/term => github.com/golang/term v0.17.0

replace gopkg.in/check.v1 => github.com/go-check/check v0.0.0-20200902074654-038fdea0a05b

replace golang.org/x/tools => github.com/golang/tools v0.6.0

replace golang.org/x/mod => github.com/golang/mod v0.15.0
