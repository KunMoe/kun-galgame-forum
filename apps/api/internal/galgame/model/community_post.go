package model

type GalgameCommentCommunityMap struct {
	OldCommentID int   `gorm:"column:old_comment_id;primaryKey" json:"old_comment_id"`
	ThreadID     int64 `gorm:"column:thread_id;not null" json:"thread_id"`
	PostID       int64 `gorm:"column:post_id;not null" json:"post_id"`
	GalgameID    int   `gorm:"column:galgame_id;not null" json:"galgame_id"`
}

func (GalgameCommentCommunityMap) TableName() string { return "galgame_comment_community_map" }
