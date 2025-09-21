package compliance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aegisshield/compliance-engine/internal/config"
	"go.uber.org/zap"
)

// RuleEngine manages compliance rules and their evaluation
type RuleEngine struct {
	config      config.RulesEngineConfig
	logger      *zap.Logger
	rules       map[string]*Rule
	ruleCache   map[string]*RuleResult
	mu          sync.RWMutex
	running     bool
	stopChan    chan struct{}
}

// NewRuleEngine creates a new rule engine instance
func NewRuleEngine(cfg config.RulesEngineConfig, logger *zap.Logger) *RuleEngine {
	return &RuleEngine{
		config:    cfg,
		logger:    logger,
		rules:     make(map[string]*Rule),
		ruleCache: make(map[string]*RuleResult),
		stopChan:  make(chan struct{}),
	}
}

// Start starts the rule engine
func (re *RuleEngine) Start(ctx context.Context) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if re.running {
		return fmt.Errorf("rule engine is already running")
	}

	re.logger.Info("Starting rule engine")

	// Load default rules
	if err := re.loadDefaultRules(); err != nil {
		return fmt.Errorf("failed to load default rules: %w", err)
	}

	// Start background tasks
	go re.ruleEvaluationLoop(ctx)
	if re.config.EnableRuleCaching {
		go re.cacheCleanupLoop(ctx)
	}

	re.running = true
	re.logger.Info("Rule engine started successfully")

	return nil
}

// Stop stops the rule engine
func (re *RuleEngine) Stop(ctx context.Context) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if !re.running {
		return nil
	}

	re.logger.Info("Stopping rule engine")

	close(re.stopChan)
	re.running = false

	re.logger.Info("Rule engine stopped")
	return nil
}

// GetApplicableRules returns rules applicable to the given data
func (re *RuleEngine) GetApplicableRules(ctx context.Context, data interface{}) ([]Rule, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	if !re.running {
		return nil, fmt.Errorf("rule engine is not running")
	}

	var applicableRules []Rule

	for _, rule := range re.rules {
		if re.isRuleApplicable(rule, data) {
			applicableRules = append(applicableRules, *rule)
		}
	}

	return applicableRules, nil
}

// EvaluateRule evaluates a specific rule against data
func (re *RuleEngine) EvaluateRule(ctx context.Context, rule Rule, data interface{}) (*RuleResult, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	if !re.running {
		return nil, fmt.Errorf("rule engine is not running")
	}

	// Check cache first if enabled
	if re.config.EnableRuleCaching {
		cacheKey := re.generateCacheKey(rule.ID, data)
		if cachedResult, exists := re.ruleCache[cacheKey]; exists {
			if time.Since(cachedResult.EvaluatedAt) < re.config.CacheTTL {
				return cachedResult, nil
			}
		}
	}

	// Create evaluation context with timeout
	evalCtx, cancel := context.WithTimeout(ctx, re.config.RuleTimeout)
	defer cancel()

	result := &RuleResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		EvaluatedAt: time.Now(),
	}

	// Evaluate rule based on type
	switch rule.Type {
	case "transaction_limit":
		result = re.evaluateTransactionLimit(evalCtx, rule, data)
	case "suspicious_pattern":
		result = re.evaluateSuspiciousPattern(evalCtx, rule, data)
	case "sanctions_screening":
		result = re.evaluateSanctionsScreening(evalCtx, rule, data)
	case "aml_threshold":
		result = re.evaluateAMLThreshold(evalCtx, rule, data)
	case "kyc_validation":
		result = re.evaluateKYCValidation(evalCtx, rule, data)
	case "pci_compliance":
		result = re.evaluatePCICompliance(evalCtx, rule, data)
	case "gdpr_compliance":
		result = re.evaluateGDPRCompliance(evalCtx, rule, data)
	case "custom":
		result = re.evaluateCustomRule(evalCtx, rule, data)
	default:
		result.Passed = false
		result.Description = fmt.Sprintf("Unknown rule type: %s", rule.Type)
		result.Details = map[string]interface{}{
			"error": "unsupported_rule_type",
		}
	}

	// Cache result if enabled
	if re.config.EnableRuleCaching {
		cacheKey := re.generateCacheKey(rule.ID, data)
		re.ruleCache[cacheKey] = result
	}

	return result, nil
}

