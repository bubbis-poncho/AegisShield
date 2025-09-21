package visualization

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ChartData represents chart visualization data
type ChartData struct {
	Labels   []string      `json:"labels"`
	Datasets []Dataset     `json:"datasets"`
	Metadata ChartMetadata `json:"metadata"`
}

// Dataset represents a data series
type Dataset struct {
	Label           string      `json:"label"`
	Data            []DataPoint `json:"data"`
	BackgroundColor string      `json:"backgroundColor,omitempty"`
	BorderColor     string      `json:"borderColor,omitempty"`
	Fill            bool        `json:"fill,omitempty"`
	Tension         float64     `json:"tension,omitempty"`
	Type            string      `json:"type,omitempty"`
}

// DataPoint represents a single data point
type DataPoint struct {
	X     interface{} `json:"x"`
	Y     interface{} `json:"y"`
	Label string      `json:"label,omitempty"`
	Meta  interface{} `json:"meta,omitempty"`
}

// ChartMetadata contains chart metadata
type ChartMetadata struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Source      string            `json:"source"`
	LastUpdated time.Time         `json:"last_updated"`
	TotalCount  int64             `json:"total_count"`
	Filters     map[string]string `json:"filters"`
}

// TableData represents table visualization data
type TableData struct {
	Columns  []TableColumn `json:"columns"`
	Rows     []TableRow    `json:"rows"`
	Metadata TableMetadata `json:"metadata"`
}

// TableColumn represents a table column
type TableColumn struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	DataType  string `json:"dataType"`
	Sortable  bool   `json:"sortable"`
	Filterable bool  `json:"filterable"`
	Width     int    `json:"width,omitempty"`
	Align     string `json:"align,omitempty"`
	Format    string `json:"format,omitempty"`
}

// TableRow represents a table row
type TableRow struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

// TableMetadata contains table metadata
type TableMetadata struct {
	TotalRows   int64             `json:"total_rows"`
	Page        int               `json:"page"`
	PageSize    int               `json:"page_size"`
	Sort        string            `json:"sort,omitempty"`
	SortOrder   string            `json:"sort_order,omitempty"`
	Filters     map[string]string `json:"filters"`
	LastUpdated time.Time         `json:"last_updated"`
}

// KPIData represents KPI visualization data
type KPIData struct {
	Value       float64     `json:"value"`
	PreviousValue float64   `json:"previous_value"`
	Target      float64     `json:"target,omitempty"`
	Unit        string      `json:"unit"`
	Change      ChangeData  `json:"change"`
	Status      KPIStatus   `json:"status"`
	Trend       TrendData   `json:"trend"`
	Metadata    KPIMetadata `json:"metadata"`
}

// ChangeData represents value change information
type ChangeData struct {
	Absolute   float64 `json:"absolute"`
	Percentage float64 `json:"percentage"`
	Direction  string  `json:"direction"` // "up", "down", "neutral"
}

// KPIStatus represents KPI status
type KPIStatus string

const (
	KPIStatusNormal   KPIStatus = "normal"
	KPIStatusWarning  KPIStatus = "warning"
	KPIStatusCritical KPIStatus = "critical"
	KPIStatusUnknown  KPIStatus = "unknown"
)

// TrendData represents trend information
type TrendData struct {
	Direction string      `json:"direction"` // "up", "down", "stable"
	Strength  float64     `json:"strength"`  // 0-1 scale
	Points    []DataPoint `json:"points"`
}

// KPIMetadata contains KPI metadata
type KPIMetadata struct {
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Category     string            `json:"category"`
	Source       string            `json:"source"`
	LastUpdated  time.Time         `json:"last_updated"`
	ThresholdConfig ThresholdConfig `json:"threshold_config"`
}

// ThresholdConfig represents KPI threshold configuration
type ThresholdConfig struct {
	Warning   float64 `json:"warning"`
	Critical  float64 `json:"critical"`
	Direction string  `json:"direction"` // "above" or "below"
}

// MapData represents map visualization data
type MapData struct {
	Markers   []MapMarker   `json:"markers"`
	Clusters  []MapCluster  `json:"clusters"`
	Layers    []MapLayer    `json:"layers"`
	Bounds    MapBounds     `json:"bounds"`
	Metadata  MapMetadata   `json:"metadata"`
}

// MapMarker represents a map marker
type MapMarker struct {
	ID        string      `json:"id"`
	Latitude  float64     `json:"latitude"`
	Longitude float64     `json:"longitude"`
	Title     string      `json:"title"`
	Content   string      `json:"content"`
	Icon      string      `json:"icon"`
	Color     string      `json:"color"`
	Size      string      `json:"size"`
	Data      interface{} `json:"data"`
}

// MapCluster represents clustered markers
type MapCluster struct {
	ID        string    `json:"id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Count     int       `json:"count"`
	Markers   []string  `json:"markers"` // Marker IDs
}

// MapLayer represents a map layer
type MapLayer struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Type    string      `json:"type"` // "markers", "heatmap", "choropleth"
	Visible bool        `json:"visible"`
	Data    interface{} `json:"data"`
	Style   interface{} `json:"style"`
}

// MapBounds represents map viewport bounds
type MapBounds struct {
	North float64 `json:"north"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	West  float64 `json:"west"`
}

