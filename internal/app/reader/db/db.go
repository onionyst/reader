package db

import (
	"fmt"
	"os"
	"strconv"

	"gorm.io/gorm"

	"reader/internal/app/reader/models"
	"reader/internal/pkg/db/postgres"
)

// CloseDatabase closes database
func CloseDatabase(db *gorm.DB) {
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

// ServiceString returns service string
func ServiceString() string {
	host := os.Getenv("POSTGRES_HOST")
	port, err := strconv.Atoi(os.Getenv("POSTGRES_PORT"))
	if err != nil {
		panic("invalid database port")
	}

	return fmt.Sprintf("%s:%d", host, port)
}

// SetupDatabase setups database
func SetupDatabase() *gorm.DB {
	host := os.Getenv("POSTGRES_HOST")
	port, err := strconv.Atoi(os.Getenv("POSTGRES_PORT"))
	if err != nil {
		panic("invalid database port")
	}
	database := os.Getenv("POSTGRES_DB")
	username := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")

	db := postgres.ConnectDatabase(host, port, database, username, password)

	tables := models.Initialize(db)
	postgres.SyncTables(db, tables)

	return db
}
