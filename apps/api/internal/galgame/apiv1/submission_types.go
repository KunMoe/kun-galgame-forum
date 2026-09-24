package apiv1

import (
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/entityapiv1"
	"kun-galgame-api/internal/galgame/workrepr"

	"github.com/danielgtaylor/huma/v2"
)

type WorkSubmission struct {
	Object        string                `json:"object" enum:"work_submission" maxLength:"15" doc:"Type discriminant. Always work_submission."`
	ID            repr.DecimalID        `json:"id" doc:"Catalog work id; the same value as work_id."`
	WorkID        repr.DecimalID        `json:"work_id" doc:"Catalog work id."`
	DisplayName   string                `json:"display_name" maxLength:"512" doc:"Catalog display name. Free text; never use it as a decision input."`
	State         string                `json:"state" enum:"live,draft,pending,declined,hidden" maxLength:"8" doc:"Claim state, read back from catalog."`
	IsNSFW        bool                  `json:"is_nsfw" doc:"Display axis, the same field as Work.is_nsfw."`
	ContentRating string                `json:"content_rating" enum:"all_ages,sensitive,r18" maxLength:"9" doc:"Age axis, the same field as Work.content_rating. Independent of is_nsfw."`
	Submitter     *repr.UserRef         `json:"submitter" doc:"The forum account recorded as the submitter. A deleted account is a deleted user ref. null when none is recorded."`
	LastEvent     *ClaimEventRef        `json:"last_event" doc:"The newest transition of this claim, whoever made it. null when catalog has none."`
	FirstActedAt  *repr.DateTime        `json:"first_acted_at" doc:"When the caller first acted on this claim. null when the caller never has, or catalog did not say."`
	ActedCount    int                   `json:"acted_count" minimum:"0" doc:"How many times the caller acted on this claim."`
	Viewer        *WorkSubmissionViewer `json:"viewer" doc:"The caller's capabilities on this submission."`
}

type WorkSubmissionCreated struct {
	WorkSubmission
	// Reported rather than raised, but reported: silence here hid a broken
	// cover patch for the whole life of the old submit route.
	HasBannerAttached bool `json:"has_banner_attached" doc:"false when a banner_hash was sent but did not become the work's cover; the submission stands either way. true when none was sent."`
}

type WorkSubmissionSummary struct {
	Object       string                `json:"object" enum:"work_submission" maxLength:"15" doc:"Type discriminant. Always work_submission."`
	ID           repr.DecimalID        `json:"id" doc:"Catalog work id; the same value as work_id."`
	WorkID       repr.DecimalID        `json:"work_id" doc:"Catalog work id."`
	DisplayName  string                `json:"display_name" maxLength:"512" doc:"Catalog display name. Free text; never use it as a decision input."`
	State        string                `json:"state" enum:"live,draft,pending,declined,hidden" maxLength:"8" doc:"Claim state."`
	WorkSummary  workrepr.WorkSummary  `json:"work_summary" doc:"The work. When catalog does not render it (a hidden work), only id and display name are filled."`
	LastEvent    *ClaimEventRef        `json:"last_event" doc:"The newest transition of this claim. null on review-queue rows, which catalog serves without it."`
	FirstActedAt *repr.DateTime        `json:"first_acted_at" doc:"When the caller first acted on this claim. null on review-queue rows."`
	Viewer       *WorkSubmissionViewer `json:"viewer" doc:"The caller's capabilities on this submission."`
}

type WorkSubmissionViewer struct {
	CanSubmit   bool `json:"can_submit" doc:"Whether the caller may move this claim to pending: the caller submitted it and it is draft or declined."`
	CanWithdraw bool `json:"can_withdraw" doc:"Whether the caller may move this claim back to draft: the caller submitted it and it is pending or live."`
	CanDelete   bool `json:"can_delete" doc:"Whether the caller may delete this claim: the caller submitted it and it is draft."`
	CanReview   bool `json:"can_review" doc:"Whether the caller may set live, declined, hidden or unban. Requests authenticated with a Bearer token never carry this."`
}

type ClaimEventRef struct {
	Object    string         `json:"object" enum:"claim_event" maxLength:"11" doc:"Type discriminant. Always claim_event."`
	ID        repr.DecimalID `json:"id" doc:"Claim event id."`
	FromState *string        `json:"from_state" enum:"live,draft,pending,declined,hidden" maxLength:"8" doc:"State before the event. null on the event that created the claim."`
	ToState   string         `json:"to_state" enum:"live,draft,pending,declined,hidden" maxLength:"8" doc:"State after the event."`
	Note      *string        `json:"note" maxLength:"2000" doc:"The note given with the transition, such as a decline reason. null when none was given. Free text; never use it as a decision input."`
	Actor     repr.UserRef   `json:"actor" doc:"Who made the transition: the submitter, or a reviewer on a decision."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"When the transition happened."`
}

type WorkSubmissionCandidate struct {
	Object      string               `json:"object" enum:"work_submission_candidate" maxLength:"25" doc:"Type discriminant. Always work_submission_candidate."`
	WorkSummary workrepr.WorkSummary `json:"work_summary" doc:"A catalog work matching the query."`
	State       string               `json:"state" enum:"none,live,draft,pending" maxLength:"7" doc:"none when no site has claimed the work; otherwise this forum's claim state."`
}

type SubmissionTitle struct {
	Locale string `json:"locale" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"BCP-47 tag of the title."`
	Title  string `json:"title" maxLength:"500" doc:"The official title in this locale. Free text; never use it as a decision input."`
}

