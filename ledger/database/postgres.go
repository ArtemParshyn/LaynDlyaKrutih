package database

import (
	"errors"
	"fmt"
	"ledger/migrations"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
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

	// 1. Run UP migrations using golang-migrate
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)

	if err := runMigrations(dbURL); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// 2. Open GORM connection
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

func runMigrations(dbURL string) error {
	driver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration iofs source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", driver, dbURL)
	if err != nil {
		return fmt.Errorf("failed to init migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	log.Println("Database migrations applied (.up.sql)")
	return nil
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
