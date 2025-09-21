// Shared Data Models - T025
// Constitutional Principle: Data Integrity & Modular Code

package main

import (
	"time"
	"encoding/json"
)

// Core Entity Models
type Entity struct {
	ID          string            `json:"id" db:"id"`
	Type        EntityType        `json:"type" db:"type"`
	Name        string            `json:"name" db:"name"`
	Status      EntityStatus      `json:"status" db:"status"`
	RiskLevel   RiskLevel         `json:"risk_level" db:"risk_level"`
	RiskScore   float64           `json:"risk_score" db:"risk_score"`
	Attributes  map[string]string `json:"attributes" db:"attributes"`
	Addresses   []Address         `json:"addresses,omitempty"`
	ContactInfo []ContactInfo     `json:"contact_info,omitempty"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
	Version     int               `json:"version" db:"version"`
	Metadata    map[string]string `json:"metadata,omitempty" db:"metadata"`
}

type EntityStatus string

const (
	EntityStatusActive    EntityStatus = "ACTIVE"
	EntityStatusInactive  EntityStatus = "INACTIVE"
	EntityStatusSuspended EntityStatus = "SUSPENDED"
	EntityStatusBlocked   EntityStatus = "BLOCKED"
	EntityStatusDeleted   EntityStatus = "DELETED"
)

type Address struct {
	ID           string      `json:"id" db:"id"`
	EntityID     string      `json:"entity_id" db:"entity_id"`
	Type         AddressType `json:"type" db:"type"`
	Street       string      `json:"street" db:"street"`
	City         string      `json:"city" db:"city"`
	State        string      `json:"state" db:"state"`
	PostalCode   string      `json:"postal_code" db:"postal_code"`
	CountryCode  string      `json:"country_code" db:"country_code"`
	IsPrimary    bool        `json:"is_primary" db:"is_primary"`
	IsVerified   bool        `json:"is_verified" db:"is_verified"`
	VerifiedAt   *time.Time  `json:"verified_at,omitempty" db:"verified_at"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at" db:"updated_at"`
}

type AddressType string

const (
	AddressTypeHome     AddressType = "HOME"
	AddressTypeBusiness AddressType = "BUSINESS"
	AddressTypeMailing  AddressType = "MAILING"
	AddressTypeOther    AddressType = "OTHER"
)

type ContactInfo struct {
	ID         string      `json:"id" db:"id"`
	EntityID   string      `json:"entity_id" db:"entity_id"`
	Type       ContactType `json:"type" db:"type"`
	Value      string      `json:"value" db:"value"`
	IsPrimary  bool        `json:"is_primary" db:"is_primary"`
	IsVerified bool        `json:"is_verified" db:"is_verified"`
	VerifiedAt *time.Time  `json:"verified_at,omitempty" db:"verified_at"`
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at" db:"updated_at"`
}

type ContactType string

const (
	ContactTypeEmail ContactType = "EMAIL"
	ContactTypePhone ContactType = "PHONE"
	ContactTypeFax   ContactType = "FAX"
	ContactTypeOther ContactType = "OTHER"
)

// Transaction Models
type Transaction struct {
	ID                  string              `json:"id" db:"id"`
	ExternalID          string              `json:"external_id" db:"external_id"`
	Type                TransactionType     `json:"type" db:"type"`
	Status              TransactionStatus   `json:"status" db:"status"`
	Amount              float64             `json:"amount" db:"amount"`
	Currency            string              `json:"currency" db:"currency"`
	Description         string              `json:"description" db:"description"`
	FromEntity          string              `json:"from_entity" db:"from_entity"`
	ToEntity            string              `json:"to_entity" db:"to_entity"`
	FromAccount         string              `json:"from_account" db:"from_account"`
	ToAccount           string              `json:"to_account" db:"to_account"`
	PaymentMethod       PaymentMethod       `json:"payment_method" db:"payment_method"`
	ProcessedAt         *time.Time          `json:"processed_at,omitempty" db:"processed_at"`
	SettledAt           *time.Time          `json:"settled_at,omitempty" db:"settled_at"`
	RiskLevel           RiskLevel           `json:"risk_level" db:"risk_level"`
	RiskScore           float64             `json:"risk_score" db:"risk_score"`
	RiskFactors         []string            `json:"risk_factors,omitempty" db:"risk_factors"`
	ComplianceChecks    []ComplianceCheck   `json:"compliance_checks,omitempty"`
	GeographicData      *GeographicData     `json:"geographic_data,omitempty"`
	Metadata            map[string]string   `json:"metadata,omitempty" db:"metadata"`
	CreatedAt           time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at" db:"updated_at"`
	SourceSystem        string              `json:"source_system" db:"source_system"`
	BatchID             string              `json:"batch_id,omitempty" db:"batch_id"`
	ValidationErrors    []ValidationError   `json:"validation_errors,omitempty"`
}

