package test

import (
	"context"
	"testing"
	"time"

	"github.com/aegisshield/data-integration/internal/config"
	"github.com/aegisshield/data-integration/internal/etl"
	"github.com/aegisshield/data-integration/internal/validation"
	"github.com/aegisshield/data-integration/internal/quality"
	"github.com/aegisshield/data-integration/internal/lineage"
	"github.com/aegisshield/data-integration/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Unit tests for Data Integration components

func TestETLPipeline(t *testing.T) {
	logger := zap.NewNop()
	
	cfg := config.ETLConfig{
		WorkerPoolSize:    4,
		QueueSize:        100,
		BatchSize:        1000,
		ProcessingTimeout: 300,
		EnableMetrics:     true,
	}

	pipeline := etl.NewPipeline(cfg, logger)
	require.NotNil(t, pipeline)

	ctx := context.Background()

	t.Run("CreateJob", func(t *testing.T) {
		jobConfig := etl.JobConfig{
			Name: "test-job",
			Type: "batch",
			Source: etl.SourceConfig{
				Type: "database",
				Config: map[string]interface{}{
					"table": "test_table",
				},
			},
			Target: etl.TargetConfig{
				Type: "storage",
				Config: map[string]interface{}{
					"path": "/test/output",
				},
			},
		}

		job, err := pipeline.CreateJob(ctx, jobConfig)
		assert.NoError(t, err)
		assert.NotNil(t, job)
		assert.Equal(t, "test-job", job.Name)
		assert.Equal(t, "created", job.Status)
	})

	t.Run("ProcessData", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "name": "John", "email": "john@example.com"},
			{"id": 2, "name": "Jane", "email": "jane@example.com"},
		}

		options := etl.ProcessingOptions{
			BatchSize:    2,
			Validate:     true,
			QualityCheck: true,
		}

		result, err := pipeline.ProcessData(ctx, data, options)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.RecordsProcessed)
		assert.Equal(t, 0, result.RecordsFailed)
	})

	t.Run("GetMetrics", func(t *testing.T) {
		metrics := pipeline.GetMetrics()
		assert.NotNil(t, metrics)
		assert.GreaterOrEqual(t, metrics.JobsCreated, int64(1))
	})
}

func TestDataValidation(t *testing.T) {
	logger := zap.NewNop()
	
	cfg := config.ValidationConfig{
		EnableProfiling:   true,
		EnableBusinessRules: true,
		MaxErrors:         100,
	}

	validator := validation.NewValidator(cfg, logger)
	require.NotNil(t, validator)

	ctx := context.Background()

	t.Run("ValidateData", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "name": "John", "email": "john@example.com", "age": 30},
			{"id": 2, "name": "Jane", "email": "invalid-email", "age": -5},
		}

		rules := []validation.Rule{
			{
				Field:   "email",
				Type:    "pattern",
				Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			},
			{
				Field:    "age",
				Type:     "range",
				MinValue: 0,
				MaxValue: 150,
			},
		}

		result, err := validator.ValidateData(ctx, data, rules)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 2) // invalid email and negative age
	})

	t.Run("ProfileData", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "name": "John", "email": "john@example.com"},
			{"id": 2, "name": "Jane", "email": "jane@example.com"},
			{"id": 3, "name": "", "email": "bob@example.com"},
		}

		profile, err := validator.ProfileData(ctx, data)
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, 3, profile.TotalRecords)
		assert.Equal(t, 3, profile.TotalFields)
		assert.Contains(t, profile.FieldProfiles, "name")
		assert.Contains(t, profile.FieldProfiles, "email")
		
		emailProfile := profile.FieldProfiles["email"]
		assert.Equal(t, 1.0, emailProfile.Completeness)
		assert.Equal(t, 1.0, emailProfile.Uniqueness)
	})

	t.Run("CreateRule", func(t *testing.T) {
		rule := validation.Rule{
			Name:    "email-validation",
			Field:   "email",
			Type:    "pattern",
			Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		}

		createdRule, err := validator.CreateRule(ctx, rule)
		assert.NoError(t, err)
		assert.NotNil(t, createdRule)
		assert.NotEmpty(t, createdRule.ID)
		assert.Equal(t, "email-validation", createdRule.Name)
	})
}

