package apiv1

import (
	"bytes"
	"encoding/json"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"

	"github.com/danielgtaylor/huma/v2"
)

type WorkCoverEngagement struct {
	Object    string                     `json:"object" enum:"work_cover_engagement" maxLength:"22" doc:"Type discriminant. Always work_cover_engagement."`
	WorkID    repr.DecimalID             `json:"work_id" doc:"Work id."`
	CoverID   repr.DecimalID             `json:"cover_id" doc:"Catalog cover row id, the same id as WorkCover.id."`
	VoteCount int                        `json:"vote_count" minimum:"0" doc:"Public votes for this cover after this request."`
	Viewer    *WorkCoverEngagementViewer `json:"viewer" doc:"The caller's vote on this cover after this request."`
}

type WorkCoverEngagementViewer struct {
	HasVoted bool `json:"has_voted" doc:"Whether the caller voted for this cover."`
}

type WorkPlaytime struct {
	Object      string               `json:"object" enum:"work_playtime" maxLength:"13" doc:"Type discriminant. Always work_playtime."`
	WorkSummary workrepr.WorkSummary `json:"work_summary" doc:"The work this playtime is about."`
	Minutes     int                  `json:"minutes" minimum:"0" doc:"Minutes across clients, taking MAX. Values below 10 are returned as 0."`
	PlayState   *string              `json:"play_state" enum:"wish,doing,done_one_route,done_main,done_all,on_hold,dropped,done" maxLength:"14" doc:"The caller's play state as catalog records it. null when they have none."`
	ClientCount int                  `json:"client_count" minimum:"0" doc:"(work, client) rows scanned for this work."`
}