type TransactionType string

const (
	TransactionTypeTransfer     TransactionType = "TRANSFER"
	TransactionTypeDeposit      TransactionType = "DEPOSIT"
	TransactionTypeWithdrawal   TransactionType = "WITHDRAWAL"
	TransactionTypePayment      TransactionType = "PAYMENT"
	TransactionTypeRefund       TransactionType = "REFUND"
	TransactionTypeExchange     TransactionType = "EXCHANGE"
	TransactionTypeTrade        TransactionType = "TRADE"
	TransactionTypeInvestment   TransactionType = "INVESTMENT"
	TransactionTypeLoan         TransactionType = "LOAN"
	TransactionTypeOther        TransactionType = "OTHER"
)

type ComplianceCheck struct {
	ID          string                `json:"id"`
	Type        ComplianceCheckType   `json:"type"`
	Status      ComplianceStatus      `json:"status"`
	Result      ComplianceResult      `json:"result"`
	Score       float64               `json:"score,omitempty"`
	Details     map[string]string     `json:"details,omitempty"`
	CheckedAt   time.Time             `json:"checked_at"`
	CheckedBy   string                `json:"checked_by"`
	Findings    []ComplianceFinding   `json:"findings,omitempty"`
}

type ComplianceCheckType string

const (
	ComplianceCheckTypeSanctions    ComplianceCheckType = "SANCTIONS"
	ComplianceCheckTypePEP          ComplianceCheckType = "PEP"
	ComplianceCheckTypeAdverseMedia ComplianceCheckType = "ADVERSE_MEDIA"
	ComplianceCheckTypeKYC          ComplianceCheckType = "KYC"
	ComplianceCheckTypeAML          ComplianceCheckType = "AML"
	ComplianceCheckTypeCFT          ComplianceCheckType = "CFT"
)

type ComplianceStatus string

const (
	ComplianceStatusPending    ComplianceStatus = "PENDING"
	ComplianceStatusInProgress ComplianceStatus = "IN_PROGRESS"
	ComplianceStatusCompleted  ComplianceStatus = "COMPLETED"
	ComplianceStatusFailed     ComplianceStatus = "FAILED"
	ComplianceStatusExpired    ComplianceStatus = "EXPIRED"
)

type ComplianceResult string

const (
	ComplianceResultPass ComplianceResult = "PASS"
	ComplianceResultFail ComplianceResult = "FAIL"
	ComplianceResultHit  ComplianceResult = "HIT"
	ComplianceResultReview ComplianceResult = "REVIEW"
)