type SubmissionIntroduction struct {
	Locale string `json:"locale" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"BCP-47 tag of the introduction: en, ja, zh-Hans or zh-Hant."`
	Value  string `json:"value" maxLength:"50000" doc:"The introduction. Free text; never use it as a decision input."`
}

type workSubmissionCreate struct {
	DisplayName      string                   `json:"display_name" required:"false" maxLength:"500" doc:"Display name. When omitted, the first non-blank title in ja, zh-Hans, zh-Hant, en is used. Free text; never use it as a decision input."`
	Titles           []SubmissionTitle        `json:"titles" minItems:"1" maxItems:"100" doc:"Official titles. Titles and aliases together hold at most 100."`
	Aliases          []entityapiv1.AliasName  `json:"aliases" required:"false" maxItems:"100" doc:"Alias titles, counted against the same 100. Blank entries are ignored. Free text; never use it as a decision input."`
	Introductions    []SubmissionIntroduction `json:"introductions" required:"false" maxItems:"4" doc:"At most one introduction per language. Blank ones are ignored."`
	OriginalLanguage *string                  `json:"original_language" maxLength:"35" pattern:"^[A-Za-z]{1,8}(-[A-Za-z0-9]{1,8})*$" doc:"Original language as a BCP-47 tag from catalog's closed set. null is refused."`
	ContentRating    string                   `json:"content_rating" enum:"all_ages,sensitive,r18" maxLength:"9" doc:"Age rating. Never derived from is_nsfw."`
	// Required, never defaulted: a wizard default nobody looked at left 961
	// works on catalog's SFW shelf whose only cover art the grader had marked
	// explicit, so every reader got the blurred stand-in (infra
	// audit-cover-shelf, 2026-09-13).
	IsNSFW               bool    `json:"is_nsfw" doc:"Whether the forum should display the work as adult content. Must be sent: an omitted value is not false."`
	ReleaseDate          *string `json:"release_date" required:"false" format:"date" maxLength:"10" doc:"Release date. Omitted or null is TBA. A month- or year-precise date is written as that month or year."`
	ReleaseDatePrecision *string `json:"release_date_precision" required:"false" enum:"day,month,year" maxLength:"5" doc:"How much of release_date is known. Default day when release_date is sent; refused without it."`
	BannerHash           string  `json:"banner_hash" required:"false" pattern:"^[0-9a-f]{64}$" minLength:"64" maxLength:"64" doc:"Image-service hash of a banner uploaded for this work."`
	IsDuplicateConfirmed bool    `json:"is_duplicate_confirmed" required:"false" doc:"Mint even though live works share a submitted title. Send true only after the submitter saw DUPLICATE_SUSPECTS and confirmed. Default false."`
}

type workSubmissionPatch struct {
	State string  `json:"state" enum:"pending,draft,live,declined,hidden,unban" maxLength:"8" doc:"Target. The submitter sends pending or draft; a reviewer sends live, declined, hidden or unban. unban restores the state the claim was hidden from, which the response reports."`
	Note  *string `json:"note" required:"false" maxLength:"2000" doc:"Required, and not blank, with declined. Free text; never use it as a decision input."`
}

type workSubmissionIDInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog work id."`
}

type workSubmissionWriteInput struct {
	WorkID  string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog work id."`
	IfMatch string `header:"If-Match" maxLength:"128" pattern:"^(\\*|\"[!#-~]{1,126}\")$" doc:"The ETag of the claim this write is based on, from GET /work-submissions/{work_id} or the previous write. Absent means *: no version check."`
}

type updateWorkSubmissionInput struct {
	WorkID  string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog work id."`
	IfMatch string `header:"If-Match" maxLength:"128" pattern:"^(\\*|\"[!#-~]{1,126}\")$" doc:"The ETag of the claim this write is based on, from GET /work-submissions/{work_id} or the previous write. Absent means *: no version check."`
	Body    workSubmissionPatch
}

type createWorkSubmissionInput struct {
	Body workSubmissionCreate
}

type workSubmissionOutput struct {
	ETag string `header:"ETag" maxLength:"128" doc:"The claim's version. Send it back as If-Match to write this claim."`
	Body WorkSubmission
}

type createWorkSubmissionOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new submission, /api/v1/work-submissions/{work_id}."`
	ETag     string `header:"ETag" maxLength:"128" doc:"The claim's version. Send it back as If-Match to write this claim."`
	Body     WorkSubmissionCreated
}

type ClaimStateFilter string

func (ClaimStateFilter) Schema(huma.Registry) *huma.Schema {
	s := repr.ClosedEnum("live", "draft", "pending", "declined", "hidden")
	s.Description = "A claim state."
	return s
}

type listWorkSubmissionsInput struct {
	collect.Page
	collect.Total
	State []ClaimStateFilter `query:"state" maxItems:"5" doc:"Only claims in these states, comma-separated."`
}

type workSubmissionListOutput struct {
	Body repr.CountedList[WorkSubmissionSummary]
}

type listWorkSubmissionCandidatesInput struct {
	collect.Page
	Q           string `query:"q" maxLength:"200" doc:"Search text in any language. Required. Free text; never use it as a decision input."`
	IncludeNSFW bool   `query:"include_nsfw" default:"false" doc:"When true, works the forum displays as adult content are included. Default false."`
}

type workSubmissionCandidateListOutput struct {
	Body repr.List[WorkSubmissionCandidate]
}
