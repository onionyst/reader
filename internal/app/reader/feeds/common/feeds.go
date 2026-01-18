package common

import (
	"context"

	"reader/internal/app/reader/models"
)

type Category string

const (
	CategoryGame Category = "Games"
)

// RegisterFeed ensures a category and feed exist, returning the feed ID.
func RegisterFeed(ctx context.Context, repo *models.Repo, category Category, name string, priority int8, url, website, iconURL string) (int64, error) {
	categoryID, found, err := repo.GetCategoryIDForName(ctx, string(category))
	if err != nil {
		return 0, err
	}

	if !found {
		if categoryID, err = repo.AddCategory(ctx, string(category)); err != nil {
			return 0, err
		}
	}

	feedID, found, err := repo.GetFeedIDForURL(ctx, url)
	if err != nil {
		return 0, err
	}

	if !found {
		if feedID, err = repo.AddFeed(ctx, name, priority, url, website, iconURL, categoryID); err != nil {
			return 0, err
		}
	}

	return feedID, nil
}
