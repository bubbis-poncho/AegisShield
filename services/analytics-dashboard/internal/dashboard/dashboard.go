package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Dashboard represents a user dashboard
type Dashboard struct {
	ID          string    `json:"id" gorm:"primarykey"`
	UserID      string    `json:"user_id" gorm:"not null;index"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Layout      Layout    `json:"layout" gorm:"type:jsonb"`
	Settings    Settings  `json:"settings" gorm:"type:jsonb"`
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	IsPublic    bool      `json:"is_public" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Widgets     []Widget  `json:"widgets" gorm:"foreignKey:DashboardID"`
}

// Widget represents a dashboard widget
type Widget struct {
	ID           string         `json:"id" gorm:"primarykey"`
	DashboardID  string         `json:"dashboard_id" gorm:"not null;index"`
	Type         WidgetType     `json:"type" gorm:"not null"`
	Title        string         `json:"title" gorm:"not null"`
	Position     Position       `json:"position" gorm:"type:jsonb"`
	Size         Size           `json:"size" gorm:"type:jsonb"`
	Config       WidgetConfig   `json:"config" gorm:"type:jsonb"`
	DataSource   DataSource     `json:"data_source" gorm:"type:jsonb"`
	RefreshRate  int            `json:"refresh_rate" gorm:"default:30"`
	IsVisible    bool           `json:"is_visible" gorm:"default:true"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// Layout represents dashboard layout configuration
type Layout struct {
	Columns       int    `json:"columns"`
	Rows          int    `json:"rows"`
	GridType      string `json:"grid_type"`
	AutoResize    bool   `json:"auto_resize"`
	ResponsiveBreakpoints map[string]int `json:"responsive_breakpoints"`
}

// Settings represents dashboard settings
type Settings struct {
	Theme           string            `json:"theme"`
	RefreshInterval int               `json:"refresh_interval"`
	TimeZone        string            `json:"timezone"`
	AutoRefresh     bool              `json:"auto_refresh"`
	ShowGrid        bool              `json:"show_grid"`
	ShowToolbar     bool              `json:"show_toolbar"`
	CustomCSS       string            `json:"custom_css"`
	Filters         map[string]string `json:"filters"`
}

// Position represents widget position
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Size represents widget size
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// WidgetType represents the type of widget
type WidgetType string

const (
	WidgetTypeChart       WidgetType = "chart"
	WidgetTypeTable       WidgetType = "table"
	WidgetTypeKPI         WidgetType = "kpi"
	WidgetTypeMap         WidgetType = "map"
	WidgetTypeText        WidgetType = "text"
	WidgetTypeGauge       WidgetType = "gauge"
	WidgetTypeHeatmap     WidgetType = "heatmap"
	WidgetTypePieChart    WidgetType = "pie_chart"
	WidgetTypeBarChart    WidgetType = "bar_chart"
	WidgetTypeLineChart   WidgetType = "line_chart"
	WidgetTypeScatterPlot WidgetType = "scatter_plot"
	WidgetTypeTreeMap     WidgetType = "tree_map"
	WidgetTypeSankey      WidgetType = "sankey"
	WidgetTypeNetwork     WidgetType = "network"
)

// WidgetConfig represents widget-specific configuration
type WidgetConfig struct {
	ChartType     string            `json:"chart_type,omitempty"`
	XAxis         AxisConfig        `json:"x_axis,omitempty"`
	YAxis         AxisConfig        `json:"y_axis,omitempty"`
	Series        []SeriesConfig    `json:"series,omitempty"`
	Colors        []string          `json:"colors,omitempty"`
	Legend        LegendConfig      `json:"legend,omitempty"`
	Tooltip       TooltipConfig     `json:"tooltip,omitempty"`
	Animation     AnimationConfig   `json:"animation,omitempty"`
	Aggregation   AggregationConfig `json:"aggregation,omitempty"`
	Threshold     ThresholdConfig   `json:"threshold,omitempty"`
	CustomOptions map[string]interface{} `json:"custom_options,omitempty"`
}

// AxisConfig represents axis configuration
type AxisConfig struct {
	Label    string `json:"label"`
	Type     string `json:"type"`
	Format   string `json:"format"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	LogScale bool   `json:"log_scale"`
}

// SeriesConfig represents series configuration
type SeriesConfig struct {
	Name   string `json:"name"`
	Field  string `json:"field"`
	Color  string `json:"color"`
	Type   string `json:"type"`
	Smooth bool   `json:"smooth"`
}

