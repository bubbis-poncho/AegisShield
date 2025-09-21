package regulatory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aegisshield/compliance-engine/internal/compliance"
	"github.com/aegisshield/compliance-engine/internal/config"
	"go.uber.org/zap"
)

// RegulationManager manages regulatory information and monitoring
type RegulationManager struct {
	config      config.RegulationsConfig
	logger      *zap.Logger
	regulations map[string]*compliance.RegulationInfo
	changes     map[string]*compliance.RegulationChange
	sources     map[string]*RegulationSource
	watchers    map[string]*RegulationWatcher
	mu          sync.RWMutex
	running     bool
	stopChan    chan struct{}
}

// RegulationSource represents an external regulation source
type RegulationSource struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // api, rss, file, manual
	URL         string                 `json:"url"`
	Jurisdiction string                `json:"jurisdiction"`
	Enabled     bool                   `json:"enabled"`
	Frequency   time.Duration          `json:"frequency"`
	LastUpdated time.Time              `json:"last_updated"`
	Credentials map[string]interface{} `json:"credentials"`
}

// RegulationWatcher monitors regulation changes
type RegulationWatcher struct {
	SourceID      string                      `json:"source_id"`
	LastCheck     time.Time                   `json:"last_check"`
	ChangeHandler func(*compliance.RegulationChange) error `json:"-"`
	Running       bool                        `json:"running"`
	StopChan      chan struct{}               `json:"-"`
}

// NewRegulationManager creates a new regulation manager instance
func NewRegulationManager(cfg config.RegulationsConfig, logger *zap.Logger) *RegulationManager {
	return &RegulationManager{
		config:      cfg,
		logger:      logger,
		regulations: make(map[string]*compliance.RegulationInfo),
		changes:     make(map[string]*compliance.RegulationChange),
		sources:     make(map[string]*RegulationSource),
		watchers:    make(map[string]*RegulationWatcher),
		stopChan:    make(chan struct{}),
	}
}

// Start starts the regulation manager
func (rm *RegulationManager) Start(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.running {
		return fmt.Errorf("regulation manager is already running")
	}

	rm.logger.Info("Starting regulation manager")

	// Load default regulations and sources
	if err := rm.loadDefaultRegulations(); err != nil {
		return fmt.Errorf("failed to load default regulations: %w", err)
	}

	if err := rm.loadRegulationSources(); err != nil {
		return fmt.Errorf("failed to load regulation sources: %w", err)
	}

	// Start monitoring
	if rm.config.EnableAutoUpdate {
		go rm.monitoringLoop(ctx)
	}

	rm.running = true
	rm.logger.Info("Regulation manager started successfully")

	return nil
}

// Stop stops the regulation manager
func (rm *RegulationManager) Stop(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return nil
	}

	rm.logger.Info("Stopping regulation manager")

	close(rm.stopChan)

	// Stop all watchers
	for _, watcher := range rm.watchers {
		if watcher.Running {
			close(watcher.StopChan)
		}
	}

	rm.running = false
	rm.logger.Info("Regulation manager stopped")

	return nil
}

// GetRegulation retrieves a regulation by ID
func (rm *RegulationManager) GetRegulation(ctx context.Context, regulationID string) (*compliance.RegulationInfo, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if !rm.running {
		return nil, fmt.Errorf("regulation manager is not running")
	}

	regulation, exists := rm.regulations[regulationID]
	if !exists {
		return nil, fmt.Errorf("regulation not found: %s", regulationID)
	}

	return regulation, nil
}

// GetRegulationsByJurisdiction returns regulations for a specific jurisdiction
func (rm *RegulationManager) GetRegulationsByJurisdiction(ctx context.Context, jurisdiction string) ([]*compliance.RegulationInfo, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if !rm.running {
		return nil, fmt.Errorf("regulation manager is not running")
	}

	var regulations []*compliance.RegulationInfo
	for _, regulation := range rm.regulations {
		if regulation.Jurisdiction == jurisdiction {
			regulations = append(regulations, regulation)
		}
	}

	return regulations, nil
}

// GetApplicableRegulations returns regulations applicable to a specific context
func (rm *RegulationManager) GetApplicableRegulations(ctx context.Context, jurisdiction string, regulationType string) ([]*compliance.RegulationInfo, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if !rm.running {
		return nil, fmt.Errorf("regulation manager is not running")
	}

	var regulations []*compliance.RegulationInfo
	for _, regulation := range rm.regulations {
		if rm.isRegulationApplicable(regulation, jurisdiction, regulationType) {
			regulations = append(regulations, regulation)
		}
	}

	return regulations, nil
}

