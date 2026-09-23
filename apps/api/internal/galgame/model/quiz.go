package model

import (
	"encoding/json"
	"time"
)

type GalgameQuiz struct {
	ID           int             `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int             `gorm:"column:user_id;not null" json:"user_id"`
	Category     string          `gorm:"type:varchar(16)" json:"category"`
	SpoilerLevel string          `gorm:"column:spoiler_level;type:varchar(16)" json:"spoiler_level"`
	Type         string          `gorm:"type:varchar(16)" json:"type"`
	Difficulty   int             `gorm:"type:smallint" json:"difficulty"`
	Question     string          `gorm:"type:text" json:"question"`
	Description  string          `gorm:"type:text" json:"description"`
	Content      json.RawMessage `gorm:"type:jsonb" json:"content"`
	Explanation  string          `gorm:"type:text" json:"explanation"`
	HideGalgame  bool            `gorm:"column:hide_galgame;default:false" json:"hide_galgame"`
	View         int             `gorm:"default:0" json:"view"`

	AnswerCount   int `gorm:"column:answer_count;default:0" json:"answer_count"`
	CorrectCount  int `gorm:"column:correct_count;default:0" json:"correct_count"`
	FavoriteCount int `gorm:"column:favorite_count;default:0" json:"favorite_count"`
	CommentCount  int `gorm:"column:comment_count;default:0" json:"comment_count"`
	QualitySum    int `gorm:"column:quality_sum;default:0" json:"quality_sum"`
	QualityCount  int `gorm:"column:quality_count;default:0" json:"quality_count"`

	StatusUpdateTime time.Time `gorm:"column:status_update_time;default:now()" json:"status_update_time"`
	CreatedAt        time.Time `gorm:"column:created" json:"created"`
	UpdatedAt        time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameQuiz) TableName() string { return "galgame_quiz" }

const quizBumpWindowDays = 3

func QuizBumpCutoff(now time.Time) time.Time {
	return now.AddDate(0, 0, -quizBumpWindowDays)
}

type GalgameQuizAnswer struct {
	ID            int             `gorm:"primaryKey;autoIncrement" json:"id"`
	QuizID        int             `gorm:"column:quiz_id;not null" json:"quiz_id"`
	UserID        int             `gorm:"column:user_id;not null" json:"user_id"`
	Role          string          `gorm:"type:varchar(16);default:answerer" json:"role"`
	Submitted     json.RawMessage `gorm:"type:jsonb" json:"submitted"`
	IsCorrect     *bool           `gorm:"column:is_correct" json:"is_correct"`
	Rewarded      bool            `gorm:"column:rewarded;default:false" json:"rewarded"`
	QualityRating *int            `gorm:"column:quality_rating" json:"quality_rating"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (GalgameQuizAnswer) TableName() string { return "galgame_quiz_answer" }

type GalgameQuizGalgame struct {
	QuizID int `gorm:"column:quiz_id;primaryKey"`
	WorkID int `gorm:"column:work_id;primaryKey"`
}

func (GalgameQuizGalgame) TableName() string { return "galgame_quiz_galgame" }

type GalgameQuizRow struct {
	ID               int    `gorm:"column:id"`
	UserID           int    `gorm:"column:user_id"`
	Category         string `gorm:"column:category"`
	SpoilerLevel     string `gorm:"column:spoiler_level"`
	Type             string `gorm:"column:type"`
	Difficulty       int    `gorm:"column:difficulty"`
	Question         string `gorm:"column:question"`
	View             int    `gorm:"column:view"`
	AnswerCount      int    `gorm:"column:answer_count"`
	CorrectCount     int    `gorm:"column:correct_count"`
	FavoriteCount    int    `gorm:"column:favorite_count"`
	CommentCount     int    `gorm:"column:comment_count"`
	QualitySum       int    `gorm:"column:quality_sum"`
	QualityCount     int    `gorm:"column:quality_count"`
	StatusUpdateTime string `gorm:"column:status_update_time"`
	Created          string `gorm:"column:created"`
	Updated          string `gorm:"column:updated"`
}

type GalgameQuizAnswererRow struct {
	UserID    int             `gorm:"column:user_id"`
	Submitted json.RawMessage `gorm:"column:submitted"`
	IsCorrect *bool           `gorm:"column:is_correct"`
	Created   string          `gorm:"column:created"`
}

type QuizFilter struct {
	Category   string
	Type       string
	SortField  string
	SortOrder  string
	Difficulty int
	WorkID     int
	UserID     int
	Page       int
	Limit      int
}
