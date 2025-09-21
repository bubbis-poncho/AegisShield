package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/aegisshield/graph-engine/internal/config"
	"github.com/aegisshield/graph-engine/internal/neo4j"
	"github.com/google/uuid"
)

// GraphAnalytics provides advanced graph analysis capabilities
type GraphAnalytics struct {
	neo4jClient *neo4j.Client
	config      config.GraphEngineConfig
	logger      *slog.Logger
}

// NetworkMetrics represents comprehensive network analysis metrics
type NetworkMetrics struct {
	ID                string                 `json:"id"`
	CalculatedAt      time.Time              `json:"calculated_at"`
	NetworkSize       int                    `json:"network_size"`
	EdgeCount         int                    `json:"edge_count"`
	Density           float64                `json:"density"`
	Clustering        float64                `json:"clustering_coefficient"`
	Diameter          int                    `json:"diameter"`
	AveragePathLength float64                `json:"average_path_length"`
	Assortativity     float64                `json:"assortativity"`
	Components        []*Component           `json:"components"`
	CentralityStats   *CentralityStatistics  `json:"centrality_stats"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// Component represents a connected component in the graph
type Component struct {
	ID         string   `json:"id"`
	Size       int      `json:"size"`
	Entities   []string `json:"entities"`
	IsGiant    bool     `json:"is_giant"`
	Density    float64  `json:"density"`
	Centrality float64  `json:"avg_centrality"`
}

// CentralityStatistics contains statistical measures for centrality metrics
type CentralityStatistics struct {
	DegreeCentrality      *CentralityStats `json:"degree_centrality"`
	BetweennessCentrality *CentralityStats `json:"betweenness_centrality"`
	ClosenessCentrality   *CentralityStats `json:"closeness_centrality"`
	EigenvectorCentrality *CentralityStats `json:"eigenvector_centrality"`
	PageRank              *CentralityStats `json:"pagerank"`
}

// CentralityStats contains statistical measures for a centrality metric
type CentralityStats struct {
	Mean      float64 `json:"mean"`
	Median    float64 `json:"median"`
	StdDev    float64 `json:"std_dev"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Skewness  float64 `json:"skewness"`
	Kurtosis  float64 `json:"kurtosis"`
}