type ComplianceFinding struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Severity    Severity          `json:"severity"`
	Description string            `json:"description"`
	Source      string            `json:"source"`
	MatchScore  float64           `json:"match_score,omitempty"`
	Details     map[string]string `json:"details,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

type GeographicData struct {
	FromCountry      string  `json:"from_country,omitempty"`
	ToCountry        string  `json:"to_country,omitempty"`
	FromRegion       string  `json:"from_region,omitempty"`
	ToRegion         string  `json:"to_region,omitempty"`
	FromCoordinates  *LatLng `json:"from_coordinates,omitempty"`
	ToCoordinates    *LatLng `json:"to_coordinates,omitempty"`
	Distance         float64 `json:"distance,omitempty"`
	IsHighRiskRoute  bool    `json:"is_high_risk_route"`
	RiskFactors      []string `json:"risk_factors,omitempty"`
}

type LatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Alert Models
type Alert struct {
	ID               string            `json:"id" db:"id"`
	RuleID           string            `json:"rule_id" db:"rule_id"`
	RuleName         string            `json:"rule_name" db:"rule_name"`
	TransactionID    string            `json:"transaction_id,omitempty" db:"transaction_id"`
	EntityID         string            `json:"entity_id,omitempty" db:"entity_id"`
	Status           AlertStatus       `json:"status" db:"status"`
	Severity         Severity          `json:"severity" db:"severity"`
	Title            string            `json:"title" db:"title"`
	Description      string            `json:"description" db:"description"`
	RiskScore        float64           `json:"risk_score" db:"risk_score"`
	AlertData        map[string]string `json:"alert_data,omitempty" db:"alert_data"`
	Evidence         []AlertEvidence   `json:"evidence,omitempty"`
	ActionsTaken     []AlertAction     `json:"actions_taken,omitempty"`
	AssignedTo       string            `json:"assigned_to,omitempty" db:"assigned_to"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at" db:"updated_at"`
	ClosedAt         *time.Time        `json:"closed_at,omitempty" db:"closed_at"`
	CreatedBySystem  string            `json:"created_by_system" db:"created_by_system"`
	Metadata         map[string]string `json:"metadata,omitempty" db:"metadata"`
	Comments         []AlertComment    `json:"comments,omitempty"`
}

type AlertStatus string

const (
	AlertStatusOpen              AlertStatus = "OPEN"
	AlertStatusInProgress        AlertStatus = "IN_PROGRESS"
	AlertStatusEscalated         AlertStatus = "ESCALATED"
	AlertStatusClosedTruePositive AlertStatus = "CLOSED_TRUE_POSITIVE"
	AlertStatusClosedFalsePositive AlertStatus = "CLOSED_FALSE_POSITIVE"
	AlertStatusClosedBenign      AlertStatus = "CLOSED_BENIGN"
	AlertStatusSuppressed        AlertStatus = "SUPPRESSED"
	AlertStatusPendingReview     AlertStatus = "PENDING_REVIEW"
)

type AlertEvidence struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Description   string            `json:"description"`
	RelevanceScore float64          `json:"relevance_score"`
	Data          map[string]string `json:"data,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

type AlertAction struct {
	ID          string            `json:"id"`
	Type        AlertActionType   `json:"type"`
	Status      ActionStatus      `json:"status"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	ExecutedAt  *time.Time        `json:"executed_at,omitempty"`
	ExecutedBy  string            `json:"executed_by,omitempty"`
	Result      string            `json:"result,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

type AlertActionType string

const (
	AlertActionTypeCreateAlert       AlertActionType = "CREATE_ALERT"
	AlertActionTypeSendEmail         AlertActionType = "SEND_EMAIL"
	AlertActionTypeSendSMS           AlertActionType = "SEND_SMS"
	AlertActionTypeWebhookCall       AlertActionType = "WEBHOOK_CALL"
	AlertActionTypeBlockTransaction  AlertActionType = "BLOCK_TRANSACTION"
	AlertActionTypeFlagEntity        AlertActionType = "FLAG_ENTITY"
	AlertActionTypeCreateCase        AlertActionType = "CREATE_CASE"
	AlertActionTypeEscalateToHuman   AlertActionType = "ESCALATE_TO_HUMAN"
	AlertActionTypeLogEvent          AlertActionType = "LOG_EVENT"
	AlertActionTypeUpdateRiskScore   AlertActionType = "UPDATE_RISK_SCORE"
)

type ActionStatus string

const (
	ActionStatusPending   ActionStatus = "PENDING"
	ActionStatusExecuting ActionStatus = "EXECUTING"
	ActionStatusCompleted ActionStatus = "COMPLETED"
	ActionStatusFailed    ActionStatus = "FAILED"
	ActionStatusSkipped   ActionStatus = "SKIPPED"
)

type AlertComment struct {
	ID        string      `json:"id" db:"id"`
	AlertID   string      `json:"alert_id" db:"alert_id"`
	Comment   string      `json:"comment" db:"comment"`
	Author    string      `json:"author" db:"author"`
	Type      CommentType `json:"type" db:"type"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
}

