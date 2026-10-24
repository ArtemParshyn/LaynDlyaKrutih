package database

import (
	"fmt"
	"ledger/model"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() *gorm.DB {
	host := getEnv("DB_HOST", "ledger-postgres")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "ledger")
	password := getEnv("DB_PASSWORD", "secret123")
	dbname := getEnv("DB_NAME", "ledger_db")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	// Auto-migrate tables
	err = db.AutoMigrate(
		&model.Operation{},
		&model.Statement{},
		&model.ForecastDay{},
		&model.ForecastAlert{},
		&model.ForecastHistory{},
	)
	if err != nil {
		log.Fatalf("Database auto-migration failed: %v", err)
	}

	DB = db
	log.Println("PostgreSQL connected and tables migrated successfully")
	return DB
}

func GetDB() *gorm.DB {
	return DB
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
