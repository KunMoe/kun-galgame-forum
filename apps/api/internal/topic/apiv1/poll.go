package apiv1

import (
	"encoding/json"
	"reflect"

	"kun-galgame-api/internal/apiv1/repr"

	"github.com/danielgtaylor/huma/v2"
)

const (
	pollMinOptions   = 2
	pollMaxOptions   = 20
	pollSampleVoters = 5
)

// G8 compares one property name's schema across the whole document, and a $ref
// is part of that schema: Poll.options, PollCreate.options and
// PollResults.options are three different objects under one name, so a
// registered component for any of them fails the gate. huma skips the registry
// for a type that carries its own schema, so these three inline instead, and
// the twin type is what keeps the field tags as the single source.
func inlineObjectSchema[T any](r huma.Registry) *huma.Schema {
	var zero T
	return huma.SchemaFromType(r, reflect.TypeOf(zero))
}

type PollChoiceType string

func (PollChoiceType) Schema(huma.Registry) *huma.Schema {
	s := repr.ClosedEnum("single", "multiple")
	s.Description = "Whether a voter picks exactly one option or several. " +
		"min_choice and max_choice are both 1 when it is single."
	return s
}

type PollResultVisibility string

func (PollResultVisibility) Schema(huma.Registry) *huma.Schema {
	s := repr.ClosedEnum("always", "after_vote", "after_deadline")
	s.Description = "Who may see the tallies: everyone, only those who have voted, or only after closes_at has passed. " +
		"The poll's author and staff holding the view permission always may."
	return s
}

type PollOption struct {
	Object string         `json:"object" enum:"poll_option" maxLength:"11" doc:"Type discriminant. Always poll_option."`
	ID     repr.DecimalID `json:"id" doc:"Option id. JSON string of a decimal integer."`
	Text   string         `json:"text" maxLength:"100" doc:"Option label as stored. Free text; never use it as a decision input."`
}

type pollOptionFields PollOption

func (PollOption) Schema(r huma.Registry) *huma.Schema {
	return inlineObjectSchema[pollOptionFields](r)
}

type PollOptionResult struct {
	OptionID  repr.DecimalID `json:"option_id" doc:"Id of the option these votes are for."`
	VoteCount int            `json:"vote_count" minimum:"0" doc:"Number of votes this option holds."`
}

type pollOptionResultFields PollOptionResult

func (PollOptionResult) Schema(r huma.Registry) *huma.Schema {
	return inlineObjectSchema[pollOptionResultFields](r)
}

type PollResults struct {
	TotalVoteCount int                `json:"total_vote_count" minimum:"0" doc:"Number of votes cast, counting every option a voter picked."`
	VoterCount     int                `json:"voter_count" minimum:"0" doc:"Number of distinct users who have voted."`
	Options        []PollOptionResult `json:"options" maxItems:"20" doc:"One entry per option of this poll, in the same order as options. Empty array if the poll has no options."`
	SampleVoters   []repr.UserRef     `json:"sample_voters" maxItems:"5" doc:"Up to five of the earliest voters, oldest first. Empty array when the poll is anonymous, and banned users are left out, so it can hold fewer than min(voter_count, 5)."`
}

type PollViewer struct {
	HasVoted        bool             `json:"has_voted" doc:"Whether the caller has voted in this poll."`
	ChosenOptionIDs []repr.DecimalID `json:"chosen_option_ids" maxItems:"20" doc:"Ids of the options the caller picked, in option order. Empty array when the caller has not voted."`
	CanVote         bool             `json:"can_vote" doc:"Whether setPollVote would be accepted now: the poll is open, and either the caller has not voted or the poll allows changing a vote."`
	CanChangeVote   bool             `json:"can_change_vote" doc:"Whether the caller may replace or retract a vote they already cast. It is the poll's can_change_vote."`
	CanEdit         bool             `json:"can_edit" doc:"Whether the caller may edit the poll: the poll's author, the topic's author, or staff holding the edit permission. Requests authenticated with a Bearer token never carry staff powers."`
	CanDelete       bool             `json:"can_delete" doc:"Whether the caller may delete the poll: the poll's author, the topic's author, or staff holding the delete permission."`
	CanViewResults  bool             `json:"can_view_results" doc:"Whether the caller may see results now. It says the same thing as results being non-null."`
}

