package compliance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aegisshield/compliance-engine/internal/config"
	"go.uber.org/zap"
)

// ViolationManager manages compliance violations and their lifecycle
type ViolationManager struct {
	config          config.ViolationHandlingConfig
	logger          *zap.Logger
	violations      map[string]*Violation
	escalationRules map[string]*EscalationRule
	mu              sync.RWMutex
	running         bool
	stopChan        chan struct{}
}

// NewViolationManager creates a new violation manager instance
func NewViolationManager(cfg config.ViolationHandlingConfig, logger *zap.Logger) *ViolationManager {
	return &ViolationManager{
		config:          cfg,
		logger:          logger,
		violations:      make(map[string]*Violation),
		escalationRules: make(map[string]*EscalationRule),
		stopChan:        make(chan struct{}),
	}
}

// Start starts the violation manager
func (vm *ViolationManager) Start(ctx context.Context) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.running {
		return fmt.Errorf("violation manager is already running")
	}

	vm.logger.Info("Starting violation manager")

	// Load escalation rules
	if err := vm.loadEscalationRules(); err != nil {
		return fmt.Errorf("failed to load escalation rules: %w", err)
	}

	// Start background processes
	go vm.escalationLoop(ctx)
	go vm.cleanupLoop(ctx)

	vm.running = true
	vm.logger.Info("Violation manager started successfully")

	return nil
}

// Stop stops the violation manager
func (vm *ViolationManager) Stop(ctx context.Context) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.running {
		return nil
	}

	vm.logger.Info("Stopping violation manager")

	close(vm.stopChan)
	vm.running = false

	vm.logger.Info("Violation manager stopped")
	return nil
}

// RecordViolation records a new compliance violation
func (vm *ViolationManager) RecordViolation(ctx context.Context, violation Violation) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.running {
		return fmt.Errorf("violation manager is not running")
	}

	// Generate violation ID if not provided
	if violation.ID == "" {
		violation.ID = vm.generateViolationID()
	}

	// Set timestamps
	violation.CreatedAt = time.Now()
	violation.UpdatedAt = time.Now()

	// Set initial status
	if violation.Status == "" {
		violation.Status = "open"
	}

	// Calculate risk score
	violation.RiskScore = vm.calculateRiskScore(violation)

	// Store violation
	vm.violations[violation.ID] = &violation

	vm.logger.Info("Violation recorded",
		zap.String("violation_id", violation.ID),
		zap.String("rule_id", violation.RuleID),
		zap.String("severity", violation.Severity),
		zap.Float64("risk_score", violation.RiskScore),
	)

	// Trigger immediate escalation if needed
	if vm.shouldEscalateImmediately(violation) {
		go vm.escalateViolation(ctx, violation)
	}

	return nil
}

// GetViolation retrieves a violation by ID
func (vm *ViolationManager) GetViolation(ctx context.Context, violationID string) (*Violation, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if !vm.running {
		return nil, fmt.Errorf("violation manager is not running")
	}

	violation, exists := vm.violations[violationID]
	if !exists {
		return nil, fmt.Errorf("violation not found: %s", violationID)
	}

	return violation, nil
}

// UpdateViolationStatus updates the status of a violation
func (vm *ViolationManager) UpdateViolationStatus(ctx context.Context, violationID string, status string, notes string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.running {
		return fmt.Errorf("violation manager is not running")
	}

	violation, exists := vm.violations[violationID]
	if !exists {
		return fmt.Errorf("violation not found: %s", violationID)
	}

	oldStatus := violation.Status
	violation.Status = status
	violation.UpdatedAt = time.Now()

	// Add status change to history
	statusChange := ViolationStatusChange{
		FromStatus: oldStatus,
		ToStatus:   status,
		ChangedAt:  time.Now(),
		Notes:      notes,
	}
	violation.StatusHistory = append(violation.StatusHistory, statusChange)

	vm.logger.Info("Violation status updated",
		zap.String("violation_id", violationID),
		zap.String("old_status", oldStatus),
		zap.String("new_status", status),
	)

	return nil
}

