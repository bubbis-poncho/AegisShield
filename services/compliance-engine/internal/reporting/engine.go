package reporting

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aegisshield/compliance-engine/internal/compliance"
	"github.com/aegisshield/compliance-engine/internal/config"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

// ReportEngine manages report generation and distribution
type ReportEngine struct {
	config         config.ReportingConfig
	logger         *zap.Logger
	templates      map[string]*compliance.ReportTemplate
	schedules      map[string]*compliance.ReportSchedule
	activeReports  map[string]*ReportStatus
	mu             sync.RWMutex
	running        bool
	stopChan       chan struct{}
}

// ReportStatus represents the status of a report generation
type ReportStatus struct {
	ReportID    string    `json:"report_id"`
	Status      string    `json:"status"`
	Progress    float64   `json:"progress"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Error       string    `json:"error,omitempty"`
}

// NewReportEngine creates a new report engine instance
func NewReportEngine(cfg config.ReportingConfig, logger *zap.Logger) *ReportEngine {
	return &ReportEngine{
		config:        cfg,
		logger:        logger,
		templates:     make(map[string]*compliance.ReportTemplate),
		schedules:     make(map[string]*compliance.ReportSchedule),
		activeReports: make(map[string]*ReportStatus),
		stopChan:      make(chan struct{}),
	}
}

// Start starts the report engine
func (re *ReportEngine) Start(ctx context.Context) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if re.running {
		return fmt.Errorf("report engine is already running")
	}

	re.logger.Info("Starting report engine")

	// Load default templates
	if err := re.loadDefaultTemplates(); err != nil {
		return fmt.Errorf("failed to load default templates: %w", err)
	}

	// Start background scheduler
	go re.schedulerLoop(ctx)

	re.running = true
	re.logger.Info("Report engine started successfully")

	return nil
}

// Stop stops the report engine
func (re *ReportEngine) Stop(ctx context.Context) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if !re.running {
		return nil
	}

	re.logger.Info("Stopping report engine")

	close(re.stopChan)
	re.running = false

	re.logger.Info("Report engine stopped")
	return nil
}

// GenerateReport generates a report based on template and parameters
func (re *ReportEngine) GenerateReport(ctx context.Context, templateID string, parameters map[string]interface{}) (*compliance.Report, error) {
	re.mu.RLock()
	template, exists := re.templates[templateID]
	re.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	report := &compliance.Report{
		ID:          re.generateReportID(),
		Name:        fmt.Sprintf("%s_%s", template.Name, time.Now().Format("20060102_150405")),
		Type:        template.Type,
		Status:      "generating",
		Format:      template.Format,
		TemplateID:  templateID,
		Parameters:  parameters,
		GeneratedAt: time.Now(),
	}

	// Track report generation
	re.mu.Lock()
	re.activeReports[report.ID] = &ReportStatus{
		ReportID:  report.ID,
		Status:    "generating",
		Progress:  0.0,
		StartedAt: time.Now(),
	}
	re.mu.Unlock()

	// Generate report content asynchronously
	go re.generateReportContent(ctx, report, template)

	return report, nil
}

// GetReportStatus returns the status of a report generation
func (re *ReportEngine) GetReportStatus(ctx context.Context, reportID string) (*ReportStatus, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	status, exists := re.activeReports[reportID]
	if !exists {
		return nil, fmt.Errorf("report not found: %s", reportID)
	}

	return status, nil
}

// GetTemplate returns a report template by ID
func (re *ReportEngine) GetTemplate(ctx context.Context, templateID string) (*compliance.ReportTemplate, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	template, exists := re.templates[templateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	return template, nil
}

// ListTemplates returns all available report templates
func (re *ReportEngine) ListTemplates(ctx context.Context) ([]*compliance.ReportTemplate, error) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	templates := make([]*compliance.ReportTemplate, 0, len(re.templates))
	for _, template := range re.templates {
		templates = append(templates, template)
	}

	return templates, nil
}

// CreateTemplate creates a new report template
func (re *ReportEngine) CreateTemplate(ctx context.Context, template *compliance.ReportTemplate) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if template.ID == "" {
		template.ID = re.generateTemplateID()
	}

	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	re.templates[template.ID] = template

	re.logger.Info("Report template created",
		zap.String("template_id", template.ID),
		zap.String("name", template.Name),
	)

	return nil
}

// UpdateTemplate updates an existing report template
func (re *ReportEngine) UpdateTemplate(ctx context.Context, template *compliance.ReportTemplate) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if _, exists := re.templates[template.ID]; !exists {
		return fmt.Errorf("template not found: %s", template.ID)
	}

	template.UpdatedAt = time.Now()
	re.templates[template.ID] = template

	re.logger.Info("Report template updated",
		zap.String("template_id", template.ID),
		zap.String("name", template.Name),
	)

	return nil
}

// DeleteTemplate deletes a report template
func (re *ReportEngine) DeleteTemplate(ctx context.Context, templateID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if _, exists := re.templates[templateID]; !exists {
		return fmt.Errorf("template not found: %s", templateID)
	}

	delete(re.templates, templateID)

	re.logger.Info("Report template deleted",
		zap.String("template_id", templateID),
	)

	return nil
}

// ScheduleReport schedules a report for periodic generation
func (re *ReportEngine) ScheduleReport(ctx context.Context, schedule *compliance.ReportSchedule) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if schedule.ID == "" {
		schedule.ID = re.generateScheduleID()
	}

	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()
	schedule.NextRun = re.calculateNextRun(schedule.Frequency)

	re.schedules[schedule.ID] = schedule

	re.logger.Info("Report scheduled",
		zap.String("schedule_id", schedule.ID),
		zap.String("frequency", schedule.Frequency),
		zap.Time("next_run", schedule.NextRun),
	)

	return nil
}

// Private methods

func (re *ReportEngine) generateReportContent(ctx context.Context, report *compliance.Report, template *compliance.ReportTemplate) {
	re.updateReportStatus(report.ID, "generating", 10.0, "")

	// Generate content based on format
	var content []byte
	var err error

	switch template.Format {
	case compliance.ReportFormatPDF:
		content, err = re.generatePDFReport(ctx, report, template)
	case compliance.ReportFormatExcel:
		content, err = re.generateExcelReport(ctx, report, template)
	case compliance.ReportFormatCSV:
		content, err = re.generateCSVReport(ctx, report, template)
	case compliance.ReportFormatJSON:
		content, err = re.generateJSONReport(ctx, report, template)
	case compliance.ReportFormatXML:
		content, err = re.generateXMLReport(ctx, report, template)
	default:
		err = fmt.Errorf("unsupported report format: %s", template.Format)
	}

	if err != nil {
		re.updateReportStatus(report.ID, "failed", 0.0, err.Error())
		re.logger.Error("Failed to generate report",
			zap.String("report_id", report.ID),
			zap.Error(err),
		)
		return
	}

	// Update report with content
	re.mu.Lock()
	report.Content = content
	report.Status = "completed"
	re.mu.Unlock()

	re.updateReportStatus(report.ID, "completed", 100.0, "")

	re.logger.Info("Report generated successfully",
		zap.String("report_id", report.ID),
		zap.String("format", template.Format),
		zap.Int("size_bytes", len(content)),
	)
}

func (re *ReportEngine) generatePDFReport(ctx context.Context, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 30.0, "Generating PDF content")

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	// Title
	pdf.Cell(40, 10, report.Name)
	pdf.Ln(12)

	// Add content based on template type
	switch template.Type {
	case compliance.ReportTypeViolation:
		return re.generateViolationPDFContent(ctx, pdf, report, template)
	case compliance.ReportTypeRegulatory:
		return re.generateRegulatoryPDFContent(ctx, pdf, report, template)
	case compliance.ReportTypeMetrics:
		return re.generateMetricsPDFContent(ctx, pdf, report, template)
	default:
		return re.generateGenericPDFContent(ctx, pdf, report, template)
	}
}

func (re *ReportEngine) generateExcelReport(ctx context.Context, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 30.0, "Generating Excel content")

	f := excelize.NewFile()
	defer f.Close()

	// Create main sheet
	sheetName := "Report"
	f.SetSheetName("Sheet1", sheetName)

	// Add headers
	headers := []string{"ID", "Name", "Type", "Severity", "Status", "Created At"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// Add data based on template type
	switch template.Type {
	case compliance.ReportTypeViolation:
		return re.generateViolationExcelContent(ctx, f, sheetName, report, template)
	case compliance.ReportTypeRegulatory:
		return re.generateRegulatoryExcelContent(ctx, f, sheetName, report, template)
	case compliance.ReportTypeMetrics:
		return re.generateMetricsExcelContent(ctx, f, sheetName, report, template)
	default:
		return re.generateGenericExcelContent(ctx, f, sheetName, report, template)
	}
}

func (re *ReportEngine) generateCSVReport(ctx context.Context, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 30.0, "Generating CSV content")

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Add headers
	headers := []string{"ID", "Name", "Type", "Severity", "Status", "Created At"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Add data based on template type
	switch template.Type {
	case compliance.ReportTypeViolation:
		return re.generateViolationCSVContent(ctx, writer, &buf, report, template)
	case compliance.ReportTypeRegulatory:
		return re.generateRegulatoryCSVContent(ctx, writer, &buf, report, template)
	case compliance.ReportTypeMetrics:
		return re.generateMetricsCSVContent(ctx, writer, &buf, report, template)
	default:
		return re.generateGenericCSVContent(ctx, writer, &buf, report, template)
	}
}

func (re *ReportEngine) generateJSONReport(ctx context.Context, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 30.0, "Generating JSON content")

	// Create report data structure
	reportData := map[string]interface{}{
		"report_id":    report.ID,
		"name":         report.Name,
		"type":         report.Type,
		"generated_at": report.GeneratedAt,
		"parameters":   report.Parameters,
	}

	// Add content based on template type
	switch template.Type {
	case compliance.ReportTypeViolation:
		data, err := re.getViolationData(ctx, report, template)
		if err != nil {
			return nil, err
		}
		reportData["violations"] = data
	case compliance.ReportTypeRegulatory:
		data, err := re.getRegulatoryData(ctx, report, template)
		if err != nil {
			return nil, err
		}
		reportData["regulatory_info"] = data
	case compliance.ReportTypeMetrics:
		data, err := re.getMetricsData(ctx, report, template)
		if err != nil {
			return nil, err
		}
		reportData["metrics"] = data
	}

	re.updateReportStatus(report.ID, "generating", 80.0, "Serializing JSON")

	content, err := json.MarshalIndent(reportData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return content, nil
}

func (re *ReportEngine) generateXMLReport(ctx context.Context, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 30.0, "Generating XML content")

	// Create report data structure
	type XMLReport struct {
		XMLName     xml.Name    `xml:"report"`
		ID          string      `xml:"id"`
		Name        string      `xml:"name"`
		Type        string      `xml:"type"`
		GeneratedAt time.Time   `xml:"generated_at"`
		Data        interface{} `xml:"data"`
	}

	xmlReport := XMLReport{
		ID:          report.ID,
		Name:        report.Name,
		Type:        report.Type,
		GeneratedAt: report.GeneratedAt,
	}

	// Add content based on template type
	switch template.Type {
	case compliance.ReportTypeViolation:
		data, err := re.getViolationData(ctx, report, template)
		if err != nil {
			return nil, err
		}
		xmlReport.Data = data
	case compliance.ReportTypeRegulatory:
		data, err := re.getRegulatoryData(ctx, report, template)
		if err != nil {
			return nil, err
		}
		xmlReport.Data = data
	case compliance.ReportTypeMetrics:
		data, err := re.getMetricsData(ctx, report, template)
		if err != nil {
			return nil, err
		}
		xmlReport.Data = data
	}

	re.updateReportStatus(report.ID, "generating", 80.0, "Serializing XML")

	content, err := xml.MarshalIndent(xmlReport, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}

	return content, nil
}

// Data retrieval methods (simplified implementations)

func (re *ReportEngine) getViolationData(ctx context.Context, report *compliance.Report, template *compliance.ReportTemplate) (interface{}, error) {
	// This would integrate with the violation manager to get actual violation data
	// For now, return mock data
	return []map[string]interface{}{
		{
			"id":         "VIO_001",
			"rule_id":    "RULE_001",
			"severity":   "high",
			"status":     "open",
			"created_at": time.Now().AddDate(0, 0, -1),
		},
		{
			"id":         "VIO_002",
			"rule_id":    "RULE_002",
			"severity":   "medium",
			"status":     "resolved",
			"created_at": time.Now().AddDate(0, 0, -2),
		},
	}, nil
}

func (re *ReportEngine) getRegulatoryData(ctx context.Context, report *compliance.Report, template *compliance.ReportTemplate) (interface{}, error) {
	// This would integrate with the regulation manager to get actual regulatory data
	// For now, return mock data
	return map[string]interface{}{
		"regulations": []map[string]interface{}{
			{
				"id":           "REG_001",
				"name":         "AML Regulation",
				"jurisdiction": "US",
				"status":       "active",
			},
		},
		"compliance_status": "compliant",
	}, nil
}

func (re *ReportEngine) getMetricsData(ctx context.Context, report *compliance.Report, template *compliance.ReportTemplate) (interface{}, error) {
	// This would integrate with the metrics system to get actual metrics data
	// For now, return mock data
	return map[string]interface{}{
		"total_violations":     150,
		"resolved_violations":  120,
		"pending_violations":   30,
		"compliance_score":     85.5,
		"trend_direction":      "improving",
	}, nil
}

// PDF content generation methods (simplified implementations)

func (re *ReportEngine) generateViolationPDFContent(ctx context.Context, pdf *gofpdf.Fpdf, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding violation data to PDF")

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 10, "Violation Report")
	pdf.Ln(8)

	// Add mock violation data
	violations := []string{"VIO_001 - High Severity", "VIO_002 - Medium Severity"}
	for _, violation := range violations {
		pdf.Cell(40, 6, violation)
		pdf.Ln(6)
	}

	return re.finalizePDF(pdf)
}

func (re *ReportEngine) generateRegulatoryPDFContent(ctx context.Context, pdf *gofpdf.Fpdf, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding regulatory data to PDF")

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 10, "Regulatory Compliance Report")
	pdf.Ln(8)

	pdf.Cell(40, 6, "Overall Status: Compliant")
	pdf.Ln(6)

	return re.finalizePDF(pdf)
}

func (re *ReportEngine) generateMetricsPDFContent(ctx context.Context, pdf *gofpdf.Fpdf, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding metrics data to PDF")

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 10, "Compliance Metrics Report")
	pdf.Ln(8)

	pdf.Cell(40, 6, "Total Violations: 150")
	pdf.Ln(6)
	pdf.Cell(40, 6, "Compliance Score: 85.5%")
	pdf.Ln(6)

	return re.finalizePDF(pdf)
}

func (re *ReportEngine) generateGenericPDFContent(ctx context.Context, pdf *gofpdf.Fpdf, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding generic content to PDF")

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 10, "Generic Report")
	pdf.Ln(8)

	return re.finalizePDF(pdf)
}

func (re *ReportEngine) finalizePDF(pdf *gofpdf.Fpdf) ([]byte, error) {
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// Excel content generation methods (simplified implementations)

func (re *ReportEngine) generateViolationExcelContent(ctx context.Context, f *excelize.File, sheetName string, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding violation data to Excel")

	// Add sample data
	violations := [][]interface{}{
		{"VIO_001", "Transaction Limit Violation", "violation", "high", "open", time.Now().AddDate(0, 0, -1)},
		{"VIO_002", "Suspicious Pattern", "violation", "medium", "resolved", time.Now().AddDate(0, 0, -2)},
	}

	for i, violation := range violations {
		row := i + 2
		for j, value := range violation {
			cell := fmt.Sprintf("%c%d", 'A'+j, row)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	return re.finalizeExcel(f)
}

func (re *ReportEngine) generateRegulatoryExcelContent(ctx context.Context, f *excelize.File, sheetName string, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding regulatory data to Excel")

	// Add regulatory summary
	f.SetCellValue(sheetName, "A2", "Overall Compliance Status")
	f.SetCellValue(sheetName, "B2", "Compliant")

	return re.finalizeExcel(f)
}

func (re *ReportEngine) generateMetricsExcelContent(ctx context.Context, f *excelize.File, sheetName string, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding metrics data to Excel")

	// Add metrics data
	metrics := [][]interface{}{
		{"Total Violations", 150},
		{"Resolved Violations", 120},
		{"Pending Violations", 30},
		{"Compliance Score", 85.5},
	}

	for i, metric := range metrics {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), metric[0])
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), metric[1])
	}

	return re.finalizeExcel(f)
}

func (re *ReportEngine) generateGenericExcelContent(ctx context.Context, f *excelize.File, sheetName string, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding generic content to Excel")

	f.SetCellValue(sheetName, "A2", "Report Type")
	f.SetCellValue(sheetName, "B2", report.Type)

	return re.finalizeExcel(f)
}

func (re *ReportEngine) finalizeExcel(f *excelize.File) ([]byte, error) {
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate Excel file: %w", err)
	}
	return buf.Bytes(), nil
}

// CSV content generation methods (simplified implementations)

func (re *ReportEngine) generateViolationCSVContent(ctx context.Context, writer *csv.Writer, buf *bytes.Buffer, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding violation data to CSV")

	violations := [][]string{
		{"VIO_001", "Transaction Limit Violation", "violation", "high", "open", time.Now().AddDate(0, 0, -1).Format("2006-01-02")},
		{"VIO_002", "Suspicious Pattern", "violation", "medium", "resolved", time.Now().AddDate(0, 0, -2).Format("2006-01-02")},
	}

	for _, violation := range violations {
		if err := writer.Write(violation); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	return buf.Bytes(), nil
}

func (re *ReportEngine) generateRegulatoryCSVContent(ctx context.Context, writer *csv.Writer, buf *bytes.Buffer, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding regulatory data to CSV")

	record := []string{"Overall Status", "Compliant", "regulatory", "info", "active", time.Now().Format("2006-01-02")}
	if err := writer.Write(record); err != nil {
		return nil, fmt.Errorf("failed to write CSV row: %w", err)
	}

	writer.Flush()
	return buf.Bytes(), nil
}

func (re *ReportEngine) generateMetricsCSVContent(ctx context.Context, writer *csv.Writer, buf *bytes.Buffer, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding metrics data to CSV")

	metrics := [][]string{
		{"Total Violations", "150", "metric", "info", "current", time.Now().Format("2006-01-02")},
		{"Compliance Score", "85.5", "metric", "info", "current", time.Now().Format("2006-01-02")},
	}

	for _, metric := range metrics {
		if err := writer.Write(metric); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	return buf.Bytes(), nil
}

func (re *ReportEngine) generateGenericCSVContent(ctx context.Context, writer *csv.Writer, buf *bytes.Buffer, report *compliance.Report, template *compliance.ReportTemplate) ([]byte, error) {
	re.updateReportStatus(report.ID, "generating", 60.0, "Adding generic content to CSV")

	record := []string{report.ID, report.Name, report.Type, "info", "generated", report.GeneratedAt.Format("2006-01-02")}
	if err := writer.Write(record); err != nil {
		return nil, fmt.Errorf("failed to write CSV row: %w", err)
	}

	writer.Flush()
	return buf.Bytes(), nil
}

// Helper methods

func (re *ReportEngine) updateReportStatus(reportID string, status string, progress float64, message string) {
	re.mu.Lock()
	defer re.mu.Unlock()

	if reportStatus, exists := re.activeReports[reportID]; exists {
		reportStatus.Status = status
		reportStatus.Progress = progress
		if status == "completed" || status == "failed" {
			reportStatus.CompletedAt = time.Now()
		}
		if status == "failed" {
			reportStatus.Error = message
		}
	}
}

func (re *ReportEngine) loadDefaultTemplates() error {
	defaultTemplates := []*compliance.ReportTemplate{
		{
			ID:          "violation_summary",
			Name:        "Violation Summary Report",
			Description: "Summary of all compliance violations",
			Type:        compliance.ReportTypeViolation,
			Format:      compliance.ReportFormatPDF,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "regulatory_compliance",
			Name:        "Regulatory Compliance Report",
			Description: "Overall regulatory compliance status",
			Type:        compliance.ReportTypeRegulatory,
			Format:      compliance.ReportFormatExcel,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "compliance_metrics",
			Name:        "Compliance Metrics Dashboard",
			Description: "Key compliance metrics and trends",
			Type:        compliance.ReportTypeMetrics,
			Format:      compliance.ReportFormatJSON,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, template := range defaultTemplates {
		re.templates[template.ID] = template
	}

	re.logger.Info("Default report templates loaded", zap.Int("count", len(defaultTemplates)))
	return nil
}

func (re *ReportEngine) schedulerLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-re.stopChan:
			return
		case <-ticker.C:
			re.checkScheduledReports(ctx)
		}
	}
}

func (re *ReportEngine) checkScheduledReports(ctx context.Context) {
	re.mu.RLock()
	schedules := make([]*compliance.ReportSchedule, 0, len(re.schedules))
	for _, schedule := range re.schedules {
		schedules = append(schedules, schedule)
	}
	re.mu.RUnlock()

	now := time.Now()
	for _, schedule := range schedules {
		if schedule.Enabled && schedule.NextRun.Before(now) {
			go re.executeScheduledReport(ctx, schedule)
		}
	}
}

func (re *ReportEngine) executeScheduledReport(ctx context.Context, schedule *compliance.ReportSchedule) {
	re.logger.Info("Executing scheduled report",
		zap.String("schedule_id", schedule.ID),
		zap.String("template_id", schedule.TemplateID),
	)

	_, err := re.GenerateReport(ctx, schedule.TemplateID, schedule.Parameters)
	if err != nil {
		re.logger.Error("Failed to execute scheduled report",
			zap.String("schedule_id", schedule.ID),
			zap.Error(err),
		)
		return
	}

	// Update next run time
	re.mu.Lock()
	schedule.LastRun = time.Now()
	schedule.NextRun = re.calculateNextRun(schedule.Frequency)
	re.mu.Unlock()
}

func (re *ReportEngine) calculateNextRun(frequency string) time.Time {
	now := time.Now()
	switch frequency {
	case "daily":
		return now.AddDate(0, 0, 1)
	case "weekly":
		return now.AddDate(0, 0, 7)
	case "monthly":
		return now.AddDate(0, 1, 0)
	case "quarterly":
		return now.AddDate(0, 3, 0)
	default:
		return now.AddDate(0, 0, 1) // Default to daily
	}
}

func (re *ReportEngine) generateReportID() string {
	return fmt.Sprintf("RPT_%d", time.Now().UnixNano())
}

func (re *ReportEngine) generateTemplateID() string {
	return fmt.Sprintf("TPL_%d", time.Now().UnixNano())
}

func (re *ReportEngine) generateScheduleID() string {
	return fmt.Sprintf("SCH_%d", time.Now().UnixNano())
}