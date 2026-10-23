package apiv1

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func pollDeadlineInvalid() problem.FieldError {
	return problem.AtPointer("/closes_at", problem.ReasonInvalidFormat,
		"must be a real instant in RFC 3339 UTC with second precision", nil)
}

func pollUnknownOption(pointer string) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonUnknownReference,
		"the option does not belong to this poll", nil)
}

func pollTooManyPolls() *problem.Problem {
	max := constants.MaxPollsPerTopic
	return validationFailed(problem.AtParameter("topic_id", problem.ReasonTooManyItems,
		"this topic already holds as many polls as it may", &problem.FieldParams{MaxItems: &max}))
}

func pollOptionCountFields(n int) []problem.FieldError {
	switch {
	case n < pollMinOptions:
		min := pollMinOptions
		return []problem.FieldError{problem.AtPointer("/option_changes", problem.ReasonTooFewItems,
			"a poll keeps at least two options", &problem.FieldParams{MinItems: &min})}
	case n > pollMaxOptions:
		max := pollMaxOptions
		return []problem.FieldError{problem.AtPointer("/option_changes", problem.ReasonTooManyItems,
			"a poll holds at most twenty options", &problem.FieldParams{MaxItems: &max})}
	default:
		return nil
	}
}

// The schema's own maxLength is the raw value (K19); this is the trim that
// happens after it, so a label of only whitespace is TOO_SHORT and not a blank
// option in the database.
func trimPollTexts(options []PollOptionCreate, pointerPrefix string) ([]string, *problem.Problem) {
	out := make([]string, len(options))
	for i, opt := range options {
		text := strings.TrimSpace(opt.Text)
		if text == "" {
			return nil, validationFailed(tooShort(fmt.Sprintf("%s/%d/text", pointerPrefix, i)))
		}
		out[i] = text
	}
	return out, nil
}

func parsePollDeadline(d PollDeadline) (*time.Time, *problem.Problem) {
	if !d.Present || d.Value == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, string(*d.Value))
	if err != nil {
		return nil, validationFailed(pollDeadlineInvalid())
	}
	utc := t.UTC()
	return &utc, nil
}

type pollChoiceBounds struct {
	min int
	max int
}

func resolvePollChoice(choiceType string, min, max *int, optionCount int) (pollChoiceBounds, *problem.Problem) {
	if choiceType == "single" {
		return pollChoiceBounds{min: 1, max: 1}, nil
	}
	bounds := pollChoiceBounds{min: 1, max: optionCount}
	if min != nil {
		bounds.min = *min
	}
	if max != nil {
		bounds.max = *max
	}
	if bounds.max > optionCount {
		limit := float64(optionCount)
		return bounds, validationFailed(problem.AtPointer("/max_choice", problem.ReasonOutOfRange,
			"must not ask for more options than the poll holds", &problem.FieldParams{Maximum: &limit}))
	}
	if bounds.min > bounds.max {
		return bounds, validationFailed(problem.AtPointer("/min_choice", problem.ReasonInconsistentWith,
			"/max_choice", nil))
	}
	return bounds, nil
}

func pollModerationText(title, description string, optionTexts []string) string {
	parts := make([]string, 0, 2+len(optionTexts))
	parts = append(parts, title, description)
	parts = append(parts, optionTexts...)
	return gate.ComposeText(parts...)
}

