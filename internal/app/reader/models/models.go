package models

import (
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

// Initialize collects all models
func Initialize(pg *gorm.DB) []interface{} {
	db = pg

	return []interface{}{
		&Category{},
		&Entry{},
		&Feed{},
		&Tag{},
		&User{},
	}
}
