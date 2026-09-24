package apiv1

import (
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"

	"github.com/danielgtaylor/huma/v2"
)

const editFieldKeyPattern = `^[a-z0-9_]+(\.[a-z0-9_]+)+$`

// EditFieldKey is an editing-engine field key inside an array, where a struct
// tag cannot reach the element schema.
type EditFieldKey string

func (EditFieldKey) Schema(huma.Registry) *huma.Schema {
	n := 128
	return &huma.Schema{
		Type: huma.TypeString, MaxLength: &n, Pattern: editFieldKeyPattern,
		Description: "Editing-engine field key, such as catalog.work.titles.",
	}
}

type EditForm struct {
	Object       string           `json:"object" enum:"edit_form" maxLength:"9" doc:"Type discriminant. Always edit_form."`
	WorkID       repr.DecimalID   `json:"work_id" doc:"Work id."`
	FieldValues  map[string]any   `json:"field_values" doc:"Current value of every editable field, keyed by catalog.work.*. Empty object, never null."`
	Fields       []EditField      `json:"fields" doc:"Editable fields. Empty array, never null. The same for every caller."`
	Vocabularies []EditVocabulary `json:"vocabularies" doc:"Closed vocabularies the fields name. Empty array when catalog could not be read; render those fields read-only."`
}

type EditField struct {
	Key           string            `json:"key" maxLength:"128" pattern:"^[a-z0-9_]+(\\.[a-z0-9_]+)+$" doc:"Editing-engine field key, such as catalog.work.titles."`
	FieldType     string            `json:"field_type" enum:"text,i18nmap,enum,int,date,list,ref,imagehash" maxLength:"9" doc:"Control type. Not a domain vocabulary."`
	DiffHint      string            `json:"diff_hint" enum:"inline,lines,items,image" maxLength:"6" doc:"How a diff of this field should be rendered."`
	IsDeprecated  bool              `json:"is_deprecated" doc:"Whether catalog rejects writes of this key."`
	MaxElements   int               `json:"max_elements" minimum:"0" doc:"Cap on a list field's element count. 0 for scalar fields."`
	MaxSuppressed int               `json:"max_suppressed" minimum:"0" doc:"Cap on this field's suppression set. 0 when the field has none."`
	Vocabulary    string            `json:"vocabulary" maxLength:"64" pattern:"^[a-z_]*$" doc:"Name of the vocabulary whose tokens this field accepts. Empty when it has none."`
	Encoding      *string           `json:"encoding" enum:"token,int" maxLength:"5" doc:"How a write carries the vocabulary value: token is the token itself, int is base plus the token's index in the vocabulary's order. null when vocabulary is empty."`
	Base          int               `json:"base" minimum:"0" doc:"Wire code of the vocabulary's first token on an int-encoded field."`
	IsNullable    bool              `json:"is_nullable" doc:"Whether null is accepted and clears the stored value."`
	Element       *EditFieldElement `json:"element" doc:"Shape of one list element. null on scalar fields and on lists whose shape catalog does not declare."`
}

type EditFieldElement struct {
	ElementType string                   `json:"element_type" enum:"object,text,ref" maxLength:"6" doc:"text elements are strings, ref elements are decimal ids of another entity, object elements have members."`
	Members     []EditFieldElementMember `json:"members" doc:"Members of an object element, in validation order. Empty array for scalar elements, never null."`
}

type EditFieldElementMember struct {
	Key        string `json:"key" maxLength:"64" pattern:"^[a-z0-9_]+$" doc:"Member key inside the element object."`
	MemberType string `json:"member_type" enum:"text,int,enum,bool,ref,imagehash" maxLength:"9" doc:"Member value type. ref is the decimal id of another entity."`
	Vocabulary string `json:"vocabulary" maxLength:"64" pattern:"^[a-z_]*$" doc:"Name of the vocabulary whose tokens this member accepts. Empty when it has none."`
	Base       int    `json:"base" minimum:"0" doc:"Wire code of the vocabulary's first token on an int member."`
	IsNullable bool   `json:"is_nullable" doc:"Whether the member may be absent or empty in at least one valid element."`
}

type EditVocabulary struct {
	Object     string                `json:"object" enum:"vocabulary" maxLength:"10" doc:"Type discriminant. Always vocabulary."`
	Vocabulary string                `json:"vocabulary" maxLength:"64" pattern:"^[a-z_]*$" doc:"The name fields use to point at this vocabulary."`
	IsClosed   bool                  `json:"is_closed" doc:"Whether a value outside values is refused."`
	Values     []EditVocabularyValue `json:"values" doc:"Tokens in the vocabulary's published order. Empty array, never null."`
}

