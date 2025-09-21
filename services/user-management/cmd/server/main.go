package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// User represents a system user
type User struct {
	ID           uint         `json:"id" gorm:"primaryKey"`
	Username     string       `json:"username" gorm:"uniqueIndex;not null"`
	Email        string       `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string       `json:"-" gorm:"not null"`
	FirstName    string       `json:"first_name"`
	LastName     string       `json:"last_name"`
	Role         string       `json:"role" gorm:"not null;default:'analyst'"` // analyst, investigator, admin, compliance
	Department   string       `json:"department"`
	IsActive     bool         `json:"is_active" gorm:"default:true"`
	LastLogin    *time.Time   `json:"last_login"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Permissions  []Permission `json:"permissions" gorm:"many2many:user_permissions;"`
}

// Permission represents system permissions
type Permission struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"uniqueIndex;not null"`
	Resource    string `json:"resource" gorm:"not null"` // alerts, investigations, entities, etc.
	Action      string `json:"action" gorm:"not null"`   // read, write, delete, approve
	Description string `json:"description"`
}

// UserSession represents active user sessions
type UserSession struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

// AuditLog represents user activity logs
type AuditLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	Action    string    `json:"action" gorm:"not null"`
	Resource  string    `json:"resource"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
}

// Request/Response models
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