// AddRegulation adds a new regulation
func (rm *RegulationManager) AddRegulation(ctx context.Context, regulation *compliance.RegulationInfo) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return fmt.Errorf("regulation manager is not running")
	}

	if regulation.ID == "" {
		regulation.ID = rm.generateRegulationID()
	}

	regulation.UpdatedAt = time.Now()
	rm.regulations[regulation.ID] = regulation

	rm.logger.Info("Regulation added",
		zap.String("regulation_id", regulation.ID),
		zap.String("name", regulation.Name),
		zap.String("jurisdiction", regulation.Jurisdiction),
	)

	return nil
}

// UpdateRegulation updates an existing regulation
func (rm *RegulationManager) UpdateRegulation(ctx context.Context, regulation *compliance.RegulationInfo) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return fmt.Errorf("regulation manager is not running")
	}

	if _, exists := rm.regulations[regulation.ID]; !exists {
		return fmt.Errorf("regulation not found: %s", regulation.ID)
	}

	regulation.UpdatedAt = time.Now()
	rm.regulations[regulation.ID] = regulation

	rm.logger.Info("Regulation updated",
		zap.String("regulation_id", regulation.ID),
		zap.String("name", regulation.Name),
	)

	return nil
}

// GetRegulationChanges returns recent regulation changes
func (rm *RegulationManager) GetRegulationChanges(ctx context.Context, since time.Time) ([]*compliance.RegulationChange, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if !rm.running {
		return nil, fmt.Errorf("regulation manager is not running")
	}

	var changes []*compliance.RegulationChange
	for _, change := range rm.changes {
		if change.DetectedAt.After(since) {
			changes = append(changes, change)
		}
	}

	return changes, nil
}

// MonitorRegulationChanges sets up monitoring for regulation changes
func (rm *RegulationManager) MonitorRegulationChanges(ctx context.Context, sourceID string, handler func(*compliance.RegulationChange) error) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.running {
		return fmt.Errorf("regulation manager is not running")
	}

	source, exists := rm.sources[sourceID]
	if !exists {
		return fmt.Errorf("regulation source not found: %s", sourceID)
	}

	if !source.Enabled {
		return fmt.Errorf("regulation source is disabled: %s", sourceID)
	}

	watcher := &RegulationWatcher{
		SourceID:      sourceID,
		LastCheck:     time.Now(),
		ChangeHandler: handler,
		Running:       true,
		StopChan:      make(chan struct{}),
	}

	rm.watchers[sourceID] = watcher

	// Start watching
	go rm.watchRegulationSource(ctx, watcher, source)

	rm.logger.Info("Started monitoring regulation source",
		zap.String("source_id", sourceID),
		zap.String("source_name", source.Name),
	)

	return nil
}

// CheckComplianceRequirements checks compliance requirements against regulations
func (rm *RegulationManager) CheckComplianceRequirements(ctx context.Context, jurisdiction string, entityType string, requirements []string) (*ComplianceCheck, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if !rm.running {
		return nil, fmt.Errorf("regulation manager is not running")
	}

	applicableRegulations, err := rm.GetApplicableRegulations(ctx, jurisdiction, entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to get applicable regulations: %w", err)
	}

	check := &ComplianceCheck{
		ID:            rm.generateCheckID(),
		Jurisdiction:  jurisdiction,
		EntityType:    entityType,
		Requirements:  requirements,
		CheckedAt:     time.Now(),
		Regulations:   applicableRegulations,
		Results:       make(map[string]*RequirementResult),
	}

	// Check each requirement against applicable regulations
	for _, requirement := range requirements {
		result := rm.checkRequirement(requirement, applicableRegulations)
		check.Results[requirement] = result
	}

	// Calculate overall compliance status
	check.OverallStatus = rm.calculateOverallStatus(check.Results)

	return check, nil
}

// Private methods

