module github.com/aegisshield/analytics-dashboard

go 1.21

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/golang-migrate/migrate/v4 v4.16.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.1
	github.com/lib/pq v1.10.9
	github.com/prometheus/client_golang v1.19.1
	github.com/segmentio/kafka-go v0.4.47
	github.com/spf13/viper v1.18.2
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.63.2
	google.golang.org/protobuf v1.34.1
	gorm.io/driver/postgres v1.5.7
	gorm.io/gorm v1.25.10
)