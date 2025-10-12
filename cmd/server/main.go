package main

import (
	"log"
	"verification-api/internal/api"
	"verification-api/internal/config"
	"verification-api/internal/database"
	"verification-api/pkg/logging"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize configuration
	if err := config.InitConfig(); err != nil {
		log.Fatal("Failed to initialize config:", err)
	}

	// Initialize logging
	logging.InitLogging()

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Set Gin mode
	gin.SetMode(config.AppConfig.Mode)

	// Create Gin engine
	r := gin.Default()

	// Setup routes
	api.SetupRoutes(r)

	// Start server
	port := config.AppConfig.Port
	logging.Infof("Starting server on port %s", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
