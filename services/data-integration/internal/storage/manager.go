package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aegisshield/data-integration/internal/config"
	"go.uber.org/zap"
)

// Manager handles data storage operations
type Manager struct {
	config config.StorageConfig
	logger *zap.Logger
	client StorageClient
}

// StorageClient defines the interface for storage operations
type StorageClient interface {
	Put(ctx context.Context, key string, data io.Reader, metadata map[string]interface{}) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Exists(ctx context.Context, key string) (bool, error)
	GetMetadata(ctx context.Context, key string) (map[string]interface{}, error)
}

// StorageMetadata represents metadata for stored data
type StorageMetadata struct {
	Key         string                 `json:"key"`
	Size        int64                  `json:"size"`
	ContentType string                 `json:"content_type"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Checksum    string                 `json:"checksum,omitempty"`
}

// ListResult represents the result of a list operation
type ListResult struct {
	Keys     []string `json:"keys"`
	HasMore  bool     `json:"has_more"`
	NextToken string  `json:"next_token,omitempty"`
}

// NewManager creates a new storage manager
func NewManager(config config.StorageConfig, logger *zap.Logger) (*Manager, error) {
	var client StorageClient
	var err error

	switch strings.ToLower(config.Type) {
	case "s3":
		client, err = NewS3Client(config, logger)
	case "gcs":
		client, err = NewGCSClient(config, logger)
	case "azure":
		client, err = NewAzureClient(config, logger)
	case "local":
		client, err = NewLocalClient(config, logger)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &Manager{
		config: config,
		logger: logger,
		client: client,
	}, nil
}

// Store stores data with the given key and metadata
func (m *Manager) Store(ctx context.Context, key string, data interface{}, metadata map[string]interface{}) error {
	// Convert data to io.Reader
	reader, err := m.convertToReader(data)
	if err != nil {
		return fmt.Errorf("failed to convert data to reader: %w", err)
	}

	// Add storage metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["stored_at"] = time.Now()
	metadata["storage_type"] = m.config.Type

	// Generate full key with prefix
	fullKey := m.buildKey(key)

	m.logger.Info("Storing data",
		zap.String("key", fullKey),
		zap.String("storage_type", m.config.Type))

	return m.client.Put(ctx, fullKey, reader, metadata)
}

// Retrieve retrieves data by key
func (m *Manager) Retrieve(ctx context.Context, key string) (io.ReadCloser, error) {
	fullKey := m.buildKey(key)

	m.logger.Debug("Retrieving data",
		zap.String("key", fullKey))

	return m.client.Get(ctx, fullKey)
}

// Delete deletes data by key
func (m *Manager) Delete(ctx context.Context, key string) error {
	fullKey := m.buildKey(key)

	m.logger.Info("Deleting data",
		zap.String("key", fullKey))

	return m.client.Delete(ctx, fullKey)
}

// List lists keys with the given prefix
func (m *Manager) List(ctx context.Context, prefix string) (*ListResult, error) {
	fullPrefix := m.buildKey(prefix)

	m.logger.Debug("Listing keys",
		zap.String("prefix", fullPrefix))

	keys, err := m.client.List(ctx, fullPrefix)
	if err != nil {
		return nil, err
	}

	// Remove prefix from keys for cleaner response
	cleanKeys := make([]string, len(keys))
	for i, key := range keys {
		cleanKeys[i] = m.removePrefix(key)
	}

	return &ListResult{
		Keys:    cleanKeys,
		HasMore: false, // Simplified implementation
	}, nil
}

// Exists checks if data exists at the given key
func (m *Manager) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := m.buildKey(key)
	return m.client.Exists(ctx, fullKey)
}

// GetMetadata retrieves metadata for the given key
func (m *Manager) GetMetadata(ctx context.Context, key string) (*StorageMetadata, error) {
	fullKey := m.buildKey(key)

	metadata, err := m.client.GetMetadata(ctx, fullKey)
	if err != nil {
		return nil, err
	}

	return &StorageMetadata{
		Key:      key,
		Metadata: metadata,
	}, nil
}

// Archive archives data to long-term storage
func (m *Manager) Archive(ctx context.Context, key string, archiveMetadata map[string]interface{}) error {
	// In a real implementation, this would move data to archive storage class
	// For now, just add archive metadata
	
	fullKey := m.buildKey(key)
	
	if archiveMetadata == nil {
		archiveMetadata = make(map[string]interface{})
	}
	archiveMetadata["archived_at"] = time.Now()
	archiveMetadata["storage_class"] = "archive"

	m.logger.Info("Archiving data",
		zap.String("key", fullKey))

	// This would be implemented based on the storage provider
	return nil
}

// Restore restores data from archive
func (m *Manager) Restore(ctx context.Context, key string) error {
	fullKey := m.buildKey(key)

	m.logger.Info("Restoring data from archive",
		zap.String("key", fullKey))

	// This would be implemented based on the storage provider
	return nil
}

// Copy copies data from one key to another
func (m *Manager) Copy(ctx context.Context, sourceKey, targetKey string) error {
	// Retrieve source data
	data, err := m.Retrieve(ctx, sourceKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve source data: %w", err)
	}
	defer data.Close()

	// Store to target
	metadata := map[string]interface{}{
		"copied_from": sourceKey,
		"copied_at":   time.Now(),
	}

	return m.Store(ctx, targetKey, data, metadata)
}

// Helper methods

func (m *Manager) buildKey(key string) string {
	if m.config.Prefix == "" {
		return key
	}
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(m.config.Prefix, "/"), key)
}

func (m *Manager) removePrefix(key string) string {
	if m.config.Prefix == "" {
		return key
	}
	prefix := fmt.Sprintf("%s/", strings.TrimSuffix(m.config.Prefix, "/"))
	return strings.TrimPrefix(key, prefix)
}

func (m *Manager) convertToReader(data interface{}) (io.Reader, error) {
	switch v := data.(type) {
	case io.Reader:
		return v, nil
	case string:
		return strings.NewReader(v), nil
	case []byte:
		return strings.NewReader(string(v)), nil
	default:
		return nil, fmt.Errorf("unsupported data type: %T", data)
	}
}

// Local storage client implementation
type LocalClient struct {
	basePath string
	logger   *zap.Logger
}

// NewLocalClient creates a new local storage client
func NewLocalClient(config config.StorageConfig, logger *zap.Logger) (*LocalClient, error) {
	basePath := config.Endpoint
	if basePath == "" {
		basePath = "./data"
	}

	return &LocalClient{
		basePath: basePath,
		logger:   logger,
	}, nil
}

func (c *LocalClient) Put(ctx context.Context, key string, data io.Reader, metadata map[string]interface{}) error {
	// This would implement local file storage
	c.logger.Debug("Local storage put", zap.String("key", key))
	return nil
}

func (c *LocalClient) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	// This would implement local file retrieval
	c.logger.Debug("Local storage get", zap.String("key", key))
	return nil, fmt.Errorf("not implemented")
}

func (c *LocalClient) Delete(ctx context.Context, key string) error {
	// This would implement local file deletion
	c.logger.Debug("Local storage delete", zap.String("key", key))
	return nil
}

func (c *LocalClient) List(ctx context.Context, prefix string) ([]string, error) {
	// This would implement local directory listing
	c.logger.Debug("Local storage list", zap.String("prefix", prefix))
	return []string{}, nil
}

func (c *LocalClient) Exists(ctx context.Context, key string) (bool, error) {
	// This would check if local file exists
	c.logger.Debug("Local storage exists", zap.String("key", key))
	return false, nil
}

func (c *LocalClient) GetMetadata(ctx context.Context, key string) (map[string]interface{}, error) {
	// This would get local file metadata
	c.logger.Debug("Local storage get metadata", zap.String("key", key))
	return map[string]interface{}{}, nil
}

// S3 storage client implementation (placeholder)
type S3Client struct {
	config config.StorageConfig
	logger *zap.Logger
}

func NewS3Client(config config.StorageConfig, logger *zap.Logger) (*S3Client, error) {
	return &S3Client{
		config: config,
		logger: logger,
	}, nil
}

func (c *S3Client) Put(ctx context.Context, key string, data io.Reader, metadata map[string]interface{}) error {
	c.logger.Debug("S3 put", zap.String("key", key))
	// Implementation would use AWS SDK
	return nil
}

func (c *S3Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	c.logger.Debug("S3 get", zap.String("key", key))
	return nil, fmt.Errorf("not implemented")
}

func (c *S3Client) Delete(ctx context.Context, key string) error {
	c.logger.Debug("S3 delete", zap.String("key", key))
	return nil
}

func (c *S3Client) List(ctx context.Context, prefix string) ([]string, error) {
	c.logger.Debug("S3 list", zap.String("prefix", prefix))
	return []string{}, nil
}

func (c *S3Client) Exists(ctx context.Context, key string) (bool, error) {
	c.logger.Debug("S3 exists", zap.String("key", key))
	return false, nil
}

func (c *S3Client) GetMetadata(ctx context.Context, key string) (map[string]interface{}, error) {
	c.logger.Debug("S3 get metadata", zap.String("key", key))
	return map[string]interface{}{}, nil
}

// GCS storage client implementation (placeholder)
type GCSClient struct {
	config config.StorageConfig
	logger *zap.Logger
}

func NewGCSClient(config config.StorageConfig, logger *zap.Logger) (*GCSClient, error) {
	return &GCSClient{
		config: config,
		logger: logger,
	}, nil
}

func (c *GCSClient) Put(ctx context.Context, key string, data io.Reader, metadata map[string]interface{}) error {
	c.logger.Debug("GCS put", zap.String("key", key))
	return nil
}

func (c *GCSClient) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	c.logger.Debug("GCS get", zap.String("key", key))
	return nil, fmt.Errorf("not implemented")
}

func (c *GCSClient) Delete(ctx context.Context, key string) error {
	c.logger.Debug("GCS delete", zap.String("key", key))
	return nil
}

func (c *GCSClient) List(ctx context.Context, prefix string) ([]string, error) {
	c.logger.Debug("GCS list", zap.String("prefix", prefix))
	return []string{}, nil
}

func (c *GCSClient) Exists(ctx context.Context, key string) (bool, error) {
	c.logger.Debug("GCS exists", zap.String("key", key))
	return false, nil
}

func (c *GCSClient) GetMetadata(ctx context.Context, key string) (map[string]interface{}, error) {
	c.logger.Debug("GCS get metadata", zap.String("key", key))
	return map[string]interface{}{}, nil
}

// Azure storage client implementation (placeholder)
type AzureClient struct {
	config config.StorageConfig
	logger *zap.Logger
}

func NewAzureClient(config config.StorageConfig, logger *zap.Logger) (*AzureClient, error) {
	return &AzureClient{
		config: config,
		logger: logger,
	}, nil
}

func (c *AzureClient) Put(ctx context.Context, key string, data io.Reader, metadata map[string]interface{}) error {
	c.logger.Debug("Azure put", zap.String("key", key))
	return nil
}

func (c *AzureClient) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	c.logger.Debug("Azure get", zap.String("key", key))
	return nil, fmt.Errorf("not implemented")
}

func (c *AzureClient) Delete(ctx context.Context, key string) error {
	c.logger.Debug("Azure delete", zap.String("key", key))
	return nil
}

func (c *AzureClient) List(ctx context.Context, prefix string) ([]string, error) {
	c.logger.Debug("Azure list", zap.String("prefix", prefix))
	return []string{}, nil
}

func (c *AzureClient) Exists(ctx context.Context, key string) (bool, error) {
	c.logger.Debug("Azure exists", zap.String("key", key))
	return false, nil
}

func (c *AzureClient) GetMetadata(ctx context.Context, key string) (map[string]interface{}, error) {
	c.logger.Debug("Azure get metadata", zap.String("key", key))
	return map[string]interface{}{}, nil
}