package apiv1

import (
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

type ReactionToken string

func (ReactionToken) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("reaction", 16)
	s.Pattern = "^[a-z][a-z0-9_]*$"
	s.Description = "Reaction token, such as like, dislike, heart or clap. The vocabulary grows; show an unknown token with a neutral fallback."
	return s
}

type Topic struct {
	Object            string                  `json:"object" enum:"topic" maxLength:"5" doc:"Type discriminant. Always topic."`
	ID                repr.DecimalID          `json:"id" doc:"Topic id. JSON string of a decimal integer."`
	Title             string                  `json:"title" maxLength:"233" doc:"Topic title as stored. Free text; never use it as a decision input."`
	State             string                  `json:"state" enum:"published,hidden" maxLength:"9" doc:"Lifecycle state. A hidden topic is visible only to its author and to staff."`
	HiddenBy          *string                 `json:"hidden_by" enum:"author,moderator,trust" maxLength:"9" doc:"Who hid the topic: its author, a moderator, or the trust-and-safety service. null when state is published."`
	AccessScope       string                  `json:"access_scope" enum:"public,login,role,users" maxLength:"6" doc:"Who may read the topic: everyone, signed-in users, holders of granted roles, or granted users. The author and staff always may."`
	Category          string                  `json:"category" enum:"galgame,technique,others" maxLength:"9" doc:"Topic category."`
	Sections          []SectionSlug           `json:"sections" maxItems:"3" doc:"Section slugs, in stored order. Empty array if none. Hyphenated URL segments of /section/{key}."`
	CoverImages       []repr.Image            `json:"cover_images" maxItems:"9" doc:"Cover images in stored token order. Tokens that do not parse are skipped, and so are stickers: a sticker is never a cover. Empty array if none."`
	IsNSFW            bool                    `json:"is_nsfw" doc:"Whether the author marked the topic NSFW. The body is returned either way; a client that hides NSFW content gates it."`
	Author            repr.UserRef            `json:"author" doc:"Topic author."`
	AuthorMoemoepoint int                     `json:"author_moemoepoint" minimum:"-2147483648" doc:"The author's moemoepoint balance as this forum last cached it. It can be negative."`
	Content           content.ContentDocument `json:"content" doc:"Topic body as a node tree."`
	ViewCount         int                     `json:"view_count" minimum:"0" doc:"Lifetime view count."`
	LikeCount         int                     `json:"like_count" minimum:"0" doc:"Like count. Equals the count of the like entry in reactions."`
	DislikeCount      int                     `json:"dislike_count" minimum:"0" doc:"Dislike count. Equals the count of the dislike entry in reactions."`
	FavoriteCount     int                     `json:"favorite_count" minimum:"0" doc:"Number of users who favorited the topic."`
	UpvoteCount       int                     `json:"upvote_count" minimum:"0" doc:"Number of upvotes."`
	ReplyCount        int                     `json:"reply_count" minimum:"0" doc:"Reply count."`
	CommentCount      int                     `json:"comment_count" minimum:"0" doc:"Comment count."`
	Reactions         []ReactionSummary       `json:"reactions" maxItems:"64" doc:"One entry per reaction token that has at least one reaction, likes and dislikes included, in first-used order. Empty array if none."`
	MiniApps          []MiniAppKind           `json:"mini_apps" maxItems:"2" doc:"Mini-apps attached to the topic, in registry order. Empty array if none."`
	PinnedReply       *Reply                  `json:"pinned_reply" doc:"The reply the author pinned. null when none is pinned or it is not visible. It also appears in the replies collection at its floor."`
	BestAnswer        *Reply                  `json:"best_answer" doc:"The reply marked as the best answer. null when none is marked or it is not visible. It also appears in the replies collection at its floor."`
	CreatedAt         repr.DateTime           `json:"created_at" doc:"Creation time."`
	EditedAt          *repr.DateTime          `json:"edited_at" doc:"Time of the latest edit of the title or body. null when never edited."`
	BumpedAt          repr.DateTime           `json:"bumped_at" doc:"Bump time. A reply, a comment, an upvote, a new best answer, an edit of the title or body, and creating a poll or a lottery set it to now, but only for topics created within the last 3 months. Casting a vote and entering a lottery do not. It is not a last-activity time."`
	UpvotedAt         *repr.DateTime          `json:"upvoted_at" doc:"Time of the latest upvote. null when the topic has never been upvoted."`
	Viewer            *TopicViewer            `json:"viewer" doc:"The caller's own state on this topic. null for an anonymous caller."`
}

type TopicViewer struct {
	HasLiked         bool `json:"has_liked" doc:"Whether the caller liked the topic."`
	HasDisliked      bool `json:"has_disliked" doc:"Whether the caller disliked the topic."`
	HasFavorited     bool `json:"has_favorited" doc:"Whether the caller favorited the topic."`
	HasUpvoted       bool `json:"has_upvoted" doc:"Whether the caller has upvoted the topic."`
	CanEdit          bool `json:"can_edit" doc:"Whether the caller may edit the topic: its author, or staff holding the edit permission. Requests authenticated with a Bearer token never carry staff powers."`
	CanHide          bool `json:"can_hide" doc:"Whether the caller may hide the topic now: its author or staff holding the hide permission, while it is published."`
	CanUnhide        bool `json:"can_unhide" doc:"Whether the caller may publish the hidden topic again. Its author may undo only a hide of their own; staff holding the hide permission may undo any."`
	CanLike          bool `json:"can_like" doc:"Whether the caller may like the topic: anyone but its author, while it is published. Other reactions and favorites are open to every signed-in reader of a published topic and have no flag."`
	CanUpvote        bool `json:"can_upvote" doc:"Whether the caller may upvote the topic: anyone but its author, while it is published. An upvote can still fail on the caller's moemoepoint balance."`
	CanSetBestAnswer bool `json:"can_set_best_answer" doc:"Whether the caller may set or clear the best answer: its author or staff holding that permission, while the topic is published."`
	CanPinReply      bool `json:"can_pin_reply" doc:"Whether the caller may pin or unpin a reply: its author or staff holding that permission, while the topic is published."`
}

type ReactionSummary struct {
	Reaction ReactionToken   `json:"reaction" doc:"Reaction token, such as like, dislike, heart or clap. The vocabulary grows; show an unknown token with a neutral fallback."`
	Count    int             `json:"count" minimum:"1" doc:"Number of reactions with this token."`
	Reactors []repr.UserRef  `json:"reactors" maxItems:"3" doc:"Up to three of the earliest reactors, oldest first. Banned users are left out, so it can hold fewer than min(count, 3). Empty array if none remain."`
	Viewer   *ReactionViewer `json:"viewer" doc:"The caller's own state on this reaction. null for an anonymous caller."`
}

type ReactionViewer struct {
	HasReacted bool `json:"has_reacted" doc:"Whether the caller reacted with this token."`
}
