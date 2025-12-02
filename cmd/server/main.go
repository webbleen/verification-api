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
	logging.Infof("Initializing database connection...")
	if err := database.InitDatabase(); err != nil {
		logging.Errorf("Failed to initialize database: %v", err)
		log.Fatal("Failed to initialize database:", err)
	}

	// Set Gin mode
	gin.SetMode(config.AppConfig.Mode)
	logging.Infof("Gin mode set to: %s", config.AppConfig.Mode)

	// Create Gin engine
	logging.Infof("Creating Gin engine...")
	r := gin.Default()
	logging.Infof("Gin engine created successfully")

	// Setup routes
	logging.Infof("Setting up routes...")
	api.SetupRoutes(r)
	logging.Infof("Routes setup completed")

	// Start server
	port := config.AppConfig.Port
	if port == "" {
		port = "8080"
	}
	logging.Infof("Starting server on port %s", port)

	// Use explicit address binding
	addr := "0.0.0.0:" + port
	logging.Infof("Binding to address: %s", addr)

	if err := r.Run(addr); err != nil {
		logging.Errorf("Failed to start server: %v", err)
		log.Fatal("Failed to start server:", err)
	}
}
