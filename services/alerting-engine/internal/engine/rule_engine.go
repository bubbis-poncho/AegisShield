package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
)

// RuleEngine evaluates alerting rules against events and data
type RuleEngine struct {
	config           *config.Config
	logger           *slog.Logger
	ruleRepo         *database.RuleRepository
	alertRepo        *database.AlertRepository
	compiledRules    map[string]*CompiledRule
	rulesMutex       sync.RWMutex
	evaluationCache  map[string]*CacheEntry
	cacheMutex       sync.RWMutex
	evaluationPool   *EvaluationPool
	shutdownChan     chan struct{}
	wg               sync.WaitGroup
}

// CompiledRule represents a compiled rule for efficient evaluation
type CompiledRule struct {
	Rule       *database.Rule
	Conditions []*vm.Program
	Actions    []ActionHandler
	LastUsed   time.Time
}

// CacheEntry represents a cached evaluation result
type CacheEntry struct {
	Result    bool
	Timestamp time.Time
	TTL       time.Duration
}

// EvaluationContext contains data for rule evaluation
type EvaluationContext struct {
	Event       map[string]interface{}
	Alert       *database.Alert
	Historical  map[string]interface{}
	Aggregated  map[string]interface{}
	Metadata    map[string]interface{}
	Timestamp   time.Time
}

// EvaluationResult contains the result of rule evaluation
type EvaluationResult struct {
	RuleID       string
	RuleName     string
	Matched      bool
	Actions      []string
	Context      *EvaluationContext
	ExecutionTime time.Duration
	Error        error
}

