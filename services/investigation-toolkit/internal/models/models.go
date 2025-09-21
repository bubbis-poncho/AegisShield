package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Investigation represents an investigation case
type Investigation struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	Title          string         `json:"title" db:"title" validate:"required,min=1,max=255"`
	Description    *string        `json:"description,omitempty" db:"description"`
	CaseType       CaseType       `json:"case_type" db:"case_type" validate:"required"`
	Priority       Priority       `json:"priority" db:"priority" validate:"required"`
	Status         Status         `json:"status" db:"status" validate:"required"`
	AssignedTo     *uuid.UUID     `json:"assigned_to,omitempty" db:"assigned_to"`
	CreatedBy      uuid.UUID      `json:"created_by" db:"created_by" validate:"required"`
	ExternalCaseID *string        `json:"external_case_id,omitempty" db:"external_case_id"`
	Tags           pq.StringArray `json:"tags" db:"tags"`
	Metadata       JSONB          `json:"metadata" db:"metadata"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
	DueDate        *time.Time     `json:"due_date,omitempty" db:"due_date"`
	ClosedAt       *time.Time     `json:"closed_at,omitempty" db:"closed_at"`
	ArchivedAt     *time.Time     `json:"archived_at,omitempty" db:"archived_at"`
}

// Evidence represents a piece of evidence in an investigation
type Evidence struct {
	ID                   uuid.UUID      `json:"id" db:"id"`
	InvestigationID      uuid.UUID      `json:"investigation_id" db:"investigation_id" validate:"required"`
	Name                 string         `json:"name" db:"name" validate:"required,min=1,max=255"`
	Description          *string        `json:"description,omitempty" db:"description"`
	EvidenceType         EvidenceType   `json:"evidence_type" db:"evidence_type" validate:"required"`
	Source               *string        `json:"source,omitempty" db:"source"`
	CollectionMethod     *string        `json:"collection_method,omitempty" db:"collection_method"`
	FilePath             *string        `json:"file_path,omitempty" db:"file_path"`
	FileSize             *int64         `json:"file_size,omitempty" db:"file_size"`
	FileHash             *string        `json:"file_hash,omitempty" db:"file_hash"`
	MimeType             *string        `json:"mime_type,omitempty" db:"mime_type"`
	CollectedBy          uuid.UUID      `json:"collected_by" db:"collected_by" validate:"required"`
	CollectedAt          time.Time      `json:"collected_at" db:"collected_at"`
	ChainOfCustody       JSONB          `json:"chain_of_custody" db:"chain_of_custody"`
	Metadata             JSONB          `json:"metadata" db:"metadata"`
	Tags                 pq.StringArray `json:"tags" db:"tags"`
	IsAuthenticated      bool           `json:"is_authenticated" db:"is_authenticated"`
	AuthenticationMethod *string        `json:"authentication_method,omitempty" db:"authentication_method"`
	AuthenticationDate   *time.Time     `json:"authentication_date,omitempty" db:"authentication_date"`
	AuthenticationBy     *uuid.UUID     `json:"authentication_by,omitempty" db:"authentication_by"`
	RetentionDate        *time.Time     `json:"retention_date,omitempty" db:"retention_date"`
	Status               EvidenceStatus `json:"status" db:"status" validate:"required"`
	CreatedAt            time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at" db:"updated_at"`
}

// Timeline represents a timeline event in an investigation
type Timeline struct {
	ID                  uuid.UUID         `json:"id" db:"id"`
	InvestigationID     uuid.UUID         `json:"investigation_id" db:"investigation_id" validate:"required"`
	Title               string            `json:"title" db:"title" validate:"required,min=1,max=255"`
	Description         *string           `json:"description,omitempty" db:"description"`
	EventType           EventType         `json:"event_type" db:"event_type" validate:"required"`
	EventDate           time.Time         `json:"event_date" db:"event_date" validate:"required"`
	DurationMinutes     *int              `json:"duration_minutes,omitempty" db:"duration_minutes"`
	Location            *string           `json:"location,omitempty" db:"location"`
	Participants        pq.StringArray    `json:"participants" db:"participants"`
	RelatedEvidenceIDs  UUIDArray         `json:"related_evidence_ids" db:"related_evidence_ids"`
	ExternalReferences  JSONB             `json:"external_references" db:"external_references"`
	Metadata            JSONB             `json:"metadata" db:"metadata"`
	Tags                pq.StringArray    `json:"tags" db:"tags"`
	CreatedBy           uuid.UUID         `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt           time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at" db:"updated_at"`
}

