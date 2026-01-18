package models

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"reader/internal/app/reader/domain"
	"reader/internal/pkg/db/postgres"
)

var ErrFeedsURLAlreadyExists = errors.New("url already exists")

const (
	constraintFeedsURL = "uidx_feeds_url"
)

var feedsUniqueConstraintErr = map[string]error{
	constraintFeedsURL: ErrFeedsURLAlreadyExists,
}

type Feed struct {
	ID int64 `gorm:"primaryKey"`

	Name     string `gorm:"not null"`
	Priority int8   `gorm:"not null;default:10;index:idx_feeds_category_priority,priority:2"`
	URL      string `gorm:"not null;uniqueIndex:uidx_feeds_url"`
	Website  string `gorm:"not null"`
	IconURL  string `gorm:"not null"`

	Category   *Category
	CategoryID int64 `gorm:"not null;index:idx_feeds_category_priority,priority:1"`
	Entries    []*Entry
}

// AddFeed creates a feed.
func (r *Repo) AddFeed(ctx context.Context, name string, priority int8, url, website, iconURL string, categoryID int64) (int64, error) {
	feed := &Feed{
		Name:       name,
		Priority:   priority,
		URL:        url,
		Website:    website,
		IconURL:    iconURL,
		CategoryID: categoryID,
	}
	if err := r.db.WithContext(ctx).Create(feed).Error; err != nil {
		return 0, postgres.MapUniqueConstraint(err, feedsUniqueConstraintErr)
	}
	return feed.ID, nil
}

// GetFeedAndCategoryNames returns feed and category names indexed by feed ID.
func (r *Repo) GetFeedAndCategoryNames(ctx context.Context) (map[int64]domain.FeedCategoryName, error) {
	type result struct {
		FeedID       int64
		FeedName     string
		CategoryName string
	}
	var results []result
	if err := r.db.WithContext(ctx).Model(&Feed{}).
		Select(
			"feeds.id AS feed_id",
			"feeds.name AS feed_name",
			"categories.name AS category_name").
		Joins("JOIN categories ON categories.id = feeds.category_id").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	names := make(map[int64]domain.FeedCategoryName, len(results))
	for _, res := range results {
		names[res.FeedID] = domain.FeedCategoryName{
			CategoryName: res.CategoryName,
			FeedName:     res.FeedName,
		}
	}
	return names, nil
}

// GetFeedIDForURL returns the feed ID for a URL.
func (r *Repo) GetFeedIDForURL(ctx context.Context, url string) (int64, bool, error) {
	var feed Feed
	if err := r.db.WithContext(ctx).Where("url = ?", url).Take(&feed).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return feed.ID, true, nil
}