type CommentType string

const (
	CommentTypeInvestigationNote CommentType = "INVESTIGATION_NOTE"
	CommentTypeResolutionNote    CommentType = "RESOLUTION_NOTE"
	CommentTypeEscalationNote    CommentType = "ESCALATION_NOTE"
	CommentTypeSystemNote        CommentType = "SYSTEM_NOTE"
)

// Graph Models
type GraphNode struct {
	ID               string            `json:"id"`
	Labels           []string          `json:"labels"`
	Properties       map[string]string `json:"properties"`
	Degree           int               `json:"degree,omitempty"`
	CentralityScore  float64           `json:"centrality_score,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type GraphRelationship struct {
	ID           string                  `json:"id"`
	SourceNodeID string                  `json:"source_node_id"`
	TargetNodeID string                  `json:"target_node_id"`
	Type         string                  `json:"type"`
	Properties   map[string]string       `json:"properties"`
	Weight       float64                 `json:"weight,omitempty"`
	Direction    RelationshipDirection   `json:"direction"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

type RelationshipDirection string

const (
	RelationshipDirectionIncoming    RelationshipDirection = "INCOMING"
	RelationshipDirectionOutgoing    RelationshipDirection = "OUTGOING"
	RelationshipDirectionUndirected  RelationshipDirection = "UNDIRECTED"
)

type GraphPath struct {
	NodeIDs         []string  `json:"node_ids"`
	RelationshipIDs []string  `json:"relationship_ids"`
	Length          int       `json:"length"`
	Weight          float64   `json:"weight,omitempty"`
	Type            PathType  `json:"type"`
}

type PathType string

const (
	PathTypeShortestPath     PathType = "SHORTEST_PATH"
	PathTypeAllPaths         PathType = "ALL_PATHS"
	PathTypeWeightedPath     PathType = "WEIGHTED_PATH"
	PathTypeTransactionChain PathType = "TRANSACTION_CHAIN"
)

// Risk Assessment Models
type RiskAssessment struct {
	EntityID        string                `json:"entity_id"`
	TransactionID   string                `json:"transaction_id,omitempty"`
	OverallRiskLevel RiskLevel            `json:"overall_risk_level"`
	RiskScore       float64               `json:"risk_score"`
	RiskFactors     []RiskFactor          `json:"risk_factors"`
	Recommendations []string              `json:"recommendations,omitempty"`
	AssessedAt      time.Time             `json:"assessed_at"`
	AssessedBy      string                `json:"assessed_by"`
	Version         string                `json:"version"`
	Methodology     RiskMethodology       `json:"methodology"`
}

type RiskFactor struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Category    string      `json:"category"`
	Severity    Severity    `json:"severity"`
	Score       float64     `json:"score"`
	Weight      float64     `json:"weight"`
	Description string      `json:"description"`
	Evidence    []string    `json:"evidence,omitempty"`
	Source      string      `json:"source"`
	DetectedAt  time.Time   `json:"detected_at"`
}

type RiskMethodology struct {
	Algorithm     string            `json:"algorithm"`
	Version       string            `json:"version"`
	Parameters    map[string]string `json:"parameters"`
	ModelType     string            `json:"model_type"`
	TrainingDate  *time.Time        `json:"training_date,omitempty"`
	Accuracy      float64           `json:"accuracy,omitempty"`
}

// Data Quality Models
type DataQualityReport struct {
	ID              string                 `json:"id"`
	DatasetID       string                 `json:"dataset_id"`
	SourceSystem    string                 `json:"source_system"`
	OverallScore    float64                `json:"overall_score"`
	DimensionScores map[string]float64     `json:"dimension_scores"`
	Issues          []DataQualityIssue     `json:"issues"`
	Recommendations []string               `json:"recommendations"`
	GeneratedAt     time.Time              `json:"generated_at"`
	GeneratedBy     string                 `json:"generated_by"`
	RecordCount     int                    `json:"record_count"`
	IssueCount      int                    `json:"issue_count"`
}