type EditVocabularyValue struct {
	Value       string `json:"value" maxLength:"128" pattern:"^\\S+$" doc:"Token."`
	DisplayName string `json:"display_name" maxLength:"512" doc:"Label. Empty when catalog gives none. Free text; never use it as a decision input."`
	Description string `json:"description" maxLength:"2048" doc:"Description. Empty when catalog gives none. Free text; never use it as a decision input."`
}

type EditProposal struct {
	Object          string               `json:"object" enum:"edit_proposal" maxLength:"13" doc:"Type discriminant. Always edit_proposal."`
	ID              repr.DecimalID       `json:"id" doc:"Proposal id."`
	State           string               `json:"state" enum:"open,merged,declined,withdrawn" maxLength:"9" doc:"Lifecycle state, read back from catalog."`
	WorkID          repr.DecimalID       `json:"work_id" doc:"The work this proposal edits."`
	WorkSummary     workrepr.WorkSummary `json:"work_summary" doc:"The work. display_name is empty when catalog did not render it."`
	Note            *string              `json:"note" maxLength:"2000" doc:"The proposer's summary. null when none. Free text; never use it as a decision input."`
	Proposer        repr.UserRef         `json:"proposer" doc:"Who filed it. A deleted or banned account is a deleted user ref."`
	Decider         *repr.UserRef        `json:"decider" doc:"Who merged or declined it. null while open, and when withdrawn."`
	BaseRevisionSeq int                  `json:"base_revision_seq" minimum:"0" doc:"The revision seq the proposal was written against."`
	Patch           map[string]any       `json:"patch" doc:"Proposed values keyed by catalog.work.*. Empty object, never null."`
	EffectivePatch  map[string]any       `json:"effective_patch" doc:"patch with every amendment folded in: what a merge would write. Empty object, never null."`
	Amendments      []EditAmendment      `json:"amendments" doc:"Amendments in seq order. Empty array, never null."`
	CreatedAt       repr.DateTime        `json:"created_at" doc:"When it was filed."`
	UpdatedAt       repr.DateTime        `json:"updated_at" doc:"When it last changed."`
	DecidedAt       *repr.DateTime       `json:"decided_at" doc:"When it was merged or declined. null while open."`
	Viewer          *EditProposalViewer  `json:"viewer" doc:"The caller's own capabilities. null for an anonymous caller."`
}

type EditProposalSummary struct {
	Object          string               `json:"object" enum:"edit_proposal" maxLength:"13" doc:"Type discriminant. Always edit_proposal."`
	ID              repr.DecimalID       `json:"id" doc:"Proposal id."`
	State           string               `json:"state" enum:"open,merged,declined,withdrawn" maxLength:"9" doc:"Lifecycle state."`
	WorkID          repr.DecimalID       `json:"work_id" doc:"The work this proposal edits."`
	WorkSummary     workrepr.WorkSummary `json:"work_summary" doc:"The work. display_name is empty when catalog did not render it."`
	Note            *string              `json:"note" maxLength:"2000" doc:"The proposer's summary. null when none. Free text; never use it as a decision input."`
	Proposer        repr.UserRef         `json:"proposer" doc:"Who filed it. A deleted or banned account is a deleted user ref."`
	Decider         *repr.UserRef        `json:"decider" doc:"Who merged or declined it. null while open, and when withdrawn."`
	BaseRevisionSeq int                  `json:"base_revision_seq" minimum:"0" doc:"The revision seq the proposal was written against."`
	CreatedAt       repr.DateTime        `json:"created_at" doc:"When it was filed."`
	UpdatedAt       repr.DateTime        `json:"updated_at" doc:"When it last changed."`
	DecidedAt       *repr.DateTime       `json:"decided_at" doc:"When it was merged or declined. null while open."`
	Viewer          *EditProposalViewer  `json:"viewer" doc:"The caller's own capabilities. null for an anonymous caller."`
}

type EditProposalViewer struct {
	IsProposer  bool `json:"is_proposer" doc:"Whether the caller filed it."`
	CanWithdraw bool `json:"can_withdraw" doc:"Whether the caller may withdraw it: the proposer, while open."`
	CanDecide   bool `json:"can_decide" doc:"Whether the caller may merge or decline it. A hint; catalog decides. Requests authenticated with a Bearer token never carry staff powers."`
	CanAmend    bool `json:"can_amend" doc:"Whether the caller may amend it: the proposer, or a reviewer, while open."`
}

