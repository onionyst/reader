package models

import (
	"context"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func (r *Repo) Tx(ctx context.Context, fn func(tx *Repo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(r.withDB(tx))
	})
}

func (r *Repo) withDB(db *gorm.DB) *Repo {
	return &Repo{
		db: db,
	}
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{
		db: db,
	}
}

// Models returns all model types for GORM.
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

// Register configures GORM join tables.
func Register(pg *gorm.DB) error {
	if err := pg.SetupJoinTable(&Entry{}, "Tags", &EntryTag{}); err != nil {
		return err
	}
	if err := pg.SetupJoinTable(&Tag{}, "Entries", &EntryTag{}); err != nil {
		return err
	}
	return nil
}
