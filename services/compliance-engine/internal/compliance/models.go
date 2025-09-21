package compliance

import (
	"time"
)

// Rule represents a compliance rule
type Rule struct {
	ID          string                 `json:"id" bson:"_id"`
	Name        string                 `json:"name" bson:"name"`
	Type        string                 `json:"type" bson:"type"`
	Severity    string                 `json:"severity" bson:"severity"`
	Description string                 `json:"description" bson:"description"`
	Enabled     bool                   `json:"enabled" bson:"enabled"`
	Parameters  map[string]interface{} `json:"parameters" bson:"parameters"`
	Tags        []string               `json:"tags" bson:"tags"`
	Category    string                 `json:"category" bson:"category"`
	Version     string                 `json:"version" bson:"version"`
	CreatedAt   time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" bson:"updated_at"`
	CreatedBy   string                 `json:"created_by" bson:"created_by"`
	UpdatedBy   string                 `json:"updated_by" bson:"updated_by"`
}

// RuleResult represents the result of a rule evaluation
type RuleResult struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Severity    string                 `json:"severity"`
	Passed      bool                   `json:"passed"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
	Duration    time.Duration          `json:"duration"`
}

// Violation represents a compliance violation
type Violation struct {
	ID              string                   `json:"id" bson:"_id"`
	RuleID          string                   `json:"rule_id" bson:"rule_id"`
	RuleName        string                   `json:"rule_name" bson:"rule_name"`
	Severity        string                   `json:"severity" bson:"severity"`
	Status          string                   `json:"status" bson:"status"` // open, investigating, resolved, false_positive
	Description     string                   `json:"description" bson:"description"`
	Details         map[string]interface{}   `json:"details" bson:"details"`
	EntityID        string                   `json:"entity_id" bson:"entity_id"`
	EntityType      string                   `json:"entity_type" bson:"entity_type"`
	RiskScore       float64                  `json:"risk_score" bson:"risk_score"`
	AssignedTo      string                   `json:"assigned_to" bson:"assigned_to"`
	Comments        []ViolationComment       `json:"comments" bson:"comments"`
	StatusHistory   []ViolationStatusChange  `json:"status_history" bson:"status_history"`
	EscalatedAt     time.Time                `json:"escalated_at" bson:"escalated_at"`
	EscalationLevel int                      `json:"escalation_level" bson:"escalation_level"`
	CreatedAt       time.Time                `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at" bson:"updated_at"`
	ResolvedAt      time.Time                `json:"resolved_at" bson:"resolved_at"`
}

// ViolationComment represents a comment on a violation
type ViolationComment struct {
	ID        string    `json:"id" bson:"id"`
	Author    string    `json:"author" bson:"author"`
	Content   string    `json:"content" bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// ViolationStatusChange represents a status change event
type ViolationStatusChange struct {
	FromStatus string    `json:"from_status" bson:"from_status"`
	ToStatus   string    `json:"to_status" bson:"to_status"`
	ChangedBy  string    `json:"changed_by" bson:"changed_by"`
	ChangedAt  time.Time `json:"changed_at" bson:"changed_at"`
	Notes      string    `json:"notes" bson:"notes"`
}

// ViolationStatistics represents violation statistics
type ViolationStatistics struct {
	TotalViolations    int                `json:"total_violations"`
	StatusCounts       map[string]int     `json:"status_counts"`
	SeverityCounts     map[string]int     `json:"severity_counts"`
	RuleCounts         map[string]int     `json:"rule_counts"`
	AverageRiskScore   float64            `json:"average_risk_score"`
	TrendData          []ViolationTrend   `json:"trend_data"`
	GeneratedAt        time.Time          `json:"generated_at"`
}

// ViolationTrend represents violation trend data
type ViolationTrend struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

// EscalationRule represents an escalation rule for violations
type EscalationRule struct {
	ID         string                 `json:"id" bson:"_id"`
	Name       string                 `json:"name" bson:"name"`
	Conditions map[string]interface{} `json:"conditions" bson:"conditions"`
	Actions    []string               `json:"actions" bson:"actions"`
	Delay      time.Duration          `json:"delay" bson:"delay"`
	MaxRetries int                    `json:"max_retries" bson:"max_retries"`
	Enabled    bool                   `json:"enabled" bson:"enabled"`
	CreatedAt  time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" bson:"updated_at"`
}

