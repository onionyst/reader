package models

import (
	"errors"

	"gorm.io/gorm"
)

// Tag tag
type Tag struct {
	ID int64

	Name string `gorm:"type:varchar(63);unique;not null"`

	Entries []*Entry `gorm:"many2many:entry_tags"`
}

// AddTag adds tag for name
func AddTag(name string) (int64, error) {
	// TODO: check name length

	tag := &Tag{Name: name}
	if res := db.Create(&tag); res.Error != nil {
		return 0, res.Error
	}

	return tag.ID, nil
}

// AddTagForEntries add tag for entries
func AddTagForEntries(tagID int64, entryIDs []int64) error {
	var entries []*Entry
	for _, entryID := range entryIDs {
		entries = append(entries, &Entry{ID: entryID})
	}

	if err := db.Model(&Tag{ID: tagID}).
		Association("Entries").Append(entries); err != nil {
		return err
	}

	return nil
}

// GetTagIDForName gets the tag ID for given name, -1 for not found
func GetTagIDForName(name string) (int64, error) {
	var tag *Tag
	if res := db.Where(&Tag{Name: name}).First(&tag); res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			return -1, nil
		}
		return 0, res.Error
	}

	return tag.ID, nil
}

// GetTagNamesForEntryIDs gets tag names for entry IDs
func GetTagNamesForEntryIDs(entryIDs []int64) (map[int64][]string, error) {
	type result struct {
		TagName string
		EntryID int64
	}

	var results []*result
	if res := db.Model(&Tag{}).
		Select("tags.name AS tag_name", "entry_tags.entry_id AS entry_id").
		Joins("JOIN entry_tags ON entry_tags.tag_id = tags.id AND entry_tags.entry_id IN ?", entryIDs).
		Scan(&results); res.Error != nil {
		return nil, res.Error
	}

	entryTagNames := make(map[int64][]string)
	for _, res := range results {
		entryTagNames[res.EntryID] = append(entryTagNames[res.EntryID], res.TagName)
	}

	return entryTagNames, nil
}

// RemoveTagForEntries remove tag for entries
func RemoveTagForEntries(tagID int64, entryIDs []int64) error {
	var entries []*Entry
	for _, entryID := range entryIDs {
		entries = append(entries, &Entry{ID: entryID})
	}

	if err := db.Model(&Tag{ID: tagID}).
		Association("Entries").Delete(entries); err != nil {
		return err
	}

	return nil
}
