package model

import "time"

type Message struct {
	ID      int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Content string `gorm:"type:varchar(233);default:''" json:"content"`
	Link    string `gorm:"type:varchar(100);default:''" json:"link"`
	Status  string `gorm:"default:'unread'" json:"status"`
	Type    string `gorm:"not null" json:"type"`

	SenderID   int `gorm:"column:sender_id;not null" json:"sender_id"`
	ReceiverID int `gorm:"column:receiver_id;not null" json:"receiver_id"`

	Ref          `gorm:"embedded"`
	IsSuppressed bool `gorm:"column:is_suppressed;not null;default:false" json:"-"`

	CommunityNotificationID *int64 `gorm:"column:community_notification_id" json:"community_notification_id"`
	CommunitySeq            *int64 `gorm:"column:community_seq" json:"community_seq"`
	CommunityThreadID       *int64 `gorm:"column:community_thread_id" json:"community_thread_id"`
	CommunityPostNumber     *int   `gorm:"column:community_post_number" json:"community_post_number"`
	ItemCount               int    `gorm:"column:item_count;default:1;not null" json:"item_count"`
	ActorCount              int    `gorm:"column:actor_count;default:1;not null" json:"actor_count"`

	CreatedAt time.Time `gorm:"column:created" json:"created"`
	UpdatedAt time.Time `gorm:"column:updated" json:"updated"`
}

func (Message) TableName() string { return "message" }

type Ref struct {
	SubjectKind string `gorm:"column:subject_kind;default:null" json:"-"`
	SubjectID   int64  `gorm:"column:subject_id;default:null" json:"-"`
	RootKind    string `gorm:"column:root_kind;default:null" json:"-"`
	RootID      int64  `gorm:"column:root_id;default:null" json:"-"`
}

const (
	KindTopic         = "topic"
	KindReply         = "reply"
	KindTopicComment  = "topic_comment"
	KindWork          = "work"
	KindResource      = "resource"
	KindRating        = "rating"
	KindQuiz          = "quiz"
	KindQuizAnswer    = "quiz_answer"
	KindWalkthrough   = "walkthrough"
	KindToolset       = "toolset"
	KindWebsite       = "website"
	KindLottery       = "lottery"
	KindPoll          = "poll"
	KindEditProposal  = "edit_proposal"
	KindCommunityPost = "community_post"
	KindUser          = "user"
	KindActivity      = "activity"
	KindNotice        = "notice"
)

func Of(kind string, id int64) Ref {
	return Ref{SubjectKind: kind, SubjectID: id, RootKind: kind, RootID: id}
}

func (r Ref) Under(kind string, id int64) Ref {
	r.RootKind, r.RootID = kind, id
	return r
}

func TopicRef(topicID int) Ref { return Of(KindTopic, int64(topicID)) }

func ReplyRef(replyID, topicID int) Ref {
	return Of(KindReply, int64(replyID)).Under(KindTopic, int64(topicID))
}

func TopicCommentRef(commentID, topicID int) Ref {
	return Of(KindTopicComment, int64(commentID)).Under(KindTopic, int64(topicID))
}

func WorkRef(workID int) Ref { return Of(KindWork, int64(workID)) }

func ResourceRef(resourceID, workID int) Ref {
	return Of(KindResource, int64(resourceID)).Under(KindWork, int64(workID))
}

func RatingRef(ratingID, workID int) Ref {
	return Of(KindRating, int64(ratingID)).Under(KindWork, int64(workID))
}

func QuizAnswerRef(answerID, quizID int) Ref {
	return Of(KindQuizAnswer, int64(answerID)).Under(KindQuiz, int64(quizID))
}

func LotteryRef(lotteryID, topicID int) Ref {
	return Of(KindLottery, int64(lotteryID)).Under(KindTopic, int64(topicID))
}

func PollRef(pollID, topicID int) Ref {
	return Of(KindPoll, int64(pollID)).Under(KindTopic, int64(topicID))
}

func CommunityPostRef(postID int64, wall Ref) Ref {
	return Of(KindCommunityPost, postID).Under(wall.SubjectKind, wall.SubjectID)
}

func UserRef(userID int) Ref { return Of(KindUser, int64(userID)) }
