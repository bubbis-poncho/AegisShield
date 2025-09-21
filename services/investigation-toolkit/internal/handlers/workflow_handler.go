package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"investigation-toolkit/internal/models"
	"investigation-toolkit/internal/repository"
)

type WorkflowHandler struct {
	workflowRepo repository.WorkflowRepository
	auditRepo   repository.AuditRepository
}

func NewWorkflowHandler(workflowRepo repository.WorkflowRepository, auditRepo repository.AuditRepository) *WorkflowHandler {
	return &WorkflowHandler{
		workflowRepo: workflowRepo,
		auditRepo:   auditRepo,
	}
}

// Workflow Templates
func (h *WorkflowHandler) CreateTemplate(c *gin.Context) {
	var req models.CreateWorkflowTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	template := &models.WorkflowTemplate{
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		Category:    req.Category,
		IsActive:    req.IsActive,
		Steps:       req.Steps,
		CreatedBy:   req.CreatedBy,
	}

	if err := h.workflowRepo.CreateTemplate(c.Request.Context(), template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workflow template", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.CreatedBy,
		Action:      "create_workflow_template",
		EntityType:  "workflow_template",
		EntityID:    &template.ID,
		Description: "Created workflow template: " + template.Name,
		NewValues:   map[string]interface{}{"template": template},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusCreated, template)
}

func (h *WorkflowHandler) GetTemplate(c *gin.Context) {
	idParam := c.Param("id")
	templateID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID format"})
		return
	}

	template, err := h.workflowRepo.GetTemplate(c.Request.Context(), templateID)
	if err != nil {
		if err.Error() == "workflow template not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow template", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

func (h *WorkflowHandler) UpdateTemplate(c *gin.Context) {
	idParam := c.Param("id")
	templateID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID format"})
		return
	}

	var req models.UpdateWorkflowTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get existing template for audit
	existingTemplate, err := h.workflowRepo.GetTemplate(c.Request.Context(), templateID)
	if err != nil {
		if err.Error() == "workflow template not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get existing template", "details": err.Error()})
		return
	}

	// Update template
	existingTemplate.Name = req.Name
	existingTemplate.Description = req.Description
	existingTemplate.Version = req.Version
	existingTemplate.Category = req.Category
	existingTemplate.IsActive = req.IsActive
	existingTemplate.Steps = req.Steps

	if err := h.workflowRepo.UpdateTemplate(c.Request.Context(), existingTemplate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update workflow template", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.UpdatedBy,
		Action:      "update_workflow_template",
		EntityType:  "workflow_template",
		EntityID:    &templateID,
		Description: "Updated workflow template: " + existingTemplate.Name,
		NewValues:   map[string]interface{}{"template": existingTemplate},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, existingTemplate)
}

func (h *WorkflowHandler) DeleteTemplate(c *gin.Context) {
	idParam := c.Param("id")
	templateID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID format"})
		return
	}

	userID := c.GetHeader("X-User-ID")
	userUUID, _ := uuid.Parse(userID)

	// Get template for audit before deletion
	template, err := h.workflowRepo.GetTemplate(c.Request.Context(), templateID)
	if err != nil {
		if err.Error() == "workflow template not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get template", "details": err.Error()})
		return
	}

	if err := h.workflowRepo.DeleteTemplate(c.Request.Context(), templateID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete workflow template", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &userUUID,
		Action:      "delete_workflow_template",
		EntityType:  "workflow_template",
		EntityID:    &templateID,
		Description: "Deleted workflow template: " + template.Name,
		OldValues:   map[string]interface{}{"template": template},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, gin.H{"message": "Workflow template deleted successfully"})
}

func (h *WorkflowHandler) ListTemplates(c *gin.Context) {
	var filter models.WorkflowTemplateFilter

	// Parse query parameters
	if category := c.Query("category"); category != "" {
		filter.Category = category
	}

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filter.IsActive = &isActive
		}
	}

	if createdByStr := c.Query("created_by"); createdByStr != "" {
		if createdBy, err := uuid.Parse(createdByStr); err == nil {
			filter.CreatedBy = &createdBy
		}
	}

	// Parse pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	templates, total, err := h.workflowRepo.ListTemplates(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list workflow templates", "details": err.Error()})
		return
	}

	response := models.ListWorkflowTemplatesResponse{
		Templates: templates,
		Total:     total,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
	}

	c.JSON(http.StatusOK, response)
}

// Workflow Instances
func (h *WorkflowHandler) CreateInstance(c *gin.Context) {
	var req models.CreateWorkflowInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	instance := &models.WorkflowInstance{
		TemplateID:       req.TemplateID,
		InvestigationID:  req.InvestigationID,
		Name:             req.Name,
		Status:           models.WorkflowStatusPending,
		CurrentStepIndex: 0,
		Context:          req.Context,
		CreatedBy:        req.CreatedBy,
	}

	if err := h.workflowRepo.CreateInstance(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create workflow instance", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.CreatedBy,
		Action:      "create_workflow_instance",
		EntityType:  "workflow_instance",
		EntityID:    &instance.ID,
		Description: "Created workflow instance: " + instance.Name,
		NewValues:   map[string]interface{}{"instance": instance},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusCreated, instance)
}

func (h *WorkflowHandler) GetInstance(c *gin.Context) {
	idParam := c.Param("id")
	instanceID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID format"})
		return
	}

	instance, err := h.workflowRepo.GetInstance(c.Request.Context(), instanceID)
	if err != nil {
		if err.Error() == "workflow instance not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow instance not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow instance", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, instance)
}