// LegendConfig represents legend configuration
type LegendConfig struct {
	Show     bool   `json:"show"`
	Position string `json:"position"`
}

// TooltipConfig represents tooltip configuration
type TooltipConfig struct {
	Show   bool   `json:"show"`
	Format string `json:"format"`
}

// AnimationConfig represents animation configuration
type AnimationConfig struct {
	Enabled  bool `json:"enabled"`
	Duration int  `json:"duration"`
}

// AggregationConfig represents data aggregation configuration
type AggregationConfig struct {
	Type     string `json:"type"`
	Interval string `json:"interval"`
	GroupBy  []string `json:"group_by"`
}

// ThresholdConfig represents threshold configuration for KPIs
type ThresholdConfig struct {
	Warning   float64 `json:"warning"`
	Critical  float64 `json:"critical"`
	Unit      string  `json:"unit"`
	Direction string  `json:"direction"` // "above" or "below"
}

// DataSource represents widget data source configuration
type DataSource struct {
	Type       DataSourceType    `json:"type"`
	Query      string            `json:"query"`
	Parameters map[string]interface{} `json:"parameters"`
	CacheTTL   int               `json:"cache_ttl"`
	RealTime   bool              `json:"real_time"`
}

// DataSourceType represents the type of data source
type DataSourceType string

const (
	DataSourceSQL       DataSourceType = "sql"
	DataSourceGraphQL   DataSourceType = "graphql"
	DataSourceREST      DataSourceType = "rest"
	DataSourceNeo4j     DataSourceType = "neo4j"
	DataSourceKafka     DataSourceType = "kafka"
	DataSourceRedis     DataSourceType = "redis"
	DataSourcePrometheus DataSourceType = "prometheus"
)

// Manager handles dashboard operations
type Manager struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewManager creates a new dashboard manager
func NewManager(db *gorm.DB, redis *redis.Client) *Manager {
	return &Manager{
		db:    db,
		redis: redis,
	}
}

// CreateDashboard creates a new dashboard
func (m *Manager) CreateDashboard(ctx context.Context, dashboard *Dashboard) error {
	dashboard.ID = uuid.New().String()
	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()

	if err := m.db.WithContext(ctx).Create(dashboard).Error; err != nil {
		return fmt.Errorf("failed to create dashboard: %w", err)
	}

	// Cache the dashboard
	if err := m.cacheDashboard(ctx, dashboard); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: failed to cache dashboard: %v\n", err)
	}

	return nil
}