// ComplianceResult represents the overall compliance evaluation result
type ComplianceResult struct {
	ID            string                 `json:"id"`
	EntityID      string                 `json:"entity_id"`
	EntityType    string                 `json:"entity_type"`
	RulesApplied  int                    `json:"rules_applied"`
	RulesPassed   int                    `json:"rules_passed"`
	RulesFailed   int                    `json:"rules_failed"`
	OverallStatus string                 `json:"overall_status"` // compliant, non_compliant, warning
	RiskScore     float64                `json:"risk_score"`
	Violations    []string               `json:"violations"` // Violation IDs
	RuleResults   []RuleResult           `json:"rule_results"`
	Details       map[string]interface{} `json:"details"`
	EvaluatedAt   time.Time              `json:"evaluated_at"`
	Duration      time.Duration          `json:"duration"`
}

// RegulationInfo represents information about a regulation
type RegulationInfo struct {
	ID          string            `json:"id" bson:"_id"`
	Name        string            `json:"name" bson:"name"`
	Jurisdiction string           `json:"jurisdiction" bson:"jurisdiction"`
	Type        string            `json:"type" bson:"type"` // federal, state, international
	Version     string            `json:"version" bson:"version"`
	EffectiveDate time.Time       `json:"effective_date" bson:"effective_date"`
	UpdatedAt   time.Time         `json:"updated_at" bson:"updated_at"`
	Source      string            `json:"source" bson:"source"`
	URL         string            `json:"url" bson:"url"`
	Summary     string            `json:"summary" bson:"summary"`
	Requirements []string         `json:"requirements" bson:"requirements"`
	Tags        []string          `json:"tags" bson:"tags"`
	Metadata    map[string]interface{} `json:"metadata" bson:"metadata"`
}

// RegulationChange represents a change in regulation
type RegulationChange struct {
	ID           string    `json:"id" bson:"_id"`
	RegulationID string    `json:"regulation_id" bson:"regulation_id"`
	ChangeType   string    `json:"change_type" bson:"change_type"` // amendment, repeal, new
	Description  string    `json:"description" bson:"description"`
	EffectiveDate time.Time `json:"effective_date" bson:"effective_date"`
	DetectedAt   time.Time `json:"detected_at" bson:"detected_at"`
	Source       string    `json:"source" bson:"source"`
	Impact       string    `json:"impact" bson:"impact"` // high, medium, low
	Status       string    `json:"status" bson:"status"` // pending, reviewed, implemented
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string                 `json:"id" bson:"_id"`
	EventType string                 `json:"event_type" bson:"event_type"`
	Category  string                 `json:"category" bson:"category"`
	UserID    string                 `json:"user_id" bson:"user_id"`
	EntityID  string                 `json:"entity_id" bson:"entity_id"`
	EntityType string                `json:"entity_type" bson:"entity_type"`
	Action    string                 `json:"action" bson:"action"`
	Details   map[string]interface{} `json:"details" bson:"details"`
	Timestamp time.Time              `json:"timestamp" bson:"timestamp"`
	IPAddress string                 `json:"ip_address" bson:"ip_address"`
	UserAgent string                 `json:"user_agent" bson:"user_agent"`
	Result    string                 `json:"result" bson:"result"` // success, failure, warning
}

// Report represents a compliance report
type Report struct {
	ID           string                 `json:"id" bson:"_id"`
	Name         string                 `json:"name" bson:"name"`
	Type         string                 `json:"type" bson:"type"` // regulatory, internal, audit
	Status       string                 `json:"status" bson:"status"` // pending, generating, completed, failed
	Format       string                 `json:"format" bson:"format"` // pdf, excel, csv, json
	TemplateID   string                 `json:"template_id" bson:"template_id"`
	Parameters   map[string]interface{} `json:"parameters" bson:"parameters"`
	Content      []byte                 `json:"content" bson:"content"`
	FilePath     string                 `json:"file_path" bson:"file_path"`
	GeneratedBy  string                 `json:"generated_by" bson:"generated_by"`
	GeneratedAt  time.Time              `json:"generated_at" bson:"generated_at"`
	ScheduledFor time.Time              `json:"scheduled_for" bson:"scheduled_for"`
	Recipients   []string               `json:"recipients" bson:"recipients"`
	Metadata     map[string]interface{} `json:"metadata" bson:"metadata"`
}

