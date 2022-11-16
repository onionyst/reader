package feeds

import (
	"reader/internal/app/reader/models"
)

// SetupFeed setups feed
func SetupFeed(categoryID int64, name string, priority int8, url, website string) (int64, error) {
	feedID, err := models.GetFeedIDForURL(url)
	if err != nil {
		return 0, err
	}

	if feedID == -1 {
		if feedID, err = models.AddFeed(name, priority, url, website, categoryID); err != nil {
			return 0, err
		}
	}

	return feedID, nil
}
