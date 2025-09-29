package main

import (
	"auth-mail/internal/api"
	"auth-mail/internal/config"
	"auth-mail/internal/database"
	"auth-mail/pkg/logging"
	"log"

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
