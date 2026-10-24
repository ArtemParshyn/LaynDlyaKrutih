package main

import (
	"log"
	"os"
	"planning/database"
	"planning/handlers"
	"planning/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not loaded, reading system environment")
	}

	db := database.Connect()
	svc := service.NewPlanningService(db)
	handler := handlers.NewPlanningHandler(svc)

	router := gin.Default()

	// Public health check
	router.GET("/health", handler.Health)

	// Protected routes (require X-User-ID)
	api := router.Group("/")
	api.Use(handlers.UserIDMiddleware())
	{
		api.POST("/events", handler.CreateEvent)
		api.GET("/events", handler.GetEvents)
		api.DELETE("/events/:id", handler.DeleteEvent)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Planning service starting on port :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Planning service failed to start: %v", err)
	}
}