// Collaboration represents user collaboration on an investigation
type Collaboration struct {
	ID             uuid.UUID `json:"id" db:"id"`
	InvestigationID uuid.UUID `json:"investigation_id" db:"investigation_id" validate:"required"`
	UserID         uuid.UUID `json:"user_id" db:"user_id" validate:"required"`
	Role           Role      `json:"role" db:"role" validate:"required"`
	Permissions    JSONB     `json:"permissions" db:"permissions"`
	AssignedBy     uuid.UUID `json:"assigned_by" db:"assigned_by" validate:"required"`
	AssignedAt     time.Time `json:"assigned_at" db:"assigned_at"`
	RemovedAt      *time.Time `json:"removed_at,omitempty" db:"removed_at"`
	RemovedBy      *uuid.UUID `json:"removed_by,omitempty" db:"removed_by"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	Notes          *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// CollaborationComment represents a comment in an investigation
type CollaborationComment struct {
	ID               uuid.UUID     `json:"id" db:"id"`
	InvestigationID  uuid.UUID     `json:"investigation_id" db:"investigation_id" validate:"required"`
	ParentCommentID  *uuid.UUID    `json:"parent_comment_id,omitempty" db:"parent_comment_id"`
	UserID           uuid.UUID     `json:"user_id" db:"user_id" validate:"required"`
	Content          string        `json:"content" db:"content" validate:"required,min=1"`
	CommentType      CommentType   `json:"comment_type" db:"comment_type" validate:"required"`
	MentionedUsers   UUIDArray     `json:"mentioned_users" db:"mentioned_users"`
	Attachments      JSONB         `json:"attachments" db:"attachments"`
	IsInternal       bool          `json:"is_internal" db:"is_internal"`
	Metadata         JSONB         `json:"metadata" db:"metadata"`
	CreatedAt        time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at" db:"updated_at"`
	EditedAt         *time.Time    `json:"edited_at,omitempty" db:"edited_at"`
	DeletedAt        *time.Time    `json:"deleted_at,omitempty" db:"deleted_at"`
}

// Workflow represents a workflow definition or instance
type Workflow struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	Name           string        `json:"name" db:"name" validate:"required,min=1,max=255"`
	Description    *string       `json:"description,omitempty" db:"description"`
	WorkflowType   WorkflowType  `json:"workflow_type" db:"workflow_type" validate:"required"`
	TemplateID     *uuid.UUID    `json:"template_id,omitempty" db:"template_id"`
	InvestigationID *uuid.UUID   `json:"investigation_id,omitempty" db:"investigation_id"`
	Definition     JSONB         `json:"definition" db:"definition" validate:"required"`
	CurrentStep    *string       `json:"current_step,omitempty" db:"current_step"`
	Status         WorkflowStatus `json:"status" db:"status" validate:"required"`
	StartedAt      *time.Time    `json:"started_at,omitempty" db:"started_at"`
	CompletedAt    *time.Time    `json:"completed_at,omitempty" db:"completed_at"`
	Variables      JSONB         `json:"variables" db:"variables"`
	CreatedBy      uuid.UUID     `json:"created_by" db:"created_by" validate:"required"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
}

// WorkflowStep represents a workflow step execution
type WorkflowStep struct {
	ID           uuid.UUID         `json:"id" db:"id"`
	WorkflowID   uuid.UUID         `json:"workflow_id" db:"workflow_id" validate:"required"`
	StepName     string            `json:"step_name" db:"step_name" validate:"required,min=1,max=100"`
	StepType     StepType          `json:"step_type" db:"step_type" validate:"required"`
	Status       StepStatus        `json:"status" db:"status" validate:"required"`
	AssignedTo   *uuid.UUID        `json:"assigned_to,omitempty" db:"assigned_to"`
	StartedAt    *time.Time        `json:"started_at,omitempty" db:"started_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty" db:"completed_at"`
	DueDate      *time.Time        `json:"due_date,omitempty" db:"due_date"`
	InputData    JSONB             `json:"input_data" db:"input_data"`
	OutputData   JSONB             `json:"output_data" db:"output_data"`
	ErrorMessage *string           `json:"error_message,omitempty" db:"error_message"`
	RetryCount   int               `json:"retry_count" db:"retry_count"`
	MaxRetries   int               `json:"max_retries" db:"max_retries"`
	Notes        *string           `json:"notes,omitempty" db:"notes"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

