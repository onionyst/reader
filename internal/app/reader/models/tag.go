package models

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"reader/internal/pkg/db/postgres"
)

var ErrTagsNameAlreadyExists = errors.New("tags name already exists")

const (
	constraintTagsName = "uidx_tags_name"
)

var tagsUniqueConstraintErr = map[string]error{
	constraintTagsName: ErrTagsNameAlreadyExists,
}

type Tag struct {
	ID int64 `gorm:"primaryKey"`

	Name string `gorm:"not null;uniqueIndex:uidx_tags_name"`

	Entries []*Entry `gorm:"many2many:entry_tags"`
}

// AddTag creates a tag.
func (r *Repo) AddTag(ctx context.Context, name string) (int64, error) {
	tag := &Tag{
		Name: name,
	}
	if err := r.db.WithContext(ctx).Create(tag).Error; err != nil {
		return 0, postgres.MapUniqueConstraint(err, tagsUniqueConstraintErr)
	}
	return tag.ID, nil
}

// AddTagForEntries associates a tag with entries.
func (r *Repo) AddTagForEntries(ctx context.Context, tagID int64, entryIDs []int64) error {
	if len(entryIDs) == 0 {
		return nil
	}

	rows := make([]EntryTag, len(entryIDs))
	for idx, entryID := range entryIDs {
		rows[idx] = EntryTag{
			EntryID: entryID,
			TagID:   tagID,
		}
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(rows, 500).Error
}

// GetTagIDForName returns the tag ID for a name.
func (r *Repo) GetTagIDForName(ctx context.Context, name string) (int64, bool, error) {
	var tag Tag
	if err := r.db.WithContext(ctx).Where("name = ?", name).Take(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return tag.ID, true, nil
}

// GetTagNamesForEntryIDs returns tag names grouped by entry ID.
func (r *Repo) GetTagNamesForEntryIDs(ctx context.Context, entryIDs []int64) (map[int64][]string, error) {
	entryTagNames := make(map[int64][]string)
	if len(entryIDs) == 0 {
		return entryTagNames, nil
	}

	var results []struct {
		TagName string
		EntryID int64
	}
	if err := r.db.WithContext(ctx).Model(&Tag{}).
		Select("tags.name AS tag_name", "entry_tags.entry_id AS entry_id").
		Joins("JOIN entry_tags ON entry_tags.tag_id = tags.id").
		Where("entry_tags.entry_id IN ?", entryIDs).
		Scan(&results).Error; err != nil {
		return nil, err
	}

	for _, res := range results {
		entryTagNames[res.EntryID] = append(entryTagNames[res.EntryID], res.TagName)
	}
	return entryTagNames, nil
}

// RemoveTagForEntries removes tag associations from entries.
func (r *Repo) RemoveTagForEntries(ctx context.Context, tagID int64, entryIDs []int64) error {
	if len(entryIDs) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Where("tag_id = ? AND entry_id IN ?", tagID, entryIDs).
		Delete(&EntryTag{}).Error
}