// GetViolationsByStatus returns violations with the specified status
func (vm *ViolationManager) GetViolationsByStatus(ctx context.Context, status string) ([]Violation, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if !vm.running {
		return nil, fmt.Errorf("violation manager is not running")
	}

	var violations []Violation
	for _, violation := range vm.violations {
		if violation.Status == status {
			violations = append(violations, *violation)
		}
	}

	return violations, nil
}

// GetViolationsBySeverity returns violations with the specified severity
func (vm *ViolationManager) GetViolationsBySeverity(ctx context.Context, severity string) ([]Violation, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if !vm.running {
		return nil, fmt.Errorf("violation manager is not running")
	}

	var violations []Violation
	for _, violation := range vm.violations {
		if violation.Severity == severity {
			violations = append(violations, *violation)
		}
	}

	return violations, nil
}

// GetViolationsByTimeRange returns violations within the specified time range
func (vm *ViolationManager) GetViolationsByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]Violation, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if !vm.running {
		return nil, fmt.Errorf("violation manager is not running")
	}

	var violations []Violation
	for _, violation := range vm.violations {
		if violation.CreatedAt.After(startTime) && violation.CreatedAt.Before(endTime) {
			violations = append(violations, *violation)
		}
	}

	return violations, nil
}

// GetViolationStatistics returns statistics about violations
func (vm *ViolationManager) GetViolationStatistics(ctx context.Context) (*ViolationStatistics, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if !vm.running {
		return nil, fmt.Errorf("violation manager is not running")
	}

	stats := &ViolationStatistics{
		TotalViolations: len(vm.violations),
		StatusCounts:    make(map[string]int),
		SeverityCounts:  make(map[string]int),
		RuleCounts:      make(map[string]int),
	}

	totalRiskScore := 0.0
	for _, violation := range vm.violations {
		stats.StatusCounts[violation.Status]++
		stats.SeverityCounts[violation.Severity]++
		stats.RuleCounts[violation.RuleID]++
		totalRiskScore += violation.RiskScore
	}

	if len(vm.violations) > 0 {
		stats.AverageRiskScore = totalRiskScore / float64(len(vm.violations))
	}

	return stats, nil
}

// AssignViolation assigns a violation to a user
func (vm *ViolationManager) AssignViolation(ctx context.Context, violationID string, assignedTo string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.running {
		return fmt.Errorf("violation manager is not running")
	}

	violation, exists := vm.violations[violationID]
	if !exists {
		return fmt.Errorf("violation not found: %s", violationID)
	}

	violation.AssignedTo = assignedTo
	violation.UpdatedAt = time.Now()

	vm.logger.Info("Violation assigned",
		zap.String("violation_id", violationID),
		zap.String("assigned_to", assignedTo),
	)

	return nil
}

// AddViolationComment adds a comment to a violation
func (vm *ViolationManager) AddViolationComment(ctx context.Context, violationID string, comment ViolationComment) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.running {
		return fmt.Errorf("violation manager is not running")
	}

	violation, exists := vm.violations[violationID]
	if !exists {
		return fmt.Errorf("violation not found: %s", violationID)
	}

	comment.CreatedAt = time.Now()
	violation.Comments = append(violation.Comments, comment)
	violation.UpdatedAt = time.Now()

	vm.logger.Info("Comment added to violation",
		zap.String("violation_id", violationID),
		zap.String("author", comment.Author),
	)

	return nil
}

// EscalateViolation manually escalates a violation
func (vm *ViolationManager) EscalateViolation(ctx context.Context, violationID string, reason string) error {
	vm.mu.RLock()
	violation, exists := vm.violations[violationID]
	vm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("violation not found: %s", violationID)
	}

	return vm.escalateViolation(ctx, *violation)
}

