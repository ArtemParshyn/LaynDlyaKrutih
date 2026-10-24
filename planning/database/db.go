package database

import (
	"fmt"
	"log"
	"os"
	"planning/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() *gorm.DB {
	host := getEnv("DB_HOST", "postgres")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "secret123")
	dbname := getEnv("DB_NAME", "users_db") // shares postgres container or uses dedicated DB

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Printf("Warning: failed to connect to PostgreSQL (%s): %v", dsn, err)
		log.Println("Note: planning service will retry on requests or running in memory/mock")
		return nil
	}

	// Auto-migrate tables
	if err := db.AutoMigrate(&model.PlanEvent{}); err != nil {
		log.Printf("Warning: auto-migration failed: %v", err)
	} else {
		log.Println("Planning database migrated successfully")
	}

	DB = db
	return DB
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
