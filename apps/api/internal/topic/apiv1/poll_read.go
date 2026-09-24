package apiv1

import (
	"context"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

type pollCaps struct {
	Edit   bool
	Delete bool
}

// The legacy face let the topic's author edit a poll they cannot delete and
// the poll's author delete a poll they cannot edit, because create and update
// asked for the topic's author while delete asked for the poll's.
func capsForPoll(topic *model.Topic, poll *model.TopicPoll, user *middleware.UserInfo) pollCaps {
	if topic == nil || poll == nil || user == nil {
		return pollCaps{}
	}
	own := user.ID == poll.UserID || user.ID == topic.UserID
	return pollCaps{
		Edit:   own || user.Can(perm.PollEditAny),
		Delete: own || user.Can(perm.PollDeleteAny),
	}
}

func pollIsClosed(poll *model.TopicPoll, now time.Time) bool {
	return poll.Deadline != nil && now.After(*poll.Deadline)
}

func canViewPollResults(poll *model.TopicPoll, user *middleware.UserInfo, hasVoted bool) bool {
	if user != nil && (user.ID == poll.UserID || user.Can(perm.PollViewRestricted)) {
		return true
	}
	switch poll.ResultVisibility {
	case "always":
		return true
	case "after_vote":
		return hasVoted
	case "after_deadline":
		return pollIsClosed(poll, time.Now())
	default:
		return false
	}
}

type pollBundle struct {
	options map[int][]model.TopicPollOption
	totals  map[int]repository.PollTotals
	voters  map[int][]int
	choices map[int][]int
	users   map[int]userclient.User
}

func (p *Polls) loadPollBundle(ctx context.Context, polls []model.TopicPoll, viewer *middleware.UserInfo) (*pollBundle, *problem.Problem) {
	ids := make([]int, len(polls))
	named := make([]int, 0, len(polls))
	for i, poll := range polls {
		ids[i] = poll.ID
		if !poll.IsAnonymous {
			named = append(named, poll.ID)
		}
	}
	options, err := p.polls.ListPollOptions(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	byPoll := map[int][]model.TopicPollOption{}
	for _, opt := range options {
		byPoll[opt.PollID] = append(byPoll[opt.PollID], opt)
	}
	rawTotals, err := p.polls.PollTotals(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	totals := map[int]repository.PollTotals{}
	for _, row := range rawTotals {
		totals[row.PollID] = row
	}
	samples, err := p.polls.SamplePollVoters(named, pollSampleVoters)
	if err != nil {
		return nil, problem.Internal(err)
	}
	voters := map[int][]int{}
	userIDs := make([]int, 0, len(polls)+len(samples))
	for _, row := range samples {
		voters[row.PollID] = append(voters[row.PollID], row.UserID)
		userIDs = append(userIDs, row.UserID)
	}
	choices := map[int][]int{}
	if viewer != nil {
		rows, err := p.polls.ViewerChoices(ids, viewer.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		for _, row := range rows {
			choices[row.PollID] = append(choices[row.PollID], row.OptionID)
		}
	}
	for _, poll := range polls {
		userIDs = append(userIDs, poll.UserID)
	}
	users, prob := p.reads.lookupUsers(ctx, userIDs)
	if prob != nil {
		return nil, prob
	}
	return &pollBundle{options: byPoll, totals: totals, voters: voters, choices: choices, users: users}, nil
}

func (p *Polls) buildPolls(ctx context.Context, topic *model.Topic, polls []model.TopicPoll, viewer *middleware.UserInfo) ([]Poll, *problem.Problem) {
	if len(polls) == 0 {
		return []Poll{}, nil
	}
	bundle, prob := p.loadPollBundle(ctx, polls, viewer)
	if prob != nil {
		return nil, prob
	}
	out := make([]Poll, 0, len(polls))
	for i := range polls {
		mapped, ok := p.mapPoll(topic, &polls[i], bundle, viewer)
		if !ok {
			continue
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (p *Polls) mapPoll(topic *model.Topic, poll *model.TopicPoll, bundle *pollBundle, viewer *middleware.UserInfo) (Poll, bool) {
	author, ok := bundle.users[poll.UserID]
	if ok && !userclient.IsRenderable(author) {
		return Poll{}, false
	}
	ref := repr.DeletedUserRef(poll.UserID)
	if ok {
		ref = repr.NewUserRef(p.reads.cdn, author)
	}
	options := bundle.options[poll.ID]
	mapped := make([]PollOption, len(options))
	for i, opt := range options {
		mapped[i] = PollOption{Object: "poll_option", ID: repr.ID(opt.ID), Text: opt.Text}
	}
	chosen := bundle.choices[poll.ID]
	hasVoted := len(chosen) > 0
	canView := canViewPollResults(poll, viewer, hasVoted)

	var results *PollResults
	if canView {
		results = pollResults(p.reads.cdn, poll, options, bundle)
	}
	var pv *PollViewer
	if viewer != nil {
		caps := capsForPoll(topic, poll, viewer)
		pv = &PollViewer{
			HasVoted:        hasVoted,
			ChosenOptionIDs: chosenIDs(options, chosen),
			CanVote:         !pollIsClosed(poll, time.Now()) && (!hasVoted || poll.CanChangeVote),
			CanChangeVote:   poll.CanChangeVote,
			CanEdit:         caps.Edit,
			CanDelete:       caps.Delete,
			CanViewResults:  canView,
		}
	}
	return Poll{
		Object:           "poll",
		ID:               repr.ID(poll.ID),
		TopicID:          repr.ID(poll.TopicID),
		Title:            poll.Title,
		Description:      poll.Description,
		ChoiceType:       PollChoiceType(poll.Type),
		MinChoice:        poll.MinChoice,
		MaxChoice:        poll.MaxChoice,
		ClosesAt:         repr.TimestampPtr(poll.Deadline),
		ResultVisibility: PollResultVisibility(poll.ResultVisibility),
		IsAnonymous:      poll.IsAnonymous,
		CanChangeVote:    poll.CanChangeVote,
		Author:           ref,
		Options:          mapped,
		Results:          results,
		CreatedAt:        repr.Timestamp(poll.CreatedAt),
		UpdatedAt:        repr.Timestamp(poll.UpdatedAt),
		Viewer:           pv,
	}, true
}

func pollResults(cdn string, poll *model.TopicPoll, options []model.TopicPollOption, bundle *pollBundle) *PollResults {
	tallies := make([]PollOptionResult, len(options))
	for i, opt := range options {
		tallies[i] = PollOptionResult{OptionID: repr.ID(opt.ID), VoteCount: opt.VoteCount}
	}
	sample := []repr.UserRef{}
	if !poll.IsAnonymous {
		for _, id := range bundle.voters[poll.ID] {
			u, ok := bundle.users[id]
			if !ok || !userclient.IsRenderable(u) {
				continue
			}
			sample = append(sample, repr.NewUserRef(cdn, u))
		}
	}
	totals := bundle.totals[poll.ID]
	return &PollResults{
		TotalVoteCount: totals.TotalVoteCount,
		VoterCount:     totals.VoterCount,
		Options:        tallies,
		SampleVoters:   sample,
	}
}

func chosenIDs(options []model.TopicPollOption, chosen []int) []repr.DecimalID {
	picked := map[int]bool{}
	for _, id := range chosen {
		picked[id] = true
	}
	out := []repr.DecimalID{}
	for _, opt := range options {
		if picked[opt.ID] {
			out = append(out, repr.ID(opt.ID))
		}
	}
	return out
}

func (p *Polls) listTopicPolls(ctx context.Context, in *listTopicPollsInput) (*listTopicPollsOutput, error) {
	if prob := p.ready(); prob != nil {
		return nil, prob
	}
	topic, user, prob := p.reads.visibleTopic(ctx, in.TopicID)
	if prob != nil {
		return nil, prob
	}
	rows, err := p.polls.ListPollsOfTopic(topic.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, prob := p.buildPolls(ctx, topic, rows, user)
	if prob != nil {
		return nil, prob
	}
	return &listTopicPollsOutput{Body: repr.NewList(items, nil)}, nil
}

func (p *Polls) getPoll(ctx context.Context, in *pollInput) (*pollOutput, error) {
	poll, topic, user, prob := p.visiblePoll(ctx, in.PollID)
	if prob != nil {
		return nil, prob
	}
	return p.pollOut(ctx, topic, poll, user)
}

func (p *Polls) pollOut(ctx context.Context, topic *model.Topic, poll *model.TopicPoll, user *middleware.UserInfo) (*pollOutput, error) {
	built, prob := p.buildPolls(ctx, topic, []model.TopicPoll{*poll}, user)
	if prob != nil {
		return nil, prob
	}
	if len(built) == 0 {
		return nil, notFound()
	}
	return &pollOutput{Body: built[0]}, nil
}

// A write answers with the whole poll, and the row it changed is the stale one
// the handler started from.
func (p *Polls) reloadOut(ctx context.Context, topic *model.Topic, pollID int, user *middleware.UserInfo) (*pollOutput, error) {
	fresh, err := p.polls.FindByID(pollID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return p.pollOut(ctx, topic, fresh, user)
}
