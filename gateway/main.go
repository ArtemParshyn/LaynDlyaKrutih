package main

import (
	"gateway/auth"
	"gateway/client"
	"gateway/database"
	_ "gateway/docs"
	"gateway/handlers"
	"gateway/repository"
	"gateway/service"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Financial Forecast & Auth Gateway API
// @version         1.0
// @description     Шлюз входа: авторизация (JWT + Redis), сборка финансового пакета и оркестрация сервисов выписки, планов и прогноза.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Введите: Bearer <ваш_токен>
func main() {
	// Load .env if present
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not loaded, reading environment variables")
	}

	// Connect PostgreSQL & Redis
	db := database.ConnectPostgres()
	database.ConnectRedis()

	// Initialize repository and handlers
	userRepo := repository.NewUserRepository(db)
	authHandler := handlers.NewAuthHandler(userRepo)
	userHandler := handlers.NewUserHandler(userRepo)

	// External microservice addresses from environment
	ledgerURL := getEnv("LEDGER_SERVICE_URL", "http://ledger:8081")
	planningURL := getEnv("PLANNING_SERVICE_URL", "http://planning:8082")
	forecastURL := getEnv("FORECAST_SERVICE_URL", "http://forecast:8000")

	// Initialize orchestrator service & forecast handler
	ledgerClient := client.NewLedgerClient(ledgerURL)
	planningClient := client.NewPlanningClient(planningURL)
	forecastClient := client.NewForecastClient(forecastURL)
	orchestratorService := service.NewOrchestratorService(ledgerClient, planningClient, forecastClient)
	forecastHandler := handlers.NewForecastHandler(orchestratorService)

	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://217.144.184.23:3000",
			"http://localhost:3000",
		},
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public Auth routes
	router.POST("/register", authHandler.Register)
	router.POST("/login", authHandler.Login)
	router.POST("/refresh", authHandler.Refresh)
	router.POST("/logout", authHandler.Logout)

	// Protected routes (require JWT)
	protected := router.Group("/")
	protected.Use(auth.AuthMiddleware())
	{
		// User profile & CRUD (IDOR protected: users can only view/edit/delete themselves)
		protected.GET("/me", userHandler.GetMe)
		protected.PATCH("/me", userHandler.UpdateMe)
		protected.DELETE("/me", userHandler.DeleteMe)
		protected.GET("/users", userHandler.GetAllUsers)
		protected.GET("/users/:id", userHandler.GetUser)
		protected.PATCH("/users/:id", userHandler.UpdateUser)
		protected.DELETE("/users/:id", userHandler.DeleteUser)

		// Statements & Forecast orchestration
		protected.POST("/api/statements", forecastHandler.UploadStatement)
		protected.POST("/statements", forecastHandler.UploadStatement)

		protected.GET("/api/forecast", forecastHandler.GetForecast)
		protected.GET("/forecast", forecastHandler.GetForecast)

		protected.POST("/api/events", forecastHandler.CreateEvent)
		protected.POST("/events", forecastHandler.CreateEvent)
		protected.GET("/api/events", forecastHandler.GetEvents)
		protected.GET("/events", forecastHandler.GetEvents)
		protected.DELETE("/api/events/:id", forecastHandler.DeleteEvent)
		protected.DELETE("/events/:id", forecastHandler.DeleteEvent)

		protected.POST("/api/scenarios", forecastHandler.CalculateScenario)
		protected.POST("/scenarios", forecastHandler.CalculateScenario)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Gateway starting on port :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
