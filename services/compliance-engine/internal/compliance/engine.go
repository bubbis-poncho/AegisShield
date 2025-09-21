package compliance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aegisshield/compliance-engine/internal/config"
	"go.uber.org/zap"
)

// ComplianceEngine manages compliance monitoring and rule enforcement
type ComplianceEngine struct {
	config         config.ComplianceConfig
	logger         *zap.Logger
	ruleEngine     *RuleEngine
	monitor        *ComplianceMonitor
	violations     *ViolationManager
	regulations    *RegulationManager
	dataRetention  *DataRetentionManager
	running        bool
	mu             sync.RWMutex
}

// NewComplianceEngine creates a new compliance engine instance
func NewComplianceEngine(cfg config.ComplianceConfig, logger *zap.Logger) *ComplianceEngine {
	return &ComplianceEngine{
		config:        cfg,
		logger:        logger,
		ruleEngine:    NewRuleEngine(cfg.RulesEngine, logger),
		monitor:       NewComplianceMonitor(cfg.Monitoring, logger),
		violations:    NewViolationManager(cfg.ViolationHandling, logger),
		regulations:   NewRegulationManager(cfg.Regulations, logger),
		dataRetention: NewDataRetentionManager(cfg.DataRetention, logger),
	}
}

// Start starts the compliance engine
func (e *ComplianceEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("compliance engine is already running")
	}

	e.logger.Info("Starting compliance engine")

	// Start sub-components
	if err := e.ruleEngine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start rule engine: %w", err)
	}

	if err := e.monitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start compliance monitor: %w", err)
	}

	if err := e.violations.Start(ctx); err != nil {
		return fmt.Errorf("failed to start violation manager: %w", err)
	}

	if err := e.regulations.Start(ctx); err != nil {
		return fmt.Errorf("failed to start regulation manager: %w", err)
	}

	if err := e.dataRetention.Start(ctx); err != nil {
		return fmt.Errorf("failed to start data retention manager: %w", err)
	}

	e.running = true
	e.logger.Info("Compliance engine started successfully")

	return nil
}

// Stop stops the compliance engine
func (e *ComplianceEngine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	e.logger.Info("Stopping compliance engine")

	// Stop sub-components
	if err := e.dataRetention.Stop(ctx); err != nil {
		e.logger.Error("Failed to stop data retention manager", zap.Error(err))
	}

	if err := e.regulations.Stop(ctx); err != nil {
		e.logger.Error("Failed to stop regulation manager", zap.Error(err))
	}

	if err := e.violations.Stop(ctx); err != nil {
		e.logger.Error("Failed to stop violation manager", zap.Error(err))
	}

	if err := e.monitor.Stop(ctx); err != nil {
		e.logger.Error("Failed to stop compliance monitor", zap.Error(err))
	}

	if err := e.ruleEngine.Stop(ctx); err != nil {
		e.logger.Error("Failed to stop rule engine", zap.Error(err))
	}

	e.running = false
	e.logger.Info("Compliance engine stopped")

	return nil
}

// EvaluateCompliance evaluates compliance for a given entity or transaction
func (e *ComplianceEngine) EvaluateCompliance(ctx context.Context, data interface{}) (*ComplianceResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return nil, fmt.Errorf("compliance engine is not running")
	}

	// Get applicable rules
	rules, err := e.ruleEngine.GetApplicableRules(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to get applicable rules: %w", err)
	}

	// Evaluate rules
	ruleResults := make([]RuleResult, 0, len(rules))
	violations := make([]Violation, 0)

	for _, rule := range rules {
		result, err := e.ruleEngine.EvaluateRule(ctx, rule, data)
		if err != nil {
			e.logger.Error("Failed to evaluate rule", 
				zap.String("rule_id", rule.ID), 
				zap.Error(err))
			continue
		}

		ruleResults = append(ruleResults, *result)

		if !result.Passed {
			violation := Violation{
				ID:          generateViolationID(),
				RuleID:      rule.ID,
				Severity:    rule.Severity,
				Description: result.Description,
				Data:        data,
				Timestamp:   time.Now(),
				Status:      "detected",
			}
			violations = append(violations, violation)
		}
	}

	// Create compliance result
	complianceResult := &ComplianceResult{
		EntityID:      extractEntityID(data),
		Timestamp:     time.Now(),
		RuleResults:   ruleResults,
		Violations:    violations,
		OverallStatus: calculateOverallStatus(ruleResults),
		RiskScore:     calculateRiskScore(violations),
		Recommendations: generateRecommendations(violations),
	}

	// Process violations if any
	if len(violations) > 0 {
		if err := e.violations.ProcessViolations(ctx, violations); err != nil {
			e.logger.Error("Failed to process violations", zap.Error(err))
		}
	}

	return complianceResult, nil
}