type WorkEditProposalCreateResult struct {
	EditProposal
	Revision *EditRevision `json:"revision" doc:"The revision written when catalog merged the proposal at once. null otherwise, or when it could not be read back."`
}

type EditAmendment struct {
	Object    string         `json:"object" enum:"edit_amendment" maxLength:"14" doc:"Type discriminant. Always edit_amendment."`
	ID        repr.DecimalID `json:"id" doc:"Amendment id."`
	Seq       int            `json:"seq" minimum:"1" doc:"Position in the proposal's amendment chain."`
	Amender   repr.UserRef   `json:"amender" doc:"Who amended. A deleted or banned account is a deleted user ref."`
	Note      *string        `json:"note" maxLength:"2000" doc:"The amender's summary. null when none. Free text; never use it as a decision input."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"When it was filed."`
}

type EditRevision struct {
	Object         string          `json:"object" enum:"edit_revision" maxLength:"13" doc:"Type discriminant. Always edit_revision."`
	ID             repr.DecimalID  `json:"id" doc:"Revision id."`
	Seq            int             `json:"seq" minimum:"1" doc:"Position in the work's revision chain, 1-based and contiguous."`
	RevisionAction string          `json:"revision_action" enum:"created,merged,direct,reverted" maxLength:"8" doc:"How the revision came about."`
	ChangedFields  []EditFieldKey  `json:"changed_fields" doc:"Field keys the revision touched. Empty array, never null."`
	Actor          repr.UserRef    `json:"actor" doc:"Who caused it. A deleted or banned account is a deleted user ref."`
	LastAmender    *repr.UserRef   `json:"last_amender" doc:"The last reviewer who amended the proposal before it merged. null when nobody did."`
	ProposalID     *repr.DecimalID `json:"proposal_id" doc:"The proposal it merged. null for direct edits and imports."`
	CreatedAt      repr.DateTime   `json:"created_at" doc:"When it was recorded."`
}

type EditRevisionDiff struct {
	Object       string            `json:"object" enum:"edit_revision_diff" maxLength:"18" doc:"Type discriminant. Always edit_revision_diff."`
	FromSeq      int               `json:"from_seq" minimum:"1" doc:"Base revision seq."`
	ToSeq        int               `json:"to_seq" minimum:"1" doc:"Target revision seq."`
	FieldChanges []EditFieldChange `json:"field_changes" doc:"Fields whose value differs. Empty array when none."`
}

type EditFieldChange struct {
	Key  string `json:"key" maxLength:"128" pattern:"^[a-z0-9_]+(\\.[a-z0-9_]+)+$" doc:"Editing-engine field key. Its field_type and diff_hint are on the edit form's fields."`
	From any    `json:"from" doc:"Value at from_seq. null when unset."`
	To   any    `json:"to" doc:"Value at to_seq. null when cleared."`
}

type EditRevert struct {
	Object   string        `json:"object" enum:"edit_revert" maxLength:"11" doc:"Type discriminant. Always edit_revert."`
	Proposal EditProposal  `json:"proposal" doc:"The proposal the revert filed."`
	Revision *EditRevision `json:"revision" doc:"The revision written when catalog merged the revert at once. null otherwise, or when it could not be read back."`
}

type editProposalCreate struct {
	Patch map[string]any `json:"patch" required:"true" doc:"New values keyed by catalog.work.*. Must name at least one key."`
	Note  *string        `json:"note" required:"false" maxLength:"2000" doc:"Summary for reviewers. Absent or null for none. Free text; never use it as a decision input."`
}

type editProposalPatch struct {
	State string  `json:"state" required:"true" enum:"merged,declined,withdrawn" maxLength:"9" doc:"Target state. merged and declined are reviewer decisions; withdrawn is the proposer's."`
	Note  *string `json:"note" required:"false" maxLength:"2000" doc:"Decision note. Required and not blank when declining; the proposer is told it. Free text; never use it as a decision input."`
}

type editAmendmentCreate struct {
	Set   map[string]any `json:"set" required:"false" doc:"Corrected values keyed by catalog.work.*."`
	Unset []EditFieldKey `json:"unset" required:"false" maxItems:"100" doc:"Keys to drop from the proposal."`
	Note  *string        `json:"note" required:"false" maxLength:"2000" doc:"Amendment summary. Absent or null for none. Free text; never use it as a decision input."`
}

type editRevertCreate struct {
	ToSeq int     `json:"to_seq" required:"true" minimum:"1" doc:"The revision seq to restore."`
	Note  *string `json:"note" required:"false" maxLength:"2000" doc:"Why. Absent or null for none. Free text; never use it as a decision input."`
}