// UpdateRules updates the rule set
func (re *RuleEngine) UpdateRules(ctx context.Context, rules []Rule) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if !re.running {
		return fmt.Errorf("rule engine is not running")
	}

	// Validate rules
	for _, rule := range rules {
		if err := re.validateRule(rule); err != nil {
			return fmt.Errorf("invalid rule %s: %w", rule.ID, err)
		}
	}

	// Clear existing rules and add new ones
	re.rules = make(map[string]*Rule)
	for _, rule := range rules {
		ruleCopy := rule
		re.rules[rule.ID] = &ruleCopy
	}

	// Clear cache
	if re.config.EnableRuleCaching {
		re.ruleCache = make(map[string]*RuleResult)
	}

	re.logger.Info("Rules updated", zap.Int("count", len(rules)))
	return nil
}

// GetActiveRuleCount returns the number of active rules
func (re *RuleEngine) GetActiveRuleCount() int {
	re.mu.RLock()
	defer re.mu.RUnlock()

	return len(re.rules)
}

// GetStatus returns the current status of the rule engine
func (re *RuleEngine) GetStatus() string {
	re.mu.RLock()
	defer re.mu.RUnlock()

	if re.running {
		return "running"
	}
	return "stopped"
}

// Rule evaluation methods

func (re *RuleEngine) evaluateTransactionLimit(ctx context.Context, rule Rule, data interface{}) *RuleResult {
	result := &RuleResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		EvaluatedAt: time.Now(),
	}

	// Extract transaction data
	transactionData, ok := data.(map[string]interface{})
	if !ok {
		result.Passed = false
		result.Description = "Invalid transaction data format"
		return result
	}

	amount, exists := transactionData["amount"]
	if !exists {
		result.Passed = false
		result.Description = "Transaction amount not found"
		return result
	}

	amountFloat, ok := amount.(float64)
	if !ok {
		result.Passed = false
		result.Description = "Invalid transaction amount format"
		return result
	}

	// Get threshold from rule parameters
	threshold, exists := rule.Parameters["threshold"]
	if !exists {
		result.Passed = false
		result.Description = "Rule threshold not configured"
		return result
	}

	thresholdFloat, ok := threshold.(float64)
	if !ok {
		result.Passed = false
		result.Description = "Invalid threshold format"
		return result
	}

	// Evaluate rule
	if amountFloat > thresholdFloat {
		result.Passed = false
		result.Description = fmt.Sprintf("Transaction amount %.2f exceeds limit %.2f", amountFloat, thresholdFloat)
		result.Details = map[string]interface{}{
			"amount":    amountFloat,
			"threshold": thresholdFloat,
			"excess":    amountFloat - thresholdFloat,
		}
	} else {
		result.Passed = true
		result.Description = "Transaction amount within limits"
		result.Details = map[string]interface{}{
			"amount":    amountFloat,
			"threshold": thresholdFloat,
		}
	}

	return result
}

func (re *RuleEngine) evaluateSuspiciousPattern(ctx context.Context, rule Rule, data interface{}) *RuleResult {
	result := &RuleResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		EvaluatedAt: time.Now(),
	}

	// Extract pattern data
	patternData, ok := data.(map[string]interface{})
	if !ok {
		result.Passed = false
		result.Description = "Invalid pattern data format"
		return result
	}

	// Check for suspicious patterns
	suspiciousIndicators := []string{}

	// Check for rapid transactions
	if re.checkRapidTransactions(patternData) {
		suspiciousIndicators = append(suspiciousIndicators, "rapid_transactions")
	}

	// Check for round amounts
	if re.checkRoundAmounts(patternData) {
		suspiciousIndicators = append(suspiciousIndicators, "round_amounts")
	}

	// Check for unusual times
	if re.checkUnusualTimes(patternData) {
		suspiciousIndicators = append(suspiciousIndicators, "unusual_times")
	}

	// Check for geographic anomalies
	if re.checkGeographicAnomalies(patternData) {
		suspiciousIndicators = append(suspiciousIndicators, "geographic_anomalies")
	}

	// Evaluate threshold
	threshold := 2 // Default threshold
	if t, exists := rule.Parameters["threshold"]; exists {
		if tInt, ok := t.(int); ok {
			threshold = tInt
		}
	}

	if len(suspiciousIndicators) >= threshold {
		result.Passed = false
		result.Description = fmt.Sprintf("Suspicious pattern detected: %v", suspiciousIndicators)
		result.Details = map[string]interface{}{
			"indicators": suspiciousIndicators,
			"count":      len(suspiciousIndicators),
			"threshold":  threshold,
		}
	} else {
		result.Passed = true
		result.Description = "No suspicious patterns detected"
		result.Details = map[string]interface{}{
			"indicators": suspiciousIndicators,
			"count":      len(suspiciousIndicators),
		}
	}

	return result
}