// WorkflowStepHistory represents the history of workflow step changes
type WorkflowStepHistory struct {
	ID             uuid.UUID `json:"id" db:"id"`
	WorkflowStepID uuid.UUID `json:"workflow_step_id" db:"workflow_step_id" validate:"required"`
	Action         string    `json:"action" db:"action" validate:"required"`
	PreviousStatus *string   `json:"previous_status,omitempty" db:"previous_status"`
	NewStatus      *string   `json:"new_status,omitempty" db:"new_status"`
	PerformedBy    uuid.UUID `json:"performed_by" db:"performed_by" validate:"required"`
	Reason         *string   `json:"reason,omitempty" db:"reason"`
	Metadata       JSONB     `json:"metadata" db:"metadata"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	InvestigationID *uuid.UUID `json:"investigation_id,omitempty" db:"investigation_id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id" validate:"required"`
	Action         string     `json:"action" db:"action" validate:"required,min=1,max=100"`
	ResourceType   string     `json:"resource_type" db:"resource_type" validate:"required,min=1,max=50"`
	ResourceID     *uuid.UUID `json:"resource_id,omitempty" db:"resource_id"`
	OldValues      JSONB      `json:"old_values,omitempty" db:"old_values"`
	NewValues      JSONB      `json:"new_values,omitempty" db:"new_values"`
	IPAddress      *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent      *string    `json:"user_agent,omitempty" db:"user_agent"`
	SessionID      *string    `json:"session_id,omitempty" db:"session_id"`
	RequestID      *string    `json:"request_id,omitempty" db:"request_id"`
	Endpoint       *string    `json:"endpoint,omitempty" db:"endpoint"`
	HTTPMethod     *string    `json:"http_method,omitempty" db:"http_method"`
	ResponseStatus *int       `json:"response_status,omitempty" db:"response_status"`
	DurationMS     *int       `json:"duration_ms,omitempty" db:"duration_ms"`
	Metadata       JSONB      `json:"metadata" db:"metadata"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// Enum types
type CaseType string

const (
	CaseTypeFraud           CaseType = "fraud"
	CaseTypeMoneyLaundering CaseType = "money_laundering"
	CaseTypeSanctions       CaseType = "sanctions"
	CaseTypeKYC             CaseType = "kyc"
	CaseTypeOther           CaseType = "other"
)

type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusUnderReview Status = "under_review"
	StatusClosed     Status = "closed"
	StatusArchived   Status = "archived"
)

type EvidenceType string

const (
	EvidenceTypeDocument      EvidenceType = "document"
	EvidenceTypeImage         EvidenceType = "image"
	EvidenceTypeVideo         EvidenceType = "video"
	EvidenceTypeAudio         EvidenceType = "audio"
	EvidenceTypeTransaction   EvidenceType = "transaction"
	EvidenceTypeCommunication EvidenceType = "communication"
	EvidenceTypeDigital       EvidenceType = "digital"
	EvidenceTypePhysical      EvidenceType = "physical"
	EvidenceTypeOther         EvidenceType = "other"
)

