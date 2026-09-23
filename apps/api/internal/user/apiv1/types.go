package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

type Me struct {
	Object                  string         `json:"object" enum:"me" maxLength:"2" doc:"Type discriminant. Always me."`
	ID                      repr.DecimalID `json:"id" doc:"The caller's user id. JSON string of a decimal integer."`
	Moemoepoint             int            `json:"moemoepoint" minimum:"-2147483648" doc:"The caller's moemoepoint balance as this forum last cached it from OAuth. It can lag the live balance. It can be negative."`
	HasCheckedInToday       bool           `json:"has_checked_in_today" doc:"Whether the caller has already checked in on the current Asia/Shanghai calendar day."`
	HasUnreadMessages       bool           `json:"has_unread_messages" doc:"Whether the caller has an unread notification of a type they have not muted, or an unread private message while private messages are not muted."`
	IsCreator               bool           `json:"is_creator" doc:"Whether the caller holds the creator role, including via a site role."`
	ToolsetUploadTodayBytes int64          `json:"toolset_upload_today_bytes" minimum:"0" doc:"Bytes of toolset uploads counted against the caller today."`
}

type CheckIn struct {
	Object             string            `json:"object" enum:"check_in" maxLength:"8" doc:"Type discriminant. Always check_in."`
	CheckInDate        repr.CalendarDate `json:"check_in_date" doc:"Asia/Shanghai calendar day this check-in is for."`
	MoemoepointAwarded int               `json:"moemoepoint_awarded" minimum:"0" maximum:"7" doc:"Moemoepoint credited for this check-in. Zero when the determined reward is zero."`
	Moemoepoint        int               `json:"moemoepoint" minimum:"-2147483648" doc:"The caller's moemoepoint balance as this forum last cached it after the check-in. It can be negative."`
}

type MoemoepointReason string

func (MoemoepointReason) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("moemoepoint_reason", 40)
	s.Pattern = `^[a-z][a-z0-9_]{0,39}$`
	s.Description = "Moemoepoint ledger reason token. The vocabulary grows; show an unknown token with a localized fallback."
	return s
}

type MoemoepointEntry struct {
	Object         string            `json:"object" enum:"moemoepoint_entry" maxLength:"17" doc:"Type discriminant. Always moemoepoint_entry."`
	ID             repr.DecimalID    `json:"id" doc:"Ledger entry id. JSON string of a decimal integer."`
	Delta          int               `json:"delta" minimum:"-2147483648" doc:"Signed change. Negative when moemoepoint was deducted."`
	Reason         MoemoepointReason `json:"reason" doc:"Ledger reason token."`
	Ref            string            `json:"ref" maxLength:"80" doc:"Triggering entity reference as stored upstream. Empty string when none. Free text; never use it as a decision input."`
	CreatedAt      repr.DateTime     `json:"created_at" doc:"When the ledger entry was written."`
	Source         string            `json:"source" enum:"this_site,account_center,other_site" maxLength:"14" doc:"Who issued the entry: this_site, account_center (the OAuth account service itself, e.g. a rename charge or an admin adjustment), or other_site. Closed."`
}

type Preferences struct {
	Object    string         `json:"object" enum:"preferences" maxLength:"11" doc:"Type discriminant. Always preferences."`
	Doc       map[string]any `json:"doc" doc:"Cloud preference document. A JSON object."`
	Version   int            `json:"version" minimum:"0" doc:"Document version. Zero when the namespace has never been written."`
	WrittenAt *repr.DateTime `json:"written_at" doc:"When the document was last written. null when it has never been written."`
}

type NsfwDisplay struct {
	Object      string `json:"object" enum:"nsfw_display" maxLength:"12" doc:"Type discriminant. Always nsfw_display."`
	NsfwDisplay string `json:"nsfw_display" enum:"hide,blur,show" maxLength:"4" doc:"How adult content is shown: hide, blur, or show."`
}

type MyProfile struct {
	Object string         `json:"object" enum:"user" maxLength:"4" doc:"Type discriminant. Always user."`
	ID     repr.DecimalID `json:"id" doc:"The caller's user id. JSON string of a decimal integer."`
	Name   *string        `json:"name" maxLength:"64" doc:"Display name. null when the account no longer exists; show a localized label. Free text; never use it as a decision input."`
	Avatar *repr.Image    `json:"avatar" doc:"Avatar image. null when the account has no image-service hash."`
	Bio    *string        `json:"bio" maxLength:"107" doc:"Profile bio as stored. Empty string when none. Free text; never use it as a decision input."`
}

type CreatorEligibility struct {
	IsEligible                    bool `json:"is_eligible" doc:"Whether the caller currently meets at least one creator-application threshold."`
	MergedPrCount                 int  `json:"merged_pr_count" minimum:"0" doc:"Merged catalog edit proposals attributed to the caller."`
	PublishedGalgameCount         int  `json:"published_galgame_count" minimum:"0" doc:"Galgames the caller owns that are published."`
	LongReviewCount               int  `json:"long_review_count" minimum:"0" doc:"Galgame ratings by the caller whose short summary is at least 100 characters."`
	Moemoepoint                   int  `json:"moemoepoint" minimum:"-2147483648" doc:"Live OAuth moemoepoint balance used for eligibility. It can be negative."`
	RequiredMergedPrCount         int  `json:"required_merged_pr_count" minimum:"0" doc:"Merged-proposal count that by itself makes the caller eligible."`
	RequiredPublishedGalgameCount int  `json:"required_published_galgame_count" minimum:"0" doc:"Published-galgame count that by itself makes the caller eligible."`
	RequiredLongReviewCount       int  `json:"required_long_review_count" minimum:"0" doc:"Long-review count that by itself makes the caller eligible."`
	RequiredMoemoepoint           int  `json:"required_moemoepoint" minimum:"0" doc:"Moemoepoint balance that by itself makes the caller eligible."`
}

type CreatorApplication struct {
	Object        string         `json:"object" enum:"creator_application" maxLength:"20" doc:"Type discriminant. Always creator_application."`
	ID            repr.DecimalID `json:"id" doc:"Application id. JSON string of a decimal integer."`
	State         string         `json:"state" enum:"pending,approved,declined" maxLength:"8" doc:"Application lifecycle state."`
	Statement     string         `json:"statement" maxLength:"1000" doc:"Applicant statement as stored. Empty string when none. Free text; never use it as a decision input."`
	DeclineReason *string        `json:"decline_reason" maxLength:"500" doc:"Reason the application was declined. null when it was not declined. Free text; never use it as a decision input."`
	CreatedAt     repr.DateTime  `json:"created_at" doc:"When the application was submitted."`
	ReviewedAt    *repr.DateTime `json:"reviewed_at" doc:"When the application was reviewed. null while it is pending."`
}

type CreatorStatus struct {
	Object      string              `json:"object" enum:"creator_status" maxLength:"14" doc:"Type discriminant. Always creator_status."`
	IsCreator   bool                `json:"is_creator" doc:"Whether the caller already holds the creator role."`
	Eligibility CreatorEligibility  `json:"eligibility" doc:"Current eligibility snapshot."`
	Application *CreatorApplication `json:"application" doc:"The caller's latest creator application. null when they have never applied."`
}
