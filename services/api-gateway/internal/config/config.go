package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port     int          `json:"port"`
	Auth     AuthConfig   `json:"auth"`
	CORS     CORSConfig   `json:"cors"`
	Services ServiceConfig `json:"services"`
	Database DatabaseConfig `json:"database"`
}

type AuthConfig struct {
	JWTSecret     string `json:"jwt_secret"`
	TokenDuration int    `json:"token_duration"` // in minutes
	Issuer        string `json:"issuer"`
}

type CORSConfig struct {
	AllowedOrigins []string `json:"allowed_origins"`
}

type ServiceConfig struct {
	DataIngestionURL   string `json:"data_ingestion_url"`
	EntityResolutionURL string `json:"entity_resolution_url"`
	AlertingEngineURL  string `json:"alerting_engine_url"`
	GraphEngineURL     string `json:"graph_engine_url"`
	AnalyticsURL       string `json:"analytics_url"`
}

type DatabaseConfig struct {
	PostgreSQLURL string `json:"postgresql_url"`
	Neo4jURL      string `json:"neo4j_url"`
	Neo4jUser     string `json:"neo4j_user"`
	Neo4jPassword string `json:"neo4j_password"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Port: getEnvAsInt("PORT", 8080),
		Auth: AuthConfig{
			JWTSecret:     getEnv("JWT_SECRET", "aegisshield-secret-key"),
			TokenDuration: getEnvAsInt("JWT_TOKEN_DURATION", 60),
			Issuer:        getEnv("JWT_ISSUER", "aegisshield"),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:3001"}),
		},
		Services: ServiceConfig{
			DataIngestionURL:   getEnv("DATA_INGESTION_URL", "localhost:50051"),
			EntityResolutionURL: getEnv("ENTITY_RESOLUTION_URL", "localhost:50052"),
			AlertingEngineURL:  getEnv("ALERTING_ENGINE_URL", "localhost:50053"),
			GraphEngineURL:     getEnv("GRAPH_ENGINE_URL", "localhost:50054"),
			AnalyticsURL:       getEnv("ANALYTICS_URL", "localhost:50055"),
		},
		Database: DatabaseConfig{
			PostgreSQLURL: getEnv("POSTGRESQL_URL", "postgres://aegisshield:password@localhost:5432/aegisshield?sslmode=disable"),
			Neo4jURL:      getEnv("NEO4J_URL", "bolt://localhost:7687"),
			Neo4jUser:     getEnv("NEO4J_USER", "neo4j"),
			Neo4jPassword: getEnv("NEO4J_PASSWORD", "password"),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}