type CreateUserRequest struct {
	Username    string `json:"username" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	FirstName   string `json:"first_name" binding:"required"`
	LastName    string `json:"last_name" binding:"required"`
	Role        string `json:"role" binding:"required"`
	Department  string `json:"department"`
	Permissions []uint `json:"permission_ids"`
}

type UpdateUserRequest struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	Email       *string `json:"email"`
	Role        *string `json:"role"`
	Department  *string `json:"department"`
	IsActive    *bool   `json:"is_active"`
	Permissions []uint  `json:"permission_ids"`
}

// UserManagementService handles user operations
type UserManagementService struct {
	db        *gorm.DB
	jwtSecret []byte
}

// NewUserManagementService creates a new user management service
func NewUserManagementService(db *gorm.DB) *UserManagementService {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "aegisshield-default-secret-change-in-production"
	}

	return &UserManagementService{
		db:        db,
		jwtSecret: []byte(jwtSecret),
	}
}

// HashPassword creates a bcrypt hash of the password
func (s *UserManagementService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPassword compares a password with its hash
func (s *UserManagementService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateJWT creates a JWT token for the user
func (s *UserManagementService) GenerateJWT(user *User) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour)

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)

	return tokenString, expiresAt, err
}

// Login authenticates a user and returns a JWT token
func (s *UserManagementService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	if err := s.db.Preload("Permissions").Where("username = ? OR email = ?", req.Username, req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is deactivated"})
		return
	}

	if !s.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, expiresAt, err := s.GenerateJWT(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Save session
	session := UserSession{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: expiresAt,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
	s.db.Create(&session)

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	s.db.Save(&user)

	// Log audit event
	s.LogAuditEvent(user.ID, "login", "authentication", "User logged in", c.ClientIP())

	// Remove password hash from response
	user.PasswordHash = ""

	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	})
}

// CreateUser creates a new user account
func (s *UserManagementService) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if username or email already exists
	var existingUser User
	if err := s.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
		return
	}

	// Hash password
	passwordHash, err := s.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	user := User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
		Department:   req.Department,
		IsActive:     true,
	}

	if err := s.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Assign permissions
	if len(req.Permissions) > 0 {
		var permissions []Permission
		s.db.Where("id IN ?", req.Permissions).Find(&permissions)
		s.db.Model(&user).Association("Permissions").Append(permissions)
	}

	// Get current user for audit log
	currentUserID := s.GetUserIDFromContext(c)
	s.LogAuditEvent(currentUserID, "create_user", "user_management",
		fmt.Sprintf("Created user: %s", user.Username), c.ClientIP())

	// Remove password hash from response
	user.PasswordHash = ""

	c.JSON(http.StatusCreated, user)
}

// GetUsers returns a list of users with optional filtering
func (s *UserManagementService) GetUsers(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "50")
	role := c.Query("role")
	department := c.Query("department")
	active := c.Query("active")

	var users []User
	query := s.db.Preload("Permissions").Offset((getIntFromString(page) - 1) * getIntFromString(limit)).Limit(getIntFromString(limit))

	if role != "" {
		query = query.Where("role = ?", role)
	}
	if department != "" {
		query = query.Where("department = ?", department)
	}
	if active != "" {
		query = query.Where("is_active = ?", active == "true")
	}

	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Remove password hashes from response
	for i := range users {
		users[i].PasswordHash = ""
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// UpdateUser updates user information
func (s *UserManagementService) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	if err := s.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Department != nil {
		user.Department = *req.Department
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := s.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Update permissions
	if req.Permissions != nil {
		var permissions []Permission
		s.db.Where("id IN ?", req.Permissions).Find(&permissions)
		s.db.Model(&user).Association("Permissions").Replace(permissions)
	}

	// Get current user for audit log
	currentUserID := s.GetUserIDFromContext(c)
	s.LogAuditEvent(currentUserID, "update_user", "user_management",
		fmt.Sprintf("Updated user: %s", user.Username), c.ClientIP())

	// Remove password hash from response
	user.PasswordHash = ""

	c.JSON(http.StatusOK, user)
}

// LogAuditEvent logs user actions for audit purposes
func (s *UserManagementService) LogAuditEvent(userID uint, action, resource, details, ipAddress string) {
	auditLog := AuditLog{
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Details:   details,
		IPAddress: ipAddress,
		Timestamp: time.Now(),
	}
	s.db.Create(&auditLog)
}

// GetUserIDFromContext extracts user ID from JWT token in context
func (s *UserManagementService) GetUserIDFromContext(c *gin.Context) uint {
	// This would be implemented by the JWT middleware
	// For now, returning a default value
	return 1
}

// Helper function to convert string to int
func getIntFromString(s string) int {
	if s == "" {
		return 0
	}
	// Simple conversion, should use strconv.Atoi in production
	switch s {
	case "1":
		return 1
	case "2":
		return 2
	default:
		return 1
	}
}

// SetupRoutes configures the HTTP routes
func SetupRoutes(service *UserManagementService) *gin.Engine {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "user-management",
			"timestamp": time.Now(),
		})
	})

	// Authentication routes
	auth := r.Group("/auth")
	{
		auth.POST("/login", service.Login)
		auth.POST("/logout", func(c *gin.Context) {
			// Implement logout logic
			c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
		})
	}

	// User management routes (protected)
	users := r.Group("/users")
	// users.Use(AuthMiddleware()) // JWT middleware would go here
	{
		users.POST("/", service.CreateUser)
		users.GET("/", service.GetUsers)
		users.PUT("/:id", service.UpdateUser)
		users.GET("/:id", func(c *gin.Context) {
			// Get single user implementation
			c.JSON(http.StatusOK, gin.H{"message": "Get user endpoint"})
		})
	}

	// Permissions routes
	permissions := r.Group("/permissions")
	{
		permissions.GET("/", func(c *gin.Context) {
			var permissions []Permission
			service.db.Find(&permissions)
			c.JSON(http.StatusOK, gin.H{"permissions": permissions})
		})
	}

	return r
}

// SeedDefaultData creates default users and permissions
func SeedDefaultData(db *gorm.DB) error {
	// Create default permissions
	permissions := []Permission{
		{Name: "read_alerts", Resource: "alerts", Action: "read", Description: "Read alerts"},
		{Name: "write_alerts", Resource: "alerts", Action: "write", Description: "Create and update alerts"},
		{Name: "read_investigations", Resource: "investigations", Action: "read", Description: "Read investigations"},
		{Name: "write_investigations", Resource: "investigations", Action: "write", Description: "Create and update investigations"},
		{Name: "admin_users", Resource: "users", Action: "admin", Description: "Manage users"},
		{Name: "read_entities", Resource: "entities", Action: "read", Description: "Read entities"},
		{Name: "write_entities", Resource: "entities", Action: "write", Description: "Update entities"},
	}

	for _, perm := range permissions {
		db.FirstOrCreate(&perm, Permission{Name: perm.Name})
	}

	// Create default admin user
	service := NewUserManagementService(db)
	passwordHash, _ := service.HashPassword("admin123")

	adminUser := User{
		Username:     "admin",
		Email:        "admin@aegisshield.com",
		PasswordHash: passwordHash,
		FirstName:    "System",
		LastName:     "Administrator",
		Role:         "admin",
		Department:   "IT",
		IsActive:     true,
	}

	db.FirstOrCreate(&adminUser, User{Username: "admin"})

	return nil
}

func main() {
	// Database connection
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=password dbname=aegisshield port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate schemas
	err = db.AutoMigrate(&User{}, &Permission{}, &UserSession{}, &AuditLog{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed default data
	if err := SeedDefaultData(db); err != nil {
		log.Fatal("Failed to seed default data:", err)
	}

	// Create service
	service := NewUserManagementService(db)

	// Setup routes
	router := SetupRoutes(service)

	// Server configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8070"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	log.Printf("User Management Service started on port %s", port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