// ReportTemplate represents a report template
type ReportTemplate struct {
	ID          string                 `json:"id" bson:"_id"`
	Name        string                 `json:"name" bson:"name"`
	Description string                 `json:"description" bson:"description"`
	Type        string                 `json:"type" bson:"type"`
	Format      string                 `json:"format" bson:"format"`
	Template    string                 `json:"template" bson:"template"`
	Parameters  []TemplateParameter    `json:"parameters" bson:"parameters"`
	Enabled     bool                   `json:"enabled" bson:"enabled"`
	CreatedAt   time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" bson:"updated_at"`
	CreatedBy   string                 `json:"created_by" bson:"created_by"`
}

// TemplateParameter represents a template parameter
type TemplateParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, number, date, boolean
	Required    bool        `json:"required"`
	DefaultValue interface{} `json:"default_value"`
	Description string      `json:"description"`
}

// ReportSchedule represents a scheduled report
type ReportSchedule struct {
	ID          string                 `json:"id" bson:"_id"`
	Name        string                 `json:"name" bson:"name"`
	TemplateID  string                 `json:"template_id" bson:"template_id"`
	Frequency   string                 `json:"frequency" bson:"frequency"` // daily, weekly, monthly, quarterly
	Parameters  map[string]interface{} `json:"parameters" bson:"parameters"`
	Recipients  []string               `json:"recipients" bson:"recipients"`
	NextRun     time.Time              `json:"next_run" bson:"next_run"`
	LastRun     time.Time              `json:"last_run" bson:"last_run"`
	Enabled     bool                   `json:"enabled" bson:"enabled"`
	CreatedAt   time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" bson:"updated_at"`
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID       string `json:"id" bson:"_id"`
	Name     string `json:"name" bson:"name"`
	Type     string `json:"type" bson:"type"` // email, sms, webhook
	Subject  string `json:"subject" bson:"subject"`
	Content  string `json:"content" bson:"content"`
	Format   string `json:"format" bson:"format"` // text, html
	Enabled  bool   `json:"enabled" bson:"enabled"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// ComplianceMetrics represents compliance metrics
type ComplianceMetrics struct {
	TotalRulesEvaluated    int64     `json:"total_rules_evaluated"`
	TotalViolations        int64     `json:"total_violations"`
	ViolationsByDate       map[string]int64 `json:"violations_by_date"`
	ViolationsBySeverity   map[string]int64 `json:"violations_by_severity"`
	ViolationsByStatus     map[string]int64 `json:"violations_by_status"`
	AverageResolutionTime  float64   `json:"average_resolution_time_hours"`
	ComplianceScore        float64   `json:"compliance_score"`
	TrendDirection         string    `json:"trend_direction"` // improving, declining, stable
	LastUpdated            time.Time `json:"last_updated"`
}

// Event types for the compliance engine
const (
	EventTypeRuleEvaluated    = "rule_evaluated"
	EventTypeViolationCreated = "violation_created"
	EventTypeViolationUpdated = "violation_updated"
	EventTypeViolationResolved = "violation_resolved"
	EventTypeReportGenerated  = "report_generated"
	EventTypeRegulationUpdated = "regulation_updated"
)

// Violation statuses
const (
	ViolationStatusOpen          = "open"
	ViolationStatusInvestigating = "investigating"
	ViolationStatusResolved      = "resolved"
	ViolationStatusFalsePositive = "false_positive"
	ViolationStatusEscalated     = "escalated"
)

// Rule severities
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
)

// Compliance statuses
const (
	ComplianceStatusCompliant    = "compliant"
	ComplianceStatusNonCompliant = "non_compliant"
	ComplianceStatusWarning      = "warning"
	ComplianceStatusUnknown      = "unknown"
)

// Report formats
const (
	ReportFormatPDF   = "pdf"
	ReportFormatExcel = "excel"
	ReportFormatCSV   = "csv"
	ReportFormatJSON  = "json"
	ReportFormatXML   = "xml"
)

// Report types
const (
	ReportTypeRegulatory = "regulatory"
	ReportTypeInternal   = "internal"
	ReportTypeAudit      = "audit"
	ReportTypeViolation  = "violation"
	ReportTypeMetrics    = "metrics"
)