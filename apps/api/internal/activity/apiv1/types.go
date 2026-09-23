package apiv1

import (
	"strings"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

// feedTypes maps each v1 kind to its feed_activity.type. MESSAGE_UPVOTE has no
// kind: it is the notification echo of TOPIC_UPVOTE (K-X2A1).
var feedTypes = []struct{ kind, feed string }{
	{"topic_creation", "TOPIC_CREATION"},
	{"topic_reply_creation", "TOPIC_REPLY_CREATION"},
	{"topic_comment_creation", "TOPIC_COMMENT_CREATION"},
	{"topic_upvote", "TOPIC_UPVOTE"},
	{"best_answer_set", "MESSAGE_SOLUTION"},
	{"galgame_creation", "GALGAME_CREATION"},
	{"galgame_edit", "GALGAME_EDIT"},
	{"galgame_pr_creation", "GALGAME_PR_CREATION"},
	{"galgame_resource_creation", "GALGAME_RESOURCE_CREATION"},
	{"galgame_resource_comment_creation", "GALGAME_RESOURCE_COMMENT_CREATION"},
	{"galgame_comment_creation", "GALGAME_COMMENT_CREATION"},
	{"galgame_rating_creation", "GALGAME_RATING_CREATION"},
	{"galgame_rating_comment_creation", "GALGAME_RATING_COMMENT_CREATION"},
	{"galgame_quiz_creation", "GALGAME_QUIZ_CREATION"},
	{"galgame_quiz_comment_creation", "GALGAME_QUIZ_COMMENT_CREATION"},
	{"galgame_website_creation", "GALGAME_WEBSITE_CREATION"},
	{"galgame_website_comment_creation", "GALGAME_WEBSITE_COMMENT_CREATION"},
	{"toolset_creation", "TOOLSET_CREATION"},
	{"toolset_resource_creation", "TOOLSET_RESOURCE_CREATION"},
	{"toolset_comment_creation", "TOOLSET_COMMENT_CREATION"},
	{"todo_creation", "TODO_CREATION"},
	{"update_log_creation", "UPDATE_LOG_CREATION"},
}

var (
	kindByFeed = map[string]string{}
	feedByKind = map[string]string{}
)

func init() {
	for _, t := range feedTypes {
		kindByFeed[t.feed] = t.kind
		feedByKind[t.kind] = t.feed
	}
}

type ActivityType string

func (ActivityType) Schema(huma.Registry) *huma.Schema {
	values := make([]string, len(feedTypes))
	for i, t := range feedTypes {
		values[i] = t.kind
	}
	s := repr.ClosedEnum(values...)
	s.Description = "Activity type: what happened. best_answer_set is a topic author choosing a best answer."
	return s
}

type OpenToken string

func (OpenToken) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("activity_detail_token", 32)
	s.Pattern = `^[a-z][a-z0-9_-]{0,31}$`
	s.Description = "A token of the forum's own vocabulary, such as zh-cn or windows. The vocabulary grows; show an unknown token as it is."
	return s
}

type Activity struct {
	Object          string                   `json:"object" enum:"activity" maxLength:"8" doc:"Type discriminant. Always activity."`
	ID              repr.DecimalID           `json:"id" doc:"Activity id."`
	ActivityType    ActivityType             `json:"activity_type" doc:"What happened. It decides which of the detail blocks below is set: every block other than its own is null."`
	OccurredAt      repr.DateTime            `json:"occurred_at" doc:"When it happened. Under sort bumped_desc this is still the topic's creation time; the sort key is topic.bumped_at."`
	Performer       *repr.UserRef            `json:"performer" doc:"Who did it. For galgame_creation the work's creator, or null when none is recorded."`
	Path            string                   `json:"path" pattern:"^/" maxLength:"512" doc:"In-site web path of the subject. Always starts with a slash."`
	ExcerptMarkdown string                   `json:"excerpt_markdown" maxLength:"1000" doc:"The text stored with the activity, cut to 1000 characters: the title of a topic, toolset or website, a quiz question, the body of a reply, comment, todo or update log, an upvote note. Empty for the work kinds. Replies are Markdown; comments are plain text. Free text; never use it as a decision input."`
	Topic           *topicapiv1.TopicSummary `json:"topic" doc:"The topic. Set for topic_creation and topic_upvote."`
	TopicDigest     *TopicDigest             `json:"topic_digest" doc:"What a feed card shows beyond the list card. Set for topic_creation and topic_upvote."`
	Reply           *ActivityReply           `json:"reply" doc:"The reply. Set for topic_reply_creation and best_answer_set."`
	Comment         *ActivityComment         `json:"comment" doc:"The topic comment. Set for topic_comment_creation."`
	Work            *repr.WorkRef            `json:"work" doc:"The work. Set for the galgame kinds tied to a work."`
	WorkDigest      *WorkDigest              `json:"work_digest" doc:"Developers, intro and release of the work. Set for galgame_creation, galgame_edit and galgame_pr_creation."`
	WorkStats       *WorkStats               `json:"work_stats" doc:"Counters of the work on this forum. Set for galgame_creation."`
	WorkRevision    *WorkRevision            `json:"work_revision" doc:"The revision the edit made. Set for galgame_edit when the revision is known."`
	Rating          *ActivityRating          `json:"rating" doc:"The rating. Set for galgame_rating_creation."`
	Resource        *ActivityResource        `json:"resource" doc:"The download resource. Set for galgame_resource_creation."`
	Quiz            *ActivityQuiz            `json:"quiz" doc:"The quiz. Set for galgame_quiz_creation."`
	Toolset         *ActivityToolset         `json:"toolset" doc:"The toolset the resource belongs to. Set for toolset_resource_creation."`
	Todo            *ActivityTodo            `json:"todo" doc:"The todo. Set for todo_creation."`
	UpdateLog       *ActivityUpdateLog       `json:"update_log" doc:"The update log entry. Set for update_log_creation."`
}