// GetComplianceStatus returns the current compliance status
func (e *ComplianceEngine) GetComplianceStatus(ctx context.Context) (*ComplianceStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return nil, fmt.Errorf("compliance engine is not running")
	}

	// Get status from sub-components
	ruleEngineStatus := e.ruleEngine.GetStatus()
	monitorStatus := e.monitor.GetStatus()
	violationStatus := e.violations.GetStatus()

	status := &ComplianceStatus{
		Timestamp:         time.Now(),
		EngineStatus:      "running",
		RuleEngineStatus:  ruleEngineStatus,
		MonitorStatus:     monitorStatus,
		ViolationStatus:   violationStatus,
		ActiveRules:       e.ruleEngine.GetActiveRuleCount(),
		TotalViolations:   e.violations.GetTotalViolationCount(),
		PendingViolations: e.violations.GetPendingViolationCount(),
	}

	return status, nil
}

// UpdateRules updates the compliance rules
func (e *ComplianceEngine) UpdateRules(ctx context.Context, rules []Rule) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return fmt.Errorf("compliance engine is not running")
	}

	return e.ruleEngine.UpdateRules(ctx, rules)
}

// GetViolations retrieves violations based on criteria
func (e *ComplianceEngine) GetViolations(ctx context.Context, criteria ViolationCriteria) ([]Violation, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return nil, fmt.Errorf("compliance engine is not running")
	}

	return e.violations.GetViolations(ctx, criteria)
}

// AcknowledgeViolation acknowledges a violation
func (e *ComplianceEngine) AcknowledgeViolation(ctx context.Context, violationID string, userID string, notes string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return fmt.Errorf("compliance engine is not running")
	}

	return e.violations.AcknowledgeViolation(ctx, violationID, userID, notes)
}

// GetComplianceMetrics returns compliance metrics
func (e *ComplianceEngine) GetComplianceMetrics(ctx context.Context, timeRange TimeRange) (*ComplianceMetrics, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return nil, fmt.Errorf("compliance engine is not running")
	}

	return e.monitor.GetComplianceMetrics(ctx, timeRange)
}

// ComplianceResult represents the result of a compliance evaluation
type ComplianceResult struct {
	EntityID        string           `json:"entity_id"`
	Timestamp       time.Time        `json:"timestamp"`
	RuleResults     []RuleResult     `json:"rule_results"`
	Violations      []Violation      `json:"violations"`
	OverallStatus   string           `json:"overall_status"`
	RiskScore       float64          `json:"risk_score"`
	Recommendations []Recommendation `json:"recommendations"`
}

// ComplianceStatus represents the current status of the compliance engine
type ComplianceStatus struct {
	Timestamp         time.Time `json:"timestamp"`
	EngineStatus      string    `json:"engine_status"`
	RuleEngineStatus  string    `json:"rule_engine_status"`
	MonitorStatus     string    `json:"monitor_status"`
	ViolationStatus   string    `json:"violation_status"`
	ActiveRules       int       `json:"active_rules"`
	TotalViolations   int       `json:"total_violations"`
	PendingViolations int       `json:"pending_violations"`
}

// ComplianceMetrics represents compliance metrics
type ComplianceMetrics struct {
	TimeRange              TimeRange                    `json:"time_range"`
	TotalEvaluations       int64                        `json:"total_evaluations"`
	PassedEvaluations      int64                        `json:"passed_evaluations"`
	FailedEvaluations      int64                        `json:"failed_evaluations"`
	ComplianceRate         float64                      `json:"compliance_rate"`
	ViolationsByType       map[string]int               `json:"violations_by_type"`
	ViolationsBySeverity   map[string]int               `json:"violations_by_severity"`
	ViolationTrends        []ViolationTrendPoint        `json:"violation_trends"`
	RiskScoreDistribution  map[string]int               `json:"risk_score_distribution"`
	TopViolatedRules       []RuleViolationSummary       `json:"top_violated_rules"`
	ComplianceByRegulation map[string]ComplianceByRegion `json:"compliance_by_regulation"`
}

// ViolationTrendPoint represents a point in violation trend data
type ViolationTrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int       `json:"count"`
	Severity  string    `json:"severity"`
}