// MapMetadata contains map metadata
type MapMetadata struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	LastUpdated time.Time `json:"last_updated"`
	ZoomLevel   int       `json:"zoom_level"`
	Center      struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"center"`
}

// HeatmapData represents heatmap visualization data
type HeatmapData struct {
	XAxis    []string         `json:"x_axis"`
	YAxis    []string         `json:"y_axis"`
	Values   [][]float64      `json:"values"`
	ColorScale ColorScale     `json:"color_scale"`
	Metadata HeatmapMetadata  `json:"metadata"`
}

// ColorScale represents color scale configuration
type ColorScale struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Colors []string `json:"colors"`
}

// HeatmapMetadata contains heatmap metadata
type HeatmapMetadata struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	XAxisLabel  string    `json:"x_axis_label"`
	YAxisLabel  string    `json:"y_axis_label"`
	ValueLabel  string    `json:"value_label"`
	LastUpdated time.Time `json:"last_updated"`
}

// NetworkData represents network visualization data
type NetworkData struct {
	Nodes    []NetworkNode `json:"nodes"`
	Edges    []NetworkEdge `json:"edges"`
	Layout   NetworkLayout `json:"layout"`
	Metadata NetworkMetadata `json:"metadata"`
}

// NetworkNode represents a network node
type NetworkNode struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Group    string                 `json:"group"`
	Size     float64                `json:"size"`
	Color    string                 `json:"color"`
	Shape    string                 `json:"shape"`
	X        float64                `json:"x,omitempty"`
	Y        float64                `json:"y,omitempty"`
	Data     map[string]interface{} `json:"data"`
}

// NetworkEdge represents a network edge
type NetworkEdge struct {
	ID     string                 `json:"id"`
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Label  string                 `json:"label,omitempty"`
	Weight float64                `json:"weight,omitempty"`
	Color  string                 `json:"color,omitempty"`
	Width  float64                `json:"width,omitempty"`
	Data   map[string]interface{} `json:"data"`
}

// NetworkLayout represents network layout configuration
type NetworkLayout struct {
	Algorithm string                 `json:"algorithm"` // "force", "circular", "hierarchical"
	Options   map[string]interface{} `json:"options"`
}

// NetworkMetadata contains network metadata
type NetworkMetadata struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	NodeCount   int       `json:"node_count"`
	EdgeCount   int       `json:"edge_count"`
	LastUpdated time.Time `json:"last_updated"`
}

// Engine handles visualization data processing
type Engine struct {
	redis *redis.Client
}

// NewEngine creates a new visualization engine
func NewEngine(redis *redis.Client) *Engine {
	return &Engine{
		redis: redis,
	}
}

// ProcessChartData processes and caches chart data
func (e *Engine) ProcessChartData(ctx context.Context, widgetID string, data *ChartData) error {
	// Add metadata
	data.Metadata.LastUpdated = time.Now()

	// Cache the processed data
	cacheKey := fmt.Sprintf("chart_data:%s", widgetID)
	return e.cacheVisualizationData(ctx, cacheKey, data, 5*time.Minute)
}

// ProcessTableData processes and caches table data
func (e *Engine) ProcessTableData(ctx context.Context, widgetID string, data *TableData) error {
	// Add metadata
	data.Metadata.LastUpdated = time.Now()

	// Cache the processed data
	cacheKey := fmt.Sprintf("table_data:%s", widgetID)
	return e.cacheVisualizationData(ctx, cacheKey, data, 5*time.Minute)
}

// ProcessKPIData processes and caches KPI data
func (e *Engine) ProcessKPIData(ctx context.Context, widgetID string, data *KPIData) error {
	// Calculate change
	if data.PreviousValue != 0 {
		data.Change.Absolute = data.Value - data.PreviousValue
		data.Change.Percentage = (data.Change.Absolute / data.PreviousValue) * 100
		
		if data.Change.Absolute > 0 {
			data.Change.Direction = "up"
		} else if data.Change.Absolute < 0 {
			data.Change.Direction = "down"
		} else {
			data.Change.Direction = "neutral"
		}
	}

	// Determine status based on thresholds
	data.Status = e.calculateKPIStatus(data.Value, data.Metadata.ThresholdConfig)

	// Add metadata
	data.Metadata.LastUpdated = time.Now()

	// Cache the processed data
	cacheKey := fmt.Sprintf("kpi_data:%s", widgetID)
	return e.cacheVisualizationData(ctx, cacheKey, data, 1*time.Minute)
}

// ProcessMapData processes and caches map data
func (e *Engine) ProcessMapData(ctx context.Context, widgetID string, data *MapData) error {
	// Calculate bounds if not provided
	if len(data.Markers) > 0 && (data.Bounds == MapBounds{}) {
		data.Bounds = e.calculateMapBounds(data.Markers)
	}

	// Add metadata
	data.Metadata.LastUpdated = time.Now()

	// Cache the processed data
	cacheKey := fmt.Sprintf("map_data:%s", widgetID)
	return e.cacheVisualizationData(ctx, cacheKey, data, 5*time.Minute)
}

// ProcessHeatmapData processes and caches heatmap data
func (e *Engine) ProcessHeatmapData(ctx context.Context, widgetID string, data *HeatmapData) error {
	// Calculate color scale if not provided
	if len(data.ColorScale.Colors) == 0 {
		data.ColorScale = e.calculateColorScale(data.Values)
	}

	// Add metadata
	data.Metadata.LastUpdated = time.Now()

	// Cache the processed data
	cacheKey := fmt.Sprintf("heatmap_data:%s", widgetID)
	return e.cacheVisualizationData(ctx, cacheKey, data, 5*time.Minute)
}

// ProcessNetworkData processes and caches network data
func (e *Engine) ProcessNetworkData(ctx context.Context, widgetID string, data *NetworkData) error {
	// Add metadata
	data.Metadata.NodeCount = len(data.Nodes)
	data.Metadata.EdgeCount = len(data.Edges)
	data.Metadata.LastUpdated = time.Now()

	// Cache the processed data
	cacheKey := fmt.Sprintf("network_data:%s", widgetID)
	return e.cacheVisualizationData(ctx, cacheKey, data, 10*time.Minute)
}

// GetVisualizationData retrieves cached visualization data
func (e *Engine) GetVisualizationData(ctx context.Context, widgetID, dataType string) (interface{}, error) {
	cacheKey := fmt.Sprintf("%s_data:%s", dataType, widgetID)
	
	data, err := e.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cached data: %w", err)
	}

	switch dataType {
	case "chart":
		var chartData ChartData
		if err := json.Unmarshal([]byte(data), &chartData); err != nil {
			return nil, err
		}
		return &chartData, nil
	case "table":
		var tableData TableData
		if err := json.Unmarshal([]byte(data), &tableData); err != nil {
			return nil, err
		}
		return &tableData, nil
	case "kpi":
		var kpiData KPIData
		if err := json.Unmarshal([]byte(data), &kpiData); err != nil {
			return nil, err
		}
		return &kpiData, nil
	case "map":
		var mapData MapData
		if err := json.Unmarshal([]byte(data), &mapData); err != nil {
			return nil, err
		}
		return &mapData, nil
	case "heatmap":
		var heatmapData HeatmapData
		if err := json.Unmarshal([]byte(data), &heatmapData); err != nil {
			return nil, err
		}
		return &heatmapData, nil
	case "network":
		var networkData NetworkData
		if err := json.Unmarshal([]byte(data), &networkData); err != nil {
			return nil, err
		}
		return &networkData, nil
	default:
		return nil, fmt.Errorf("unknown data type: %s", dataType)
	}
}

// cacheVisualizationData caches visualization data in Redis
func (e *Engine) cacheVisualizationData(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return e.redis.Set(ctx, key, jsonData, ttl).Err()
}

// calculateKPIStatus determines KPI status based on thresholds
func (e *Engine) calculateKPIStatus(value float64, config ThresholdConfig) KPIStatus {
	if config.Direction == "above" {
		if value >= config.Critical {
			return KPIStatusCritical
		} else if value >= config.Warning {
			return KPIStatusWarning
		}
	} else if config.Direction == "below" {
		if value <= config.Critical {
			return KPIStatusCritical
		} else if value <= config.Warning {
			return KPIStatusWarning
		}
	}
	return KPIStatusNormal
}

// calculateMapBounds calculates map bounds from markers
func (e *Engine) calculateMapBounds(markers []MapMarker) MapBounds {
	if len(markers) == 0 {
		return MapBounds{}
	}

	bounds := MapBounds{
		North: markers[0].Latitude,
		South: markers[0].Latitude,
		East:  markers[0].Longitude,
		West:  markers[0].Longitude,
	}

	for _, marker := range markers {
		if marker.Latitude > bounds.North {
			bounds.North = marker.Latitude
		}
		if marker.Latitude < bounds.South {
			bounds.South = marker.Latitude
		}
		if marker.Longitude > bounds.East {
			bounds.East = marker.Longitude
		}
		if marker.Longitude < bounds.West {
			bounds.West = marker.Longitude
		}
	}

	return bounds
}

// calculateColorScale calculates color scale from values
func (e *Engine) calculateColorScale(values [][]float64) ColorScale {
	var min, max float64
	hasValues := false

	for _, row := range values {
		for _, val := range row {
			if !hasValues {
				min = val
				max = val
				hasValues = true
			} else {
				if val < min {
					min = val
				}
				if val > max {
					max = val
				}
			}
		}
	}

	return ColorScale{
		Min:    min,
		Max:    max,
		Colors: []string{"#3182bd", "#6baed6", "#9ecae1", "#c6dbef", "#e6f3ff"},
	}
}