package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/aegis-shield/services/alerting-engine/internal/config"
	"github.com/aegis-shield/services/alerting-engine/internal/database"
	"github.com/aegis-shield/services/alerting-engine/internal/engine"
	"github.com/aegis-shield/services/alerting-engine/internal/notification"
)

// Scheduler manages periodic tasks and scheduled operations
type Scheduler struct {
	config           *config.Config
	logger           *slog.Logger
	cron             *cron.Cron
	alertRepo        *database.AlertRepository
	ruleRepo         *database.RuleRepository
	notificationRepo *database.NotificationRepository
	escalationRepo   *database.EscalationRepository
	ruleEngine       *engine.RuleEngine
	notificationMgr  *notification.Manager
	tasks            map[string]*ScheduledTask
	tasksMutex       sync.RWMutex
	shutdownChan     chan struct{}
	wg               sync.WaitGroup
}

// ScheduledTask represents a scheduled task
type ScheduledTask struct {
	ID          string
	Name        string
	Description string
	Schedule    string
	Handler     TaskHandler
	LastRun     time.Time
	NextRun     time.Time
	RunCount    int64
	ErrorCount  int64
	Enabled     bool
	cronEntryID cron.EntryID
}

// TaskHandler defines the interface for scheduled task handlers
type TaskHandler interface {
	Execute(ctx context.Context) error
	GetName() string
	GetDescription() string
}

// NewScheduler creates a new scheduler
func NewScheduler(
	cfg *config.Config,
	logger *slog.Logger,
	alertRepo *database.AlertRepository,
	ruleRepo *database.RuleRepository,
	notificationRepo *database.NotificationRepository,
	escalationRepo *database.EscalationRepository,
	ruleEngine *engine.RuleEngine,
	notificationMgr *notification.Manager,
) (*Scheduler, error) {
	// Create cron with second precision and timezone support
	cronScheduler := cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC))

	scheduler := &Scheduler{
		config:           cfg,
		logger:           logger,
		cron:             cronScheduler,
		alertRepo:        alertRepo,
		ruleRepo:         ruleRepo,
		notificationRepo: notificationRepo,
		escalationRepo:   escalationRepo,
		ruleEngine:       ruleEngine,
		notificationMgr:  notificationMgr,
		tasks:            make(map[string]*ScheduledTask),
		shutdownChan:     make(chan struct{}),
	}

	// Initialize default tasks
	if err := scheduler.initializeDefaultTasks(); err != nil {
		return nil, fmt.Errorf("failed to initialize default tasks: %w", err)
	}

	return scheduler, nil
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("Starting scheduler")

	// Schedule all enabled tasks
	s.tasksMutex.RLock()
	for _, task := range s.tasks {
		if task.Enabled {
			if err := s.scheduleTask(task); err != nil {
				s.logger.Error("Failed to schedule task",
					"task_id", task.ID,
					"task_name", task.Name,
					"error", err)
			}
		}
	}
	s.tasksMutex.RUnlock()

	// Start cron scheduler
	s.cron.Start()

	// Start monitoring goroutine
	s.wg.Add(1)
	go s.monitoringRoutine(ctx)

	s.logger.Info("Scheduler started", "scheduled_tasks", len(s.tasks))
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping scheduler")
	
	// Stop cron scheduler
	ctx := s.cron.Stop()
	<-ctx.Done()

	// Signal shutdown and wait for goroutines
	close(s.shutdownChan)
	s.wg.Wait()

	s.logger.Info("Scheduler stopped")
}

// AddTask adds a new scheduled task
func (s *Scheduler) AddTask(task *ScheduledTask) error {
	s.tasksMutex.Lock()
	defer s.tasksMutex.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	s.tasks[task.ID] = task

	if task.Enabled {
		return s.scheduleTask(task)
	}

	return nil
}

// RemoveTask removes a scheduled task
func (s *Scheduler) RemoveTask(taskID string) error {
	s.tasksMutex.Lock()
	defer s.tasksMutex.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	// Remove from cron
	if task.cronEntryID != 0 {
		s.cron.Remove(task.cronEntryID)
	}

	delete(s.tasks, taskID)
	return nil
}

// EnableTask enables a scheduled task
func (s *Scheduler) EnableTask(taskID string) error {
	s.tasksMutex.Lock()
	defer s.tasksMutex.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	if !task.Enabled {
		task.Enabled = true
		return s.scheduleTask(task)
	}

	return nil
}

// DisableTask disables a scheduled task
func (s *Scheduler) DisableTask(taskID string) error {
	s.tasksMutex.Lock()
	defer s.tasksMutex.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	if task.Enabled {
		task.Enabled = false
		if task.cronEntryID != 0 {
			s.cron.Remove(task.cronEntryID)
			task.cronEntryID = 0
		}
	}

	return nil
}