func (p *Polls) createPoll(ctx context.Context, in *createPollInput) (*createPollOutput, error) {
	if prob := p.ready(); prob != nil {
		return nil, prob
	}
	topic, user, prob := p.reads.visibleTopic(ctx, in.TopicID)
	if prob != nil {
		return nil, prob
	}
	if user.ID != topic.UserID && !user.Can(perm.PollCreateAny) {
		return nil, permissionRequired()
	}
	count, err := p.polls.CountByTopicID(topic.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if count >= int64(constants.MaxPollsPerTopic) {
		return nil, pollTooManyPolls()
	}

	title := strings.TrimSpace(in.Body.Title)
	if title == "" {
		return nil, validationFailed(tooShort("/title"))
	}
	description := ""
	if in.Body.Description != nil {
		description = strings.TrimSpace(*in.Body.Description)
	}
	texts, prob := trimPollTexts(in.Body.Options, "/options")
	if prob != nil {
		return nil, prob
	}
	deadline, prob := parsePollDeadline(in.Body.ClosesAt)
	if prob != nil {
		return nil, prob
	}
	bounds, prob := resolvePollChoice(string(in.Body.ChoiceType), in.Body.MinChoice, in.Body.MaxChoice, len(texts))
	if prob != nil {
		return nil, prob
	}

	moderation := pollModerationText(title, description, texts)
	verdict, prob := p.rejectContent(ctx, moderation, user.ID)
	if prob != nil {
		return nil, prob
	}

	row := &model.TopicPoll{
		Title:            title,
		Description:      description,
		Type:             string(in.Body.ChoiceType),
		MinChoice:        bounds.min,
		MaxChoice:        bounds.max,
		Deadline:         deadline,
		ResultVisibility: string(in.Body.ResultVisibility),
		IsAnonymous:      in.Body.IsAnonymous != nil && *in.Body.IsAnonymous,
		CanChangeVote:    in.Body.CanChangeVote == nil || *in.Body.CanChangeVote,
		TopicID:          topic.ID,
		UserID:           user.ID,
	}
	var pollID int
	err = p.db().Transaction(func(tx *gorm.DB) error {
		id, _, err := p.polls.InsertPoll(tx, row)
		if err != nil {
			return err
		}
		pollID = id
		if err := p.polls.InsertPollOptions(tx, id, texts); err != nil {
			return err
		}
		return p.polls.TouchTopicStatusUpdateTime(tx, topic.ID, time.Now())
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	p.scanPoll(verdict, pollID, user.ID, moderation)

	out, herr := p.reloadOut(ctx, topic, pollID, user)
	if herr != nil {
		return nil, herr
	}
	return &createPollOutput{
		Location: "/api/v1/polls/" + strconv.Itoa(pollID),
		Body:     out.Body,
	}, nil
}

type pollOptionPlan struct {
	add    []string
	rename map[int]string
	remove []int
	count  int
}

func (p *Polls) planOptionChanges(stored []model.TopicPollOption, changes *PollOptionChanges) (pollOptionPlan, *problem.Problem) {
	plan := pollOptionPlan{rename: map[int]string{}, count: len(stored)}
	if changes == nil {
		return plan, nil
	}
	byID := map[int]model.TopicPollOption{}
	for _, opt := range stored {
		byID[opt.ID] = opt
	}
	texts, prob := trimPollTexts(changes.Add, "/option_changes/add")
	if prob != nil {
		return plan, prob
	}
	plan.add = texts

	for i, u := range changes.Update {
		id, ok := repr.ParseID(u.OptionID)
		opt, known := byID[id]
		if !ok || !known {
			return plan, validationFailed(pollUnknownOption(fmt.Sprintf("/option_changes/update/%d/option_id", i)))
		}
		text := strings.TrimSpace(u.Text)
		if text == "" {
			return plan, validationFailed(tooShort(fmt.Sprintf("/option_changes/update/%d/text", i)))
		}
		if opt.VoteCount > 0 && opt.Text != text {
			return plan, validationFailed(problem.AtPointer(fmt.Sprintf("/option_changes/update/%d/text", i),
				problem.ReasonImmutable, "an option that already holds votes keeps its label", nil))
		}
		plan.rename[id] = text
	}
	for i, raw := range changes.Remove {
		id, ok := repr.ParseID(raw)
		opt, known := byID[id]
		if !ok || !known {
			return plan, validationFailed(pollUnknownOption(fmt.Sprintf("/option_changes/remove/%d", i)))
		}
		if opt.VoteCount > 0 {
			return plan, validationFailed(problem.AtPointer(fmt.Sprintf("/option_changes/remove/%d", i),
				problem.ReasonImmutable, "an option that already holds votes cannot be removed", nil))
		}
		plan.remove = append(plan.remove, id)
	}
	plan.count = len(stored) + len(plan.add) - len(plan.remove)
	if fields := pollOptionCountFields(plan.count); fields != nil {
		return plan, validationFailed(fields...)
	}
	return plan, nil
}

func (p *Polls) updatePoll(ctx context.Context, in *updatePollInput) (*pollOutput, error) {
	poll, topic, user, prob := p.visiblePoll(ctx, in.PollID)
	if prob != nil {
		return nil, prob
	}
	if !capsForPoll(topic, poll, user).Edit {
		return nil, permissionRequired()
	}
	stored, err := p.polls.FindOptionsByPollID(poll.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	totals, err := p.polls.PollTotals([]int{poll.ID})
	if err != nil {
		return nil, problem.Internal(err)
	}
	hasVotes := len(totals) > 0 && totals[0].TotalVoteCount > 0

	body := in.Body
	fields := map[string]any{}
	title := poll.Title
	if body.Title != nil {
		title = strings.TrimSpace(*body.Title)
		if title == "" {
			return nil, validationFailed(tooShort("/title"))
		}
		fields["title"] = title
	}
	description := poll.Description
	if body.Description != nil {
		description = strings.TrimSpace(*body.Description)
		fields["description"] = description
	}
	choiceType := poll.Type
	if body.ChoiceType != nil {
		choiceType = string(*body.ChoiceType)
		// Votes already cast in a multiple-choice poll stay in the table, so a
		// poll flipped to single would keep voters holding three options.
		if hasVotes && choiceType != poll.Type {
			return nil, validationFailed(problem.AtPointer("/choice_type", problem.ReasonImmutable,
				"a poll that already holds votes keeps its choice type", nil))
		}
		fields["type"] = choiceType
	}
	if body.IsAnonymous != nil {
		// Those votes were cast under a promise of anonymity, and the vote log
		// would name every one of them.
		if hasVotes && poll.IsAnonymous && !*body.IsAnonymous {
			return nil, validationFailed(problem.AtPointer("/is_anonymous", problem.ReasonImmutable,
				"an anonymous poll that already holds votes cannot be made public", nil))
		}
		fields["is_anonymous"] = *body.IsAnonymous
	}
	if body.CanChangeVote != nil {
		fields["can_change_vote"] = *body.CanChangeVote
	}
	if body.ResultVisibility != nil {
		fields["result_visibility"] = string(*body.ResultVisibility)
	}
	if body.ClosesAt.Present {
		deadline, prob := parsePollDeadline(body.ClosesAt)
		if prob != nil {
			return nil, prob
		}
		fields["deadline"] = deadline
	}

	plan, prob := p.planOptionChanges(stored, body.OptionChanges)
	if prob != nil {
		return nil, prob
	}
	lo, hi := poll.MinChoice, min(poll.MaxChoice, plan.count)
	if body.MinChoice != nil {
		lo = *body.MinChoice
	}
	if body.MaxChoice != nil {
		hi = *body.MaxChoice
	}
	bounds, prob := resolvePollChoice(choiceType, &lo, &hi, plan.count)
	if prob != nil {
		return nil, prob
	}
	fields["min_choice"] = bounds.min
	fields["max_choice"] = bounds.max

	// K18: only the text this request submits is checked again.
	var submitted []string
	if body.Title != nil && title != poll.Title {
		submitted = append(submitted, title)
	}
	if body.Description != nil && description != poll.Description {
		submitted = append(submitted, description)
	}
	submitted = append(submitted, plan.add...)
	for _, text := range plan.rename {
		submitted = append(submitted, text)
	}
	verdict := gate.Allowed
	moderation := ""
	if len(submitted) > 0 {
		moderation = gate.ComposeText(submitted...)
		verdict, prob = p.rejectContent(ctx, moderation, poll.UserID)
		if prob != nil {
			return nil, prob
		}
	}

	err = p.db().Transaction(func(tx *gorm.DB) error {
		if err := p.polls.UpdatePollRow(tx, poll.ID, fields); err != nil {
			return err
		}
		if err := p.polls.InsertPollOptions(tx, poll.ID, plan.add); err != nil {
			return err
		}
		for id, text := range plan.rename {
			if err := p.polls.UpdatePollOptionText(tx, id, text); err != nil {
				return err
			}
		}
		return p.polls.DeletePollOptions(tx, plan.remove)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	if moderation != "" {
		p.scanPoll(verdict, poll.ID, poll.UserID, moderation)
	}
	return p.reloadOut(ctx, topic, poll.ID, user)
}

func (p *Polls) deletePoll(ctx context.Context, in *pollInput) (*struct{}, error) {
	poll, topic, user, prob := p.visiblePoll(ctx, in.PollID)
	if prob != nil {
		return nil, prob
	}
	if !capsForPoll(topic, poll, user).Delete {
		return nil, permissionRequired()
	}
	err := p.db().Transaction(func(tx *gorm.DB) error {
		return p.polls.DeletePollCascade(tx, poll.ID)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	return nil, nil
}
