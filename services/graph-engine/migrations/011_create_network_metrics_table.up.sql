-- Create network_metrics table
CREATE TABLE IF NOT EXISTS network_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id VARCHAR(255) NOT NULL,
    degree_centrality DECIMAL(10,6) NOT NULL DEFAULT 0,
    betweenness_centrality DECIMAL(10,6) NOT NULL DEFAULT 0,
    closeness_centrality DECIMAL(10,6) NOT NULL DEFAULT 0,
    eigenvector_centrality DECIMAL(10,6) NOT NULL DEFAULT 0,
    page_rank DECIMAL(10,6) NOT NULL DEFAULT 0,
    clustering_coefficient DECIMAL(10,6) NOT NULL DEFAULT 0,
    community_id VARCHAR(255),
    calculated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    analysis_job_id UUID,
    metadata JSONB,
    graph_size INTEGER,
    calculation_method VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for network_metrics
CREATE INDEX IF NOT EXISTS idx_network_metrics_entity_id ON network_metrics(entity_id);
CREATE INDEX IF NOT EXISTS idx_network_metrics_calculated_at ON network_metrics(calculated_at);
CREATE INDEX IF NOT EXISTS idx_network_metrics_degree_centrality ON network_metrics(degree_centrality);
CREATE INDEX IF NOT EXISTS idx_network_metrics_betweenness_centrality ON network_metrics(betweenness_centrality);
CREATE INDEX IF NOT EXISTS idx_network_metrics_closeness_centrality ON network_metrics(closeness_centrality);
CREATE INDEX IF NOT EXISTS idx_network_metrics_page_rank ON network_metrics(page_rank);
CREATE INDEX IF NOT EXISTS idx_network_metrics_community_id ON network_metrics(community_id);
CREATE INDEX IF NOT EXISTS idx_network_metrics_analysis_job_id ON network_metrics(analysis_job_id);

-- Create composite index for entity and time
CREATE INDEX IF NOT EXISTS idx_network_metrics_entity_calculated ON network_metrics(entity_id, calculated_at DESC);

-- Add foreign key constraints
ALTER TABLE network_metrics 
ADD CONSTRAINT fk_network_metrics_analysis_job_id 
FOREIGN KEY (analysis_job_id) REFERENCES analysis_jobs(id) ON DELETE SET NULL;

-- Add unique constraint to ensure one metric per entity per calculation
CREATE UNIQUE INDEX IF NOT EXISTS idx_network_metrics_unique_entity_job 
ON network_metrics(entity_id, analysis_job_id) 
WHERE analysis_job_id IS NOT NULL;

-- Add comments
COMMENT ON TABLE network_metrics IS 'Stores calculated network analysis metrics for entities';
COMMENT ON COLUMN network_metrics.id IS 'Unique identifier for the metrics record';
COMMENT ON COLUMN network_metrics.entity_id IS 'ID of the entity these metrics belong to';
COMMENT ON COLUMN network_metrics.degree_centrality IS 'Degree centrality score (0.0-1.0)';
COMMENT ON COLUMN network_metrics.betweenness_centrality IS 'Betweenness centrality score (0.0-1.0)';
COMMENT ON COLUMN network_metrics.closeness_centrality IS 'Closeness centrality score (0.0-1.0)';
COMMENT ON COLUMN network_metrics.eigenvector_centrality IS 'Eigenvector centrality score (0.0-1.0)';
COMMENT ON COLUMN network_metrics.page_rank IS 'PageRank score (0.0-1.0)';
COMMENT ON COLUMN network_metrics.clustering_coefficient IS 'Clustering coefficient (0.0-1.0)';
COMMENT ON COLUMN network_metrics.community_id IS 'ID of the community this entity belongs to';
COMMENT ON COLUMN network_metrics.graph_size IS 'Size of the graph when metrics were calculated';
COMMENT ON COLUMN network_metrics.calculation_method IS 'Method used for calculation (networkx, neo4j, custom)';