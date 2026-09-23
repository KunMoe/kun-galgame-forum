package model

import "time"

const (
	ContributorSourceWikiSeed int16 = 0
	ContributorSourceRevision int16 = 1
)

type GalgameContributor struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	WorkID        int64     `gorm:"column:work_id;not null" json:"galgame_id"`
	UserID        int64     `gorm:"column:user_id;not null" json:"user_id"`
	FirstAt       time.Time `gorm:"column:first_at;not null" json:"first_at"`
	LastAt        time.Time `gorm:"column:last_at;not null" json:"last_at"`
	RevisionCount int       `gorm:"column:revision_count;not null" json:"revision_count"`
	Source        int16     `gorm:"column:source;not null" json:"source"`
}

func (GalgameContributor) TableName() string { return "galgame_contributor" }