// ActionHandler defines an interface for rule actions
type ActionHandler interface {
	Execute(ctx context.Context, result *EvaluationResult) error
	GetType() string
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(
	cfg *config.Config,
	logger *slog.Logger,
	ruleRepo *database.RuleRepository,
	alertRepo *database.AlertRepository,
) (*RuleEngine, error) {
	engine := &RuleEngine{
		config:          cfg,
		logger:          logger,
		ruleRepo:        ruleRepo,
		alertRepo:       alertRepo,
		compiledRules:   make(map[string]*CompiledRule),
		evaluationCache: make(map[string]*CacheEntry),
		shutdownChan:    make(chan struct{}),
	}

	// Initialize evaluation pool
	engine.evaluationPool = NewEvaluationPool(cfg.Rules.MaxConcurrentEvaluations)

	return engine, nil
}

// Start starts the rule engine
func (r *RuleEngine) Start(ctx context.Context) error {
	r.logger.Info("Starting rule engine")

	// Load and compile rules
	if err := r.loadRules(ctx); err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	// Start cache cleanup routine
	r.wg.Add(1)
	go r.cacheCleanupRoutine(ctx)

	// Start rule refresh routine
	r.wg.Add(1)
	go r.ruleRefreshRoutine(ctx)

	r.logger.Info("Rule engine started", "loaded_rules", len(r.compiledRules))
	return nil
}

// Stop stops the rule engine
func (r *RuleEngine) Stop() {
	r.logger.Info("Stopping rule engine")
	close(r.shutdownChan)
	r.wg.Wait()
	r.evaluationPool.Close()
	r.logger.Info("Rule engine stopped")
}

// EvaluateEvent evaluates an event against all enabled rules
func (r *RuleEngine) EvaluateEvent(ctx context.Context, event map[string]interface{}) ([]*EvaluationResult, error) {
	r.rulesMutex.RLock()
	rules := make([]*CompiledRule, 0, len(r.compiledRules))
	for _, rule := range r.compiledRules {
		if rule.Rule.Enabled {
			rules = append(rules, rule)
		}
	}
	r.rulesMutex.RUnlock()

	if len(rules) == 0 {
		return nil, nil
	}

	// Create evaluation context
	evalContext := &EvaluationContext{
		Event:     event,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Add historical and aggregated data if needed
	if err := r.enrichContext(ctx, evalContext); err != nil {
		r.logger.Error("Failed to enrich evaluation context", "error", err)
	}

	// Evaluate rules in parallel
	results := make([]*EvaluationResult, 0, len(rules))
	resultChan := make(chan *EvaluationResult, len(rules))
	errorChan := make(chan error, len(rules))

	// Submit evaluation tasks
	for _, rule := range rules {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			r.evaluationPool.Submit(func(rule *CompiledRule) {
				result := r.evaluateRule(ctx, rule, evalContext)
				if result.Error != nil {
					errorChan <- result.Error
				} else {
					resultChan <- result
				}
			}(rule))
		}
	}

	// Collect results
	for i := 0; i < len(rules); i++ {
		select {
		case result := <-resultChan:
			results = append(results, result)
		case err := <-errorChan:
			r.logger.Error("Rule evaluation error", "error", err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Filter matched results
	matchedResults := make([]*EvaluationResult, 0)
	for _, result := range results {
		if result.Matched {
			matchedResults = append(matchedResults, result)
		}
	}

	r.logger.Debug("Event evaluation completed",
		"total_rules", len(rules),
		"matched_rules", len(matchedResults),
		"event_type", event["type"])

	return matchedResults, nil
}

// EvaluateRule evaluates a single rule against an event
func (r *RuleEngine) evaluateRule(ctx context.Context, compiledRule *CompiledRule, evalContext *EvaluationContext) *EvaluationResult {
	startTime := time.Now()
	
	result := &EvaluationResult{
		RuleID:   compiledRule.Rule.ID,
		RuleName: compiledRule.Rule.Name,
		Context:  evalContext,
		Matched:  false,
	}

	// Check cache first
	if r.config.Rules.CacheEnabled {
		if cached := r.getCachedResult(compiledRule.Rule.ID, evalContext); cached != nil {
			result.Matched = cached.Result
			result.ExecutionTime = time.Since(startTime)
			return result
		}
	}

	// Evaluate conditions
	matched, err := r.evaluateConditions(ctx, compiledRule, evalContext)
	if err != nil {
		result.Error = fmt.Errorf("failed to evaluate conditions: %w", err)
		return result
	}

	result.Matched = matched
	result.ExecutionTime = time.Since(startTime)

	// Cache result if enabled
	if r.config.Rules.CacheEnabled && result.Error == nil {
		r.cacheResult(compiledRule.Rule.ID, evalContext, matched, time.Duration(r.config.Rules.CacheTTLSeconds)*time.Second)
	}

	// Extract actions if matched
	if matched {
		actions := make([]string, 0)
		var actionsData []map[string]interface{}
		if err := json.Unmarshal(compiledRule.Rule.Actions, &actionsData); err == nil {
			for _, action := range actionsData {
				if actionType, ok := action["type"].(string); ok {
					actions = append(actions, actionType)
				}
			}
		}
		result.Actions = actions
	}

	// Update rule usage
	compiledRule.LastUsed = time.Now()

	return result
}

// EvaluateConditions evaluates all conditions for a rule
func (r *RuleEngine) evaluateConditions(ctx context.Context, compiledRule *CompiledRule, evalContext *EvaluationContext) (bool, error) {
	if len(compiledRule.Conditions) == 0 {
		return true, nil
	}

	// Create evaluation environment
	env := r.createEvaluationEnvironment(evalContext)

	// Evaluate each condition (AND logic)
	for i, condition := range compiledRule.Conditions {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			result, err := vm.Run(condition, env)
			if err != nil {
				return false, fmt.Errorf("condition %d evaluation failed: %w", i, err)
			}

			matched, ok := result.(bool)
			if !ok {
				return false, fmt.Errorf("condition %d did not return boolean", i)
			}

			if !matched {
				return false, nil
			}
		}
	}

	return true, nil
}

// LoadRules loads and compiles all enabled rules
func (r *RuleEngine) loadRules(ctx context.Context) error {
	rules, err := r.ruleRepo.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to list enabled rules: %w", err)
	}

	r.rulesMutex.Lock()
	defer r.rulesMutex.Unlock()

	// Clear existing rules
	r.compiledRules = make(map[string]*CompiledRule)

	// Compile each rule
	for _, rule := range rules {
		compiledRule, err := r.compileRule(rule)
		if err != nil {
			r.logger.Error("Failed to compile rule",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"error", err)
			continue
		}

		r.compiledRules[rule.ID] = compiledRule
		r.logger.Debug("Rule compiled",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"conditions", len(compiledRule.Conditions))
	}

	return nil
}

// CompileRule compiles a rule for efficient evaluation
func (r *RuleEngine) compileRule(rule *database.Rule) (*CompiledRule, error) {
	compiledRule := &CompiledRule{
		Rule:       rule,
		Conditions: make([]*vm.Program, 0),
		Actions:    make([]ActionHandler, 0),
		LastUsed:   time.Now(),
	}

	// Parse and compile conditions
	var conditions []map[string]interface{}
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		return nil, fmt.Errorf("failed to parse rule conditions: %w", err)
	}

	for i, condition := range conditions {
		if expression, ok := condition["expression"].(string); ok {
			program, err := expr.Compile(expression)
			if err != nil {
				return nil, fmt.Errorf("failed to compile condition %d: %w", i, err)
			}
			compiledRule.Conditions = append(compiledRule.Conditions, program)
		}
	}

	// Parse and compile actions
	var actions []map[string]interface{}
	if err := json.Unmarshal(rule.Actions, &actions); err != nil {
		return nil, fmt.Errorf("failed to parse rule actions: %w", err)
	}

	for _, action := range actions {
		handler, err := r.createActionHandler(action)
		if err != nil {
			r.logger.Error("Failed to create action handler",
				"rule_id", rule.ID,
				"action", action,
				"error", err)
			continue
		}
		compiledRule.Actions = append(compiledRule.Actions, handler)
	}

	return compiledRule, nil
}

// CreateEvaluationEnvironment creates the environment for rule evaluation
func (r *RuleEngine) createEvaluationEnvironment(evalContext *EvaluationContext) map[string]interface{} {
	env := map[string]interface{}{
		"event":      evalContext.Event,
		"timestamp":  evalContext.Timestamp,
		"metadata":   evalContext.Metadata,
		"now":        time.Now(),
		"today":      time.Now().Truncate(24 * time.Hour),
		"yesterday":  time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour),
	}

	// Add alert data if available
	if evalContext.Alert != nil {
		env["alert"] = map[string]interface{}{
			"id":          evalContext.Alert.ID,
			"title":       evalContext.Alert.Title,
			"description": evalContext.Alert.Description,
			"severity":    evalContext.Alert.Severity,
			"status":      evalContext.Alert.Status,
			"type":        evalContext.Alert.Type,
			"created_at":  evalContext.Alert.CreatedAt,
		}
	}

	// Add historical data if available
	if evalContext.Historical != nil {
		env["historical"] = evalContext.Historical
	}

	// Add aggregated data if available
	if evalContext.Aggregated != nil {
		env["aggregated"] = evalContext.Aggregated
	}

	// Add utility functions
	env["len"] = func(v interface{}) int {
		switch val := v.(type) {
		case []interface{}:
			return len(val)
		case map[string]interface{}:
			return len(val)
		case string:
			return len(val)
		default:
			return 0
		}
	}

	env["contains"] = func(haystack, needle interface{}) bool {
		switch h := haystack.(type) {
		case []interface{}:
			for _, item := range h {
				if item == needle {
					return true
				}
			}
		case string:
			if n, ok := needle.(string); ok {
				return strings.Contains(h, n)
			}
		}
		return false
	}

	env["matches"] = func(pattern, text string) bool {
		// Simple pattern matching (could be enhanced with regex)
		return strings.Contains(text, pattern)
	}

	return env
}