type EvidenceStatus string

const (
	EvidenceStatusActive        EvidenceStatus = "active"
	EvidenceStatusUnderReview   EvidenceStatus = "under_review"
	EvidenceStatusAuthenticated EvidenceStatus = "authenticated"
	EvidenceStatusRejected      EvidenceStatus = "rejected"
	EvidenceStatusArchived      EvidenceStatus = "archived"
)

type EventType string

const (
	EventTypeTransaction         EventType = "transaction"
	EventTypeCommunication       EventType = "communication"
	EventTypeMeeting             EventType = "meeting"
	EventTypeDocument            EventType = "document"
	EventTypeInvestigationAction EventType = "investigation_action"
	EventTypeSystemEvent         EventType = "system_event"
	EventTypeOther               EventType = "other"
)

type Role string

const (
	RoleLeadInvestigator Role = "lead_investigator"
	RoleInvestigator     Role = "investigator"
	RoleAnalyst          Role = "analyst"
	RoleReviewer         Role = "reviewer"
	RoleObserver         Role = "observer"
	RoleConsultant       Role = "consultant"
)

type CommentType string

const (
	CommentTypeGeneral         CommentType = "general"
	CommentTypeQuestion        CommentType = "question"
	CommentTypeFinding         CommentType = "finding"
	CommentTypeRecommendation  CommentType = "recommendation"
	CommentTypeStatusUpdate    CommentType = "status_update"
	CommentTypeEvidenceComment CommentType = "evidence_comment"
)

type WorkflowType string

const (
	WorkflowTypeTemplate WorkflowType = "template"
	WorkflowTypeInstance WorkflowType = "instance"
)

type WorkflowStatus string

const (
	WorkflowStatusDraft     WorkflowStatus = "draft"
	WorkflowStatusActive    WorkflowStatus = "active"
	WorkflowStatusSuspended WorkflowStatus = "suspended"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

type StepType string

const (
	StepTypeManual         StepType = "manual"
	StepTypeAutomated      StepType = "automated"
	StepTypeApproval       StepType = "approval"
	StepTypeDecision       StepType = "decision"
	StepTypeNotification   StepType = "notification"
	StepTypeDataCollection StepType = "data_collection"
)

type StepStatus string

const (
	StepStatusPending    StepStatus = "pending"
	StepStatusInProgress StepStatus = "in_progress"
	StepStatusCompleted  StepStatus = "completed"
	StepStatusFailed     StepStatus = "failed"
	StepStatusSkipped    StepStatus = "skipped"
	StepStatusCancelled  StepStatus = "cancelled"
)

// Custom types for database handling
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return json.Unmarshal([]byte(value.(string)), j)
	}
	return json.Unmarshal(bytes, j)
}

type UUIDArray []uuid.UUID

func (u UUIDArray) Value() (driver.Value, error) {
	if u == nil {
		return nil, nil
	}
	
	strs := make([]string, len(u))
	for i, uid := range u {
		strs[i] = uid.String()
	}
	
	return pq.Array(strs).Value()
}

func (u *UUIDArray) Scan(value interface{}) error {
	if value == nil {
		*u = nil
		return nil
	}
	
	var strs pq.StringArray
	if err := strs.Scan(value); err != nil {
		return err
	}
	
	uuids := make([]uuid.UUID, len(strs))
	for i, str := range strs {
		uid, err := uuid.Parse(str)
		if err != nil {
			return err
		}
		uuids[i] = uid
	}
	
	*u = uuids
	return nil
}

