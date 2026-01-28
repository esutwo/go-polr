package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/nnnc-org/go-polr/internal/config"
	"github.com/nnnc-org/go-polr/internal/database"
	"github.com/nnnc-org/go-polr/internal/router"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Don't leak sensitive config details in error messages
		if err == config.ErrDefaultSecrets {
			log.Fatalf("Security error: %v", err)
		}
		log.Fatal("Failed to load configuration")
	}

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		// Don't leak database connection details
		log.Fatal("Failed to connect to database")
	}
	defer database.Close(db)

	log.Println("Database connection established")

	// Set Gin mode based on environment
	if len(cfg.AppURL) >= 5 && cfg.AppURL[:5] == "https" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Setup router
	r := router.Setup(db, cfg)

	// Start server
	log.Printf("Starting %s on port %s", cfg.AppName, cfg.AppPort)

	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatal("Failed to start server")
	}
}