// GetTasks returns all scheduled tasks
func (s *Scheduler) GetTasks() []*ScheduledTask {
	s.tasksMutex.RLock()
	defer s.tasksMutex.RUnlock()

	tasks := make([]*ScheduledTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetTask returns a specific task
func (s *Scheduler) GetTask(taskID string) (*ScheduledTask, error) {
	s.tasksMutex.RLock()
	defer s.tasksMutex.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", taskID)
	}

	return task, nil
}

// scheduleTask schedules a task with cron
func (s *Scheduler) scheduleTask(task *ScheduledTask) error {
	// Remove existing entry if any
	if task.cronEntryID != 0 {
		s.cron.Remove(task.cronEntryID)
	}

	// Add new entry
	entryID, err := s.cron.AddFunc(task.Schedule, func() {
		s.executeTask(task)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule task %s: %w", task.ID, err)
	}

	task.cronEntryID = entryID
	
	// Update next run time
	entries := s.cron.Entries()
	for _, entry := range entries {
		if entry.ID == entryID {
			task.NextRun = entry.Next
			break
		}
	}

	s.logger.Debug("Task scheduled",
		"task_id", task.ID,
		"task_name", task.Name,
		"schedule", task.Schedule,
		"next_run", task.NextRun)

	return nil
}

// executeTask executes a scheduled task
func (s *Scheduler) executeTask(task *ScheduledTask) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	startTime := time.Now()
	task.LastRun = startTime
	task.RunCount++

	s.logger.Debug("Executing scheduled task",
		"task_id", task.ID,
		"task_name", task.Name,
		"run_count", task.RunCount)

	if err := task.Handler.Execute(ctx); err != nil {
		task.ErrorCount++
		s.logger.Error("Scheduled task failed",
			"task_id", task.ID,
			"task_name", task.Name,
			"error", err,
			"execution_time", time.Since(startTime))
	} else {
		s.logger.Debug("Scheduled task completed",
			"task_id", task.ID,
			"task_name", task.Name,
			"execution_time", time.Since(startTime))
	}

	// Update next run time
	entries := s.cron.Entries()
	for _, entry := range entries {
		if entry.ID == task.cronEntryID {
			task.NextRun = entry.Next
			break
		}
	}
}

// initializeDefaultTasks initializes the default scheduled tasks
func (s *Scheduler) initializeDefaultTasks() error {
	// Alert cleanup task
	alertCleanupTask := &ScheduledTask{
		ID:          "alert_cleanup",
		Name:        "Alert Cleanup",
		Description: "Cleanup old resolved and closed alerts",
		Schedule:    s.config.Scheduler.AlertCleanupSchedule,
		Handler:     NewAlertCleanupHandler(s.alertRepo, s.config, s.logger),
		Enabled:     s.config.Scheduler.AlertCleanupEnabled,
	}
	s.tasks[alertCleanupTask.ID] = alertCleanupTask

	// Notification cleanup task
	notificationCleanupTask := &ScheduledTask{
		ID:          "notification_cleanup",
		Name:        "Notification Cleanup",
		Description: "Cleanup old delivered and failed notifications",
		Schedule:    s.config.Scheduler.NotificationCleanupSchedule,
		Handler:     NewNotificationCleanupHandler(s.notificationRepo, s.config, s.logger),
		Enabled:     s.config.Scheduler.NotificationCleanupEnabled,
	}
	s.tasks[notificationCleanupTask.ID] = notificationCleanupTask

	// Health check task
	healthCheckTask := &ScheduledTask{
		ID:          "health_check",
		Name:        "Health Check",
		Description: "Perform system health checks and generate alerts if needed",
		Schedule:    s.config.Scheduler.HealthCheckSchedule,
		Handler:     NewHealthCheckHandler(s.alertRepo, s.ruleEngine, s.config, s.logger),
		Enabled:     s.config.Scheduler.HealthCheckEnabled,
	}
	s.tasks[healthCheckTask.ID] = healthCheckTask

	// Escalation processor task
	escalationTask := &ScheduledTask{
		ID:          "escalation_processor",
		Name:        "Escalation Processor",
		Description: "Process alert escalations based on escalation policies",
		Schedule:    s.config.Scheduler.EscalationProcessorSchedule,
		Handler:     NewEscalationProcessorHandler(s.alertRepo, s.escalationRepo, s.notificationRepo, s.config, s.logger),
		Enabled:     s.config.Scheduler.EscalationProcessorEnabled,
	}
	s.tasks[escalationTask.ID] = escalationTask

	// Metrics collection task
	metricsTask := &ScheduledTask{
		ID:          "metrics_collection",
		Name:        "Metrics Collection",
		Description: "Collect and update system metrics",
		Schedule:    s.config.Scheduler.MetricsCollectionSchedule,
		Handler:     NewMetricsCollectionHandler(s.alertRepo, s.notificationRepo, s.ruleEngine, s.config, s.logger),
		Enabled:     s.config.Scheduler.MetricsCollectionEnabled,
	}
	s.tasks[metricsTask.ID] = metricsTask

	// Pending notifications processor
	pendingNotificationsTask := &ScheduledTask{
		ID:          "pending_notifications",
		Name:        "Pending Notifications Processor",
		Description: "Process pending notifications that need to be sent",
		Schedule:    s.config.Scheduler.PendingNotificationsSchedule,
		Handler:     NewPendingNotificationsHandler(s.notificationMgr, s.config, s.logger),
		Enabled:     s.config.Scheduler.PendingNotificationsEnabled,
	}
	s.tasks[pendingNotificationsTask.ID] = pendingNotificationsTask

	return nil
}

// monitoringRoutine monitors the scheduler and tasks
func (s *Scheduler) monitoringRoutine(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.shutdownChan:
			return
		case <-ticker.C:
			s.logSchedulerStats()
		}
	}
}