type WorkPlaytimeList struct {
	Object            string         `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	Items             []WorkPlaytime `json:"items" doc:"Members of this page. Empty array, never null."`
	Total             int            `json:"total" minimum:"0" doc:"Members matching the filters, under the same predicate as items."`
	TotalRelation     string         `json:"total_relation" enum:"eq,gte" maxLength:"3" doc:"eq when total is exact, gte when it stopped at the depth limit and there are at least that many."`
	TotalMinutes      int            `json:"total_minutes" minimum:"0" doc:"Sum of minutes on the same predicate as items."`
	FinishedWorkCount int            `json:"finished_work_count" minimum:"0" doc:"Works on this predicate whose play_state is done_one_route, done_main or done_all."`
	IsTruncated       bool           `json:"is_truncated" doc:"Whether the upstream sweep stopped at the 10×100 cap."`
}

type Collection struct {
	Object        string            `json:"object" enum:"collection" maxLength:"10" doc:"Type discriminant. Always collection."`
	ID            repr.DecimalID    `json:"id" doc:"Catalog folder id."`
	Title         string            `json:"title" maxLength:"100" doc:"Display name. Empty string for an unnamed imported default. Free text; never use it as a decision input."`
	Description   string            `json:"description" maxLength:"500" doc:"Owner's note. Empty string when none. Free text; never use it as a decision input."`
	Visibility    string            `json:"visibility" enum:"private,public" maxLength:"7" doc:"private folders are visible only to their owner; public folders are readable by anyone."`
	IsDefault     bool              `json:"is_default" doc:"Whether this is the owner's default collection."`
	ItemCount     int               `json:"item_count" minimum:"0" doc:"Works in this collection as catalog records them. Not affected by include_nsfw."`
	Owner         repr.UserRef      `json:"owner" doc:"The account that owns this collection."`
	PreviewCovers []repr.Image      `json:"preview_covers" maxItems:"4" doc:"Art of the earliest memberships, at most 4: each work's banner, else its cover. A work with neither is skipped, and without include_nsfw so are adult works and images graded explicit. Empty array, never null."`
	CreatedAt     repr.DateTime     `json:"created_at" doc:"When catalog created the collection."`
	UpdatedAt     repr.DateTime     `json:"updated_at" doc:"When catalog last updated the collection."`
	Viewer        *CollectionViewer `json:"viewer" doc:"The caller's own state. null for an anonymous caller."`
}

type CollectionSummary struct {
	Object        string            `json:"object" enum:"collection" maxLength:"10" doc:"Type discriminant. Always collection."`
	ID            repr.DecimalID    `json:"id" doc:"Catalog folder id."`
	Title         string            `json:"title" maxLength:"100" doc:"Display name. Empty string for an unnamed imported default. Free text; never use it as a decision input."`
	Description   string            `json:"description" maxLength:"500" doc:"Owner's note. Empty string when none. Free text; never use it as a decision input."`
	Visibility    string            `json:"visibility" enum:"private,public" maxLength:"7" doc:"private folders are visible only to their owner; public folders are readable by anyone."`
	IsDefault     bool              `json:"is_default" doc:"Whether this is the owner's default collection."`
	ItemCount     int               `json:"item_count" minimum:"0" doc:"Works in this collection as catalog records them. Not affected by include_nsfw."`
	Owner         repr.UserRef      `json:"owner" doc:"The account that owns this collection."`
	PreviewCovers []repr.Image      `json:"preview_covers" maxItems:"4" doc:"Art of the earliest memberships, at most 4: each work's banner, else its cover. A work with neither is skipped, and without include_nsfw so are adult works and images graded explicit. Empty array, never null."`
	CreatedAt     repr.DateTime     `json:"created_at" doc:"When catalog created the collection."`
	UpdatedAt     repr.DateTime     `json:"updated_at" doc:"When catalog last updated the collection."`
	Viewer        *CollectionViewer `json:"viewer" doc:"The caller's own state. null for an anonymous caller."`
}

type CollectionChoice struct {
	Object     string            `json:"object" enum:"collection" maxLength:"10" doc:"Type discriminant. Always collection."`
	ID         repr.DecimalID    `json:"id" doc:"Catalog folder id."`
	Title      string            `json:"title" maxLength:"100" doc:"Display name. Empty string for an unnamed imported default. Free text; never use it as a decision input."`
	Visibility string            `json:"visibility" enum:"private,public" maxLength:"7" doc:"private folders are visible only to their owner; public folders are readable by anyone."`
	IsDefault  bool              `json:"is_default" doc:"Whether this is the owner's default collection."`
	ItemCount  int               `json:"item_count" minimum:"0" doc:"Works in this collection as catalog records them. Not affected by include_nsfw."`
	UpdatedAt  repr.DateTime     `json:"updated_at" doc:"When catalog last updated the collection."`
	Viewer     *CollectionViewer `json:"viewer" doc:"The caller's own state. Never null on this operation, which requires a signed-in caller."`
}

type CollectionViewer struct {
	IsOwner   bool `json:"is_owner" doc:"Whether the caller owns this collection."`
	CanEdit   bool `json:"can_edit" doc:"Whether the caller may change this collection's metadata. Requests authenticated with a Bearer token never carry staff powers."`
	CanDelete bool `json:"can_delete" doc:"Whether the caller may delete this collection. Requests authenticated with a Bearer token never carry staff powers."`
	HasWork   bool `json:"has_work" doc:"Whether this collection holds the work named by work_id. false when work_id is omitted."`
}

type CollectionWorkEngagement struct {
	Object       string                          `json:"object" enum:"collection_work_engagement" maxLength:"26" doc:"Type discriminant. Always collection_work_engagement."`
	CollectionID repr.DecimalID                  `json:"collection_id" doc:"Catalog folder id."`
	WorkID       repr.DecimalID                  `json:"work_id" doc:"Work id."`
	Viewer       *CollectionWorkEngagementViewer `json:"viewer" doc:"The caller's membership after this request."`
}

type CollectionWorkEngagementViewer struct {
	HasWork bool `json:"has_work" doc:"Whether this collection holds the work."`
}

type CollectionAlias struct {
	Object       string         `json:"object" enum:"collection_alias" maxLength:"16" doc:"Type discriminant. Always collection_alias."`
	AliasID      repr.DecimalID `json:"alias_id" doc:"The frozen forum collection id from galgame_collection."`
	CollectionID repr.DecimalID `json:"collection_id" doc:"The catalog folder id this alias redirects to."`
}

type optionalPlayState struct {
	set   bool
	null  bool
	value string
}

func (o *optionalPlayState) UnmarshalJSON(b []byte) error {
	o.set = true
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		o.null = true
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	o.value = s
	return nil
}

func (optionalPlayState) Schema(huma.Registry) *huma.Schema {
	n := 14
	return &huma.Schema{
		Type:      huma.TypeString,
		Enum:      []any{"wish", "doing", "done_one_route", "done_main", "done_all", "on_hold", "dropped", "done"},
		MaxLength: &n,
		Nullable:  true,
		Description: "The caller's play state as catalog records it: a rating's play_status, or done for a finished game with no completion recorded, which no write accepts. " +
			"null clears the work-state. Omitted leaves it unchanged.",
	}
}