// Private methods

func (vm *ViolationManager) loadEscalationRules() error {
	// Load default escalation rules
	defaultRules := []*EscalationRule{
		{
			ID:          "critical_immediate",
			Name:        "Critical Violations - Immediate Escalation",
			Conditions:  map[string]interface{}{"severity": "critical"},
			Actions:     []string{"notify_manager", "create_ticket", "send_alert"},
			Delay:       0,
			MaxRetries:  3,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "high_30min",
			Name:        "High Severity - 30 Minute Escalation",
			Conditions:  map[string]interface{}{"severity": "high"},
			Actions:     []string{"notify_manager", "create_ticket"},
			Delay:       30 * time.Minute,
			MaxRetries:  3,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "medium_2hour",
			Name:        "Medium Severity - 2 Hour Escalation",
			Conditions:  map[string]interface{}{"severity": "medium"},
			Actions:     []string{"notify_team"},
			Delay:       2 * time.Hour,
			MaxRetries:  2,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "high_risk_score",
			Name:        "High Risk Score Escalation",
			Conditions:  map[string]interface{}{"risk_score_threshold": 80.0},
			Actions:     []string{"notify_manager", "create_priority_ticket"},
			Delay:       15 * time.Minute,
			MaxRetries:  3,
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
	}

	for _, rule := range defaultRules {
		vm.escalationRules[rule.ID] = rule
	}

	vm.logger.Info("Escalation rules loaded", zap.Int("count", len(defaultRules)))
	return nil
}

func (vm *ViolationManager) shouldEscalateImmediately(violation Violation) bool {
	// Check if violation should be escalated immediately
	for _, rule := range vm.escalationRules {
		if !rule.Enabled {
			continue
		}

		if rule.Delay == 0 && vm.violationMatchesRule(violation, rule) {
			return true
		}
	}
	return false
}

func (vm *ViolationManager) violationMatchesRule(violation Violation, rule *EscalationRule) bool {
	// Check if violation matches escalation rule conditions
	for key, value := range rule.Conditions {
		switch key {
		case "severity":
			if violation.Severity != value.(string) {
				return false
			}
		case "risk_score_threshold":
			threshold := value.(float64)
			if violation.RiskScore < threshold {
				return false
			}
		case "rule_id":
			if violation.RuleID != value.(string) {
				return false
			}
		}
	}
	return true
}

func (vm *ViolationManager) escalateViolation(ctx context.Context, violation Violation) error {
	vm.logger.Info("Escalating violation",
		zap.String("violation_id", violation.ID),
		zap.String("severity", violation.Severity),
	)

	// Find applicable escalation rules
	for _, rule := range vm.escalationRules {
		if !rule.Enabled {
			continue
		}

		if vm.violationMatchesRule(violation, rule) {
			// Execute escalation actions
			for _, action := range rule.Actions {
				if err := vm.executeEscalationAction(ctx, action, violation); err != nil {
					vm.logger.Error("Failed to execute escalation action",
						zap.String("action", action),
						zap.String("violation_id", violation.ID),
						zap.Error(err),
					)
				}
			}

			// Update violation status
			vm.mu.Lock()
			if storedViolation, exists := vm.violations[violation.ID]; exists {
				storedViolation.EscalatedAt = time.Now()
				storedViolation.EscalationLevel++
				storedViolation.UpdatedAt = time.Now()
			}
			vm.mu.Unlock()

			break
		}
	}

	return nil
}

func (vm *ViolationManager) executeEscalationAction(ctx context.Context, action string, violation Violation) error {
	switch action {
	case "notify_manager":
		return vm.notifyManager(ctx, violation)
	case "notify_team":
		return vm.notifyTeam(ctx, violation)
	case "create_ticket":
		return vm.createTicket(ctx, violation)
	case "create_priority_ticket":
		return vm.createPriorityTicket(ctx, violation)
	case "send_alert":
		return vm.sendAlert(ctx, violation)
	default:
		return fmt.Errorf("unknown escalation action: %s", action)
	}
}

func (vm *ViolationManager) notifyManager(ctx context.Context, violation Violation) error {
	vm.logger.Info("Notifying manager about violation",
		zap.String("violation_id", violation.ID),
	)
	// Implementation would send notification to manager
	return nil
}

func (vm *ViolationManager) notifyTeam(ctx context.Context, violation Violation) error {
	vm.logger.Info("Notifying team about violation",
		zap.String("violation_id", violation.ID),
	)
	// Implementation would send notification to team
	return nil
}

func (vm *ViolationManager) createTicket(ctx context.Context, violation Violation) error {
	vm.logger.Info("Creating ticket for violation",
		zap.String("violation_id", violation.ID),
	)
	// Implementation would create support ticket
	return nil
}

func (vm *ViolationManager) createPriorityTicket(ctx context.Context, violation Violation) error {
	vm.logger.Info("Creating priority ticket for violation",
		zap.String("violation_id", violation.ID),
	)
	// Implementation would create high-priority support ticket
	return nil
}

func (vm *ViolationManager) sendAlert(ctx context.Context, violation Violation) error {
	vm.logger.Info("Sending alert for violation",
		zap.String("violation_id", violation.ID),
	)
	// Implementation would send immediate alert
	return nil
}

func (vm *ViolationManager) generateViolationID() string {
	return fmt.Sprintf("VIO_%d", time.Now().UnixNano())
}

func (vm *ViolationManager) calculateRiskScore(violation Violation) float64 {
	score := 0.0

	// Base score based on severity
	switch violation.Severity {
	case "critical":
		score = 90.0
	case "high":
		score = 70.0
	case "medium":
		score = 50.0
	case "low":
		score = 30.0
	default:
		score = 40.0
	}

	// Adjust based on rule factors
	if factors, exists := violation.Details["risk_factors"]; exists {
		if factorsSlice, ok := factors.([]interface{}); ok {
			score += float64(len(factorsSlice)) * 5.0
		}
	}

	// Cap at 100
	if score > 100.0 {
		score = 100.0
	}

	return score
}

func (vm *ViolationManager) escalationLoop(ctx context.Context) {
	ticker := time.NewTicker(vm.config.EscalationCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-vm.stopChan:
			return
		case <-ticker.C:
			vm.checkPendingEscalations(ctx)
		}
	}
}

