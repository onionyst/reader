package models

import (
	"errors"

	"gorm.io/gorm"

	"reader/internal/app/reader"
)

// Feed feed
type Feed struct {
	ID int64

	Name     string `gorm:"type:varchar(255);not null;index"`
	Priority int8   `gorm:"default:10;not null;index"`
	URL      string `gorm:"type:varchar(255);not null;unique"`
	Website  string `gorm:"type:varchar(255)"`

	Category   *Category
	CategoryID int64
	Entries    []*Entry
}

// AddFeed adds a feed
func AddFeed(name string, priority int8, url, website string, categoryID int64) (int64, error) {
	feed := &Feed{
		Name:       name,
		Priority:   priority,
		URL:        url,
		Website:    website,
		CategoryID: categoryID,
	}
	if res := db.Create(&feed); res.Error != nil {
		return 0, res.Error
	}

	return feed.ID, nil
}

// GetFeedAndCategoryNames gets the feed names that have category names
func GetFeedAndCategoryNames() (map[int64]*reader.FeedCategoryName, error) {
	type result struct {
		FeedID       int64
		FeedName     string
		CategoryName string
	}

	var results []*result
	if res := db.Model(&Feed{}).
		Select(
			"feeds.id AS feed_id",
			"feeds.name AS feed_name",
			"categories.name AS category_name").
		Joins("JOIN categories ON categories.id = feeds.category_id").
		Scan(&results); res.Error != nil {
		return nil, res.Error
	}

	names := make(map[int64]*reader.FeedCategoryName)
	for _, res := range results {
		names[res.FeedID] = &reader.FeedCategoryName{
			CategoryName: res.CategoryName,
			FeedName:     res.FeedName,
		}
	}

	return names, nil
}

// GetFeedIDForURL gets the feed ID for given URL, -1 for not found
func GetFeedIDForURL(url string) (int64, error) {
	var feed *Feed
	if res := db.Where(&Feed{URL: url}).First(&feed); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return -1, nil
		}
		return 0, res.Error
	}

	return feed.ID, nil
}
