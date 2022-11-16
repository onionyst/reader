package postgres

import (
	"fmt"

	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectDatabase initializes PostgreSQL
func ConnectDatabase(host string, port int, database, username, password string) *gorm.DB {
	connectStr := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=disable TimeZone=UTC", username, password, host, port, database)

	db, err := gorm.Open(pg.Open(connectStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect database")
	}

	return db
}

// SyncTables syncs all tables
func SyncTables(db *gorm.DB, tables []interface{}) {
	if err := db.AutoMigrate(tables...); err != nil {
		panic("failed to sync database")
	}
}