type Poll struct {
	Object           string               `json:"object" enum:"poll" maxLength:"4" doc:"Type discriminant. Always poll."`
	ID               repr.DecimalID       `json:"id" doc:"Poll id. JSON string of a decimal integer."`
	TopicID          repr.DecimalID       `json:"topic_id" doc:"Id of the topic the poll belongs to."`
	Title            string               `json:"title" maxLength:"100" doc:"Poll question as stored. Free text; never use it as a decision input."`
	Description      string               `json:"description" maxLength:"500" doc:"Longer explanation as stored. Empty string when there is none, never null. Free text; never use it as a decision input."`
	ChoiceType       PollChoiceType       `json:"choice_type" doc:"Whether a voter picks exactly one option or several."`
	MinChoice        int                  `json:"min_choice" minimum:"1" doc:"Fewest options a vote may hold. Always 1 when choice_type is single."`
	MaxChoice        int                  `json:"max_choice" minimum:"1" doc:"Most options a vote may hold. Always 1 when choice_type is single."`
	ClosesAt         *repr.DateTime       `json:"closes_at" doc:"When the poll stops accepting votes. null when it never closes."`
	ResultVisibility PollResultVisibility `json:"result_visibility" doc:"Who may see the tallies."`
	IsAnonymous      bool                 `json:"is_anonymous" doc:"Whether who voted for what is hidden. An anonymous poll never lists voters and has no vote log."`
	CanChangeVote    bool                 `json:"can_change_vote" doc:"Whether a voter may replace or retract their vote."`
	Author           repr.UserRef         `json:"author" doc:"The user who created the poll."`
	Options          []PollOption         `json:"options" maxItems:"20" doc:"The options, oldest first. Tallies are not here; they are in results."`
	Results          *PollResults         `json:"results" doc:"The tallies. null as one block when the caller may not see them yet, never a scattering of null counts."`
	CreatedAt        repr.DateTime        `json:"created_at" doc:"Creation time."`
	UpdatedAt        repr.DateTime        `json:"updated_at" doc:"Time of the latest change to the poll or its options."`
	Viewer           *PollViewer          `json:"viewer" doc:"The caller's own state on this poll. null for an anonymous caller."`
}

type PollVote struct {
	Object    string         `json:"object" enum:"poll_vote" maxLength:"9" doc:"Type discriminant. Always poll_vote."`
	ID        repr.DecimalID `json:"id" doc:"Vote id. JSON string of a decimal integer."`
	PollID    repr.DecimalID `json:"poll_id" doc:"Id of the poll this vote belongs to."`
	OptionID  repr.DecimalID `json:"option_id" doc:"Id of the option that was picked. A voter in a multiple-choice poll has one entry per option they picked."`
	Voter     repr.UserRef   `json:"voter" doc:"The user who cast it."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"When it was cast."`
}