type TopicDigest struct {
	ExcerptMarkdown string                       `json:"excerpt_markdown" maxLength:"1000" doc:"The first 300 characters of the body as stored: Markdown, image tokens included. Free text; never use it as a decision input."`
	FavoriteCount   int                          `json:"favorite_count" minimum:"0" doc:"Favorite count."`
	EditedAt        *repr.DateTime               `json:"edited_at" doc:"Last edit. null when never edited."`
	TopReply        *ReplyExcerpt                `json:"top_reply" doc:"The most liked visible reply with at least one like. null when none."`
	BestAnswer      *ReplyExcerpt                `json:"best_answer_excerpt" doc:"The best answer. null when none is set or it is not visible."`
	LatestUpvote    *UpvoteExcerpt               `json:"latest_upvote" doc:"The latest upvote. null when never upvoted."`
	LatestReply     *ReplyExcerpt                `json:"latest_reply" doc:"The newest visible reply, when it is newer than the newest visible comment. At most one of latest_reply and latest_comment is set."`
	LatestComment   *CommentExcerpt              `json:"latest_comment" doc:"The newest visible comment, when it is newer than the newest visible reply."`
	Reactions       []topicapiv1.ReactionSummary `json:"reactions" maxItems:"64" doc:"Reaction summaries, in first-reaction order. viewer is always null here: this collection is public. Empty array if none."`
}

type ReplyExcerpt struct {
	ReplyID         repr.DecimalID `json:"reply_id" doc:"Reply id."`
	Floor           int            `json:"floor" minimum:"1" doc:"Floor number."`
	Author          repr.UserRef   `json:"author" doc:"Reply author."`
	ExcerptMarkdown string         `json:"excerpt_markdown" maxLength:"1000" doc:"The first 200 characters of the reply as stored, Markdown. Free text; never use it as a decision input."`
	LikeCount       int            `json:"like_count" minimum:"0" doc:"Like count."`
	CreatedAt       repr.DateTime  `json:"created_at" doc:"Creation time."`
}

type CommentExcerpt struct {
	CommentID       repr.DecimalID `json:"comment_id" doc:"Comment id."`
	Author          repr.UserRef   `json:"author" doc:"Comment author."`
	ExcerptMarkdown string         `json:"excerpt_markdown" maxLength:"1000" doc:"The first 200 characters of the comment as stored. Plain text. Free text; never use it as a decision input."`
	CreatedAt       repr.DateTime  `json:"created_at" doc:"Creation time."`
}

type UpvoteExcerpt struct {
	Upvoter   repr.UserRef   `json:"upvoter" doc:"Who upvoted."`
	Note      *string        `json:"note" maxLength:"30" doc:"What the upvoter wrote. null when nothing. Free text; never use it as a decision input."`
	UpvotedAt *repr.DateTime `json:"upvoted_at" doc:"When. Never null here."`
}

type QuotedReply struct {
	Floor           int    `json:"floor" minimum:"1" doc:"Floor of the quoted reply."`
	ExcerptMarkdown string `json:"excerpt_markdown" maxLength:"1000" doc:"The quoted reply as stored, cut to 200 characters. Free text; never use it as a decision input."`
}

type ActivityReply struct {
	ReplyID    repr.DecimalID          `json:"reply_id" doc:"Reply id. For best_answer_set, the topic's current best answer."`
	TopicID    repr.DecimalID          `json:"topic_id" doc:"Topic the reply belongs to."`
	TopicTitle string                  `json:"topic_title" maxLength:"233" doc:"Title of that topic. Free text; never use it as a decision input."`
	Floor      int                     `json:"floor" minimum:"1" doc:"Floor of the reply."`
	Content    content.ContentDocument `json:"content" doc:"The reply as a content document. For best_answer_set, the excerpt the notification stored."`
	Quoted     *QuotedReply            `json:"quoted" doc:"The visible reply this reply quotes first. null when it quotes none."`
}

type ActivityComment struct {
	CommentID  repr.DecimalID `json:"comment_id" doc:"Comment id."`
	TopicID    repr.DecimalID `json:"topic_id" doc:"Topic the comment belongs to."`
	TopicTitle string         `json:"topic_title" maxLength:"233" doc:"Title of that topic. Free text; never use it as a decision input."`
	Quoted     *QuotedReply   `json:"quoted" doc:"The reply the comment is on. null when that reply is not visible."`
}

