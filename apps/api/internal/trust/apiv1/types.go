package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

type TrustSubjectKind string

func (TrustSubjectKind) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("trust_subject_kind", 64)
	s.Pattern = `^[a-z][a-z0-9_]{0,63}$`
	s.Description = "Trust-and-safety subject kind as registered with the trust service, such as forum_topic or galgame. " +
		"The vocabulary grows whenever a new kind of content becomes reportable; show an unknown kind with a localized fallback."
	return s
}

type ReportReasonKey string

func (ReportReasonKey) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("report_reason", 64)
	s.Pattern = `^[a-z][a-z0-9_]{0,63}$`
	s.Description = "Report reason key. The trust service's administrators add and retire reasons, so the vocabulary is data: " +
		"list the current ones with listReportReasons."
	return s
}

type ReviewItemOrigin string

func (ReviewItemOrigin) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("review_item_origin", 32)
	s.Pattern = `^[a-z][a-z0-9_]{0,31}$`
	s.Description = "What opened the review item: reports, ai_text, ai_image, community_forward, mislabel, manual or ai_sample. " +
		"unknown when the trust service reports an origin this forum does not know yet."
	return s
}

type ReportReason struct {
	Object      string          `json:"object" enum:"report_reason" maxLength:"13" doc:"Type discriminant. Always report_reason."`
	Key         ReportReasonKey `json:"key" doc:"The reason key a report is filed under."`
	DisplayName string          `json:"display_name" maxLength:"64" doc:"The label the trust service registered for this reason, in its original language. Free text; never use it as a decision input."`
}

type ReportCreate struct {
	SubjectKind TrustSubjectKind `json:"subject_kind" doc:"Kind of the reported content. One of the kinds this forum registers with the trust service."`
	SubjectID   repr.DecimalID   `json:"subject_id" doc:"Id of the reported content within its kind."`
	ReasonKey   ReportReasonKey  `json:"reason_key" doc:"One of the keys listReportReasons returns."`
	Note        *string          `json:"note" required:"false" maxLength:"1000" doc:"What is wrong with the content, for the moderators. Absent or null for none. Free text; never use it as a decision input."`
	Snapshot    *string          `json:"snapshot" required:"false" maxLength:"2000" doc:"The reporter's copy of the content as they saw it. Absent or null for none. Free text; never use it as a decision input."`
	SubjectURL  *string          `json:"subject_url" required:"false" format:"uri" maxLength:"512" doc:"Link to the content on this forum. It must start with https://www.kungal.com/. Absent or null for none."`
}

type ReviewItemSummary struct {
	Object          string           `json:"object" enum:"review_item" maxLength:"11" doc:"Type discriminant. Always review_item."`
	ID              repr.DecimalID   `json:"id" doc:"Review item id."`
	SubjectKind     TrustSubjectKind `json:"subject_kind" doc:"Kind of the content under review."`
	SubjectID       repr.DecimalID   `json:"subject_id" doc:"Id of the content under review within its kind."`
	OpenedBy        ReviewItemOrigin `json:"opened_by" doc:"What opened the item."`
	State           string           `json:"state" enum:"pending,claimed,actioned,dismissed" maxLength:"9" doc:"pending until a moderator claims it; actioned or dismissed once decided."`
	Priority        float32          `json:"priority" minimum:"0" doc:"Queue rank; the inbox lists the highest first."`
	Severity        *int             `json:"severity" minimum:"0" doc:"Severity of the reason that opened the item. null when the item was not opened by reports."`
	ClassifierScore *float32         `json:"classifier_score" minimum:"0" maximum:"1" doc:"The AI classifier's score. null when no classifier judged the content."`
	ReportWeightSum *float32         `json:"report_weight_sum" minimum:"0" doc:"Summed weight of the reports behind the item. null when reports did not open it."`
	ReachCount      *int64           `json:"reach_count" minimum:"0" doc:"How many people the content had reached when the item opened. null when the product does not report it."`
	ContextNote     *string          `json:"context_note" maxLength:"4000" doc:"Evidence for an item that reports did not open, such as the flagged excerpt. Free text; never use it as a decision input."`
	Claimant        *repr.UserRef    `json:"claimant" doc:"The moderator who claimed the item. null before anyone has."`
	ClaimedAt       *repr.DateTime   `json:"claimed_at" doc:"When the item was claimed. null before anyone has."`
	Decider         *repr.UserRef    `json:"decider" doc:"The moderator who decided the item. null while it is open."`
	DecidedAt       *repr.DateTime   `json:"decided_at" doc:"When the item was decided. null while it is open."`
	CreatedAt       repr.DateTime    `json:"created_at" doc:"When the item opened."`
}

type ReviewItem struct {
	ReviewItemSummary
	Reports []ReviewReport `json:"reports" maxItems:"1000" doc:"The reports linked to the item, oldest first. Empty for an item that reports did not open."`
}

type ReviewReport struct {
	Object       string         `json:"object" enum:"report" maxLength:"6" doc:"Type discriminant. Always report."`
	ID           repr.DecimalID `json:"id" doc:"Report id."`
	Reporter     repr.UserRef   `json:"reporter" doc:"Who filed the report."`
	ReportReason *ReportReason  `json:"report_reason" doc:"The reason it was filed under. null when that reason has since been retired."`
	Note         *string        `json:"note" maxLength:"1000" doc:"The reporter's note. Free text; never use it as a decision input."`
	Snapshot     *string        `json:"snapshot" maxLength:"2000" doc:"The reporter's copy of the content. Free text; never use it as a decision input."`
	SubjectURL   *string        `json:"subject_url" format:"uri" maxLength:"512" doc:"The reporter's link to the content. null unless it points at this forum."`
	Weight       float32        `json:"weight" minimum:"0" doc:"The weight the trust service gave this reporter."`
	CreatedAt    repr.DateTime  `json:"created_at" doc:"When the report was filed."`
}

type ReviewItemPatch struct {
	State      string  `json:"state" enum:"claimed,actioned,dismissed" maxLength:"9" doc:"claimed takes a pending item; actioned or dismissed decides a pending or claimed one."`
	Action     *string `json:"action,omitempty" enum:"none,hide,remove,warn_user,restrict,escalate_idp" maxLength:"12" doc:"What to do to the content. Required with actioned and not allowed otherwise."`
	ReasonCode *string `json:"reason_code,omitempty" pattern:"^[a-z][a-z0-9_]{0,63}$" maxLength:"64" doc:"Why, as a code; a report reason key where one fits. Required with actioned and not allowed otherwise."`
	Statement  *string `json:"statement,omitempty" maxLength:"1000" doc:"A statement of reasons for the author. Only with actioned. Free text; never use it as a decision input."`
}
