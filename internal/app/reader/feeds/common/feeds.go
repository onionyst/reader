package common

import (
	"reader/internal/app/reader/models"
)

type Category string

const (
	CategoryGame Category = "Games"
)

// RegisterFeed registers feed under category
func RegisterFeed(category Category, name string, priority int8, url, website, iconURL string) (int64, error) {
	categoryID, err := models.GetCategoryIDForName(string(category))
	if err != nil {
		return 0, err
	}

	if categoryID == -1 {
		if categoryID, err = models.AddCategory(string(category)); err != nil {
			return 0, err
		}
	}

	feedID, err := models.GetFeedIDForURL(url)
	if err != nil {
		return 0, err
	}

	if feedID == -1 {
		if feedID, err = models.AddFeed(name, priority, url, website, iconURL, categoryID); err != nil {
			return 0, err
		}
	}

	return feedID, nil
}
