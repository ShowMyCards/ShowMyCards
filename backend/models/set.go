package models

// tygo:export
type Set struct {
	ScryfallID    string  `gorm:"primaryKey;type:varchar(36);not null" json:"scryfall_id"`
	Code          string  `gorm:"uniqueIndex;type:varchar(10);not null" json:"code"`
	Name          string  `gorm:"index;type:varchar(255);not null" json:"name"`
	SetType       string  `gorm:"index;type:varchar(50)" json:"set_type"`
	ReleasedAt    *string `gorm:"type:varchar(10)" json:"released_at"`
	CardCount     int     `gorm:"type:integer" json:"card_count"`
	Digital       bool    `gorm:"type:boolean" json:"digital"`
	IconFilename  string  `gorm:"type:varchar(255)" json:"icon_filename"`
	ParentSetCode string  `gorm:"type:varchar(10)" json:"parent_set_code"`
}

func (Set) TableName() string {
	return "sets"
}
