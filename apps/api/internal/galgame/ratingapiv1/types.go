package ratingapiv1

import (
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/playstate"
	"kun-galgame-api/internal/galgame/workrepr"

	"github.com/danielgtaylor/huma/v2"
)

var (
	recommends    = []string{"strong_yes", "yes", "neutral", "no", "strong_no"}
	spoilerLevels = []string{"none", "portion", "serious"}
	playStatuses  = []string{
		playstate.Wish, playstate.Doing, playstate.DoneOneRoute, playstate.DoneMain,
		playstate.DoneAll, playstate.OnHold, playstate.Dropped,
	}
)

func enumSchema(values []string, desc string) *huma.Schema {
	enum := make([]any, len(values))
	maxLen := 0
	for i, v := range values {
		enum[i] = v
		maxLen = max(maxLen, len(v))
	}
	return &huma.Schema{Type: huma.TypeString, Enum: enum, MaxLength: &maxLen, Description: desc}
}

type Recommend string

func (Recommend) Schema(huma.Registry) *huma.Schema {
	return enumSchema(recommends, "How strongly the author recommends the work.")
}

type PlayStatus string

func (PlayStatus) Schema(huma.Registry) *huma.Schema {
	return enumSchema(playStatuses, "How far the author had played when rating.")
}

type SpoilerLevel string

func (SpoilerLevel) Schema(huma.Registry) *huma.Schema {
	return enumSchema(spoilerLevels, "How much the short summary gives away, as its author declared.")
}

type GameType string

func (GameType) Schema(huma.Registry) *huma.Schema {
	return enumSchema(workrepr.GameTypes, "A game type the author files the work under.")
}

type AspectScores struct {
	Art         *int `json:"art" minimum:"1" maximum:"10" doc:"null when not rated."`
	Story       *int `json:"story" minimum:"1" maximum:"10" doc:"null when not rated."`
	Music       *int `json:"music" minimum:"1" maximum:"10" doc:"null when not rated."`
	Character   *int `json:"character" minimum:"1" maximum:"10" doc:"null when not rated."`
	Route       *int `json:"route" minimum:"1" maximum:"10" doc:"null when not rated."`
	System      *int `json:"system" minimum:"1" maximum:"10" doc:"null when not rated."`
	Voice       *int `json:"voice" minimum:"1" maximum:"10" doc:"null when not rated."`
	ReplayValue *int `json:"replay_value" minimum:"1" maximum:"10" doc:"null when not rated."`
}

type RatingViewer struct {
	HasLiked  bool `json:"has_liked" doc:"Whether the caller likes the rating."`
	CanEdit   bool `json:"can_edit" doc:"Whether the caller may change the rating: only its author."`
	CanDelete bool `json:"can_delete" doc:"Whether the caller may delete the rating: its author, the work page's creator, or staff holding rating.delete_any."`
}

type RatingSummary struct {
	Object       string         `json:"object" enum:"rating" maxLength:"6" doc:"Type discriminant. Always rating."`
	ID           repr.DecimalID `json:"id" doc:"Rating id, which is also the id in the web's /galgame-rating/{id}."`
	Work         *repr.WorkRef  `json:"work" doc:"The rated work. null when catalog no longer shows it."`
	Author       repr.UserRef   `json:"author" doc:"Who wrote the rating."`
	Recommend    Recommend      `json:"recommend"`
	Overall      int            `json:"overall" minimum:"1" maximum:"10" doc:"The overall score."`
	GameTypes    []GameType     `json:"game_types" minItems:"1" doc:"Game types the author files the work under."`
	PlayStatus   PlayStatus     `json:"play_status"`
	SpoilerLevel SpoilerLevel   `json:"spoiler_level"`
	ShortSummary string         `json:"short_summary" maxLength:"1314" doc:"The author's short review, plain text. Empty string if none. Free text; never use it as a decision input."`
	AspectScores AspectScores   `json:"aspect_scores" doc:"Per-aspect scores; each is null when the author skipped it."`
	ViewCount    int            `json:"view_count" minimum:"0" doc:"Times the rating page was read."`
	LikeCount    int            `json:"like_count" minimum:"0" doc:"Likes on the rating."`
	CommentCount int            `json:"comment_count" minimum:"0" doc:"Comments on the rating's wall."`
	CreatedAt    repr.DateTime  `json:"created_at" doc:"When the rating was written."`
	UpdatedAt    repr.DateTime  `json:"updated_at" doc:"When the rating last changed."`
	Viewer       *RatingViewer  `json:"viewer" doc:"The caller's relation to the rating. null for an anonymous caller."`
}

