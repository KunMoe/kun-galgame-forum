package notifytype

import (
	"github.com/danielgtaylor/huma/v2"
)

type Type string

const (
	Upvoted                Type = "upvoted"
	Liked                  Type = "liked"
	Favorited              Type = "favorited"
	Replied                Type = "replied"
	Commented              Type = "commented"
	Mentioned              Type = "mentioned"
	FollowedThreadActivity Type = "followed_thread_activity"
	BestAnswerChosen       Type = "best_answer_chosen"
	ReplyPinned            Type = "reply_pinned"
	QuizAnswered           Type = "quiz_answered"
	ResourceLinkReported   Type = "resource_link_reported"
	EditRequested          Type = "edit_requested"
	EditMerged             Type = "edit_merged"
	EditDeclined           Type = "edit_declined"
	LotteryWon             Type = "lottery_won"
	LotteryDrawn           Type = "lottery_drawn"
	LotteryCodeExpired     Type = "lottery_code_expired"
	PollClosed             Type = "poll_closed"
)

const KeyChat = "chat"

var all = []Type{
	Upvoted,
	Liked,
	Favorited,
	Replied,
	Commented,
	Mentioned,
	FollowedThreadActivity,
	BestAnswerChosen,
	ReplyPinned,
	QuizAnswered,
	ResourceLinkReported,
	EditRequested,
	EditMerged,
	EditDeclined,
	LotteryWon,
	LotteryDrawn,
	LotteryCodeExpired,
	PollClosed,
}

var toDB = map[Type]string{
	Upvoted:                "upvoted",
	Liked:                  "liked",
	Favorited:              "favorite",
	Replied:                "replied",
	Commented:              "commented",
	Mentioned:              "mentioned",
	FollowedThreadActivity: "followed",
	BestAnswerChosen:       "solution",
	ReplyPinned:            "pin-reply",
	QuizAnswered:           "quiz-answered",
	ResourceLinkReported:   "expired",
	EditRequested:          "requested",
	EditMerged:             "merged",
	EditDeclined:           "declined",
	LotteryWon:             "lottery-won",
	LotteryDrawn:           "lottery-closed",
	LotteryCodeExpired:     "lottery-expired",
	PollClosed:             "poll-closed",
}

var fromDB = func() map[string]Type {
	m := make(map[string]Type, len(toDB))
	for tok, db := range toDB {
		m[db] = tok
	}
	return m
}()

func All() []Type {
	out := make([]Type, len(all))
	copy(out, all)
	return out
}

func ToDB(t Type) string {
	return toDB[t]
}

func FromDB(db string) (Type, bool) {
	t, ok := fromDB[db]
	return t, ok
}

func (Type) Schema(huma.Registry) *huma.Schema {
	enum := make([]any, len(all))
	maxLen := 0
	for i, t := range all {
		s := string(t)
		enum[i] = s
		if n := len(s); n > maxLen {
			maxLen = n
		}
	}
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        enum,
		MaxLength:   &maxLen,
		Description: "Notification type. Closed vocabulary of v1 tokens; stored values are mapped in Go and never appear here.",
	}
}
