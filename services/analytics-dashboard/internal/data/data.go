package data

import "context"

type DataSourceType string

const (
	DataSourcePostgreSQL DataSourceType = "postgresql"
	DataSourceKafka      DataSourceType = "kafka"
	DataSourceAPI        DataSourceType = "api"
)

type DataSource struct {
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Type     DataSourceType         `json:"type"`
	Status   string                 `json:"status,omitempty"`
	Query    string                 `json:"query,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

type QueryRequest struct {
	Source DataSource             `json:"source"`
	Query  string                 `json:"query"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type QueryResponse struct {
	Data     interface{}       `json:"data"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Processor struct {
	cache interface{}
}

func NewProcessor(cache interface{}) *Processor {
	return &Processor{cache: cache}
}

func (p *Processor) ExecuteQuery(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	_ = ctx

	sampleData := []map[string]interface{}{
		{
			"id":        "record_1",
			"value":     123.45,
			"source":    req.Source.Type,
			"query":     req.Query,
			"timestamp": "2024-01-01T00:00:00Z",
		},
		{
			"id":        "record_2",
			"value":     678.90,
			"source":    req.Source.Type,
			"query":     req.Query,
			"timestamp": "2024-01-01T00:01:00Z",
		},
	}

	return &QueryResponse{
		Data: sampleData,
		Metadata: map[string]string{
			"source": string(req.Source.Type),
		},
	}, nil
}