// RuleViolationSummary represents a summary of rule violations
type RuleViolationSummary struct {
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Count       int    `json:"count"`
	Severity    string `json:"severity"`
	LastOccurred time.Time `json:"last_occurred"`
}

// ComplianceByRegion represents compliance statistics by regulation
type ComplianceByRegion struct {
	Regulation     string  `json:"regulation"`
	ComplianceRate float64 `json:"compliance_rate"`
	ViolationCount int     `json:"violation_count"`
	LastUpdated    time.Time `json:"last_updated"`
}

// Recommendation represents a compliance recommendation
type Recommendation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Priority    string    `json:"priority"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Actions     []string  `json:"actions"`
	CreatedAt   time.Time `json:"created_at"`
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Helper functions

func extractEntityID(data interface{}) string {
	// Extract entity ID from data based on type
	// This is a simplified implementation
	if dataMap, ok := data.(map[string]interface{}); ok {
		if id, exists := dataMap["entity_id"]; exists {
			if idStr, ok := id.(string); ok {
				return idStr
			}
		}
		if id, exists := dataMap["id"]; exists {
			if idStr, ok := id.(string); ok {
				return idStr
			}
		}
	}
	return fmt.Sprintf("unknown_%d", time.Now().Unix())
}

func calculateOverallStatus(results []RuleResult) string {
	if len(results) == 0 {
		return "unknown"
	}

	hasCritical := false
	hasViolation := false

	for _, result := range results {
		if !result.Passed {
			hasViolation = true
			if result.Severity == "critical" {
				hasCritical = true
			}
		}
	}

	if hasCritical {
		return "critical_violation"
	}
	if hasViolation {
		return "violation"
	}
	return "compliant"
}

func calculateRiskScore(violations []Violation) float64 {
	if len(violations) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, violation := range violations {
		switch violation.Severity {
		case "critical":
			totalScore += 10.0
		case "high":
			totalScore += 7.0
		case "medium":
			totalScore += 4.0
		case "low":
			totalScore += 1.0
		default:
			totalScore += 2.0
		}
	}

	// Normalize to 0-100 scale
	maxPossibleScore := float64(len(violations)) * 10.0
	if maxPossibleScore == 0 {
		return 0.0
	}

	return (totalScore / maxPossibleScore) * 100.0
}

func generateRecommendations(violations []Violation) []Recommendation {
	recommendations := make([]Recommendation, 0)
	
	// Group violations by type and generate recommendations
	violationTypes := make(map[string][]Violation)
	for _, violation := range violations {
		violationType := extractViolationType(violation)
		violationTypes[violationType] = append(violationTypes[violationType], violation)
	}

	for violationType, typeViolations := range violationTypes {
		recommendation := generateRecommendationForType(violationType, typeViolations)
		recommendations = append(recommendations, recommendation)
	}

	return recommendations
}

func extractViolationType(violation Violation) string {
	// Extract violation type from description or rule ID
	// This is a simplified implementation
	if violation.RuleID != "" {
		return violation.RuleID
	}
	return "general"
}

func generateRecommendationForType(violationType string, violations []Violation) Recommendation {
	// Generate specific recommendations based on violation type
	// This is a simplified implementation
	return Recommendation{
		ID:          generateRecommendationID(),
		Type:        violationType,
		Priority:    determinePriority(violations),
		Title:       fmt.Sprintf("Address %s violations", violationType),
		Description: fmt.Sprintf("Found %d violations of type %s", len(violations), violationType),
		Actions:     generateActions(violationType, violations),
		CreatedAt:   time.Now(),
	}
}

func determinePriority(violations []Violation) string {
	for _, violation := range violations {
		if violation.Severity == "critical" {
			return "high"
		}
	}
	for _, violation := range violations {
		if violation.Severity == "high" {
			return "medium"
		}
	}
	return "low"
}

func generateActions(violationType string, violations []Violation) []string {
	// Generate specific actions based on violation type
	// This is a simplified implementation
	actions := []string{
		"Review affected transactions",
		"Update compliance procedures",
		"Notify relevant stakeholders",
	}

	if len(violations) > 10 {
		actions = append(actions, "Implement automated controls")
	}

	return actions
}

func generateViolationID() string {
	return fmt.Sprintf("viol_%d", time.Now().UnixNano())
}

func generateRecommendationID() string {
	return fmt.Sprintf("rec_%d", time.Now().UnixNano())
}