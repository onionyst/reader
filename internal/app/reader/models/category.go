package models

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"reader/internal/pkg/db/postgres"
)

var ErrCategoriesNameAlreadyExists = errors.New("name already exists")

const (
	constraintCategoriesName = "uidx_categories_name"
)

var categoriesUniqueConstraintErr = map[string]error{
	constraintCategoriesName: ErrCategoriesNameAlreadyExists,
}

type Category struct {
	ID int64 `gorm:"primaryKey"`

	Name string `gorm:"not null;uniqueIndex:uidx_categories_name"`

	Feeds []*Feed
}

// AddCategory creates a category.
func (r *Repo) AddCategory(ctx context.Context, name string) (int64, error) {
	category := &Category{
		Name: name,
	}
	if err := r.db.WithContext(ctx).Create(category).Error; err != nil {
		return 0, postgres.MapUniqueConstraint(err, categoriesUniqueConstraintErr)
	}
	return category.ID, nil
}

// GetCategoryIDForName returns the category ID for a name.
func (r *Repo) GetCategoryIDForName(ctx context.Context, name string) (int64, bool, error) {
	var category Category
	if err := r.db.WithContext(ctx).Select("id").Where("name = ?", name).Take(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return category.ID, true, nil
}

// ListAllCategoriesWithFeeds returns all categories with feeds preloaded.
func (r *Repo) ListAllCategoriesWithFeeds(ctx context.Context) ([]*Category, error) {
	var categories []*Category
	if err := r.db.WithContext(ctx).Preload("Feeds").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}
