package models

import (
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"reader/internal/app/reader"
)

// Entry entry
type Entry struct {
	ID int64

	Author   string    `gorm:"type:varchar(255)"`
	Content  string    `gorm:"type:text"`
	Date     time.Time `gorm:"type:timestamp with time zone"`
	Favorite bool      `gorm:"default:false;index"`
	GUID     string    `gorm:"type:varchar(760);not null;index:feed_id_guid,unique"`
	Link     string    `gorm:"type:varchar(1023);not null"`
	Read     bool      `gorm:"default:false;index;index:idx_entries_feed_read"`
	Title    string    `gorm:"type:varchar(255);not null"`

	Feed   *Feed
	FeedID int64  `gorm:"index:idx_entries_feed_read;index:feed_id_guid,unique"`
	Tags   []*Tag `gorm:"many2many:entry_tags"`
}

// AddEntry adds entries and returns inserted count
func AddEntry(entry *Entry) (int64, error) {
	if res := db.Create(&entry); res.Error != nil {
		return 0, res.Error
	}

	return entry.ID, nil
}

// AddEntries inserts new entries while ignoring duplicates
func AddEntries(entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}

	const batchSize = 200

	return db.
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(&entries, batchSize).
		Error
}

// AllScope generates all scope for query
func AllScope(db *gorm.DB) *gorm.DB {
	return db.
		Joins("JOIN feeds ON feeds.id = entries.feed_id").
		Where("feeds.priority >= ?", int64(reader.PriorityNormal))
}

// CategoryScope generates category scope for query
func CategoryScope(id int64) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Joins("JOIN feeds ON feeds.id = entries.feed_id").
			Where("feeds.priority >= ?", int64(reader.PriorityNormal)).
			Where("feeds.category_id = ?", id)
	}
}

// ContinuationScope generates continuation scope for query
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

// ExistingGUIDsForFeed returns GUIDs that exist in feed
func ExistingGUIDsForFeed(feedID int64, gUIDs []string) (map[string]struct{}, error) {
	if len(gUIDs) == 0 {
		return map[string]struct{}{}, nil
	}

	var rows []struct {
		GUID string
	}
	if res := db.Model(&Entry{}).
		Select("guid").
		Where("feed_id = ?", feedID).
		Where("guid IN ?", gUIDs).
		Scan(&rows); res.Error != nil {
		return nil, res.Error
	}

	set := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		set[row.GUID] = struct{}{}
	}
	return set, nil
}

// FeedScope generates feed scope for query
func FeedScope(id int64) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("entries.feed_id = ?", id)
	}
}

// GetEntryCountForFeedDate counts entries of specific feed and day
func GetEntryCountForFeedDate(feedID int64, date time.Time) (int, error) {
	end := date.Add(time.Hour*23 + time.Minute*59 + time.Second*59)

	var count int64
	if res := db.Model(&Entry{}).
		Where("feed_id = ?", feedID).
		Where("date BETWEEN ? AND ?", date, end).
		Count(&count); res.Error != nil {
		return 0, res.Error
	}

	return int(count), nil
}

// GetLatestFeedDate gets latest date of feed
func GetLatestFeedDate(feedID int64) (time.Time, error) {
	var nt sql.NullTime
	if res := db.Model(&Entry{}).
		Where("feed_id = ?", feedID).
		Select("MAX(date)").
		Scan(&nt); res.Error != nil {
		return time.Time{}, res.Error
	}

	if !nt.Valid {
		return time.Time{}, nil
	}

	return nt.Time.UTC(), nil
}

// IsEntryExist returns true if entry with guid exists
func IsEntryExist(guid string) (bool, error) {
	var entry *Entry
	if res := db.Where(&Entry{GUID: guid}).First(&entry); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, res.Error
	}

	return true, nil
}

// ListEntryIDs list Entry IDs with conditions
func ListEntryIDs(limit int, scopes ...func(*gorm.DB) *gorm.DB) ([]int64, bool, error) {
	type EntryID struct {
		ID int64
	}

	var rows []*EntryID
	if res := db.Model(&Entry{}).
		Select("entries.id").
		Scopes(scopes...).
		Limit(limit + 1).
		Scan(&rows); res.Error != nil {
		return nil, false, res.Error
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	ids := make([]int64, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}

	return ids, hasMore, nil
}

// ListEntriesByIDs list Entries by IDs
func ListEntriesByIDs(ids []int64, asc bool) ([]*Entry, error) {
	// TODO: split for chunk ?

	var entries []*Entry
	if res := db.
		Preload("Tags").
		Scopes(OrderScope(asc)).
		Find(&entries, ids); res.Error != nil {
		return nil, res.Error
	}

	return entries, nil
}

// MarkFavorite marks entries for favorite state
func MarkFavorite(ids []int64, favorite bool) (int64, error) {
	res := db.Model(&Entry{}).Where("id IN ?", ids).Update("favorite", favorite)
	if res.Error != nil {
		return 0, res.Error
	}

	return res.RowsAffected, nil
}

// MarkRead marks entries for read state
func MarkRead(ids []int64, read bool) (int64, error) {
	res := db.Model(&Entry{}).Where("id IN ?", ids).Update("read", read)
	if res.Error != nil {
		return 0, res.Error
	}

	return res.RowsAffected, nil
}

// OrderScope generates order scope for query
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

// StarredScope generates starred scope for query
func StarredScope(db *gorm.DB) *gorm.DB {
	return db.
		Joins("JOIN feeds ON feeds.id = entries.feed_id").
		Where("feeds.priority >= ?", int64(reader.PriorityNormal)).
		Where("entries.favorite = true")
}

// StartTimeScope generates start time scope for query
func StartTimeScope(time time.Time) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("entries.date >= ?", time)
	}
}

// StateScope generates state scope for query
func StateScope(state reader.State) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if state&reader.StateNotRead != 0 {
			if state&reader.StateRead == 0 {
				db = db.Where("entries.read = false")
			}
		} else if state&reader.StateRead != 0 {
			db = db.Where("entries.read = true")
		}

		if state&reader.StateFavorite != 0 {
			if state&reader.StateNotFavorite == 0 {
				db = db.Where("entries.favorite = true")
			}
		} else if state&reader.StateNotFavorite != 0 {
			db = db.Where("entries.favorite = false")
		}

		return db
	}
}

// StopTimeScope generates stop time scope for query
func StopTimeScope(time time.Time) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("entries.date <= ?", time)
	}
}

// TagScope generates tag scope for for query
func TagScope(id int64) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Joins("JOIN feeds ON feeds.id = entries.feed_id").
			Where("feeds.priority >= ?", int64(reader.PriorityNormal)).
			Joins("JOIN entry_tags ON entry_tags.entry_id = entries.id").
			Where("entry_tags.tag_id = ?", id)
	}
}
