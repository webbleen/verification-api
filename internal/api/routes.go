package api

import (
	"auth-mail/internal/middleware"
	"auth-mail/internal/models"
	"auth-mail/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up all routes
func SetupRoutes(r *gin.Engine) {
	// Initialize project manager
	middleware.InitProjectManager()

	// API route group
	api := r.Group("/api")
	{
		// Verification code routes (require project authentication)
		verification := api.Group("/verification")
		verification.Use(middleware.ProjectAuthMiddleware())
		{
			verification.POST("/send-code", SendVerificationCode)
			verification.POST("/verify-code", VerifyCode)
		}

		// Project management routes (for admin use)
		admin := api.Group("/admin")
		{
			admin.GET("/projects", GetProjects)
			admin.POST("/projects", CreateProject)
			admin.PUT("/projects/:id", UpdateProject)
			admin.DELETE("/projects/:id", DeleteProject)
			admin.GET("/projects/:id/stats", GetProjectStats)
		}

		// Statistics and monitoring routes
		stats := api.Group("/stats")
		stats.Use(middleware.ProjectAuthMiddleware())
		{
			stats.GET("/verification", GetVerificationStats)
			stats.GET("/project", GetProjectStats)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "verification-service",
		})
	})
}

// GetProjects gets all projects
func GetProjects(c *gin.Context) {
	projectService := services.NewProjectService()
	projects, err := projectService.GetAllProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get projects",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    projects,
	})
}

// CreateProjectRequest represents create project request
type CreateProjectRequest struct {
	ProjectID    string `json:"project_id" binding:"required"`
	ProjectName  string `json:"project_name" binding:"required"`
	APIKey       string `json:"api_key" binding:"required"`
	FromName     string `json:"from_name" binding:"required"`
	TemplateID   string `json:"template_id"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	WebhookURL   string `json:"webhook_url"`
	RateLimit    int    `json:"rate_limit"`
	MaxRequests  int    `json:"max_requests"`
}

// CreateProject creates a new project
func CreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Set defaults
	if req.RateLimit == 0 {
		req.RateLimit = 60 // 60 requests per hour
	}
	if req.MaxRequests == 0 {
		req.MaxRequests = 1000 // 1000 requests per day
	}

	project := &models.Project{
		ProjectID:    req.ProjectID,
		ProjectName:  req.ProjectName,
		APIKey:       req.APIKey,
		FromName:     req.FromName,
		TemplateID:   req.TemplateID,
		Description:  req.Description,
		ContactEmail: req.ContactEmail,
		WebhookURL:   req.WebhookURL,
		RateLimit:    req.RateLimit,
		MaxRequests:  req.MaxRequests,
		IsActive:     true,
	}

	projectService := services.NewProjectService()
	if err := projectService.CreateProject(project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to create project: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Project created successfully",
		"data":    project,
	})
}

// UpdateProjectRequest represents update project request
type UpdateProjectRequest struct {
	ProjectName  string `json:"project_name"`
	FromName     string `json:"from_name"`
	TemplateID   string `json:"template_id"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	WebhookURL   string `json:"webhook_url"`
	RateLimit    int    `json:"rate_limit"`
	MaxRequests  int    `json:"max_requests"`
	IsActive     *bool  `json:"is_active"`
}

// UpdateProject updates an existing project
func UpdateProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project ID is required",
		})
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
		return
	}

	// Build update map
	updates := make(map[string]interface{})
	if req.ProjectName != "" {
		updates["project_name"] = req.ProjectName
	}
	if req.FromName != "" {
		updates["from_name"] = req.FromName
	}
	if req.TemplateID != "" {
		updates["template_id"] = req.TemplateID
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.ContactEmail != "" {
		updates["contact_email"] = req.ContactEmail
	}
	if req.WebhookURL != "" {
		updates["webhook_url"] = req.WebhookURL
	}
	if req.RateLimit > 0 {
		updates["rate_limit"] = req.RateLimit
	}
	if req.MaxRequests > 0 {
		updates["max_requests"] = req.MaxRequests
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	projectService := services.NewProjectService()
	if err := projectService.UpdateProject(projectID, updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to update project: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Project updated successfully",
	})
}

// DeleteProject deletes a project
func DeleteProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project ID is required",
		})
		return
	}

	projectService := services.NewProjectService()
	if err := projectService.DeleteProject(projectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Failed to delete project: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Project deleted successfully",
	})
}

// GetProjectStats gets project statistics
func GetProjectStats(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		// If no ID in param, get from context (for stats routes)
		if pid, exists := c.Get("project_id"); exists {
			projectID = pid.(string)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "Project ID is required",
			})
			return
		}
	}

	projectService := services.NewProjectService()
	stats, err := projectService.GetProjectStats(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get project stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetVerificationStats gets verification statistics
func GetVerificationStats(c *gin.Context) {
	projectID, exists := c.Get("project_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Project ID is required",
		})
		return
	}

	// Get days parameter (default to 7)
	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	verificationService := services.NewVerificationService()
	stats, err := verificationService.GetVerificationStats(projectID.(string), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get verification stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}