// Request and Response DTOs
type CreateInvestigationRequest struct {
	Title          string                 `json:"title" validate:"required,min=1,max=255"`
	Description    *string                `json:"description,omitempty"`
	CaseType       CaseType               `json:"case_type" validate:"required"`
	Priority       Priority               `json:"priority" validate:"required"`
	AssignedTo     *uuid.UUID             `json:"assigned_to,omitempty"`
	ExternalCaseID *string                `json:"external_case_id,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	DueDate        *time.Time             `json:"due_date,omitempty"`
}

type UpdateInvestigationRequest struct {
	Title          *string                `json:"title,omitempty" validate:"omitempty,min=1,max=255"`
	Description    *string                `json:"description,omitempty"`
	CaseType       *CaseType              `json:"case_type,omitempty"`
	Priority       *Priority              `json:"priority,omitempty"`
	Status         *Status                `json:"status,omitempty"`
	AssignedTo     *uuid.UUID             `json:"assigned_to,omitempty"`
	ExternalCaseID *string                `json:"external_case_id,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	DueDate        *time.Time             `json:"due_date,omitempty"`
}

type CreateEvidenceRequest struct {
	Name                 string                 `json:"name" validate:"required,min=1,max=255"`
	Description          *string                `json:"description,omitempty"`
	EvidenceType         EvidenceType           `json:"evidence_type" validate:"required"`
	Source               *string                `json:"source,omitempty"`
	CollectionMethod     *string                `json:"collection_method,omitempty"`
	Tags                 []string               `json:"tags,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	AuthenticationMethod *string                `json:"authentication_method,omitempty"`
	RetentionDate        *time.Time             `json:"retention_date,omitempty"`
}

type CreateTimelineRequest struct {
	Title              string                 `json:"title" validate:"required,min=1,max=255"`
	Description        *string                `json:"description,omitempty"`
	EventType          EventType              `json:"event_type" validate:"required"`
	EventDate          time.Time              `json:"event_date" validate:"required"`
	DurationMinutes    *int                   `json:"duration_minutes,omitempty"`
	Location           *string                `json:"location,omitempty"`
	Participants       []string               `json:"participants,omitempty"`
	RelatedEvidenceIDs []uuid.UUID            `json:"related_evidence_ids,omitempty"`
	ExternalReferences map[string]interface{} `json:"external_references,omitempty"`
	Tags               []string               `json:"tags,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

type AddCollaboratorRequest struct {
	UserID      uuid.UUID              `json:"user_id" validate:"required"`
	Role        Role                   `json:"role" validate:"required"`
	Permissions map[string]interface{} `json:"permissions,omitempty"`
	Notes       *string                `json:"notes,omitempty"`
}

type CreateCommentRequest struct {
	ParentCommentID *uuid.UUID             `json:"parent_comment_id,omitempty"`
	Content         string                 `json:"content" validate:"required,min=1"`
	CommentType     CommentType            `json:"comment_type" validate:"required"`
	MentionedUsers  []uuid.UUID            `json:"mentioned_users,omitempty"`
	Attachments     map[string]interface{} `json:"attachments,omitempty"`
	IsInternal      bool                   `json:"is_internal"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Filter and search structs
type InvestigationFilter struct {
	CaseTypes    []CaseType `json:"case_types,omitempty"`
	Priorities   []Priority `json:"priorities,omitempty"`
	Statuses     []Status   `json:"statuses,omitempty"`
	AssignedTo   *uuid.UUID `json:"assigned_to,omitempty"`
	CreatedBy    *uuid.UUID `json:"created_by,omitempty"`
	CreatedAfter *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	DueAfter     *time.Time `json:"due_after,omitempty"`
	DueBefore    *time.Time `json:"due_before,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
	Search       *string    `json:"search,omitempty"`
}

type EvidenceFilter struct {
	EvidenceTypes   []EvidenceType   `json:"evidence_types,omitempty"`
	Statuses        []EvidenceStatus `json:"statuses,omitempty"`
	CollectedBy     *uuid.UUID       `json:"collected_by,omitempty"`
	CollectedAfter  *time.Time       `json:"collected_after,omitempty"`
	CollectedBefore *time.Time       `json:"collected_before,omitempty"`
	IsAuthenticated *bool            `json:"is_authenticated,omitempty"`
	Tags            []string         `json:"tags,omitempty"`
	Search          *string          `json:"search,omitempty"`
}