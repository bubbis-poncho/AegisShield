package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/aegis-shield/shared/proto"
	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
	"github.com/aegis-shield/services/alerting-engine/internal/engine"
	"github.com/aegis-shield/services/alerting-engine/internal/kafka"
	"github.com/aegis-shield/services/alerting-engine/internal/notification"
)

// GRPCServer implements the alerting engine gRPC service
type GRPCServer struct {
	pb.UnimplementedAlertingEngineServer
	config           *config.Config
	logger           *slog.Logger
	alertRepo        *database.AlertRepository
	ruleRepo         *database.RuleRepository
	notificationRepo *database.NotificationRepository
	escalationRepo   *database.EscalationRepository
	ruleEngine       *engine.RuleEngine
	notificationMgr  *notification.Manager
	eventProcessor   *kafka.EventProcessor
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(
	cfg *config.Config,
	logger *slog.Logger,
	alertRepo *database.AlertRepository,
	ruleRepo *database.RuleRepository,
	notificationRepo *database.NotificationRepository,
	escalationRepo *database.EscalationRepository,
	ruleEngine *engine.RuleEngine,
	notificationMgr *notification.Manager,
	eventProcessor *kafka.EventProcessor,
) *GRPCServer {
	return &GRPCServer{
		config:           cfg,
		logger:           logger,
		alertRepo:        alertRepo,
		ruleRepo:         ruleRepo,
		notificationRepo: notificationRepo,
		escalationRepo:   escalationRepo,
		ruleEngine:       ruleEngine,
		notificationMgr:  notificationMgr,
		eventProcessor:   eventProcessor,
	}
}

// Alert Management Operations

// CreateAlert creates a new alert
func (s *GRPCServer) CreateAlert(ctx context.Context, req *pb.CreateAlertRequest) (*pb.CreateAlertResponse, error) {
	s.logger.Info("Creating alert", "title", req.Title, "severity", req.Severity)

	// Validate request
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.Severity == "" {
		return nil, status.Error(codes.InvalidArgument, "severity is required")
	}

	// Create alert
	alert := &database.Alert{
		ID:          generateID("alert"),
		RuleID:      req.RuleId,
		Title:       req.Title,
		Description: req.Description,
		Severity:    req.Severity,
		Type:        req.Type,
		Priority:    req.Priority,
		Status:      "active",
		Source:      req.Source,
		CreatedBy:   req.CreatedBy,
		UpdatedBy:   req.CreatedBy,
	}

	// Set defaults
	if alert.Type == "" {
		alert.Type = "manual"
	}
	if alert.Priority == "" {
		alert.Priority = "medium"
	}
	if alert.Source == "" {
		alert.Source = "api"
	}

	// Add event data if provided
	if len(req.EventData) > 0 {
		eventData, err := json.Marshal(req.EventData)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid event data")
		}
		alert.EventData = eventData
	}

	// Add metadata if provided
	if len(req.Metadata) > 0 {
		metadata, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid metadata")
		}
		alert.Metadata = metadata
	}

	// Save alert
	if err := s.alertRepo.Create(ctx, alert); err != nil {
		s.logger.Error("Failed to create alert", "error", err)
		return nil, status.Error(codes.Internal, "failed to create alert")
	}

	s.logger.Info("Alert created", "alert_id", alert.ID)

	// Convert to proto and return
	pbAlert := s.alertToProto(alert)
	return &pb.CreateAlertResponse{Alert: pbAlert}, nil
}

// GetAlert retrieves an alert by ID
func (s *GRPCServer) GetAlert(ctx context.Context, req *pb.GetAlertRequest) (*pb.GetAlertResponse, error) {
	if req.AlertId == "" {
		return nil, status.Error(codes.InvalidArgument, "alert_id is required")
	}

	alert, err := s.alertRepo.GetByID(ctx, req.AlertId)
	if err != nil {
		s.logger.Error("Failed to get alert", "alert_id", req.AlertId, "error", err)
		return nil, status.Error(codes.NotFound, "alert not found")
	}

	pbAlert := s.alertToProto(alert)
	return &pb.GetAlertResponse{Alert: pbAlert}, nil
}