// EnrichContext adds historical and aggregated data to evaluation context
func (r *RuleEngine) enrichContext(ctx context.Context, evalContext *EvaluationContext) error {
	// This is a placeholder for more sophisticated context enrichment
	// In practice, you would query historical data, calculate aggregations, etc.
	
	evalContext.Historical = map[string]interface{}{
		"alert_count_last_hour": 0,
		"alert_count_last_day":  0,
	}
	
	evalContext.Aggregated = map[string]interface{}{
		"avg_response_time": 0.0,
		"error_rate":        0.0,
	}

	return nil
}

// Cache management

func (r *RuleEngine) getCachedResult(ruleID string, evalContext *EvaluationContext) *CacheEntry {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()

	cacheKey := r.generateCacheKey(ruleID, evalContext)
	entry, exists := r.evaluationCache[cacheKey]
	if !exists {
		return nil
	}

	// Check TTL
	if time.Since(entry.Timestamp) > entry.TTL {
		return nil
	}

	return entry
}

func (r *RuleEngine) cacheResult(ruleID string, evalContext *EvaluationContext, result bool, ttl time.Duration) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	cacheKey := r.generateCacheKey(ruleID, evalContext)
	r.evaluationCache[cacheKey] = &CacheEntry{
		Result:    result,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

func (r *RuleEngine) generateCacheKey(ruleID string, evalContext *EvaluationContext) string {
	// Generate a cache key based on rule ID and relevant context data
	// This is simplified - in practice you'd want a more sophisticated key generation
	eventHash := fmt.Sprintf("%v", evalContext.Event)
	return fmt.Sprintf("%s:%x", ruleID, eventHash)
}

// Background routines

func (r *RuleEngine) cacheCleanupRoutine(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(time.Duration(r.config.Rules.CacheCleanupIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.shutdownChan:
			return
		case <-ticker.C:
			r.cleanupCache()
		}
	}
}

func (r *RuleEngine) ruleRefreshRoutine(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(time.Duration(r.config.Rules.RefreshIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.shutdownChan:
			return
		case <-ticker.C:
			if err := r.loadRules(ctx); err != nil {
				r.logger.Error("Failed to refresh rules", "error", err)
			} else {
				r.logger.Debug("Rules refreshed", "count", len(r.compiledRules))
			}
		}
	}
}

func (r *RuleEngine) cleanupCache() {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	now := time.Now()
	for key, entry := range r.evaluationCache {
		if now.Sub(entry.Timestamp) > entry.TTL {
			delete(r.evaluationCache, key)
		}
	}
}

// Action handler creation
func (r *RuleEngine) createActionHandler(action map[string]interface{}) (ActionHandler, error) {
	actionType, ok := action["type"].(string)
	if !ok {
		return nil, fmt.Errorf("action type not specified")
	}

	switch actionType {
	case "create_alert":
		return NewCreateAlertHandler(action, r.alertRepo, r.logger), nil
	case "send_notification":
		return NewSendNotificationHandler(action, r.logger), nil
	case "webhook":
		return NewWebhookActionHandler(action, r.logger), nil
	default:
		return nil, fmt.Errorf("unsupported action type: %s", actionType)
	}
}

// GetRuleStats returns statistics about rule usage
func (r *RuleEngine) GetRuleStats() map[string]interface{} {
	r.rulesMutex.RLock()
	defer r.rulesMutex.RUnlock()

	stats := map[string]interface{}{
		"total_rules":  len(r.compiledRules),
		"cache_size":   len(r.evaluationCache),
		"rule_details": make([]map[string]interface{}, 0),
	}

	for _, rule := range r.compiledRules {
		ruleStats := map[string]interface{}{
			"id":               rule.Rule.ID,
			"name":             rule.Rule.Name,
			"enabled":          rule.Rule.Enabled,
			"condition_count":  len(rule.Conditions),
			"action_count":     len(rule.Actions),
			"last_used":        rule.LastUsed,
		}
		stats["rule_details"] = append(stats["rule_details"].([]map[string]interface{}), ruleStats)
	}

	return stats
}