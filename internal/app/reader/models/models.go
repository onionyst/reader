package models

import (
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

// Models returns all models
func Models() []any {
	return []any{
		&Category{},
		&EntryTag{},
		&Entry{},
		&Feed{},
		&Tag{},
		&User{},
	}
}

// Register adds external indexes to database
func Register(db *gorm.DB) error {
	if err := db.SetupJoinTable(&Entry{}, "Tags", &EntryTag{}); err != nil {
		return err
	}
	if err := db.SetupJoinTable(&Tag{}, "Entries", &EntryTag{}); err != nil {
		return err
	}
	return nil
}