func (re *RuleEngine) evaluateSanctionsScreening(ctx context.Context, rule Rule, data interface{}) *RuleResult {
	result := &RuleResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		EvaluatedAt: time.Now(),
	}

	// Extract entity data
	entityData, ok := data.(map[string]interface{})
	if !ok {
		result.Passed = false
		result.Description = "Invalid entity data format"
		return result
	}

	// Get entity name for screening
	entityName, exists := entityData["name"]
	if !exists {
		result.Passed = false
		result.Description = "Entity name not found"
		return result
	}

	nameStr, ok := entityName.(string)
	if !ok {
		result.Passed = false
		result.Description = "Invalid entity name format"
		return result
	}

	// Check against sanctions lists (simplified implementation)
	sanctionedEntities := []string{
		"sanctioned_entity_1",
		"sanctioned_entity_2",
		"blocked_person",
	}

	isMatched := false
	matchType := ""

	for _, sanctioned := range sanctionedEntities {
		if nameStr == sanctioned {
			isMatched = true
			matchType = "exact"
			break
		}
		if re.fuzzyMatch(nameStr, sanctioned) {
			isMatched = true
			matchType = "fuzzy"
			break
		}
	}

	if isMatched {
		result.Passed = false
		result.Description = fmt.Sprintf("Entity matches sanctions list: %s (%s match)", nameStr, matchType)
		result.Details = map[string]interface{}{
			"entity_name": nameStr,
			"match_type":  matchType,
			"risk_level":  "high",
		}
	} else {
		result.Passed = true
		result.Description = "Entity cleared sanctions screening"
		result.Details = map[string]interface{}{
			"entity_name": nameStr,
			"screened_at": time.Now(),
		}
	}

	return result
}

func (re *RuleEngine) evaluateAMLThreshold(ctx context.Context, rule Rule, data interface{}) *RuleResult {
	result := &RuleResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		EvaluatedAt: time.Now(),
	}

	// Extract AML data
	amlData, ok := data.(map[string]interface{})
	if !ok {
		result.Passed = false
		result.Description = "Invalid AML data format"
		return result
	}

	// Calculate AML risk score
	riskScore := re.calculateAMLRiskScore(amlData)

	// Get threshold from rule parameters
	threshold := 50.0 // Default threshold
	if t, exists := rule.Parameters["threshold"]; exists {
		if tFloat, ok := t.(float64); ok {
			threshold = tFloat
		}
	}

	if riskScore > threshold {
		result.Passed = false
		result.Description = fmt.Sprintf("AML risk score %.2f exceeds threshold %.2f", riskScore, threshold)
		result.Details = map[string]interface{}{
			"risk_score": riskScore,
			"threshold":  threshold,
			"risk_level": re.getRiskLevel(riskScore),
		}
	} else {
		result.Passed = true
		result.Description = fmt.Sprintf("AML risk score %.2f within threshold", riskScore)
		result.Details = map[string]interface{}{
			"risk_score": riskScore,
			"threshold":  threshold,
		}
	}

	return result
}

func (re *RuleEngine) evaluateKYCValidation(ctx context.Context, rule Rule, data interface{}) *RuleResult {
	result := &RuleResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		EvaluatedAt: time.Now(),
	}

	// Extract KYC data
	kycData, ok := data.(map[string]interface{})
	if !ok {
		result.Passed = false
		result.Description = "Invalid KYC data format"
		return result
	}

	// Check required KYC fields
	requiredFields := []string{"full_name", "date_of_birth", "address", "identification"}
	missingFields := []string{}

	for _, field := range requiredFields {
		if _, exists := kycData[field]; !exists {
			missingFields = append(missingFields, field)
		}
	}

	// Check document verification status
	docVerified := false
	if status, exists := kycData["document_verified"]; exists {
		if verified, ok := status.(bool); ok {
			docVerified = verified
		}
	}

	if len(missingFields) > 0 || !docVerified {
		result.Passed = false
		if len(missingFields) > 0 {
			result.Description = fmt.Sprintf("KYC validation failed: missing fields %v", missingFields)
		} else {
			result.Description = "KYC validation failed: documents not verified"
		}
		result.Details = map[string]interface{}{
			"missing_fields":    missingFields,
			"document_verified": docVerified,
		}
	} else {
		result.Passed = true
		result.Description = "KYC validation passed"
		result.Details = map[string]interface{}{
			"all_fields_present": true,
			"document_verified":  docVerified,
		}
	}

	return result
}