// GetDashboard retrieves a dashboard by ID
func (m *Manager) GetDashboard(ctx context.Context, id string) (*Dashboard, error) {
	// Try cache first
	if dashboard, err := m.getDashboardFromCache(ctx, id); err == nil {
		return dashboard, nil
	}

	var dashboard Dashboard
	if err := m.db.WithContext(ctx).Preload("Widgets").First(&dashboard, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Cache the result
	if err := m.cacheDashboard(ctx, &dashboard); err != nil {
		fmt.Printf("Warning: failed to cache dashboard: %v\n", err)
	}

	return &dashboard, nil
}

// GetUserDashboards retrieves all dashboards for a user
func (m *Manager) GetUserDashboards(ctx context.Context, userID string) ([]Dashboard, error) {
	var dashboards []Dashboard
	if err := m.db.WithContext(ctx).Preload("Widgets").Where("user_id = ?", userID).Find(&dashboards).Error; err != nil {
		return nil, fmt.Errorf("failed to get user dashboards: %w", err)
	}

	return dashboards, nil
}

// UpdateDashboard updates an existing dashboard
func (m *Manager) UpdateDashboard(ctx context.Context, dashboard *Dashboard) error {
	dashboard.UpdatedAt = time.Now()

	if err := m.db.WithContext(ctx).Save(dashboard).Error; err != nil {
		return fmt.Errorf("failed to update dashboard: %w", err)
	}

	// Update cache
	if err := m.cacheDashboard(ctx, dashboard); err != nil {
		fmt.Printf("Warning: failed to cache dashboard: %v\n", err)
	}

	// Invalidate user dashboard list cache
	cacheKey := fmt.Sprintf("user_dashboards:%s", dashboard.UserID)
	m.redis.Del(ctx, cacheKey)

	return nil
}

// DeleteDashboard deletes a dashboard
func (m *Manager) DeleteDashboard(ctx context.Context, id string) error {
	// Get the dashboard to find the user ID
	dashboard, err := m.GetDashboard(ctx, id)
	if err != nil {
		return err
	}

	if err := m.db.WithContext(ctx).Delete(&Dashboard{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	// Remove from cache
	cacheKey := fmt.Sprintf("dashboard:%s", id)
	m.redis.Del(ctx, cacheKey)

	// Invalidate user dashboard list cache
	userCacheKey := fmt.Sprintf("user_dashboards:%s", dashboard.UserID)
	m.redis.Del(ctx, userCacheKey)

	return nil
}

// AddWidget adds a widget to a dashboard
func (m *Manager) AddWidget(ctx context.Context, widget *Widget) error {
	widget.ID = uuid.New().String()
	widget.CreatedAt = time.Now()
	widget.UpdatedAt = time.Now()

	if err := m.db.WithContext(ctx).Create(widget).Error; err != nil {
		return fmt.Errorf("failed to add widget: %w", err)
	}

	// Invalidate dashboard cache
	cacheKey := fmt.Sprintf("dashboard:%s", widget.DashboardID)
	m.redis.Del(ctx, cacheKey)

	return nil
}

// UpdateWidget updates an existing widget
func (m *Manager) UpdateWidget(ctx context.Context, widget *Widget) error {
	widget.UpdatedAt = time.Now()

	if err := m.db.WithContext(ctx).Save(widget).Error; err != nil {
		return fmt.Errorf("failed to update widget: %w", err)
	}

	// Invalidate dashboard cache
	cacheKey := fmt.Sprintf("dashboard:%s", widget.DashboardID)
	m.redis.Del(ctx, cacheKey)

	return nil
}

// DeleteWidget deletes a widget
func (m *Manager) DeleteWidget(ctx context.Context, id string) error {
	var widget Widget
	if err := m.db.WithContext(ctx).First(&widget, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to find widget: %w", err)
	}

	if err := m.db.WithContext(ctx).Delete(&Widget{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete widget: %w", err)
	}

	// Invalidate dashboard cache
	cacheKey := fmt.Sprintf("dashboard:%s", widget.DashboardID)
	m.redis.Del(ctx, cacheKey)

	return nil
}

// CloneDashboard creates a copy of an existing dashboard
func (m *Manager) CloneDashboard(ctx context.Context, sourceID, userID, newName string) (*Dashboard, error) {
	source, err := m.GetDashboard(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source dashboard: %w", err)
	}

	clone := &Dashboard{
		UserID:      userID,
		Name:        newName,
		Description: fmt.Sprintf("Copy of %s", source.Description),
		Layout:      source.Layout,
		Settings:    source.Settings,
		IsDefault:   false,
		IsPublic:    false,
	}

	if err := m.CreateDashboard(ctx, clone); err != nil {
		return nil, fmt.Errorf("failed to create cloned dashboard: %w", err)
	}

	// Clone widgets
	for _, widget := range source.Widgets {
		clonedWidget := &Widget{
			DashboardID: clone.ID,
			Type:        widget.Type,
			Title:       widget.Title,
			Position:    widget.Position,
			Size:        widget.Size,
			Config:      widget.Config,
			DataSource:  widget.DataSource,
			RefreshRate: widget.RefreshRate,
			IsVisible:   widget.IsVisible,
		}

		if err := m.AddWidget(ctx, clonedWidget); err != nil {
			return nil, fmt.Errorf("failed to clone widget: %w", err)
		}
	}

	return clone, nil
}

// cacheDashboard caches a dashboard in Redis
func (m *Manager) cacheDashboard(ctx context.Context, dashboard *Dashboard) error {
	data, err := json.Marshal(dashboard)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("dashboard:%s", dashboard.ID)
	return m.redis.Set(ctx, cacheKey, data, 15*time.Minute).Err()
}

// getDashboardFromCache retrieves a dashboard from Redis cache
func (m *Manager) getDashboardFromCache(ctx context.Context, id string) (*Dashboard, error) {
	cacheKey := fmt.Sprintf("dashboard:%s", id)
	data, err := m.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}

	var dashboard Dashboard
	if err := json.Unmarshal([]byte(data), &dashboard); err != nil {
		return nil, err
	}

	return &dashboard, nil
}