// ListAlerts lists alerts with filtering and pagination
func (s *GRPCServer) ListAlerts(ctx context.Context, req *pb.ListAlertsRequest) (*pb.ListAlertsResponse, error) {
	// Build filter
	filter := database.Filter{
		Limit:  int(req.PageSize),
		Offset: int(req.PageToken),
		Filters: make(map[string]interface{}),
	}

	if req.RuleId != "" {
		filter.Filters["rule_id"] = req.RuleId
	}
	if req.Severity != "" {
		filter.Filters["severity"] = req.Severity
	}
	if req.Status != "" {
		filter.Filters["status"] = req.Status
	}
	if req.Type != "" {
		filter.Filters["type"] = req.Type
	}
	if req.Source != "" {
		filter.Filters["source"] = req.Source
	}

	// Date filters
	if req.StartTime != nil {
		startTime := req.StartTime.AsTime()
		filter.DateFrom = &startTime
	}
	if req.EndTime != nil {
		endTime := req.EndTime.AsTime()
		filter.DateTo = &endTime
	}

	// Get alerts
	alerts, total, err := s.alertRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list alerts", "error", err)
		return nil, status.Error(codes.Internal, "failed to list alerts")
	}

	// Convert to proto
	pbAlerts := make([]*pb.Alert, len(alerts))
	for i, alert := range alerts {
		pbAlerts[i] = s.alertToProto(alert)
	}

	return &pb.ListAlertsResponse{
		Alerts:        pbAlerts,
		TotalCount:    int32(total),
		NextPageToken: int32(filter.Offset + len(alerts)),
	}, nil
}

// UpdateAlert updates an alert
func (s *GRPCServer) UpdateAlert(ctx context.Context, req *pb.UpdateAlertRequest) (*pb.UpdateAlertResponse, error) {
	if req.AlertId == "" {
		return nil, status.Error(codes.InvalidArgument, "alert_id is required")
	}

	// Get current alert
	alert, err := s.alertRepo.GetByID(ctx, req.AlertId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "alert not found")
	}

	// Update fields
	if req.Title != "" {
		alert.Title = req.Title
	}
	if req.Description != "" {
		alert.Description = req.Description
	}
	if req.Severity != "" {
		alert.Severity = req.Severity
	}
	if req.Priority != "" {
		alert.Priority = req.Priority
	}
	if req.UpdatedBy != "" {
		alert.UpdatedBy = req.UpdatedBy
	}

	// Update metadata if provided
	if len(req.Metadata) > 0 {
		metadata, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid metadata")
		}
		alert.Metadata = metadata
	}

	// Save alert
	if err := s.alertRepo.Update(ctx, alert); err != nil {
		s.logger.Error("Failed to update alert", "alert_id", req.AlertId, "error", err)
		return nil, status.Error(codes.Internal, "failed to update alert")
	}

	pbAlert := s.alertToProto(alert)
	return &pb.UpdateAlertResponse{Alert: pbAlert}, nil
}

// AcknowledgeAlert acknowledges an alert
func (s *GRPCServer) AcknowledgeAlert(ctx context.Context, req *pb.AcknowledgeAlertRequest) (*pb.AcknowledgeAlertResponse, error) {
	if req.AlertId == "" {
		return nil, status.Error(codes.InvalidArgument, "alert_id is required")
	}
	if req.AcknowledgedBy == "" {
		return nil, status.Error(codes.InvalidArgument, "acknowledged_by is required")
	}

	if err := s.alertRepo.Acknowledge(ctx, req.AlertId, req.AcknowledgedBy); err != nil {
		s.logger.Error("Failed to acknowledge alert", "alert_id", req.AlertId, "error", err)
		return nil, status.Error(codes.Internal, "failed to acknowledge alert")
	}

	return &pb.AcknowledgeAlertResponse{Success: true}, nil
}

