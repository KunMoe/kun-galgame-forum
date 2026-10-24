package model

import "time"

type Topic struct {
	AccessScope      string     `gorm:"column:access_scope;not null;default:public" json:"access_scope"`
	ID               int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Title            string     `gorm:"type:varchar(233);not null" json:"title"`
	Content          string     `gorm:"type:text;not null" json:"content"`
	View             int        `gorm:"default:0" json:"view"`
	IsNSFW           bool       `gorm:"column:is_nsfw;default:false" json:"is_nsfw"`
	Status           int        `gorm:"default:0" json:"status"`
	HiddenBy         string     `gorm:"column:hidden_by;default:''" json:"hidden_by"`
	Category         string     `gorm:"not null" json:"category"`
	StatusUpdateTime time.Time  `gorm:"column:status_update_time;autoCreateTime" json:"status_update_time"`
	Edited           *time.Time `gorm:"" json:"edited"`
	UpvoteTime       *time.Time `gorm:"column:upvote_time" json:"upvote_time"`

	UserID int `gorm:"column:user_id;not null" json:"user_id"`

	CoverImages ImageTokens `gorm:"column:cover_images;type:text;not null;default:''" json:"cover_images"`

	BestAnswerID  *int `gorm:"column:best_answer_id;uniqueIndex;constraint:OnDelete:SET NULL" json:"best_answer_id"`
	PinnedReplyID *int `gorm:"column:pinned_reply_id;uniqueIndex;constraint:OnDelete:SET NULL" json:"pinned_reply_id"`

	LikeCount      int `gorm:"column:like_count;default:0" json:"like_count"`
	DislikeCount   int `gorm:"column:dislike_count;default:0" json:"dislike_count"`
	ReplyCount     int `gorm:"column:reply_count;default:0" json:"reply_count"`
	LastReplyFloor int `gorm:"column:last_reply_floor;not null;default:0" json:"-"`
	CommentCount   int `gorm:"column:comment_count;default:0" json:"comment_count"`
	FavoriteCount  int `gorm:"column:favorite_count;default:0" json:"favorite_count"`
	UpvoteCount    int `gorm:"column:upvote_count;default:0" json:"upvote_count"`

	AISummary      *string    `gorm:"column:ai_summary" json:"ai_summary"`
	AISummaryModel string     `gorm:"column:ai_summary_model;not null;default:''" json:"-"`
	AISummaryAt    *time.Time `gorm:"column:ai_summary_at" json:"-"`
	AISummaryDrift int        `gorm:"column:ai_summary_drift;not null;default:0" json:"-"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (Topic) TableName() string { return "topic" }

const topicBumpWindowMonths = 3

func BumpCutoff(now time.Time) time.Time {
	return now.AddDate(0, -topicBumpWindowMonths, 0)
}

type TopicSection struct {
	ID               int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string `gorm:"not null" json:"name"`
	GroupID          int    `gorm:"column:group_id" json:"group_id"`
	Label            string `gorm:"column:label;not null" json:"label"`
	Description      string `gorm:"column:description;not null" json:"description"`
	Icon             string `gorm:"column:icon;not null" json:"icon"`
	Color            string `gorm:"column:color;not null" json:"color"`
	Position         int    `gorm:"column:position;not null" json:"position"`
	IsArchived       bool   `gorm:"column:is_archived;not null" json:"is_archived"`
	CostsMoemoepoint bool   `gorm:"column:costs_moemoepoint;not null" json:"costs_moemoepoint"`
	HiddenFromFeeds  bool   `gorm:"column:hidden_from_feeds;not null" json:"hidden_from_feeds"`

	BannerImageHash   string   `gorm:"column:banner_image_hash;not null" json:"banner_image_hash"`
	MascotImageHashes []string `gorm:"column:mascot_image_hashes;serializer:json;type:jsonb;not null" json:"mascot_image_hashes"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicSection) TableName() string { return "topic_section" }

type TopicSectionGroup struct {
	ID             int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Key            string `gorm:"column:key;not null" json:"key"`
	Label          string `gorm:"column:label;not null" json:"label"`
	Position       int    `gorm:"column:position;not null" json:"position"`
	LegacyCategory string `gorm:"column:legacy_category;not null" json:"legacy_category"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicSectionGroup) TableName() string { return "topic_section_group" }

type TopicSectionRelation struct {
	TopicID        int `gorm:"column:topic_id;primaryKey" json:"topic_id"`
	TopicSectionID int `gorm:"column:topic_section_id;primaryKey" json:"topic_section_id"`
	Position       int `gorm:"column:position;not null;default:0" json:"position"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicSectionRelation) TableName() string { return "topic_section_relation" }

type TopicUpvote struct {
	ID          int    `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicID     int    `gorm:"column:topic_id;not null" json:"topic_id"`
	UserID      int    `gorm:"column:user_id;not null" json:"user_id"`
	Description string `gorm:"column:description;default:''" json:"description"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicUpvote) TableName() string { return "topic_upvote" }

type TopicFavorite struct {
	ID      int `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicID int `gorm:"column:topic_id;not null;uniqueIndex:idx_topic_favorite" json:"topic_id"`
	UserID  int `gorm:"column:user_id;not null;uniqueIndex:idx_topic_favorite" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicFavorite) TableName() string { return "topic_favorite" }