func (rm *RegulationManager) loadDefaultRegulations() error {
	// Load default regulations for common jurisdictions
	defaultRegulations := []*compliance.RegulationInfo{
		{
			ID:           "BSA_US",
			Name:         "Bank Secrecy Act",
			Jurisdiction: "US",
			Type:         "federal",
			Version:      "2023.1",
			EffectiveDate: time.Date(1970, 10, 26, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Now(),
			Source:       "FinCEN",
			URL:          "https://www.fincen.gov/resources/statutes-regulations/bank-secrecy-act",
			Summary:      "Requires financial institutions to assist U.S. government agencies in detecting and preventing money laundering",
			Requirements: []string{
				"CTR filing for transactions over $10,000",
				"SAR filing for suspicious activities",
				"Customer due diligence",
				"Record keeping requirements",
			},
			Tags: []string{"AML", "BSA", "FinCEN"},
		},
		{
			ID:           "GDPR_EU",
			Name:         "General Data Protection Regulation",
			Jurisdiction: "EU",
			Type:         "international",
			Version:      "2016/679",
			EffectiveDate: time.Date(2018, 5, 25, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Now(),
			Source:       "European Commission",
			URL:          "https://gdpr-info.eu/",
			Summary:      "Regulation on data protection and privacy for individuals within the European Union",
			Requirements: []string{
				"Lawful basis for processing",
				"Data subject consent",
				"Right to be forgotten",
				"Data breach notification",
				"Privacy by design",
			},
			Tags: []string{"Privacy", "Data Protection", "GDPR"},
		},
		{
			ID:           "SOX_US",
			Name:         "Sarbanes-Oxley Act",
			Jurisdiction: "US",
			Type:         "federal",
			Version:      "2002",
			EffectiveDate: time.Date(2002, 7, 30, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Now(),
			Source:       "SEC",
			URL:          "https://www.sec.gov/about/laws/soa2002.pdf",
			Summary:      "Federal law that strengthened corporate disclosure requirements and the accountability of auditing firms",
			Requirements: []string{
				"Internal controls over financial reporting",
				"Management assessment of controls",
				"Auditor attestation",
				"CEO/CFO certifications",
			},
			Tags: []string{"Financial Reporting", "Corporate Governance", "SOX"},
		},
		{
			ID:           "PCI_DSS",
			Name:         "Payment Card Industry Data Security Standard",
			Jurisdiction: "International",
			Type:         "industry",
			Version:      "4.0",
			EffectiveDate: time.Date(2022, 3, 31, 0, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Now(),
			Source:       "PCI Security Standards Council",
			URL:          "https://www.pcisecuritystandards.org/",
			Summary:      "Security standard for organizations that handle branded credit cards",
			Requirements: []string{
				"Install and maintain firewall configuration",
				"Do not use vendor-supplied defaults",
				"Protect stored cardholder data",
				"Encrypt transmission of cardholder data",
				"Use and regularly update anti-virus software",
				"Develop and maintain secure systems",
			},
			Tags: []string{"Payment Security", "Data Security", "PCI"},
		},
	}

	for _, regulation := range defaultRegulations {
		rm.regulations[regulation.ID] = regulation
	}

	rm.logger.Info("Default regulations loaded", zap.Int("count", len(defaultRegulations)))
	return nil
}

func (rm *RegulationManager) loadRegulationSources() error {
	// Load default regulation sources
	defaultSources := []*RegulationSource{
		{
			ID:           "fincen_rss",
			Name:         "FinCEN RSS Feed",
			Type:         "rss",
			URL:          "https://www.fincen.gov/news-room/rss.xml",
			Jurisdiction: "US",
			Enabled:      rm.config.EnableAutoUpdate,
			Frequency:    24 * time.Hour,
			LastUpdated:  time.Now(),
		},
		{
			ID:           "sec_rss",
			Name:         "SEC RSS Feed",
			Type:         "rss",
			URL:          "https://www.sec.gov/news/rss.xml",
			Jurisdiction: "US",
			Enabled:      rm.config.EnableAutoUpdate,
			Frequency:    12 * time.Hour,
			LastUpdated:  time.Now(),
		},
		{
			ID:           "eu_gdpr_updates",
			Name:         "EU GDPR Updates",
			Type:         "api",
			URL:          "https://ec.europa.eu/info/law/law-topic/data-protection/reform/rules-business-and-organisations/legal-grounds-processing-data/what-does-lawful-basis-mean_en",
			Jurisdiction: "EU",
			Enabled:      rm.config.EnableAutoUpdate,
			Frequency:    7 * 24 * time.Hour, // Weekly
			LastUpdated:  time.Now(),
		},
	}

	for _, source := range defaultSources {
		rm.sources[source.ID] = source
	}

	rm.logger.Info("Regulation sources loaded", zap.Int("count", len(defaultSources)))
	return nil
}

func (rm *RegulationManager) isRegulationApplicable(regulation *compliance.RegulationInfo, jurisdiction string, regulationType string) bool {
	// Check jurisdiction match
	if jurisdiction != "" && regulation.Jurisdiction != jurisdiction && regulation.Jurisdiction != "International" {
		return false
	}

	// Check regulation type match
	if regulationType != "" && regulation.Type != regulationType {
		return false
	}

	// Check if regulation is still effective
	if !regulation.EffectiveDate.IsZero() && regulation.EffectiveDate.After(time.Now()) {
		return false
	}

	return true
}

func (rm *RegulationManager) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(rm.config.UpdateCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rm.stopChan:
			return
		case <-ticker.C:
			rm.checkForUpdates()
		}
	}
}

func (rm *RegulationManager) watchRegulationSource(ctx context.Context, watcher *RegulationWatcher, source *RegulationSource) {
	ticker := time.NewTicker(source.Frequency)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-watcher.StopChan:
			return
		case <-ticker.C:
			rm.checkSourceForChanges(watcher, source)
		}
	}
}

