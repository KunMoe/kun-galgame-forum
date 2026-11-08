package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/entityapiv1"
	"kun-galgame-api/internal/galgame/workrepr"
)

type listCollectionsInput struct {
	Sort           string `query:"sort" enum:"followers_desc,updated_desc,created_desc" default:"followers_desc" maxLength:"14" doc:"followers: how many accounts follow the collection. updated: when its text or works last changed. created: when it was made."`
	IncludeNSFW    bool   `query:"include_nsfw" default:"false" doc:"When true, collections holding an adult work are listed and their preview_covers may show adult works. Default false: such collections are left out."`
	ContainsWorkID string `query:"contains_work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Only collections that hold this work."`
	Tag            string `query:"tag" maxLength:"16" doc:"Only collections carrying this tag, spelled as listCollectionTags returns it. Free text; never use it as a decision input."`
	Q              string `query:"q" maxLength:"50" doc:"Only collections whose title, description or a tag contains this text, ignoring case. The sort is kept. Free text; never use it as a decision input."`
	Cursor         string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor from a previous page of this collection. Bound to sort, include_nsfw and contains_work_id."`
	Limit          int    `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
}

type listFollowedCollectionsInput struct {
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, adult works may appear in preview_covers. Default false. Collections are never left out by it."`
	Cursor      string `query:"cursor" pattern:"^cur_[A-Za-z0-9_-]+$" maxLength:"512" doc:"Opaque keyset cursor from a previous page of this collection."`
	Limit       int    `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
}

type collectionSummaryListOutput struct {
	Body repr.List[CollectionSummary]
}

type collectionFollowInput struct {
	CollectionID string `path:"collection_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog folder id."`
}

type CollectionFollowEngagement struct {
	Object        string                            `json:"object" enum:"collection_follow_engagement" maxLength:"28" doc:"Type discriminant. Always collection_follow_engagement."`
	CollectionID  repr.DecimalID                    `json:"collection_id" doc:"Catalog folder id."`
	FollowerCount *int                              `json:"follower_count" minimum:"0" doc:"Accounts that follow the collection after this request. null when the collection is no longer public and the count is not published."`
	Viewer        *CollectionFollowEngagementViewer `json:"viewer" doc:"The caller's follow after this request."`
}

type CollectionFollowEngagementViewer struct {
	IsFollowing bool `json:"is_following" doc:"Whether the caller follows this collection."`
}

type collectionFollowOutput struct {
	Body CollectionFollowEngagement
}

type listCollectionEntriesInput struct {
	CollectionID string `path:"collection_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog folder id."`
	Page         int    `query:"page" minimum:"1" default:"1" doc:"1-based page number. page × limit may not exceed 10000."`
	Limit        int    `query:"limit" minimum:"1" maximum:"100" default:"24" doc:"Page size. 1–100, default 24. Values above 100 are rejected, not clamped."`
	IncludeNSFW  bool   `query:"include_nsfw" default:"false" doc:"When true, adult works are included. Default false."`
	Sort         string `query:"sort" enum:"position_asc,favorited_desc,favorited_asc,released_desc,released_asc" default:"position_asc" maxLength:"14" doc:"position: the owner's own order. favorited: when the work was put in this collection. released: the work's release date; works without one come last either way."`
}

type CollectionEntry struct {
	Object              string                      `json:"object" enum:"collection_entry" maxLength:"16" doc:"Type discriminant. Always collection_entry."`
	WorkSummary         workrepr.WorkSummary        `json:"work_summary" doc:"The work."`
	Note                *string                     `json:"note" maxLength:"500" doc:"The owner's note on this work in this collection. null when none. Free text; never use it as a decision input."`
	AddedAt             repr.DateTime               `json:"added_at" doc:"When the work was put in this collection."`
	TopTags             []workrepr.TagRef           `json:"top_tags" maxItems:"6" doc:"The work's first tags as its page lists them, without spoilers, and without adult tags unless include_nsfw. Empty array, never null."`
	ScenarioCredits     []entityapiv1.CreditNameRef `json:"scenario_credits" maxItems:"3" doc:"Credited scenario writers, at most 3, each linking to /galgame/staff/{id}. Empty array, never null."`
	IllustrationCredits []entityapiv1.CreditNameRef `json:"illustration_credits" maxItems:"3" doc:"Credited illustrators, else character designers, at most 3. Empty array, never null."`
}

type collectionEntryPageOutput struct {
	Body repr.PageList[CollectionEntry]
}

type patchCollectionWorkInput struct {
	CollectionID string `path:"collection_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog folder id."`
	WorkID       string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	Body         collectionWorkPatch
}

type collectionWorkPatch struct {
	Note         *string `json:"note" required:"false" maxLength:"500" doc:"New note. Absent or null leaves it; an empty string clears it. Free text; never use it as a decision input."`
	AfterWorkID  *string `json:"after_work_id,omitempty" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Move the work to directly after this one in the owner's order. Not with before_work_id."`
	BeforeWorkID *string `json:"before_work_id,omitempty" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Move the work to directly before this one in the owner's order. Not with after_work_id."`
}

type CollectionItem struct {
	Object       string         `json:"object" enum:"collection_item" maxLength:"15" doc:"Type discriminant. Always collection_item."`
	CollectionID repr.DecimalID `json:"collection_id" doc:"Catalog folder id."`
	WorkID       repr.DecimalID `json:"work_id" doc:"Work id."`
	Note         *string        `json:"note" maxLength:"500" doc:"The owner's note on this work in this collection. null when none. Free text; never use it as a decision input."`
	AddedAt      repr.DateTime  `json:"added_at" doc:"When the work was put in this collection."`
}

type collectionItemOutput struct {
	Body CollectionItem
}

type createCollectionReportInput struct {
	Body collectionReportCreate
}

type collectionReportCreate struct {
	CollectionID string  `json:"collection_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The public collection being reported."`
	Reason       string  `json:"report_category" enum:"abuse,spam,illegal,rating_mislabel,copyright,other" maxLength:"15" doc:"Why the collection is reported."`
	Note         *string `json:"note" required:"false" maxLength:"500" doc:"The reporter's own words for the moderators. Absent or null for none. Free text; never use it as a decision input."`
	WorkID       *string `json:"work_id,omitempty" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Report the owner's note on this work instead of the collection's title and description. The work must be in the collection and carry a note."`
}