// logSchedulerStats logs scheduler statistics
func (s *Scheduler) logSchedulerStats() {
	s.tasksMutex.RLock()
	defer s.tasksMutex.RUnlock()

	totalTasks := len(s.tasks)
	enabledTasks := 0
	totalRuns := int64(0)
	totalErrors := int64(0)

	for _, task := range s.tasks {
		if task.Enabled {
			enabledTasks++
		}
		totalRuns += task.RunCount
		totalErrors += task.ErrorCount
	}

	s.logger.Debug("Scheduler statistics",
		"total_tasks", totalTasks,
		"enabled_tasks", enabledTasks,
		"total_runs", totalRuns,
		"total_errors", totalErrors)
}

// GetSchedulerStats returns scheduler statistics
func (s *Scheduler) GetSchedulerStats() map[string]interface{} {
	s.tasksMutex.RLock()
	defer s.tasksMutex.RUnlock()

	stats := map[string]interface{}{
		"total_tasks":   len(s.tasks),
		"enabled_tasks": 0,
		"total_runs":    int64(0),
		"total_errors":  int64(0),
		"tasks":         make([]map[string]interface{}, 0),
	}

	for _, task := range s.tasks {
		if task.Enabled {
			stats["enabled_tasks"] = stats["enabled_tasks"].(int) + 1
		}
		stats["total_runs"] = stats["total_runs"].(int64) + task.RunCount
		stats["total_errors"] = stats["total_errors"].(int64) + task.ErrorCount

		taskStats := map[string]interface{}{
			"id":          task.ID,
			"name":        task.Name,
			"description": task.Description,
			"schedule":    task.Schedule,
			"enabled":     task.Enabled,
			"last_run":    task.LastRun,
			"next_run":    task.NextRun,
			"run_count":   task.RunCount,
			"error_count": task.ErrorCount,
		}
		stats["tasks"] = append(stats["tasks"].([]map[string]interface{}), taskStats)
	}

	return stats
}

// ValidateSchedule validates a cron schedule expression
func (s *Scheduler) ValidateSchedule(schedule string) error {
	_, err := cron.ParseStandard(schedule)
	return err
}

// GetNextRuns returns the next run times for all enabled tasks
func (s *Scheduler) GetNextRuns() map[string]time.Time {
	s.tasksMutex.RLock()
	defer s.tasksMutex.RUnlock()

	nextRuns := make(map[string]time.Time)
	for _, task := range s.tasks {
		if task.Enabled {
			nextRuns[task.ID] = task.NextRun
		}
	}

	return nextRuns
}

// ExecuteTaskNow executes a task immediately (outside of its schedule)
func (s *Scheduler) ExecuteTaskNow(taskID string) error {
	s.tasksMutex.RLock()
	task, exists := s.tasks[taskID]
	s.tasksMutex.RUnlock()

	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	go s.executeTask(task)
	return nil
}

// UpdateTaskSchedule updates the schedule for a task
func (s *Scheduler) UpdateTaskSchedule(taskID, newSchedule string) error {
	// Validate schedule first
	if err := s.ValidateSchedule(newSchedule); err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	s.tasksMutex.Lock()
	defer s.tasksMutex.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task with ID %s not found", taskID)
	}

	task.Schedule = newSchedule

	// Reschedule if enabled
	if task.Enabled {
		return s.scheduleTask(task)
	}

	return nil
}