func TestDataQuality(t *testing.T) {
	logger := zap.NewNop()
	
	cfg := config.QualityConfig{
		Thresholds: config.QualityThresholds{
			Completeness: 0.95,
			Accuracy:     0.90,
			Consistency:  0.85,
			Validity:     0.90,
			Uniqueness:   0.95,
			Freshness:    0.80,
		},
		EnableRecommendations: true,
	}

	checker := quality.NewChecker(cfg, logger)
	require.NotNil(t, checker)

	ctx := context.Background()

	t.Run("CheckQuality", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "name": "John", "email": "john@example.com", "created_at": "2024-01-01T10:00:00Z"},
			{"id": 2, "name": "Jane", "email": "jane@example.com", "created_at": "2024-01-01T11:00:00Z"},
			{"id": 3, "name": "", "email": "invalid-email", "created_at": "2023-01-01T10:00:00Z"},
		}

		dimensions := []string{"completeness", "accuracy", "freshness"}
		
		result, err := checker.CheckQuality(ctx, data, dimensions)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.OverallScore, 0.0)
		assert.Less(t, result.OverallScore, 1.0)
		assert.Contains(t, result.Dimensions, "completeness")
		assert.Contains(t, result.Dimensions, "accuracy")
		assert.Contains(t, result.Dimensions, "freshness")
	})

	t.Run("DetectIssues", func(t *testing.T) {
		data := []map[string]interface{}{
			{"id": 1, "email": "john@example.com"},
			{"id": 1, "email": "duplicate@example.com"}, // duplicate ID
			{"id": 3, "email": "invalid-email"},         // invalid email
		}

		issues, err := checker.DetectIssues(ctx, data)
		assert.NoError(t, err)
		assert.NotEmpty(t, issues)
		
		// Should detect duplicate and invalid email
		duplicateFound := false
		invalidEmailFound := false
		for _, issue := range issues {
			if issue.Type == "duplicate" {
				duplicateFound = true
			}
			if issue.Type == "pattern_mismatch" {
				invalidEmailFound = true
			}
		}
		assert.True(t, duplicateFound)
		assert.True(t, invalidEmailFound)
	})

	t.Run("GenerateRecommendations", func(t *testing.T) {
		issues := []quality.Issue{
			{
				Field:       "email",
				Type:        "pattern_mismatch",
				Severity:    "major",
				Count:       5,
				Description: "Invalid email format detected",
			},
		}

		recommendations := checker.GenerateRecommendations(ctx, issues)
		assert.NotEmpty(t, recommendations)
		assert.Contains(t, recommendations[0].Title, "Email")
	})
}

func TestLineageTracking(t *testing.T) {
	logger := zap.NewNop()
	
	cfg := config.LineageConfig{
		EnableSchemaTracking: true,
		MaxDepth:            10,
		EnableVisualization: true,
	}

	tracker := lineage.NewTracker(cfg, logger)
	require.NotNil(t, tracker)

	ctx := context.Background()

	t.Run("TrackLineage", func(t *testing.T) {
		entry := lineage.LineageEntry{
			Dataset:   "customer_data",
			Operation: "transform",
			Source:    []string{"raw_customer_data"},
			Target:    "processed_customer_data",
			Schema: map[string]interface{}{
				"fields": []string{"id", "name", "email"},
			},
			Transformations: []lineage.Transformation{
				{
					Type:        "rename",
					Source:      "customer_id",
					Target:      "id",
					Description: "Rename customer_id to id",
				},
			},
		}

		err := tracker.TrackLineage(ctx, entry)
		assert.NoError(t, err)
	})

	t.Run("GetDatasetLineage", func(t *testing.T) {
		lineageInfo, err := tracker.GetDatasetLineage(ctx, "customer_data", "both", 3)
		assert.NoError(t, err)
		assert.NotNil(t, lineageInfo)
		assert.Equal(t, "customer_data", lineageInfo.Dataset)
		assert.NotEmpty(t, lineageInfo.Graph.Nodes)
	})

	t.Run("GetUpstreamDatasets", func(t *testing.T) {
		upstream, err := tracker.GetUpstreamDatasets(ctx, "customer_data", 2)
		assert.NoError(t, err)
		assert.NotNil(t, upstream)
	})

	t.Run("GetDownstreamDatasets", func(t *testing.T) {
		downstream, err := tracker.GetDownstreamDatasets(ctx, "raw_customer_data", 2)
		assert.NoError(t, err)
		assert.NotNil(t, downstream)
	})

	t.Run("TrackSchemaEvolution", func(t *testing.T) {
		change := lineage.SchemaChange{
			Dataset:    "customer_data",
			ChangeType: "field_added",
			Field:      "phone",
			OldSchema:  map[string]interface{}{"fields": []string{"id", "name", "email"}},
			NewSchema:  map[string]interface{}{"fields": []string{"id", "name", "email", "phone"}},
			Timestamp:  time.Now(),
		}

		err := tracker.TrackSchemaEvolution(ctx, change)
		assert.NoError(t, err)
	})
}

