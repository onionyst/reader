package models

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"reader/internal/app/reader/domain"
)

type Entry struct {
	ID int64 `gorm:"primaryKey"`

	Author   string    `gorm:"not null"`
	Content  string    `gorm:"not null"`
	Date     time.Time `gorm:"not null;index:idx_entries_feed_date,priority:2,sort:desc"`
	Favorite bool      `gorm:"not null;default:false"`
	GUID     string    `gorm:"not null;uniqueIndex:uidx_entries_feed_guid,priority:2"`
	Link     string    `gorm:"not null"`
	Read     bool      `gorm:"not null;default:false"`
	Title    string    `gorm:"not null"`

	Feed   *Feed
	FeedID int64  `gorm:"not null;uniqueIndex:uidx_entries_feed_guid,priority:1;index:idx_entries_feed_date,priority:1"`
	Tags   []*Tag `gorm:"many2many:entry_tags"`
}

// AddEntries inserts entries, ignoring duplicates.
func (r *Repo) AddEntries(ctx context.Context, entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(entries, 500).
		Error
}

// CategoryScope filters by category. Requires WithNormalFeeds.
func CategoryScope(id int64) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("feeds.category_id = ?", id)
	}
}

// ContinuationScope filters entries for pagination.
func ContinuationScope(id int64, asc bool) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if asc {
			db = db.Where("entries.id > ?", id)
		} else {
			db = db.Where("entries.id < ?", id)
		}
		return db
	}
}

// ExistingGUIDsForFeed returns which GUIDs already exist for a feed.
func (r *Repo) ExistingGUIDsForFeed(ctx context.Context, feedID int64, guids []string) ([]string, error) {
	if len(guids) == 0 {
		return []string{}, nil
	}

	var exists []string
	err := r.db.WithContext(ctx).Model(&Entry{}).
		Where("feed_id = ? AND guid IN ?", feedID, guids).
		Pluck("guid", &exists).Error
	return exists, err
}

// FeedScope filters by feed ID.
func FeedScope(id int64) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("entries.feed_id = ?", id)
	}
}

// GetLatestFeedDate returns the most recent entry date for a feed.
func (r *Repo) GetLatestFeedDate(ctx context.Context, feedID int64) (time.Time, error) {
	var nt sql.NullTime
	if err := r.db.WithContext(ctx).Model(&Entry{}).
		Select("MAX(date)").
		Where("feed_id = ?", feedID).
		Row().
		Scan(&nt); err != nil {
		return time.Time{}, err
	}
	if !nt.Valid {
		return time.Time{}, nil
	}
	return nt.Time.UTC(), nil
}

// ListEntryIDs returns entry IDs matching the scopes, plus whether more exist.
func (r *Repo) ListEntryIDs(ctx context.Context, limit int, scopes ...func(*gorm.DB) *gorm.DB) ([]int64, bool, error) {
	var ids []int64
	if err := r.db.WithContext(ctx).Model(&Entry{}).
		Select("entries.id").
		Scopes(scopes...).
		Limit(limit+1).
		Pluck("entries.id", &ids).Error; err != nil {
		return nil, false, err
	}

	hasMore := len(ids) > limit
	if hasMore {
		ids = ids[:limit]
	}
	return ids, hasMore, nil
}

// ListEntriesByIDs returns entries by IDs with tags preloaded.
func (r *Repo) ListEntriesByIDs(ctx context.Context, ids []int64, asc bool) ([]*Entry, error) {
	if len(ids) == 0 {
		return []*Entry{}, nil
	}

	var entries []*Entry
	if err := r.db.WithContext(ctx).
		Preload("Tags").
		Scopes(OrderScope(asc)).
		Find(&entries, ids).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// MarkFavorite updates the favorite status for entries.
func (r *Repo) MarkFavorite(ctx context.Context, ids []int64, favorite bool) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	res := r.db.WithContext(ctx).Model(&Entry{}).Where("id IN ?", ids).Update("favorite", favorite)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// MarkRead updates the read status for entries.
func (r *Repo) MarkRead(ctx context.Context, ids []int64, read bool) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	res := r.db.WithContext(ctx).Model(&Entry{}).Where("id IN ?", ids).Update("read", read)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// OrderScope orders by entry ID.
func OrderScope(asc bool) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if asc {
			db = db.Order("entries.id")
		} else {
			db = db.Order("entries.id DESC")
		}
		return db
	}
}

// StarredScope filters to favorited entries. Requires WithNormalFeeds.
func StarredScope(db *gorm.DB) *gorm.DB {
	return db.Where("entries.favorite = true")
}

// StartTimeScope filters entries on or after the given time.
func StartTimeScope(time time.Time) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("entries.date >= ?", time)
	}
}

// StateScope filters by read/favorite state.
func StateScope(state domain.State) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if state&domain.StateNotRead != 0 {
			if state&domain.StateRead == 0 {
				db = db.Where("entries.read = false")
			}
		} else if state&domain.StateRead != 0 {
			db = db.Where("entries.read = true")
		}

		if state&domain.StateFavorite != 0 {
			if state&domain.StateNotFavorite == 0 {
				db = db.Where("entries.favorite = true")
			}
		} else if state&domain.StateNotFavorite != 0 {
			db = db.Where("entries.favorite = false")
		}

		return db
	}
}

// StopTimeScope filters entries on or before the given time.
func StopTimeScope(time time.Time) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("entries.date <= ?", time)
	}
}

// TagScope filters entries by tag ID.
func TagScope(id int64) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Joins("JOIN entry_tags ON entry_tags.entry_id = entries.id").
			Where("entry_tags.tag_id = ?", id)
	}
}

// WithNormalFeeds joins feeds table and filters by normal priority.
func WithNormalFeeds(db *gorm.DB) *gorm.DB {
	return db.
		Joins("JOIN feeds ON feeds.id = entries.feed_id").
		Where("feeds.priority >= ?", int64(domain.PriorityNormal))
}
