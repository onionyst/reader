package models

import (
	"errors"

	"gorm.io/gorm"
)

// Category category
type Category struct {
	ID int64 `gorm:"primaryKey"`

	Name string `gorm:"not null;uniqueIndex:uidx_categories_name"`

	Feeds []*Feed
}

// AddCategory adds category
func AddCategory(name string) (int64, error) {
	category := &Category{Name: name}
	if res := db.Create(&category); res.Error != nil {
		return 0, res.Error
	}

	return category.ID, nil
}

// GetCategoryIDForName gets the category ID for given name, -1 for not found
func GetCategoryIDForName(name string) (int64, error) {
	var category *Category
	if res := db.Where(&Category{Name: name}).First(&category); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return -1, nil
		}
		return 0, res.Error
	}

	return category.ID, nil
}

// ListAllCategoriesWithFeeds gets all categories with feeds data
func ListAllCategoriesWithFeeds() ([]*Category, error) {
	var categories []*Category
	if res := db.Preload("Feeds").Find(&categories); res.Error != nil {
		return nil, res.Error
	}

	return categories, nil
}
