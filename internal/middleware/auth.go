package middleware

import (
	"net/http"
	"time"
	"verification-api/internal/services"

	"verification-api/internal/response"

	"github.com/gin-gonic/gin"
)

var ProjectService *services.ProjectService

// InitProjectManager initializes the project manager
func InitProjectManager() {
	ProjectService = services.NewProjectService()
}

// ProjectAuthMiddleware provides project authentication middleware
func ProjectAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get project ID and API key
		projectID := c.GetHeader("X-Project-ID")
		apiKey := c.GetHeader("X-API-Key")

		// If not passed via header, try to get from query parameters
		if projectID == "" {
			projectID = c.Query("project_id")
		}
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		// Validate project ID and API key
		if projectID == "" || apiKey == "" {
			c.JSON(http.StatusUnauthorized, response.Error(http.StatusUnauthorized, "Missing project_id or api_key"))
			c.Abort()
			return
		}

		// Validate project using database
		if !ProjectService.ValidateProject(projectID, apiKey) {
			c.JSON(http.StatusUnauthorized, response.Error(http.StatusUnauthorized, "Invalid project_id or api_key"))
			c.Abort()
			return
		}

		// Store project ID and additional info in context
		c.Set("project_id", projectID)
		c.Set("request_time", time.Now())
		c.Next()
	}
}
