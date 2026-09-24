package apiv1

import (
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
)

const editProposalLocation = "/api/v1/edit-proposals/"

type editWorkInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
}

type editFormOutput struct {
	Body EditForm
}

type createWorkEditProposalInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	Body   editProposalCreate
}

type createWorkEditProposalOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new proposal, such as /api/v1/edit-proposals/1207."`
	Body     WorkEditProposalCreateResult
}

type listWorkEditProposalsInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	State  string `query:"state" enum:"open,merged,declined,withdrawn" maxLength:"9" default:"open" doc:"Proposal state. Default open."`
	collect.Page
}

type editProposalListOutput struct {
	Body repr.List[EditProposalSummary]
}

type listWorkEditRevisionsInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	collect.PageNumber
}

type editRevisionListOutput struct {
	Body repr.PageList[EditRevision]
}

type workEditRevisionDiffInput struct {
	WorkID  string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	FromSeq int    `query:"from_seq" required:"true" minimum:"1" doc:"Base revision seq."`
	ToSeq   int    `query:"to_seq" required:"true" minimum:"1" doc:"Target revision seq."`
}

type editRevisionDiffOutput struct {
	Body EditRevisionDiff
}

type createWorkEditRevertInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Work id."`
	Body   editRevertCreate
}

type createWorkEditRevertOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the proposal the revert filed, such as /api/v1/edit-proposals/1207."`
	Body     EditRevert
}

type listMyEditProposalsInput struct {
	WorkID string `query:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Narrow to one work."`
	State  string `query:"state" enum:"open,merged,declined,withdrawn" maxLength:"9" doc:"Narrow to one state. Absent means every state."`
	collect.Page
}

type listEditProposalsInput struct {
	State string `query:"state" enum:"open,merged,declined,withdrawn" maxLength:"9" default:"open" doc:"Proposal state. Default open."`
	collect.Page
}

type editProposalIDInput struct {
	ProposalID string `path:"proposal_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Proposal id."`
}

type editProposalOutput struct {
	ETag string `header:"ETag" maxLength:"64" doc:"Catalog's validator for this proposal. Send it back as If-Match."`
	Body EditProposal
}

type createEditProposalAmendmentInput struct {
	ProposalID string `path:"proposal_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Proposal id."`
	IfMatch    string `header:"If-Match" maxLength:"64" pattern:"^(\\*|(W/)?\"[^\"]{1,60}\")$" doc:"The proposal's ETag from GET /edit-proposals/{proposal_id} or a previous write. Absent means unconditional."`
	Body       editAmendmentCreate
}

type createEditProposalAmendmentOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the amended proposal, such as /api/v1/edit-proposals/1207."`
	ETag     string `header:"ETag" maxLength:"64" doc:"The proposal's new validator."`
	Body     EditAmendment
}

type updateEditProposalInput struct {
	ProposalID string `path:"proposal_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Proposal id."`
	IfMatch    string `header:"If-Match" maxLength:"64" pattern:"^(\\*|(W/)?\"[^\"]{1,60}\")$" doc:"The proposal's ETag from GET /edit-proposals/{proposal_id} or a previous write. Absent means unconditional."`
	Body       editProposalPatch
}