func (rm *RegulationManager) checkForUpdates() {
	rm.mu.RLock()
	sources := make([]*RegulationSource, 0, len(rm.sources))
	for _, source := range rm.sources {
		if source.Enabled {
			sources = append(sources, source)
		}
	}
	rm.mu.RUnlock()

	for _, source := range sources {
		if time.Since(source.LastUpdated) >= source.Frequency {
			go rm.updateFromSource(source)
		}
	}
}

func (rm *RegulationManager) checkSourceForChanges(watcher *RegulationWatcher, source *RegulationSource) {
	rm.logger.Debug("Checking regulation source for changes",
		zap.String("source_id", source.ID),
		zap.String("source_name", source.Name),
	)

	// Simplified change detection (in production, would implement proper change detection)
	changes := rm.detectChangesFromSource(source)

	for _, change := range changes {
		// Store change
		rm.mu.Lock()
		rm.changes[change.ID] = change
		rm.mu.Unlock()

		// Notify handler
		if watcher.ChangeHandler != nil {
			if err := watcher.ChangeHandler(change); err != nil {
				rm.logger.Error("Failed to handle regulation change",
					zap.String("change_id", change.ID),
					zap.Error(err),
				)
			}
		}

		rm.logger.Info("Regulation change detected",
			zap.String("change_id", change.ID),
			zap.String("regulation_id", change.RegulationID),
			zap.String("change_type", change.ChangeType),
		)
	}

	watcher.LastCheck = time.Now()
}

func (rm *RegulationManager) updateFromSource(source *RegulationSource) {
	rm.logger.Info("Updating regulations from source",
		zap.String("source_id", source.ID),
		zap.String("source_name", source.Name),
	)

	// Simplified source update (in production, would implement proper source parsing)
	// This would fetch and parse updates from the actual source

	source.LastUpdated = time.Now()
}

func (rm *RegulationManager) detectChangesFromSource(source *RegulationSource) []*compliance.RegulationChange {
	// Simplified change detection (in production, would implement proper detection)
	// This would compare current regulations with source data and detect changes

	// Return empty for now
	return []*compliance.RegulationChange{}
}

func (rm *RegulationManager) checkRequirement(requirement string, regulations []*compliance.RegulationInfo) *RequirementResult {
	result := &RequirementResult{
		Requirement:   requirement,
		Status:        "not_covered",
		CoveredBy:     []string{},
		Gaps:          []string{},
		Recommendations: []string{},
	}

	// Check if requirement is covered by any regulation
	for _, regulation := range regulations {
		for _, regRequirement := range regulation.Requirements {
			if rm.requirementMatches(requirement, regRequirement) {
				result.Status = "covered"
				result.CoveredBy = append(result.CoveredBy, regulation.ID)
				break
			}
		}
	}

	// Generate recommendations if not covered
	if result.Status == "not_covered" {
		result.Recommendations = []string{
			fmt.Sprintf("Consider implementing controls for: %s", requirement),
			"Review applicable regulations for this requirement",
		}
		result.Gaps = []string{requirement}
	}

	return result
}

func (rm *RegulationManager) requirementMatches(requirement string, regRequirement string) bool {
	// Simplified matching (in production, would implement sophisticated matching)
	return requirement == regRequirement
}

func (rm *RegulationManager) calculateOverallStatus(results map[string]*RequirementResult) string {
	totalRequirements := len(results)
	coveredRequirements := 0

	for _, result := range results {
		if result.Status == "covered" {
			coveredRequirements++
		}
	}

	if coveredRequirements == totalRequirements {
		return "compliant"
	} else if coveredRequirements > 0 {
		return "partial"
	}

	return "non_compliant"
}

func (rm *RegulationManager) generateRegulationID() string {
	return fmt.Sprintf("REG_%d", time.Now().UnixNano())
}

func (rm *RegulationManager) generateCheckID() string {
	return fmt.Sprintf("CHECK_%d", time.Now().UnixNano())
}

// Supporting types

type ComplianceCheck struct {
	ID             string                           `json:"id"`
	Jurisdiction   string                           `json:"jurisdiction"`
	EntityType     string                           `json:"entity_type"`
	Requirements   []string                         `json:"requirements"`
	CheckedAt      time.Time                        `json:"checked_at"`
	Regulations    []*compliance.RegulationInfo     `json:"regulations"`
	Results        map[string]*RequirementResult    `json:"results"`
	OverallStatus  string                           `json:"overall_status"` // compliant, partial, non_compliant
}

type RequirementResult struct {
	Requirement     string   `json:"requirement"`
	Status          string   `json:"status"` // covered, not_covered, partial
	CoveredBy       []string `json:"covered_by"`
	Gaps            []string `json:"gaps"`
	Recommendations []string `json:"recommendations"`
}