type DataQualityIssue struct {
	ID               string    `json:"id"`
	Type             string    `json:"type"`
	Severity         Severity  `json:"severity"`
	FieldName        string    `json:"field_name"`
	Description      string    `json:"description"`
	OccurrenceCount  int       `json:"occurrence_count"`
	SampleValues     []string  `json:"sample_values,omitempty"`
	SuggestedFix     string    `json:"suggested_fix,omitempty"`
	BusinessImpact   string    `json:"business_impact,omitempty"`
	DetectedAt       time.Time `json:"detected_at"`
}

// Audit Models
type AuditRecord struct {
	ID           string            `json:"id" db:"id"`
	EntityID     string            `json:"entity_id,omitempty" db:"entity_id"`
	ActorID      string            `json:"actor_id" db:"actor_id"`
	ActorType    string            `json:"actor_type" db:"actor_type"`
	Action       string            `json:"action" db:"action"`
	Resource     string            `json:"resource" db:"resource"`
	ResourceID   string            `json:"resource_id,omitempty" db:"resource_id"`
	Result       AuditResult       `json:"result" db:"result"`
	BeforeState  map[string]string `json:"before_state,omitempty" db:"before_state"`
	AfterState   map[string]string `json:"after_state,omitempty" db:"after_state"`
	Reason       string            `json:"reason,omitempty" db:"reason"`
	IPAddress    string            `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent    string            `json:"user_agent,omitempty" db:"user_agent"`
	SessionID    string            `json:"session_id,omitempty" db:"session_id"`
	Timestamp    time.Time         `json:"timestamp" db:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty" db:"metadata"`
}

type AuditResult string

const (
	AuditResultSuccess        AuditResult = "SUCCESS"
	AuditResultFailure        AuditResult = "FAILURE"
	AuditResultPartialSuccess AuditResult = "PARTIAL_SUCCESS"
	AuditResultUnauthorized   AuditResult = "UNAUTHORIZED"
	AuditResultForbidden      AuditResult = "FORBIDDEN"
)

// Validation Models
type ValidationError struct {
	Field       string          `json:"field"`
	Code        string          `json:"code"`
	Message     string          `json:"message"`
	Value       interface{}     `json:"value,omitempty"`
	Severity    Severity        `json:"severity"`
	Context     map[string]string `json:"context,omitempty"`
}

type ValidationResult struct {
	IsValid bool              `json:"is_valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

// Common Time Range Model
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// Pagination Models
type PaginationRequest struct {
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
	Cursor   string `json:"cursor,omitempty"`
	SortBy   string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

type PaginationResponse struct {
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
	TotalCount int    `json:"total_count"`
	TotalPages int    `json:"total_pages,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// API Response Models
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Metadata  interface{} `json:"metadata,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
	Trace   string            `json:"trace,omitempty"`
}

// Configuration Models
type ServiceConfiguration struct {
	ServiceName    string                 `json:"service_name"`
	Version        string                 `json:"version"`
	Environment    string                 `json:"environment"`
	Config         map[string]interface{} `json:"config"`
	LastUpdated    time.Time              `json:"last_updated"`
	UpdatedBy      string                 `json:"updated_by"`
}

// Health Check Models
type HealthStatus struct {
	Service      string                 `json:"service"`
	Status       string                 `json:"status"`
	Timestamp    time.Time              `json:"timestamp"`
	Version      string                 `json:"version"`
	Dependencies []DependencyHealth     `json:"dependencies,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
}

type DependencyHealth struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	ResponseTime float64   `json:"response_time_ms"`
	LastChecked  time.Time `json:"last_checked"`
	Error        string    `json:"error,omitempty"`
}

// Metrics Models
type MetricPoint struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Job/Task Models
type JobStatus struct {
	JobID       string            `json:"job_id"`
	Type        string            `json:"type"`
	Status      string            `json:"status"`
	Progress    float64           `json:"progress"`
	Message     string            `json:"message,omitempty"`
	StartedAt   time.Time         `json:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	CreatedBy   string            `json:"created_by"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Errors      []ValidationError `json:"errors,omitempty"`
}