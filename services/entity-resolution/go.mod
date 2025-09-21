module github.com/aegisshield/entity-resolution

go 1.21

require (
	github.com/aegisshield/shared v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	github.com/segmentio/kafka-go v0.4.47
	github.com/prometheus/client_golang v1.19.0
	google.golang.org/grpc v1.62.1
	google.golang.org/protobuf v1.33.0
	github.com/golang-migrate/migrate/v4 v4.17.0
	github.com/texttheater/golang-levenshtein/levenshtein v0.0.0-20200805054039-cae8b0eaed6c
	github.com/kljensen/snowball v0.6.0
	github.com/agnivade/levenshtein v1.1.1
	github.com/armon/go-radix v1.0.0
	github.com/bbalet/stopwords v1.0.0
	github.com/neo4j/neo4j-go-driver/v5 v5.17.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
)

replace github.com/aegisshield/shared => ../../shared