// CommunityDetectionRequest represents a community detection request
type CommunityDetectionRequest struct {
	Algorithm       CommunityAlgorithm     `json:"algorithm"`
	EntityIDs       []string               `json:"entity_ids,omitempty"`
	Resolution      float64                `json:"resolution,omitempty"`
	MinCommunitySize int                   `json:"min_community_size,omitempty"`
	MaxCommunities  int                    `json:"max_communities,omitempty"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
}

// CommunityAlgorithm represents different community detection algorithms
type CommunityAlgorithm string

const (
	AlgorithmLouvain        CommunityAlgorithm = "louvain"
	AlgorithmLabelProp      CommunityAlgorithm = "label_propagation"
	AlgorithmFastGreedy     CommunityAlgorithm = "fast_greedy"
	AlgorithmWalktrap       CommunityAlgorithm = "walktrap"
	AlgorithmInfomap        CommunityAlgorithm = "infomap"
	AlgorithmLeiden         CommunityAlgorithm = "leiden"
)

// CommunityDetectionResult contains community detection results
type CommunityDetectionResult struct {
	Algorithm         CommunityAlgorithm `json:"algorithm"`
	Communities       []*Community       `json:"communities"`
	Modularity        float64            `json:"modularity"`
	NumCommunities    int                `json:"num_communities"`
	LargestCommunity  int                `json:"largest_community"`
	SmallestCommunity int                `json:"smallest_community"`
	ProcessingTime    time.Duration      `json:"processing_time"`
}

// Community represents a detected community
type Community struct {
	ID         string                 `json:"id"`
	Size       int                    `json:"size"`
	Entities   []*neo4j.Entity        `json:"entities"`
	Density    float64                `json:"density"`
	Modularity float64                `json:"modularity"`
	Centrality float64                `json:"avg_centrality"`
	RiskScore  float64                `json:"risk_score"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// PathAnalysisRequest represents a path analysis request
type PathAnalysisRequest struct {
	SourceID      string                 `json:"source_id"`
	TargetID      string                 `json:"target_id,omitempty"`
	MaxDepth      int                    `json:"max_depth"`
	MaxPaths      int                    `json:"max_paths"`
	WeightProperty string                `json:"weight_property,omitempty"`
	PathTypes     []string               `json:"path_types,omitempty"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
}

// PathAnalysisResult contains path analysis results
type PathAnalysisResult struct {
	Paths            []*neo4j.Path `json:"paths"`
	ShortestPath     *neo4j.Path   `json:"shortest_path"`
	ShortestDistance int           `json:"shortest_distance"`
	AverageDistance  float64       `json:"average_distance"`
	PathDiversity    float64       `json:"path_diversity"`
	ProcessingTime   time.Duration `json:"processing_time"`
}

// InfluenceAnalysisRequest represents an influence analysis request
type InfluenceAnalysisRequest struct {
	EntityIDs      []string               `json:"entity_ids"`
	InfluenceType  InfluenceType          `json:"influence_type"`
	MaxDepth       int                    `json:"max_depth"`
	DecayFactor    float64                `json:"decay_factor"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
}

// InfluenceType represents different types of influence analysis
type InfluenceType string

const (
	InfluenceTypeIncoming InfluenceType = "incoming"
	InfluenceTypeOutgoing InfluenceType = "outgoing"
	InfluenceTypeBoth     InfluenceType = "both"
)

// InfluenceAnalysisResult contains influence analysis results
type InfluenceAnalysisResult struct {
	InfluenceScores map[string]float64 `json:"influence_scores"`
	TotalInfluence  float64            `json:"total_influence"`
	TopInfluencers  []*InfluenceRanking `json:"top_influencers"`
	ProcessingTime  time.Duration      `json:"processing_time"`
}

// InfluenceRanking represents an entity's influence ranking
type InfluenceRanking struct {
	EntityID       string  `json:"entity_id"`
	InfluenceScore float64 `json:"influence_score"`
	Rank           int     `json:"rank"`
}

// NewGraphAnalytics creates a new graph analytics instance
func NewGraphAnalytics(client *neo4j.Client, config config.GraphEngineConfig, logger *slog.Logger) *GraphAnalytics {
	return &GraphAnalytics{
		neo4jClient: client,
		config:      config,
		logger:      logger,
	}
}

// CalculateNetworkMetrics calculates comprehensive network metrics
func (ga *GraphAnalytics) CalculateNetworkMetrics(ctx context.Context, entityTypes []string) (*NetworkMetrics, error) {
	startTime := time.Now()
	
	ga.logger.Info("Starting network metrics calculation",
		"entity_types", entityTypes)

	metrics := &NetworkMetrics{
		ID:           uuid.New().String(),
		CalculatedAt: time.Now(),
	}

	// Calculate basic network statistics
	if err := ga.calculateBasicNetworkStats(ctx, metrics, entityTypes); err != nil {
		return nil, fmt.Errorf("failed to calculate basic network stats: %w", err)
	}

	// Calculate centrality statistics
	centralityStats, err := ga.calculateCentralityStatistics(ctx, entityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate centrality statistics: %w", err)
	}
	metrics.CentralityStats = centralityStats

	// Find connected components
	components, err := ga.findConnectedComponents(ctx, entityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to find connected components: %w", err)
	}
	metrics.Components = components

	// Calculate network-level metrics
	if err := ga.calculateNetworkLevelMetrics(ctx, metrics, entityTypes); err != nil {
		return nil, fmt.Errorf("failed to calculate network-level metrics: %w", err)
	}

	ga.logger.Info("Network metrics calculation completed",
		"processing_time", time.Since(startTime),
		"network_size", metrics.NetworkSize,
		"components", len(metrics.Components))

	return metrics, nil
}

// calculateBasicNetworkStats calculates basic network statistics
func (ga *GraphAnalytics) calculateBasicNetworkStats(ctx context.Context, metrics *NetworkMetrics, entityTypes []string) error {
	// Count nodes and edges
	nodeQuery := `
		MATCH (n)
		WHERE n:` + entityTypes[0] + `
		RETURN COUNT(n) as nodeCount
	`
	
	edgeQuery := `
		MATCH ()-[r]->()
		RETURN COUNT(r) as edgeCount
	`

	// Execute node count query
	nodeRecords, err := ga.neo4jClient.ExecuteQuery(ctx, nodeQuery, nil)
	if err != nil {
		return fmt.Errorf("failed to count nodes: %w", err)
	}

	if len(nodeRecords) > 0 {
		if count, ok := nodeRecords[0]["nodeCount"].(int64); ok {
			metrics.NetworkSize = int(count)
		}
	}

	// Execute edge count query
	edgeRecords, err := ga.neo4jClient.ExecuteQuery(ctx, edgeQuery, nil)
	if err != nil {
		return fmt.Errorf("failed to count edges: %w", err)
	}

	if len(edgeRecords) > 0 {
		if count, ok := edgeRecords[0]["edgeCount"].(int64); ok {
			metrics.EdgeCount = int(count)
		}
	}

	// Calculate density
	if metrics.NetworkSize > 1 {
		maxPossibleEdges := metrics.NetworkSize * (metrics.NetworkSize - 1)
		metrics.Density = float64(metrics.EdgeCount) / float64(maxPossibleEdges)
	}

	return nil
}

// calculateCentralityStatistics calculates statistics for all centrality measures
func (ga *GraphAnalytics) calculateCentralityStatistics(ctx context.Context, entityTypes []string) (*CentralityStatistics, error) {
	// Use Neo4j Graph Data Science library for centrality calculations
	stats := &CentralityStatistics{}

	// Calculate degree centrality statistics
	degreeCentrality, err := ga.calculateCentralityStats(ctx, "gds.degree.stats", entityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate degree centrality stats: %w", err)
	}
	stats.DegreeCentrality = degreeCentrality

	// Calculate betweenness centrality statistics
	betweennessCentrality, err := ga.calculateCentralityStats(ctx, "gds.betweenness.stats", entityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate betweenness centrality stats: %w", err)
	}
	stats.BetweennessCentrality = betweennessCentrality

	// Calculate closeness centrality statistics
	closenessCentrality, err := ga.calculateCentralityStats(ctx, "gds.closeness.stats", entityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate closeness centrality stats: %w", err)
	}
	stats.ClosenessCentrality = closenessCentrality

	// Calculate PageRank statistics
	pageRank, err := ga.calculateCentralityStats(ctx, "gds.pageRank.stats", entityTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate PageRank stats: %w", err)
	}
	stats.PageRank = pageRank

	return stats, nil
}

// calculateCentralityStats calculates statistics for a specific centrality measure
func (ga *GraphAnalytics) calculateCentralityStats(ctx context.Context, algorithm string, entityTypes []string) (*CentralityStats, error) {
	query := fmt.Sprintf(`
		CALL %s('myGraph', {
			nodeProjection: '%s',
			relationshipProjection: '*'
		})
		YIELD centralityDistribution
		RETURN centralityDistribution.mean as mean,
			   centralityDistribution.min as min,
			   centralityDistribution.max as max,
			   centralityDistribution.p50 as median,
			   centralityDistribution.stdDev as stdDev
	`, algorithm, entityTypes[0])

	records, err := ga.neo4jClient.ExecuteQuery(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no centrality statistics returned")
	}

	record := records[0]
	stats := &CentralityStats{
		Mean:   getFloat64(record, "mean"),
		Median: getFloat64(record, "median"),
		StdDev: getFloat64(record, "stdDev"),
		Min:    getFloat64(record, "min"),
		Max:    getFloat64(record, "max"),
	}

	return stats, nil
}

// findConnectedComponents finds connected components in the graph
func (ga *GraphAnalytics) findConnectedComponents(ctx context.Context, entityTypes []string) ([]*Component, error) {
	query := fmt.Sprintf(`
		CALL gds.wcc.stream('myGraph', {
			nodeProjection: '%s',
			relationshipProjection: '*'
		})
		YIELD nodeId, componentId
		RETURN componentId, COUNT(nodeId) as size, COLLECT(nodeId) as nodes
		ORDER BY size DESC
	`, entityTypes[0])

	records, err := ga.neo4jClient.ExecuteQuery(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	components := make([]*Component, 0)
	for i, record := range records {
		componentID := fmt.Sprintf("component_%d", i)
		
		size, ok := record["size"].(int64)
		if !ok {
			continue
		}

		component := &Component{
			ID:      componentID,
			Size:    int(size),
			IsGiant: i == 0, // Largest component is the giant component
		}

		// Calculate component density
		if component.Size > 1 {
			// This would require additional queries to count edges within the component
			component.Density = 0.5 // Placeholder
		}

		components = append(components, component)
	}

	return components, nil
}

// calculateNetworkLevelMetrics calculates network-level metrics
func (ga *GraphAnalytics) calculateNetworkLevelMetrics(ctx context.Context, metrics *NetworkMetrics, entityTypes []string) error {
	// Calculate clustering coefficient
	clusteringQuery := fmt.Sprintf(`
		CALL gds.localClusteringCoefficient.stats('myGraph', {
			nodeProjection: '%s',
			relationshipProjection: '*'
		})
		YIELD averageClusteringCoefficient
		RETURN averageClusteringCoefficient
	`, entityTypes[0])

	records, err := ga.neo4jClient.ExecuteQuery(ctx, clusteringQuery, nil)
	if err == nil && len(records) > 0 {
		metrics.Clustering = getFloat64(records[0], "averageClusteringCoefficient")
	}

	// Calculate average shortest path length
	shortestPathQuery := fmt.Sprintf(`
		CALL gds.allShortestPaths.stats('myGraph', {
			nodeProjection: '%s',
			relationshipProjection: '*'
		})
		YIELD relationshipCount, nodeCount
		RETURN relationshipCount, nodeCount
	`, entityTypes[0])

	pathRecords, err := ga.neo4jClient.ExecuteQuery(ctx, shortestPathQuery, nil)
	if err == nil && len(pathRecords) > 0 {
		relationshipCount := getFloat64(pathRecords[0], "relationshipCount")
		nodeCount := getFloat64(pathRecords[0], "nodeCount")
		if nodeCount > 0 {
			metrics.AveragePathLength = relationshipCount / nodeCount
		}
	}

	return nil
}

// DetectCommunities performs community detection using specified algorithm
func (ga *GraphAnalytics) DetectCommunities(ctx context.Context, req *CommunityDetectionRequest) (*CommunityDetectionResult, error) {
	startTime := time.Now()

	ga.logger.Info("Starting community detection",
		"algorithm", req.Algorithm,
		"entity_count", len(req.EntityIDs))

	var query string
	var params map[string]interface{}

	switch req.Algorithm {
	case AlgorithmLouvain:
		query, params = ga.buildLouvainQuery(req)
	case AlgorithmLabelProp:
		query, params = ga.buildLabelPropagationQuery(req)
	case AlgorithmLeiden:
		query, params = ga.buildLeidenQuery(req)
	default:
		return nil, fmt.Errorf("unsupported community detection algorithm: %s", req.Algorithm)
	}

	records, err := ga.neo4jClient.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute community detection query: %w", err)
	}

	result := &CommunityDetectionResult{
		Algorithm:      req.Algorithm,
		ProcessingTime: time.Since(startTime),
	}

	communities := ga.buildCommunitiesFromResults(records)
	result.Communities = communities
	result.NumCommunities = len(communities)

	// Calculate statistics
	if len(communities) > 0 {
		sizes := make([]int, len(communities))
		for i, community := range communities {
			sizes[i] = community.Size
		}
		sort.Ints(sizes)
		result.LargestCommunity = sizes[len(sizes)-1]
		result.SmallestCommunity = sizes[0]
	}

	ga.logger.Info("Community detection completed",
		"algorithm", req.Algorithm,
		"communities_found", result.NumCommunities,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// buildLouvainQuery builds a Louvain community detection query
func (ga *GraphAnalytics) buildLouvainQuery(req *CommunityDetectionRequest) (string, map[string]interface{}) {
	query := `
		CALL gds.louvain.stream('myGraph', {
			maxIterations: $maxIterations,
			tolerance: $tolerance
		})
		YIELD nodeId, communityId, intermediateCommunityIds
		RETURN communityId, COUNT(nodeId) as size, COLLECT(nodeId) as members
		ORDER BY size DESC
	`

	params := map[string]interface{}{
		"maxIterations": 10,
		"tolerance":     0.0001,
	}

	if val, ok := req.Parameters["max_iterations"]; ok {
		params["maxIterations"] = val
	}

	if val, ok := req.Parameters["tolerance"]; ok {
		params["tolerance"] = val
	}

	return query, params
}

// buildLabelPropagationQuery builds a label propagation community detection query
func (ga *GraphAnalytics) buildLabelPropagationQuery(req *CommunityDetectionRequest) (string, map[string]interface{}) {
	query := `
		CALL gds.labelPropagation.stream('myGraph', {
			maxIterations: $maxIterations
		})
		YIELD nodeId, communityId
		RETURN communityId, COUNT(nodeId) as size, COLLECT(nodeId) as members
		ORDER BY size DESC
	`

	params := map[string]interface{}{
		"maxIterations": 10,
	}

	if val, ok := req.Parameters["max_iterations"]; ok {
		params["maxIterations"] = val
	}

	return query, params
}

// buildLeidenQuery builds a Leiden community detection query
func (ga *GraphAnalytics) buildLeidenQuery(req *CommunityDetectionRequest) (string, map[string]interface{}) {
	query := `
		CALL gds.leiden.stream('myGraph', {
			maxLevels: $maxLevels,
			gamma: $gamma,
			theta: $theta
		})
		YIELD nodeId, communityId
		RETURN communityId, COUNT(nodeId) as size, COLLECT(nodeId) as members
		ORDER BY size DESC
	`

	params := map[string]interface{}{
		"maxLevels": 10,
		"gamma":     1.0,
		"theta":     0.01,
	}

	if val, ok := req.Parameters["max_levels"]; ok {
		params["maxLevels"] = val
	}

	if val, ok := req.Parameters["gamma"]; ok {
		params["gamma"] = val
	}

	if val, ok := req.Parameters["theta"]; ok {
		params["theta"] = val
	}

	return query, params
}

// buildCommunitiesFromResults builds community objects from query results
func (ga *GraphAnalytics) buildCommunitiesFromResults(records []map[string]interface{}) []*Community {
	communities := make([]*Community, 0)

	for i, record := range records {
		communityID := fmt.Sprintf("community_%d", i)
		
		size, ok := record["size"].(int64)
		if !ok {
			continue
		}

		community := &Community{
			ID:   communityID,
			Size: int(size),
		}

		// Calculate community metrics
		community.Density = ga.calculateCommunityDensity(community)
		community.RiskScore = ga.calculateCommunityRiskScore(community)

		communities = append(communities, community)
	}

	return communities
}

// AnalyzePaths performs comprehensive path analysis
func (ga *GraphAnalytics) AnalyzePaths(ctx context.Context, req *PathAnalysisRequest) (*PathAnalysisResult, error) {
	startTime := time.Now()

	ga.logger.Info("Starting path analysis",
		"source_id", req.SourceID,
		"target_id", req.TargetID,
		"max_depth", req.MaxDepth)

	var query string
	var params map[string]interface{}

	if req.TargetID != "" {
		// Analyze paths between specific source and target
		query, params = ga.buildSpecificPathQuery(req)
	} else {
		// Analyze all paths from source
		query, params = ga.buildAllPathsQuery(req)
	}

	records, err := ga.neo4jClient.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute path analysis query: %w", err)
	}

	result := &PathAnalysisResult{
		ProcessingTime: time.Since(startTime),
	}

	// Build paths from results
	paths := ga.buildPathsFromResults(records)
	result.Paths = paths

	// Calculate path statistics
	if len(paths) > 0 {
		result.ShortestPath = ga.findShortestPath(paths)
		result.ShortestDistance = result.ShortestPath.Length
		result.AverageDistance = ga.calculateAverageDistance(paths)
		result.PathDiversity = ga.calculatePathDiversity(paths)
	}

	ga.logger.Info("Path analysis completed",
		"paths_found", len(paths),
		"shortest_distance", result.ShortestDistance,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// AnalyzeInfluence performs influence analysis on the network
func (ga *GraphAnalytics) AnalyzeInfluence(ctx context.Context, req *InfluenceAnalysisRequest) (*InfluenceAnalysisResult, error) {
	startTime := time.Now()

	ga.logger.Info("Starting influence analysis",
		"entity_count", len(req.EntityIDs),
		"influence_type", req.InfluenceType,
		"max_depth", req.MaxDepth)

	// Build influence analysis query based on type
	query, params := ga.buildInfluenceQuery(req)

	records, err := ga.neo4jClient.ExecuteQuery(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute influence analysis query: %w", err)
	}

	result := &InfluenceAnalysisResult{
		InfluenceScores: make(map[string]float64),
		ProcessingTime:  time.Since(startTime),
	}

	// Process influence scores
	totalInfluence := 0.0
	rankings := make([]*InfluenceRanking, 0)

	for _, record := range records {
		entityID, ok := record["entityId"].(string)
		if !ok {
			continue
		}

		score := getFloat64(record, "influenceScore")
		result.InfluenceScores[entityID] = score
		totalInfluence += score

		rankings = append(rankings, &InfluenceRanking{
			EntityID:       entityID,
			InfluenceScore: score,
		})
	}

	// Sort by influence score and assign ranks
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].InfluenceScore > rankings[j].InfluenceScore
	})

	for i, ranking := range rankings {
		ranking.Rank = i + 1
	}

	result.TotalInfluence = totalInfluence
	result.TopInfluencers = rankings

	ga.logger.Info("Influence analysis completed",
		"entities_analyzed", len(result.InfluenceScores),
		"total_influence", result.TotalInfluence,
		"processing_time", result.ProcessingTime)

	return result, nil
}

// Helper methods

func (ga *GraphAnalytics) buildSpecificPathQuery(req *PathAnalysisRequest) (string, map[string]interface{}) {
	query := `
		MATCH path = shortestPath((source {id: $sourceId})-[*1..$maxDepth]-(target {id: $targetId}))
		RETURN path,
			   length(path) as pathLength,
			   [r IN relationships(path) | r.amount] as amounts
		ORDER BY pathLength
		LIMIT $maxPaths
	`

	params := map[string]interface{}{
		"sourceId": req.SourceID,
		"targetId": req.TargetID,
		"maxDepth": req.MaxDepth,
		"maxPaths": req.MaxPaths,
	}

	return query, params
}

func (ga *GraphAnalytics) buildAllPathsQuery(req *PathAnalysisRequest) (string, map[string]interface{}) {
	query := `
		MATCH path = (source {id: $sourceId})-[*1..$maxDepth]-(target)
		WHERE source <> target
		RETURN path,
			   length(path) as pathLength,
			   target.id as targetId
		ORDER BY pathLength
		LIMIT $maxPaths
	`

	params := map[string]interface{}{
		"sourceId": req.SourceID,
		"maxDepth": req.MaxDepth,
		"maxPaths": req.MaxPaths,
	}

	return query, params
}

func (ga *GraphAnalytics) buildInfluenceQuery(req *InfluenceAnalysisRequest) (string, map[string]interface{}) {
	// Use PageRank algorithm for influence analysis
	query := `
		CALL gds.pageRank.stream('myGraph', {
			maxIterations: 20,
			dampingFactor: 0.85,
			sourceNodes: $entityIds
		})
		YIELD nodeId, score
		RETURN nodeId as entityId, score as influenceScore
		ORDER BY influenceScore DESC
	`

	params := map[string]interface{}{
		"entityIds": req.EntityIDs,
	}

	return query, params
}

func (ga *GraphAnalytics) buildPathsFromResults(records []map[string]interface{}) []*neo4j.Path {
	paths := make([]*neo4j.Path, 0)
	
	for _, record := range records {
		pathLength, ok := record["pathLength"].(int64)
		if !ok {
			continue
		}

		path := &neo4j.Path{
			Length: int(pathLength),
		}

		// Extract additional path information from record
		// This would be implemented based on the actual Neo4j record structure

		paths = append(paths, path)
	}

	return paths
}

func (ga *GraphAnalytics) findShortestPath(paths []*neo4j.Path) *neo4j.Path {
	if len(paths) == 0 {
		return nil
	}

	shortest := paths[0]
	for _, path := range paths {
		if path.Length < shortest.Length {
			shortest = path
		}
	}

	return shortest
}

func (ga *GraphAnalytics) calculateAverageDistance(paths []*neo4j.Path) float64 {
	if len(paths) == 0 {
		return 0
	}

	total := 0
	for _, path := range paths {
		total += path.Length
	}

	return float64(total) / float64(len(paths))
}

func (ga *GraphAnalytics) calculatePathDiversity(paths []*neo4j.Path) float64 {
	if len(paths) <= 1 {
		return 0
	}

	// Calculate diversity based on path length distribution
	lengths := make(map[int]int)
	for _, path := range paths {
		lengths[path.Length]++
	}

	// Shannon diversity index
	total := float64(len(paths))
	diversity := 0.0

	for _, count := range lengths {
		p := float64(count) / total
		if p > 0 {
			diversity -= p * math.Log2(p)
		}
	}

	return diversity
}

func (ga *GraphAnalytics) calculateCommunityDensity(community *Community) float64 {
	if community.Size <= 1 {
		return 0
	}

	// This would require additional query to count edges within the community
	// For now, return a placeholder value
	return 0.5
}

func (ga *GraphAnalytics) calculateCommunityRiskScore(community *Community) float64 {
	// Calculate risk score based on community characteristics
	riskScore := 0.0

	// Size factor
	if community.Size > 50 {
		riskScore += 30
	} else if community.Size > 20 {
		riskScore += 20
	} else if community.Size > 10 {
		riskScore += 10
	}

	// Density factor
	if community.Density > 0.8 {
		riskScore += 20
	} else if community.Density > 0.6 {
		riskScore += 15
	} else if community.Density > 0.4 {
		riskScore += 10
	}

	return math.Min(riskScore, 100.0)
}

// getFloat64 safely extracts a float64 value from a record
func getFloat64(record map[string]interface{}, key string) float64 {
	if val, ok := record[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int64:
			return float64(v)
		case int:
			return float64(v)
		}
	}
	return 0.0
}