func (h *WorkflowHandler) UpdateInstanceStatus(c *gin.Context) {
	idParam := c.Param("id")
	instanceID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID format"})
		return
	}

	var req models.UpdateWorkflowInstanceStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get existing instance
	instance, err := h.workflowRepo.GetInstance(c.Request.Context(), instanceID)
	if err != nil {
		if err.Error() == "workflow instance not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Workflow instance not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get instance", "details": err.Error()})
		return
	}

	oldStatus := instance.Status
	instance.Status = req.Status
	instance.CurrentStepIndex = req.CurrentStepIndex

	if req.Status == models.WorkflowStatusCompleted && instance.CompletedAt == nil {
		now := instance.UpdatedAt
		instance.CompletedAt = &now
	}

	if err := h.workflowRepo.UpdateInstance(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update workflow instance", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.UpdatedBy,
		Action:      "update_workflow_instance_status",
		EntityType:  "workflow_instance",
		EntityID:    &instanceID,
		Description: "Updated workflow instance status from " + string(oldStatus) + " to " + string(req.Status),
		OldValues:   map[string]interface{}{"status": oldStatus},
		NewValues:   map[string]interface{}{"status": req.Status, "current_step_index": req.CurrentStepIndex},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, instance)
}

func (h *WorkflowHandler) GetInstancesByInvestigation(c *gin.Context) {
	investigationIDParam := c.Param("investigation_id")
	investigationID, err := uuid.Parse(investigationIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid investigation ID format"})
		return
	}

	instances, err := h.workflowRepo.GetInstancesByInvestigation(c.Request.Context(), investigationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow instances", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"instances": instances})
}

// Workflow Steps
func (h *WorkflowHandler) GetInstanceSteps(c *gin.Context) {
	instanceIDParam := c.Param("instance_id")
	instanceID, err := uuid.Parse(instanceIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid instance ID format"})
		return
	}

	steps, err := h.workflowRepo.ListSteps(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow steps", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"steps": steps})
}

func (h *WorkflowHandler) StartStep(c *gin.Context) {
	stepIDParam := c.Param("step_id")
	stepID, err := uuid.Parse(stepIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid step ID format"})
		return
	}

	var req models.StartWorkflowStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	if err := h.workflowRepo.StartStep(c.Request.Context(), stepID, req.AssignedTo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start workflow step", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.AssignedTo,
		Action:      "start_workflow_step",
		EntityType:  "workflow_step",
		EntityID:    &stepID,
		Description: "Started workflow step",
		NewValues:   map[string]interface{}{"assigned_to": req.AssignedTo},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, gin.H{"message": "Workflow step started successfully"})
}

func (h *WorkflowHandler) CompleteStep(c *gin.Context) {
	stepIDParam := c.Param("step_id")
	stepID, err := uuid.Parse(stepIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid step ID format"})
		return
	}

	var req models.CompleteWorkflowStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	if err := h.workflowRepo.CompleteStep(c.Request.Context(), stepID, req.UserID, req.Result, req.Outputs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete workflow step", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.UserID,
		Action:      "complete_workflow_step",
		EntityType:  "workflow_step",
		EntityID:    &stepID,
		Description: "Completed workflow step: " + req.Result,
		NewValues:   map[string]interface{}{"result": req.Result, "outputs": req.Outputs},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, gin.H{"message": "Workflow step completed successfully"})
}

func (h *WorkflowHandler) SkipStep(c *gin.Context) {
	stepIDParam := c.Param("step_id")
	stepID, err := uuid.Parse(stepIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid step ID format"})
		return
	}

	var req models.SkipWorkflowStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	if err := h.workflowRepo.SkipStep(c.Request.Context(), stepID, req.UserID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to skip workflow step", "details": err.Error()})
		return
	}

	// Audit log
	auditLog := &models.AuditLog{
		UserID:      &req.UserID,
		Action:      "skip_workflow_step",
		EntityType:  "workflow_step",
		EntityID:    &stepID,
		Description: "Skipped workflow step: " + req.Reason,
		NewValues:   map[string]interface{}{"reason": req.Reason, "skipped": true},
	}
	h.auditRepo.CreateAuditLog(c.Request.Context(), auditLog)

	c.JSON(http.StatusOK, gin.H{"message": "Workflow step skipped successfully"})
}

// Workflow Automation and Management
func (h *WorkflowHandler) GetPendingSteps(c *gin.Context) {
	userIDStr := c.Query("user_id")
	var userID *uuid.UUID
	if userIDStr != "" {
		if parsed, err := uuid.Parse(userIDStr); err == nil {
			userID = &parsed
		}
	}

	steps, err := h.workflowRepo.GetPendingSteps(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending steps", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pending_steps": steps})
}

func (h *WorkflowHandler) GetOverdueSteps(c *gin.Context) {
	steps, err := h.workflowRepo.GetOverdueSteps(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get overdue steps", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"overdue_steps": steps})
}

func (h *WorkflowHandler) GetWorkflowStats(c *gin.Context) {
	var filter models.WorkflowStatsFilter

	if templateIDStr := c.Query("template_id"); templateIDStr != "" {
		if templateID, err := uuid.Parse(templateIDStr); err == nil {
			filter.TemplateID = &templateID
		}
	}

	if createdByStr := c.Query("created_by"); createdByStr != "" {
		if createdBy, err := uuid.Parse(createdByStr); err == nil {
			filter.CreatedBy = &createdBy
		}
	}

	stats, err := h.workflowRepo.GetWorkflowStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workflow stats", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}