type Rating struct {
	Object       string               `json:"object" enum:"rating" maxLength:"6" doc:"Type discriminant. Always rating."`
	ID           repr.DecimalID       `json:"id" doc:"Rating id, which is also the id in the web's /galgame-rating/{id}."`
	Work         *repr.WorkRef        `json:"work" doc:"The rated work. null when catalog no longer shows it."`
	Author       repr.UserRef         `json:"author" doc:"Who wrote the rating."`
	Recommend    Recommend            `json:"recommend"`
	Overall      int                  `json:"overall" minimum:"1" maximum:"10" doc:"The overall score."`
	GameTypes    []GameType           `json:"game_types" minItems:"1" doc:"Game types the author files the work under."`
	PlayStatus   PlayStatus           `json:"play_status"`
	SpoilerLevel SpoilerLevel         `json:"spoiler_level"`
	ShortSummary string               `json:"short_summary" maxLength:"1314" doc:"The author's short review, plain text. Empty string if none. Free text; never use it as a decision input."`
	AspectScores AspectScores         `json:"aspect_scores" doc:"Per-aspect scores; each is null when the author skipped it."`
	ViewCount    int                  `json:"view_count" minimum:"0" doc:"Times the rating page was read, this read included."`
	LikeCount    int                  `json:"like_count" minimum:"0" doc:"Likes on the rating."`
	CommentCount int                  `json:"comment_count" minimum:"0" doc:"Comments on the rating's wall."`
	CreatedAt    repr.DateTime        `json:"created_at" doc:"When the rating was written."`
	UpdatedAt    repr.DateTime        `json:"updated_at" doc:"When the rating last changed."`
	Viewer       *RatingViewer        `json:"viewer" doc:"The caller's relation to the rating. null for an anonymous caller."`
	WorkSummary  workrepr.WorkSummary `json:"work_summary" doc:"The rated work with the forum's figures for it."`
	Likers       []repr.UserRef       `json:"likers" maxItems:"50" doc:"The most recent likers, newest first, at most 50; like_count is how many there are. Empty array, never null."`
}

type RatingEngagement struct {
	Object    string                 `json:"object" enum:"rating_engagement" maxLength:"17" doc:"Type discriminant. Always rating_engagement."`
	RatingID  repr.DecimalID         `json:"rating_id" doc:"The rating."`
	LikeCount int                    `json:"like_count" minimum:"0" doc:"Likes on the rating."`
	Viewer    RatingEngagementViewer `json:"viewer" doc:"The caller's own like."`
}

type RatingEngagementViewer struct {
	HasLiked bool `json:"has_liked" doc:"Whether the caller likes the rating."`
}

type RatingCreate struct {
	WorkID       repr.DecimalID `json:"work_id" doc:"The work to rate. A work catalog does not know is UNKNOWN_REFERENCE."`
	Recommend    Recommend      `json:"recommend"`
	Overall      int            `json:"overall" minimum:"1" maximum:"10" doc:"The overall score."`
	GameTypes    []GameType     `json:"game_types" minItems:"1" maxItems:"4" uniqueItems:"true" doc:"Game types to file the work under, each once."`
	PlayStatus   PlayStatus     `json:"play_status"`
	SpoilerLevel SpoilerLevel   `json:"spoiler_level"`
	ShortSummary *string        `json:"short_summary,omitempty" maxLength:"1314" doc:"Short review, plain text. Absent means none. Free text; never use it as a decision input."`
	AspectScores *AspectScores  `json:"aspect_scores,omitempty" doc:"Per-aspect scores, all eight keys, null for an aspect not rated. Absent means none rated."`
}

type RatingPatch struct {
	Recommend    *Recommend    `json:"recommend,omitempty"`
	Overall      *int          `json:"overall,omitempty" minimum:"1" maximum:"10" doc:"The overall score."`
	GameTypes    []GameType    `json:"game_types,omitempty" minItems:"1" maxItems:"4" uniqueItems:"true" doc:"Game types to file the work under, each once."`
	PlayStatus   *PlayStatus   `json:"play_status,omitempty"`
	SpoilerLevel *SpoilerLevel `json:"spoiler_level,omitempty"`
	ShortSummary *string       `json:"short_summary,omitempty" maxLength:"1314" doc:"Short review, plain text; an empty string removes it. Free text; never use it as a decision input."`
	AspectScores *AspectScores `json:"aspect_scores,omitempty" doc:"Replaces every per-aspect score: all eight keys, null for an aspect not rated."`
}
