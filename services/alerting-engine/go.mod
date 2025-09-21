module github.com/aegisshield/alerting-engine

go 1.21

require (
	// Core dependencies
	github.com/aegisshield/shared v0.0.0
	google.golang.org/grpc v1.58.0
	google.golang.org/protobuf v1.31.0

	// Database
	github.com/lib/pq v1.10.9
	github.com/jmoiron/sqlx v1.3.5
	github.com/golang-migrate/migrate/v4 v4.16.2

	// Kafka
	github.com/IBM/sarama v1.41.2

	// HTTP framework
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.5.0

	// Metrics and monitoring
	github.com/prometheus/client_golang v1.16.0

	// Configuration
	github.com/spf13/viper v1.16.0
	github.com/spf13/pflag v1.0.5

	// Logging
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9

	// Email and SMS
	github.com/sendgrid/sendgrid-go v3.12.0+incompatible
	github.com/twilio/twilio-go v1.7.2

	// Template engine
	github.com/flosch/pongo2/v6 v6.0.0

	// Caching
	github.com/go-redis/redis/v8 v8.11.5
	github.com/patrickmn/go-cache v2.1.0+incompatible

	// Rate limiting
	golang.org/x/time v0.3.0

	// Webhook and HTTP client
	github.com/go-resty/resty/v2 v2.7.0

	// JSON and data processing
	github.com/tidwall/gjson v1.15.0
	github.com/itchyny/gojq v0.12.12

	// Rule engine
	github.com/hyperjumptech/grule-rule-engine v1.15.0

	// Scheduler
	github.com/robfig/cron/v3 v3.0.1

	// Validation
	github.com/go-playground/validator/v10 v10.15.3

	// UUID generation
	github.com/google/uuid v1.3.1

	// Encryption
	golang.org/x/crypto v0.13.0

	// Context and utilities
	golang.org/x/net v0.15.0
	golang.org/x/sync v0.3.0
)

require (
	// Indirect dependencies (auto-managed)
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/eapache/go-resiliency v1.4.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Local module replacements for development
replace github.com/aegisshield/shared => ../shared