func (re *RuleEngine) evaluatePCICompliance(ctx context.Context, rule Rule, data interface{}) *RuleResult {
	result := &RuleResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		EvaluatedAt: time.Now(),
	}

	// Extract payment data
	paymentData, ok := data.(map[string]interface{})
	if !ok {
		result.Passed = false
		result.Description = "Invalid payment data format"
		return result
	}

	violations := []string{}

	// Check for PAN encryption
	if pan, exists := paymentData["card_number"]; exists {
		if panStr, ok := pan.(string); ok {
			if !re.isPANEncrypted(panStr) {
				violations = append(violations, "unencrypted_pan")
			}
		}
	}

	// Check for CVV storage
	if _, exists := paymentData["cvv"]; exists {
		violations = append(violations, "cvv_storage_prohibited")
	}

	// Check for secure transmission
	if secure, exists := paymentData["secure_transmission"]; exists {
		if secureFlag, ok := secure.(bool); ok && !secureFlag {
			violations = append(violations, "insecure_transmission")
		}
	}

	if len(violations) > 0 {
		result.Passed = false
		result.Description = fmt.Sprintf("PCI compliance violations: %v", violations)
		result.Details = map[string]interface{}{
			"violations": violations,
			"severity":   "high",
		}
	} else {
		result.Passed = true
		result.Description = "PCI compliance requirements met"
		result.Details = map[string]interface{}{
			"compliance_status": "compliant",
		}
	}

	return result
}

func (re *RuleEngine) evaluateGDPRCompliance(ctx context.Context, rule Rule, data interface{}) *RuleResult {
	result := &RuleResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		EvaluatedAt: time.Now(),
	}

	// Extract personal data
	personalData, ok := data.(map[string]interface{})
	if !ok {
		result.Passed = false
		result.Description = "Invalid personal data format"
		return result
	}

	violations := []string{}

	// Check consent
	if consent, exists := personalData["consent_given"]; exists {
		if consentFlag, ok := consent.(bool); ok && !consentFlag {
			violations = append(violations, "missing_consent")
		}
	} else {
		violations = append(violations, "consent_not_recorded")
	}

	// Check data minimization
	if purpose, exists := personalData["processing_purpose"]; exists {
		if purposeStr, ok := purpose.(string); ok {
			if !re.isDataMinimized(personalData, purposeStr) {
				violations = append(violations, "data_not_minimized")
			}
		}
	}

	// Check retention period
	if retention, exists := personalData["retention_period"]; exists {
		if retentionStr, ok := retention.(string); ok {
			if re.isRetentionExceeded(retentionStr) {
				violations = append(violations, "retention_period_exceeded")
			}
		}
	}

	if len(violations) > 0 {
		result.Passed = false
		result.Description = fmt.Sprintf("GDPR compliance violations: %v", violations)
		result.Details = map[string]interface{}{
			"violations": violations,
			"severity":   "high",
		}
	} else {
		result.Passed = true
		result.Description = "GDPR compliance requirements met"
		result.Details = map[string]interface{}{
			"compliance_status": "compliant",
		}
	}

	return result
}

func (re *RuleEngine) evaluateCustomRule(ctx context.Context, rule Rule, data interface{}) *RuleResult {
	result := &RuleResult{
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		EvaluatedAt: time.Now(),
	}

	// Execute custom rule logic
	if script, exists := rule.Parameters["script"]; exists {
		if scriptStr, ok := script.(string); ok {
			// Execute custom script (simplified implementation)
			passed, description, details := re.executeCustomScript(scriptStr, data)
			result.Passed = passed
			result.Description = description
			result.Details = details
		} else {
			result.Passed = false
			result.Description = "Invalid custom script format"
		}
	} else {
		result.Passed = false
		result.Description = "Custom script not provided"
	}

	return result
}

// Helper methods