func (vm *ViolationManager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Daily cleanup
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-vm.stopChan:
			return
		case <-ticker.C:
			vm.cleanupOldViolations()
		}
	}
}

func (vm *ViolationManager) checkPendingEscalations(ctx context.Context) {
	vm.mu.RLock()
	violations := make([]Violation, 0, len(vm.violations))
	for _, violation := range vm.violations {
		violations = append(violations, *violation)
	}
	vm.mu.RUnlock()

	for _, violation := range violations {
		if violation.Status == "open" && violation.EscalatedAt.IsZero() {
			// Check if violation should be escalated based on time
			for _, rule := range vm.escalationRules {
				if !rule.Enabled || rule.Delay == 0 {
					continue
				}

				if vm.violationMatchesRule(violation, rule) {
					if time.Since(violation.CreatedAt) >= rule.Delay {
						go vm.escalateViolation(ctx, violation)
						break
					}
				}
			}
		}
	}
}

func (vm *ViolationManager) cleanupOldViolations() {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	cutoffTime := time.Now().AddDate(0, 0, -vm.config.RetentionDays)
	deletedCount := 0

	for id, violation := range vm.violations {
		if violation.Status == "resolved" && violation.UpdatedAt.Before(cutoffTime) {
			delete(vm.violations, id)
			deletedCount++
		}
	}

	if deletedCount > 0 {
		vm.logger.Info("Cleaned up old violations", zap.Int("count", deletedCount))
	}
}