package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectPostgres() *gorm.DB {
	host := getEnv("DB_HOST", "localhost")
	user := getEnv("DB_USER", "admin")
	password := getEnv("DB_PASSWORD", "secret123")
	dbname := getEnv("DB_NAME", "users_db")
	port := getEnv("DB_PORT", "5432")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	DB = db
	log.Println("PostgreSQL connected successfully")
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