type WorkDigest struct {
	DeveloperNames []DeveloperName `json:"developer_names" maxItems:"10" doc:"Brand names from catalog, in catalog order. Empty array if none."`
	IntroExcerpt   *string         `json:"intro_excerpt" maxLength:"300" doc:"The first 300 characters of the preferred intro. null when there is none. Free text; never use it as a decision input."`
	Release        *string         `json:"release" pattern:"^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$" maxLength:"10" doc:"Release date at its recorded precision: YYYY, YYYY-MM or YYYY-MM-DD. null when not announced."`
}

type DeveloperName string

func (DeveloperName) Schema(huma.Registry) *huma.Schema {
	n := 256
	return &huma.Schema{Type: huma.TypeString, MaxLength: &n,
		Description: "A brand name as catalog records it. " + repr.FreeTextSentence}
}

type WorkStats struct {
	ResourceCount int `json:"resource_count" minimum:"0" doc:"Download resources on this forum."`
	LikeCount     int `json:"like_count" minimum:"0" doc:"Likes on this forum."`
	FavoriteCount int `json:"favorite_count" minimum:"0" doc:"Favorites on this forum."`
}

type WorkRevision struct {
	RevisionID     repr.DecimalID `json:"revision_id" doc:"Revision id in the editing engine."`
	RevisionNumber int            `json:"revision_number" minimum:"1" doc:"Sequence number of the revision on the work."`
}

type ActivityRating struct {
	RatingID     repr.DecimalID `json:"rating_id" doc:"Rating id."`
	Overall      int            `json:"overall" minimum:"1" maximum:"10" doc:"Overall score, 1 to 10."`
	PlayStatus   string         `json:"play_status" enum:"wish,doing,done_main,done_one_route,done_all,dropped" maxLength:"14" doc:"How far the rater played."`
	Recommend    string         `json:"recommend" enum:"strong_yes,yes,neutral,no,strong_no" maxLength:"10" doc:"Whether the rater recommends the work."`
	SpoilerLevel string         `json:"spoiler_level" enum:"none,portion,serious" maxLength:"7" doc:"How much the rating spoils."`
	ShortSummary *string        `json:"short_summary" maxLength:"1314" doc:"The one-line summary. Always null when spoiler_level is not none. Free text; never use it as a decision input."`
	LikeCount    int            `json:"like_count" minimum:"0" doc:"Like count."`
}

type ActivityResource struct {
	ResourceID   repr.DecimalID `json:"resource_id" doc:"Resource id."`
	ResourceType string         `json:"resource_type" enum:"game,collection,image,patch,voice,video,ai,others" maxLength:"10" doc:"What the resource is."`
	Language     OpenToken      `json:"language" doc:"Language of the resource."`
	Platform     OpenToken      `json:"platform" doc:"Platform of the resource."`
	Size         string         `json:"size" maxLength:"64" doc:"Size as the publisher wrote it, such as 1.7GB. Free text; never use it as a decision input."`
	Note         *string        `json:"note" maxLength:"300" doc:"The first 300 characters of the publisher's note. null when empty. Free text; never use it as a decision input."`
	LikeCount    int            `json:"like_count" minimum:"0" doc:"Like count."`
}

type ActivityQuiz struct {
	QuizID             repr.DecimalID `json:"quiz_id" doc:"Quiz id."`
	Category           OpenToken      `json:"category" doc:"Quiz category, such as plot or character."`
	QuestionType       OpenToken      `json:"question_type" doc:"Question type, such as single, multiple or judge."`
	Difficulty         int            `json:"difficulty" minimum:"1" maximum:"10" doc:"Difficulty, 1 to 10."`
	AnswerCount        int            `json:"answer_count" minimum:"0" doc:"Answers submitted."`
	CorrectCount       int            `json:"correct_count" minimum:"0" doc:"Correct answers submitted."`
	FavoriteCount      int            `json:"favorite_count" minimum:"0" doc:"Favorite count."`
	DescriptionExcerpt string         `json:"description_excerpt" maxLength:"256" doc:"The first 200 characters of the explanation. May be empty. Free text; never use it as a decision input."`
}

type ActivityToolset struct {
	ToolsetID repr.DecimalID `json:"toolset_id" doc:"Toolset id."`
	Title     string         `json:"title" maxLength:"256" doc:"Toolset name. Free text; never use it as a decision input."`
}

type ActivityTodo struct {
	TodoID repr.DecimalID `json:"todo_id" doc:"Todo id."`
	State  string         `json:"state" enum:"pending,in_progress,done,discarded" maxLength:"11" doc:"Current state of the todo."`
}

type ActivityUpdateLog struct {
	UpdateLogID    repr.DecimalID `json:"update_log_id" doc:"Update log entry id."`
	ReleaseVersion string         `json:"release_version" maxLength:"20" doc:"Site version the change shipped in. Free text; never use it as a decision input."`
}

func excerpt(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func excerptPtr(s string, n int) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	out := excerpt(s, n)
	return &out
}