func (re *RuleEngine) loadDefaultRules() error {
	// Load default compliance rules
	defaultRules := []Rule{
		{
			ID:          "txn_limit_10k",
			Name:        "Transaction Limit $10,000",
			Type:        "transaction_limit",
			Severity:    "high",
			Description: "Transactions exceeding $10,000 require additional review",
			Enabled:     true,
			Parameters: map[string]interface{}{
				"threshold": 10000.0,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          "suspicious_patterns",
			Name:        "Suspicious Transaction Patterns",
			Type:        "suspicious_pattern",
			Severity:    "medium",
			Description: "Detect suspicious transaction patterns",
			Enabled:     true,
			Parameters: map[string]interface{}{
				"threshold": 2,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          "sanctions_screening",
			Name:        "Sanctions List Screening",
			Type:        "sanctions_screening",
			Severity:    "critical",
			Description: "Screen entities against sanctions lists",
			Enabled:     true,
			Parameters:  map[string]interface{}{},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, rule := range defaultRules {
		re.rules[rule.ID] = &rule
	}

	re.logger.Info("Default rules loaded", zap.Int("count", len(defaultRules)))
	return nil
}

func (re *RuleEngine) isRuleApplicable(rule *Rule, data interface{}) bool {
	if !rule.Enabled {
		return false
	}

	// Check rule conditions (simplified implementation)
	if conditions, exists := rule.Parameters["conditions"]; exists {
		if conditionsMap, ok := conditions.(map[string]interface{}); ok {
			return re.evaluateConditions(conditionsMap, data)
		}
	}

	return true
}

func (re *RuleEngine) evaluateConditions(conditions map[string]interface{}, data interface{}) bool {
	// Simplified condition evaluation
	return true
}

func (re *RuleEngine) validateRule(rule Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.Type == "" {
		return fmt.Errorf("rule type is required")
	}
	return nil
}

func (re *RuleEngine) generateCacheKey(ruleID string, data interface{}) string {
	// Generate cache key based on rule ID and data hash
	return fmt.Sprintf("%s_%d", ruleID, re.hashData(data))
}

func (re *RuleEngine) hashData(data interface{}) uint32 {
	// Simple hash function for data
	return uint32(time.Now().Unix()) % 1000000
}

func (re *RuleEngine) ruleEvaluationLoop(ctx context.Context) {
	ticker := time.NewTicker(re.config.RuleEvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-re.stopChan:
			return
		case <-ticker.C:
			// Perform periodic rule evaluation tasks
			re.performMaintenanceTasks()
		}
	}
}

func (re *RuleEngine) cacheCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(re.config.CacheTTL / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-re.stopChan:
			return
		case <-ticker.C:
			re.cleanupExpiredCache()
		}
	}
}

func (re *RuleEngine) performMaintenanceTasks() {
	// Perform rule engine maintenance tasks
	re.logger.Debug("Performing rule engine maintenance")
}

func (re *RuleEngine) cleanupExpiredCache() {
	re.mu.Lock()
	defer re.mu.Unlock()

	now := time.Now()
	for key, result := range re.ruleCache {
		if now.Sub(result.EvaluatedAt) > re.config.CacheTTL {
			delete(re.ruleCache, key)
		}
	}
}

// Pattern detection helper methods

func (re *RuleEngine) checkRapidTransactions(data map[string]interface{}) bool {
	// Check for rapid succession of transactions
	return false // Simplified implementation
}

func (re *RuleEngine) checkRoundAmounts(data map[string]interface{}) bool {
	// Check for unusually round transaction amounts
	return false // Simplified implementation
}

func (re *RuleEngine) checkUnusualTimes(data map[string]interface{}) bool {
	// Check for transactions at unusual times
	return false // Simplified implementation
}

func (re *RuleEngine) checkGeographicAnomalies(data map[string]interface{}) bool {
	// Check for geographic anomalies in transactions
	return false // Simplified implementation
}

func (re *RuleEngine) fuzzyMatch(str1, str2 string) bool {
	// Simple fuzzy matching implementation
	return false // Simplified implementation
}

func (re *RuleEngine) calculateAMLRiskScore(data map[string]interface{}) float64 {
	// Calculate AML risk score based on various factors
	return 25.0 // Simplified implementation
}

func (re *RuleEngine) getRiskLevel(score float64) string {
	if score >= 80 {
		return "high"
	} else if score >= 50 {
		return "medium"
	}
	return "low"
}

func (re *RuleEngine) isPANEncrypted(pan string) bool {
	// Check if PAN is properly encrypted
	return len(pan) > 16 // Simplified check
}

func (re *RuleEngine) isDataMinimized(data map[string]interface{}, purpose string) bool {
	// Check if data collection is minimized for the purpose
	return true // Simplified implementation
}

func (re *RuleEngine) isRetentionExceeded(retentionPeriod string) bool {
	// Check if retention period is exceeded
	return false // Simplified implementation
}

func (re *RuleEngine) executeCustomScript(script string, data interface{}) (bool, string, map[string]interface{}) {
	// Execute custom rule script
	return true, "Custom rule passed", map[string]interface{}{} // Simplified implementation
}