type TopicReply struct {
	ID      int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Content string     `gorm:"type:text;default:''" json:"content"`
	Floor   int        `gorm:"default:0" json:"floor"`
	Edited  *time.Time `gorm:"" json:"edited"`

	UserID  int `gorm:"column:user_id;not null" json:"user_id"`
	TopicID int `gorm:"column:topic_id;not null" json:"topic_id"`

	Status int `gorm:"column:status;default:0" json:"-"`

	LikeCount    int `gorm:"column:like_count;default:0" json:"like_count"`
	DislikeCount int `gorm:"column:dislike_count;default:0" json:"dislike_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicReply) TableName() string { return "topic_reply" }

type TopicReplyLike struct {
	ID           int `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int `gorm:"column:user_id;not null;uniqueIndex:idx_reply_like" json:"user_id"`
	TopicReplyID int `gorm:"column:topic_reply_id;not null;uniqueIndex:idx_reply_like" json:"topic_reply_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicReplyLike) TableName() string { return "topic_reply_like" }

type TopicReplyDislike struct {
	ID           int `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int `gorm:"column:user_id;not null;uniqueIndex:idx_reply_dislike" json:"user_id"`
	TopicReplyID int `gorm:"column:topic_reply_id;not null;uniqueIndex:idx_reply_dislike" json:"topic_reply_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicReplyDislike) TableName() string { return "topic_reply_dislike" }

type TopicReaction struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicID   int       `gorm:"column:topic_id;not null" json:"topic_id"`
	UserID    int       `gorm:"column:user_id;not null" json:"user_id"`
	Reaction  string    `gorm:"column:reaction;not null" json:"reaction"`
	CreatedAt time.Time `gorm:"column:created" json:"created"`
}

func (TopicReaction) TableName() string { return "topic_reaction" }

type TopicComment struct {
	ID           int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Content      string `gorm:"type:varchar(1007);default:''" json:"content"`
	TopicID      int    `gorm:"column:topic_id;not null" json:"topic_id"`
	TopicReplyID int    `gorm:"column:topic_reply_id;not null" json:"topic_reply_id"`
	UserID       int    `gorm:"column:user_id;not null" json:"user_id"`
	TargetUserID int    `gorm:"column:target_user_id;not null" json:"target_user_id"`

	ParentCommentID *int `gorm:"column:parent_comment_id" json:"parent_comment_id"`

	Edited *time.Time `gorm:"column:edited" json:"edited"`

	Status int `gorm:"column:status;default:0" json:"-"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicComment) TableName() string { return "topic_comment" }

type TopicCommentLike struct {
	ID             int `gorm:"primaryKey;autoIncrement" json:"id"`
	TopicCommentID int `gorm:"column:topic_comment_id;not null;uniqueIndex:idx_comment_like" json:"topic_comment_id"`
	UserID         int `gorm:"column:user_id;not null;uniqueIndex:idx_comment_like" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicCommentLike) TableName() string { return "topic_comment_like" }

type TopicPoll struct {
	ID               int        `gorm:"primaryKey;autoIncrement" json:"id"`
	Title            string     `gorm:"type:varchar(100);not null" json:"title"`
	Description      string     `gorm:"type:varchar(500);default:''" json:"description"`
	Type             string     `gorm:"default:'single'" json:"type"`
	MinChoice        int        `gorm:"column:min_choice;default:1" json:"min_choice"`
	MaxChoice        int        `gorm:"column:max_choice;default:1" json:"max_choice"`
	Deadline         *time.Time `gorm:"" json:"deadline"`
	Status           string     `gorm:"default:'open'" json:"status"`
	NotificationSent bool       `gorm:"column:notification_sent;default:false" json:"notification_sent"`
	ResultVisibility string     `gorm:"column:result_visibility;default:'always'" json:"result_visibility"`
	IsAnonymous      bool       `gorm:"column:is_anonymous;default:false" json:"is_anonymous"`
	CanChangeVote    bool       `gorm:"column:can_change_vote;default:true" json:"can_change_vote"`

	TopicID int `gorm:"column:topic_id;not null" json:"topic_id"`
	UserID  int `gorm:"column:user_id;not null" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicPoll) TableName() string { return "topic_poll" }

type TopicPollOption struct {
	ID     int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Text   string `gorm:"type:varchar(100);not null" json:"text"`
	PollID int    `gorm:"column:poll_id;not null" json:"poll_id"`

	VoteCount int `gorm:"column:vote_count;default:0" json:"vote_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicPollOption) TableName() string { return "topic_poll_option" }

type TopicPollVote struct {
	ID       int `gorm:"primaryKey;autoIncrement" json:"id"`
	PollID   int `gorm:"column:poll_id;not null;uniqueIndex:idx_poll_vote" json:"poll_id"`
	OptionID int `gorm:"column:option_id;not null;uniqueIndex:idx_poll_vote" json:"option_id"`
	UserID   int `gorm:"column:user_id;not null;uniqueIndex:idx_poll_vote;index:idx_user_poll" json:"user_id"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (TopicPollVote) TableName() string { return "topic_poll_vote" }