func TestStorageManager(t *testing.T) {
	logger := zap.NewNop()
	
	cfg := config.StorageConfig{
		DefaultProvider: "local",
		Providers: map[string]config.StorageProvider{
			"local": {
				Type: "local",
				Config: map[string]interface{}{
					"base_path": "/tmp/aegis-test",
				},
			},
		},
		EnableEncryption: false,
		EnableMetadata:   true,
	}

	manager := storage.NewManager(cfg, logger)
	require.NotNil(t, manager)

	ctx := context.Background()

	t.Run("Store", func(t *testing.T) {
		data := []byte("test data content")
		path := "/test/file1.txt"
		
		metadata := map[string]interface{}{
			"source": "test",
			"type":   "text",
		}

		result, err := manager.Store(ctx, path, data, metadata)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, path, result.Path)
		assert.Equal(t, int64(len(data)), result.Size)
	})

	t.Run("Retrieve", func(t *testing.T) {
		path := "/test/file1.txt"
		
		data, metadata, err := manager.Retrieve(ctx, path)
		assert.NoError(t, err)
		assert.Equal(t, []byte("test data content"), data)
		assert.Contains(t, metadata, "source")
		assert.Equal(t, "test", metadata["source"])
	})

	t.Run("List", func(t *testing.T) {
		prefix := "/test/"
		
		objects, err := manager.List(ctx, prefix, 10, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, objects)
		assert.Equal(t, "/test/file1.txt", objects[0].Path)
	})

	t.Run("GetMetadata", func(t *testing.T) {
		path := "/test/file1.txt"
		
		metadata, err := manager.GetMetadata(ctx, path)
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Contains(t, metadata.CustomMetadata, "source")
	})

	t.Run("Delete", func(t *testing.T) {
		path := "/test/file1.txt"
		
		err := manager.Delete(ctx, path)
		assert.NoError(t, err)
		
		// Verify file is deleted
		_, _, err = manager.Retrieve(ctx, path)
		assert.Error(t, err)
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := config.Config{
			Server: config.ServerConfig{
				HTTPPort: 8080,
				GRPCPort: 9090,
			},
			Database: config.DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "aegis",
				Username: "user",
				Password: "pass",
			},
			Kafka: config.KafkaConfig{
				Brokers:  "localhost:9092",
				GroupID:  "data-integration",
				Topics: config.KafkaTopics{
					RawData:          "raw-data",
					ProcessedData:    "processed-data",
					ValidationErrors: "validation-errors",
					QualityMetrics:   "quality-metrics",
					DataLineage:      "data-lineage",
					SchemaChanges:    "schema-changes",
				},
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		cfg := config.Config{
			Server: config.ServerConfig{
				HTTPPort: 0, // Invalid port
			},
		}

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP port")
	})
}

// Benchmark tests

func BenchmarkETLPipeline(b *testing.B) {
	logger := zap.NewNop()
	cfg := config.ETLConfig{
		WorkerPoolSize: 4,
		BatchSize:     1000,
	}
	pipeline := etl.NewPipeline(cfg, logger)
	ctx := context.Background()

	data := make([]map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		data[i] = map[string]interface{}{
			"id":    i,
			"name":  "User " + string(rune(i)),
			"email": "user" + string(rune(i)) + "@example.com",
		}
	}

	options := etl.ProcessingOptions{
		BatchSize: 100,
		Validate:  false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.ProcessData(ctx, data, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataValidation(b *testing.B) {
	logger := zap.NewNop()
	cfg := config.ValidationConfig{
		EnableProfiling: false,
	}
	validator := validation.NewValidator(cfg, logger)
	ctx := context.Background()

	data := []map[string]interface{}{
		{"id": 1, "email": "john@example.com"},
		{"id": 2, "email": "jane@example.com"},
	}

	rules := []validation.Rule{
		{
			Field:   "email",
			Type:    "pattern",
			Pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateData(ctx, data, rules)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataQuality(b *testing.B) {
	logger := zap.NewNop()
	cfg := config.QualityConfig{}
	checker := quality.NewChecker(cfg, logger)
	ctx := context.Background()

	data := []map[string]interface{}{
		{"id": 1, "name": "John", "email": "john@example.com"},
		{"id": 2, "name": "Jane", "email": "jane@example.com"},
	}

	dimensions := []string{"completeness", "accuracy"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := checker.CheckQuality(ctx, data, dimensions)
		if err != nil {
			b.Fatal(err)
		}
	}
}