// ResolveAlert resolves an alert
func (s *GRPCServer) ResolveAlert(ctx context.Context, req *pb.ResolveAlertRequest) (*pb.ResolveAlertResponse, error) {
	if req.AlertId == "" {
		return nil, status.Error(codes.InvalidArgument, "alert_id is required")
	}
	if req.ResolvedBy == "" {
		return nil, status.Error(codes.InvalidArgument, "resolved_by is required")
	}

	if err := s.alertRepo.Resolve(ctx, req.AlertId, req.ResolvedBy, req.Resolution); err != nil {
		s.logger.Error("Failed to resolve alert", "alert_id", req.AlertId, "error", err)
		return nil, status.Error(codes.Internal, "failed to resolve alert")
	}

	return &pb.ResolveAlertResponse{Success: true}, nil
}

// EscalateAlert escalates an alert
func (s *GRPCServer) EscalateAlert(ctx context.Context, req *pb.EscalateAlertRequest) (*pb.EscalateAlertResponse, error) {
	if req.AlertId == "" {
		return nil, status.Error(codes.InvalidArgument, "alert_id is required")
	}
	if req.EscalatedBy == "" {
		return nil, status.Error(codes.InvalidArgument, "escalated_by is required")
	}

	if err := s.alertRepo.Escalate(ctx, req.AlertId, req.EscalatedBy); err != nil {
		s.logger.Error("Failed to escalate alert", "alert_id", req.AlertId, "error", err)
		return nil, status.Error(codes.Internal, "failed to escalate alert")
	}

	return &pb.EscalateAlertResponse{Success: true}, nil
}

// Rule Management Operations

// CreateRule creates a new alerting rule
func (s *GRPCServer) CreateRule(ctx context.Context, req *pb.CreateRuleRequest) (*pb.CreateRuleResponse, error) {
	s.logger.Info("Creating rule", "name", req.Name, "type", req.Type)

	// Validate request
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	// Validate name uniqueness
	if err := s.ruleRepo.ValidateName(ctx, req.Name, ""); err != nil {
		return nil, status.Error(codes.AlreadyExists, err.Error())
	}

	// Create rule
	rule := &database.Rule{
		ID:          generateID("rule"),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Severity:    req.Severity,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
		CreatedBy:   req.CreatedBy,
		UpdatedBy:   req.CreatedBy,
	}

	// Set defaults
	if rule.Type == "" {
		rule.Type = "pattern"
	}
	if rule.Severity == "" {
		rule.Severity = "medium"
	}
	if rule.Priority == "" {
		rule.Priority = "medium"
	}

	// Convert conditions and actions
	if len(req.Conditions) > 0 {
		conditions, err := json.Marshal(req.Conditions)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid conditions")
		}
		rule.Conditions = conditions
	}

	if len(req.Actions) > 0 {
		actions, err := json.Marshal(req.Actions)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid actions")
		}
		rule.Actions = actions
	}

	// Save rule
	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		s.logger.Error("Failed to create rule", "error", err)
		return nil, status.Error(codes.Internal, "failed to create rule")
	}

	s.logger.Info("Rule created", "rule_id", rule.ID)

	pbRule := s.ruleToProto(rule)
	return &pb.CreateRuleResponse{Rule: pbRule}, nil
}

// GetRule retrieves a rule by ID
func (s *GRPCServer) GetRule(ctx context.Context, req *pb.GetRuleRequest) (*pb.GetRuleResponse, error) {
	if req.RuleId == "" {
		return nil, status.Error(codes.InvalidArgument, "rule_id is required")
	}

	rule, err := s.ruleRepo.GetByID(ctx, req.RuleId)
	if err != nil {
		s.logger.Error("Failed to get rule", "rule_id", req.RuleId, "error", err)
		return nil, status.Error(codes.NotFound, "rule not found")
	}

	pbRule := s.ruleToProto(rule)
	return &pb.GetRuleResponse{Rule: pbRule}, nil
}

