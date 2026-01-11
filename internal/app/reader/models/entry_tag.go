package models

type EntryTag struct {
	EntryID int64 `gorm:"primaryKey;autoIncrement:false;index:idx_entry_tags_tag_entry,priority:2"`
	TagID   int64 `gorm:"primaryKey;autoIncrement:false;index:idx_entry_tags_tag_entry,priority:1"`
}

func (EntryTag) TableName() string {
	return "entry_tags"
}
