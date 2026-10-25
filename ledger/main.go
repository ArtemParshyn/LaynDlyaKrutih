package main

import (
	"ledger/database"
	"ledger/handlers"
	"ledger/service"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not loaded, reading system environment")
	}

	// Connect PostgreSQL and auto-migrate tables
	db := database.Connect()

	// Initialize service and handlers
	ledgerService := service.NewLedgerService(db)
	ledgerHandler := handlers.NewLedgerHandler(ledgerService)

	router := gin.Default()

	// Public healthcheck
	router.GET("/health", ledgerHandler.Health)

	// Protected routes (require X-User-ID)
	api := router.Group("/")
	api.Use(handlers.UserIDMiddleware())
	{
		api.POST("/statements", ledgerHandler.UploadStatements)
		api.DELETE("/statements", ledgerHandler.Wipe)
		api.POST("/operations", ledgerHandler.AddManualOperation)
		api.GET("/operations", ledgerHandler.Operations)
		api.GET("/data", ledgerHandler.GetData)
		api.POST("/forecast/archive", ledgerHandler.ArchiveForecast)
		api.POST("/forecast", ledgerHandler.SaveForecast)
		api.GET("/forecast", ledgerHandler.GetForecast)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Ledger service starting on port :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Ledger service failed to start: %v", err)
	}
}