// ListRules lists rules with filtering and pagination
func (s *GRPCServer) ListRules(ctx context.Context, req *pb.ListRulesRequest) (*pb.ListRulesResponse, error) {
	// Build filter
	filter := database.Filter{
		Limit:  int(req.PageSize),
		Offset: int(req.PageToken),
		Filters: make(map[string]interface{}),
	}

	if req.Type != "" {
		filter.Filters["type"] = req.Type
	}
	if req.Severity != "" {
		filter.Filters["severity"] = req.Severity
	}
	if req.Enabled != nil {
		filter.Filters["enabled"] = req.Enabled.Value
	}

	// Get rules
	rules, total, err := s.ruleRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list rules", "error", err)
		return nil, status.Error(codes.Internal, "failed to list rules")
	}

	// Convert to proto
	pbRules := make([]*pb.Rule, len(rules))
	for i, rule := range rules {
		pbRules[i] = s.ruleToProto(rule)
	}

	return &pb.ListRulesResponse{
		Rules:         pbRules,
		TotalCount:    int32(total),
		NextPageToken: int32(filter.Offset + len(rules)),
	}, nil
}

// EnableRule enables a rule
func (s *GRPCServer) EnableRule(ctx context.Context, req *pb.EnableRuleRequest) (*pb.EnableRuleResponse, error) {
	if req.RuleId == "" {
		return nil, status.Error(codes.InvalidArgument, "rule_id is required")
	}
	if req.UpdatedBy == "" {
		return nil, status.Error(codes.InvalidArgument, "updated_by is required")
	}

	if err := s.ruleRepo.Enable(ctx, req.RuleId, req.UpdatedBy); err != nil {
		s.logger.Error("Failed to enable rule", "rule_id", req.RuleId, "error", err)
		return nil, status.Error(codes.Internal, "failed to enable rule")
	}

	return &pb.EnableRuleResponse{Success: true}, nil
}

// DisableRule disables a rule
func (s *GRPCServer) DisableRule(ctx context.Context, req *pb.DisableRuleRequest) (*pb.DisableRuleResponse, error) {
	if req.RuleId == "" {
		return nil, status.Error(codes.InvalidArgument, "rule_id is required")
	}
	if req.UpdatedBy == "" {
		return nil, status.Error(codes.InvalidArgument, "updated_by is required")
	}

	if err := s.ruleRepo.Disable(ctx, req.RuleId, req.UpdatedBy); err != nil {
		s.logger.Error("Failed to disable rule", "rule_id", req.RuleId, "error", err)
		return nil, status.Error(codes.Internal, "failed to disable rule")
	}

	return &pb.DisableRuleResponse{Success: true}, nil
}

// Notification Operations

// GetNotificationStatus retrieves notification status
func (s *GRPCServer) GetNotificationStatus(ctx context.Context, req *pb.GetNotificationStatusRequest) (*pb.GetNotificationStatusResponse, error) {
	if req.NotificationId == "" {
		return nil, status.Error(codes.InvalidArgument, "notification_id is required")
	}

	notification, err := s.notificationRepo.GetByID(ctx, req.NotificationId)
	if err != nil {
		s.logger.Error("Failed to get notification", "notification_id", req.NotificationId, "error", err)
		return nil, status.Error(codes.NotFound, "notification not found")
	}

	return &pb.GetNotificationStatusResponse{
		Status:       notification.Status,
		Channel:      notification.Channel,
		Recipient:    notification.Recipient,
		RetryCount:   int32(notification.RetryCount),
		ErrorMessage: notification.ErrorMessage,
		CreatedAt:    timestamppb.New(notification.CreatedAt),
		SentAt:       timeToProto(notification.SentAt),
		DeliveredAt:  timeToProto(notification.DeliveredAt),
	}, nil
}