type PollOptionCreate struct {
	Text string `json:"text" minLength:"1" maxLength:"100" doc:"Option label, stored as sent. A label of only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
}

type pollOptionCreateFields PollOptionCreate

func (PollOptionCreate) Schema(r huma.Registry) *huma.Schema {
	return inlineObjectSchema[pollOptionCreateFields](r)
}

type PollOptionUpdate struct {
	OptionID repr.DecimalID `json:"option_id" doc:"Id of an option of this poll. An id of another poll's option is refused as UNKNOWN_REFERENCE."`
	Text     string         `json:"text" minLength:"1" maxLength:"100" doc:"New label, checked as in createPoll. Free text; never use it as a decision input."`
}

type PollOptionChanges struct {
	Add    []PollOptionCreate `json:"add" required:"false" maxItems:"20" doc:"Options to append, in order."`
	Update []PollOptionUpdate `json:"update" required:"false" maxItems:"20" doc:"Labels to change. An option that already holds votes cannot be relabelled."`
	Remove []repr.DecimalID   `json:"remove" required:"false" maxItems:"20" uniqueItems:"true" doc:"Options to delete. An option that already holds votes cannot be deleted, and at least two options must remain."`
}

// A PATCH has to tell "leave closes_at alone" from "clear it", and a
// *repr.DateTime decodes an absent field and an explicit null to the same nil.
type PollDeadline struct {
	Present bool
	Value   *repr.DateTime
}

func (d *PollDeadline) UnmarshalJSON(data []byte) error {
	d.Present = true
	if string(data) == "null" {
		d.Value = nil
		return nil
	}
	var v repr.DateTime
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	d.Value = &v
	return nil
}

func (PollDeadline) Schema(r huma.Registry) *huma.Schema {
	s := repr.DateTime("").Schema(r)
	s.Nullable = true
	s.Description = "When the poll stops accepting votes, as RFC 3339 in UTC with second precision. null means it never closes."
	return s
}

type PollCreate struct {
	Title            string               `json:"title" minLength:"1" maxLength:"100" doc:"Poll question, stored as sent. A question of only whitespace is refused as TOO_SHORT. Free text; never use it as a decision input."`
	Description      *string              `json:"description,omitempty" maxLength:"500" doc:"Longer explanation. Absent means none. Free text; never use it as a decision input."`
	ChoiceType       PollChoiceType       `json:"choice_type" doc:"Whether a voter picks exactly one option or several."`
	MinChoice        *int                 `json:"min_choice,omitempty" minimum:"1" maximum:"20" doc:"Fewest options a vote may hold. Ignored when choice_type is single, which forces 1. Absent means 1."`
	MaxChoice        *int                 `json:"max_choice,omitempty" minimum:"1" maximum:"20" doc:"Most options a vote may hold, at most the number of options. Ignored when choice_type is single, which forces 1. Absent means every option."`
	ClosesAt         PollDeadline         `json:"closes_at" required:"false" doc:"When the poll stops accepting votes. Absent or null means it never closes. It is stored as sent, never rounded."`
	ResultVisibility PollResultVisibility `json:"result_visibility" doc:"Who may see the tallies."`
	IsAnonymous      *bool                `json:"is_anonymous,omitempty" doc:"Whether who voted for what is hidden. Absent means false. It cannot be turned off once the poll holds a vote."`
	CanChangeVote    *bool                `json:"can_change_vote,omitempty" doc:"Whether a voter may replace or retract their vote. Absent means true."`
	Options          []PollOptionCreate   `json:"options" minItems:"2" maxItems:"20" doc:"The options, in display order. Between 2 and 20."`
}

type PollPatch struct {
	Title            *string               `json:"title,omitempty" minLength:"1" maxLength:"100" doc:"New question, checked as in createPoll. Free text; never use it as a decision input."`
	Description      *string               `json:"description,omitempty" maxLength:"500" doc:"New explanation. An empty string removes it. Free text; never use it as a decision input."`
	ChoiceType       *PollChoiceType       `json:"choice_type,omitempty" doc:"New choice type. It cannot change once the poll holds a vote."`
	MinChoice        *int                  `json:"min_choice,omitempty" minimum:"1" maximum:"20" doc:"New lower bound. Forced to 1 when the resulting choice_type is single."`
	MaxChoice        *int                  `json:"max_choice,omitempty" minimum:"1" maximum:"20" doc:"New upper bound. Forced to 1 when the resulting choice_type is single."`
	ClosesAt         PollDeadline          `json:"closes_at" required:"false" doc:"New deadline. null clears it; leaving the field out keeps the stored one."`
	ResultVisibility *PollResultVisibility `json:"result_visibility,omitempty" doc:"New result visibility."`
	IsAnonymous      *bool                 `json:"is_anonymous,omitempty" doc:"New anonymity. Turning it off is refused once the poll holds a vote: those votes were cast under a promise of anonymity."`
	CanChangeVote    *bool                 `json:"can_change_vote,omitempty" doc:"New can_change_vote."`
	OptionChanges    *PollOptionChanges    `json:"option_changes,omitempty" doc:"Options to add, relabel or delete. The stored options are otherwise left alone."`
}

type PollVoteSet struct {
	OptionIDs []repr.DecimalID `json:"option_ids" minItems:"1" maxItems:"20" uniqueItems:"true" doc:"The options the caller picks, replacing whatever they picked before. Every id must belong to this poll, and the same id twice is refused as DUPLICATE_ITEM."`
}