// GetSystemHealth returns system health status
func (s *GRPCServer) GetSystemHealth(ctx context.Context, req *pb.GetSystemHealthRequest) (*pb.GetSystemHealthResponse, error) {
	// Get component health status
	health := &pb.SystemHealth{
		Status:    "healthy",
		Timestamp: timestamppb.Now(),
		Components: map[string]*pb.ComponentHealth{
			"rule_engine": {
				Status:  "healthy",
				Message: "Rule engine is operational",
			},
			"database": {
				Status:  "healthy", 
				Message: "Database connections are healthy",
			},
			"kafka": {
				Status:  "healthy",
				Message: "Kafka connections are healthy",
			},
			"notifications": {
				Status:  "healthy",
				Message: "Notification system is operational",
			},
		},
	}

	// Check rule engine health
	ruleStats := s.ruleEngine.GetRuleStats()
	if totalRules, ok := ruleStats["total_rules"].(int); ok && totalRules == 0 {
		health.Components["rule_engine"].Status = "warning"
		health.Components["rule_engine"].Message = "No rules loaded"
		health.Status = "degraded"
	}

	// Check event processor health
	if s.eventProcessor != nil {
		processorStats := s.eventProcessor.GetStats()
		if consumerStats, ok := processorStats["consumer"].(map[string]interface{}); ok {
			if !consumerStats["is_running"].(bool) {
				health.Components["kafka"].Status = "unhealthy"
				health.Components["kafka"].Message = "Kafka consumer is not running"
				health.Status = "unhealthy"
			}
		}
	}

	return &pb.GetSystemHealthResponse{Health: health}, nil
}

// Helper methods

func (s *GRPCServer) alertToProto(alert *database.Alert) *pb.Alert {
	pbAlert := &pb.Alert{
		Id:           alert.ID,
		RuleId:       alert.RuleID,
		Title:        alert.Title,
		Description:  alert.Description,
		Severity:     alert.Severity,
		Type:         alert.Type,
		Priority:     alert.Priority,
		Status:       alert.Status,
		Source:       alert.Source,
		CreatedBy:    alert.CreatedBy,
		UpdatedBy:    alert.UpdatedBy,
		CreatedAt:    timestamppb.New(alert.CreatedAt),
		UpdatedAt:    timestamppb.New(alert.UpdatedAt),
		AcknowledgedAt: timeToProto(alert.AcknowledgedAt),
		ResolvedAt:     timeToProto(alert.ResolvedAt),
		EscalatedAt:    timeToProto(alert.EscalatedAt),
	}

	// Add metadata if available
	if alert.Metadata != nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal(alert.Metadata, &metadata); err == nil {
			pbAlert.Metadata = metadata
		}
	}

	return pbAlert
}

func (s *GRPCServer) ruleToProto(rule *database.Rule) *pb.Rule {
	pbRule := &pb.Rule{
		Id:          rule.ID,
		Name:        rule.Name,
		Description: rule.Description,
		Type:        rule.Type,
		Severity:    rule.Severity,
		Priority:    rule.Priority,
		Enabled:     rule.Enabled,
		CreatedBy:   rule.CreatedBy,
		UpdatedBy:   rule.UpdatedBy,
		Version:     int32(rule.Version),
		CreatedAt:   timestamppb.New(rule.CreatedAt),
		UpdatedAt:   timestamppb.New(rule.UpdatedAt),
	}

	// Add conditions if available
	if rule.Conditions != nil {
		var conditions []map[string]interface{}
		if err := json.Unmarshal(rule.Conditions, &conditions); err == nil {
			pbRule.Conditions = conditions
		}
	}

	// Add actions if available
	if rule.Actions != nil {
		var actions []map[string]interface{}
		if err := json.Unmarshal(rule.Actions, &actions); err == nil {
			pbRule.Actions = actions
		}
	}

	return pbRule
}

func timeToProto(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().Unix(), time.Now().Nanosecond())
}