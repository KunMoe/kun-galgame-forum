package apiv1

import (
	"context"
	"log/slog"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

type submissionLane string

const (
	laneMine    submissionLane = "work_submissions_mine"
	laneReviews submissionLane = "work_submissions_reviewed"
	laneQueue   submissionLane = "work_submissions_queue"
)

func (s *Service) listMyWorkSubmissions(ctx context.Context, in *listWorkSubmissionsInput) (*workSubmissionListOutput, error) {
	return s.listSubmissions(ctx, in, laneMine)
}

func (s *Service) listMyWorkSubmissionReviews(ctx context.Context, in *listWorkSubmissionsInput) (*workSubmissionListOutput, error) {
	return s.listSubmissions(ctx, in, laneReviews)
}

func (s *Service) listWorkSubmissions(ctx context.Context, in *listWorkSubmissionsInput) (*workSubmissionListOutput, error) {
	return s.listSubmissions(ctx, in, laneQueue)
}

func (s *Service) listSubmissions(ctx context.Context, in *listWorkSubmissionsInput, lane submissionLane) (*workSubmissionListOutput, error) {
	c, p := s.submissionCall(ctx)
	if p != nil {
		return nil, p
	}
	if lane != laneMine && !c.user.Can(perm.GalgameClaimReview) {
		return nil, permissionRequired()
	}
	states := make([]string, 0, len(in.State))
	for _, st := range in.State {
		if !slices.Contains(states, string(st)) {
			states = append(states, string(st))
		}
	}
	if lane == laneQueue && len(states) == 0 {
		states = []string{"pending"}
	}
	slices.Sort(states)
	fp := collect.Fingerprint(strings.Join(states, ","))
	upstream, p := unwrapUpstreamCursor(in.Cursor, string(lane), fp)
	if p != nil {
		return nil, p
	}
	f := catalogclient.UserClaimFilter{ClaimStates: states, Cursor: upstream, Limit: in.Limit, IncludeTotal: in.IncludeTotal}
	var page *catalogclient.UserClaimPage
	var err error
	switch lane {
	case laneMine:
		f.Kind = catalogclient.ClaimKindSubmitted
		page, err = c.cat.MyClaims(ctx, c.token, f)
	case laneReviews:
		f.Kind = catalogclient.ClaimKindAudited
		page, err = c.cat.MyClaims(ctx, c.token, f)
	default:
		page, err = c.cat.ListModerationClaims(ctx, c.token, f)
	}
	if err != nil {
		return nil, mapUserPlane(err, true)
	}

	rows := make([]catalogclient.UserClaimItem, 0, len(page.Items))
	workIDs := make([]int, 0, len(page.Items))
	userIDs := make([]int, 0, len(page.Items))
	for _, it := range page.Items {
		if !claimStates[it.ClaimState] {
			slog.Warn("galgame submissions: claim with an unknown state dropped", "work_id", it.WorkID, "value", it.ClaimState)
			continue
		}
		rows = append(rows, it)
		workIDs = append(workIDs, int(it.WorkID))
		userIDs = append(userIDs, int(it.LastActorUID))
	}
	refs, p := s.lookupUserRefs(ctx, userIDs)
	if p != nil {
		return nil, p
	}
	owned := map[int]bool{}
	switch lane {
	case laneMine:
		for _, id := range workIDs {
			owned[id] = true
		}
	case laneQueue:
		if owned, p = s.createdBy(workIDs, c.user.ID); p != nil {
			return nil, p
		}
	}
	summaries := s.workSummaries(ctx, workIDs)
	items := make([]WorkSubmissionSummary, 0, len(rows))
	for _, it := range rows {
		id := int(it.WorkID)
		sum, ok := summaries[id]
		if !ok {
			sum = emptyWorkSummary(id, it.DisplayName)
		}
		items = append(items, WorkSubmissionSummary{
			Object: "work_submission", ID: repr.ID(id), WorkID: repr.ID(id),
			DisplayName: it.DisplayName, State: it.ClaimState, WorkSummary: sum,
			LastEvent: claimEventRef(it, refs), FirstActedAt: optionalTime(it),
			Viewer: submissionViewer(c.user, it.ClaimState, owned[id]),
		})
	}
	var total *int
	if in.IncludeTotal {
		n := int(page.Total)
		total = &n
	}
	return &workSubmissionListOutput{Body: repr.NewCountedList(items, wrapUpstreamCursor(string(lane), fp, page.NextCursor), total)}, nil
}

func (s *Service) createdBy(workIDs []int, userID int) (map[int]bool, *problem.Problem) {
	out := make(map[int]bool, len(workIDs))
	if !s.store.Ready() {
		return out, nil
	}
	creators, err := s.store.CreatorsOf(workIDs)
	if err != nil {
		return nil, problem.Internal(err)
	}
	for id, creator := range creators {
		out[id] = creator == userID
	}
	return out, nil
}

func (s *Service) toWorkSubmission(ctx context.Context, user *middleware.UserInfo, it catalogclient.UserClaimItem, owner bool, axes submissionAxes) (WorkSubmission, *problem.Problem) {
	if !claimStates[it.ClaimState] {
		slog.Warn("galgame submissions: claim with an unknown state answered as not found", "work_id", it.WorkID, "value", it.ClaimState)
		return WorkSubmission{}, notFound()
	}
	id := int(it.WorkID)
	creator, p := s.localCreator(id)
	if p != nil {
		return WorkSubmission{}, p
	}
	refs, p := s.lookupUserRefs(ctx, []int{int(it.LastActorUID), creator})
	if p != nil {
		return WorkSubmission{}, p
	}
	return WorkSubmission{
		Object: "work_submission", ID: repr.ID(id), WorkID: repr.ID(id),
		DisplayName: it.DisplayName, State: it.ClaimState,
		IsNSFW: axes.isNSFW, ContentRating: axes.rating,
		Submitter: userRefPtr(refs, creator), LastEvent: claimEventRef(it, refs),
		FirstActedAt: optionalTime(it), ActedCount: max(it.ActedCount, 0),
		Viewer: submissionViewer(user, it.ClaimState, owner),
	}, nil
}

func optionalTime(it catalogclient.UserClaimItem) *repr.DateTime {
	if it.FirstActedAt.IsZero() {
		return nil
	}
	t := repr.Timestamp(it.FirstActedAt)
	return &t
}

func claimEventRef(it catalogclient.UserClaimItem, refs map[int]repr.UserRef) *ClaimEventRef {
	if it.LastEventID == 0 {
		return nil
	}
	if !claimStates[it.LastToState] {
		slog.Warn("galgame submissions: claim event with an unknown to_state dropped",
			"work_id", it.WorkID, "event_id", it.LastEventID, "value", it.LastToState)
		return nil
	}
	from := it.LastFromState
	if from != nil && !claimStates[*from] {
		if *from != "none" {
			slog.Warn("galgame submissions: claim event from_state not in the vocabulary, sent as null",
				"work_id", it.WorkID, "event_id", it.LastEventID, "value", *from)
		}
		from = nil
	}
	return &ClaimEventRef{
		Object: "claim_event", ID: repr.ID(int(it.LastEventID)),
		FromState: from, ToState: it.LastToState, Note: it.LastReason,
		Actor:     userRefOf(refs, int(it.LastActorUID)),
		CreatedAt: repr.Timestamp(it.LastEventAt),
	}
}

func submissionViewer(user *middleware.UserInfo, state string, owner bool) *WorkSubmissionViewer {
	if user == nil {
		return nil
	}
	return &WorkSubmissionViewer{
		CanSubmit:   owner && (state == "draft" || state == "declined"),
		CanWithdraw: owner && (state == "pending" || state == "live"),
		CanDelete:   owner && state == "draft",
		CanReview:   user.Can(perm.GalgameClaimReview),
	}
}

func (s *Service) listWorkSubmissionCandidates(ctx context.Context, in *listWorkSubmissionCandidatesInput) (*workSubmissionCandidateListOutput, error) {
	if _, p := s.requireActive(ctx); p != nil {
		return nil, p
	}
	q := strings.TrimSpace(in.Q)
	if q == "" {
		return nil, invalidParameter(problem.AtParameter("q", problem.ReasonRequired, "q must contain a non-space character", nil))
	}
	search := s.worksSearch()
	if search == nil || s.hydrator == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	const scope = "work_submission_candidates"
	fp := collect.Fingerprint(q, strconv.FormatBool(in.IncludeNSFW), strconv.Itoa(in.Limit))
	pageNo := 1
	if in.Cursor != "" {
		raw, p := unwrapUpstreamCursor(in.Cursor, scope, fp)
		if p != nil {
			return nil, p
		}
		if n, err := strconv.Atoi(raw); err == nil && n > 1 {
			pageNo = n
		} else {
			return nil, invalidParameter(problem.AtParameter("cursor", problem.ReasonInvalidFormat,
				"pass the next_cursor from a previous page of this collection", nil))
		}
	}
	params := url.Values{
		"q":       {q},
		"page":    {strconv.Itoa(pageNo)},
		"limit":   {strconv.Itoa(in.Limit)},
		"include": {workrepr.RowInclude},
	}
	client.ApplyWorksGate(params, !in.IncludeNSFW)
	res, appErr := search.CatalogWorksSearch(ctx, params)
	if appErr != nil {
		return nil, catalogUnavailable(appErr)
	}
	// CatalogItemWizardEligible is the only claim-state gate: the search
	// index's claim_state facet lags a day behind the registry, and gating on
	// it hid every work that had just become adoptable.
	rows := make([]client.CatalogWorkListItem, 0, len(res.Items))
	for i := range res.Items {
		row := &res.Items[i]
		if !client.CatalogItemRenderable(row) || !client.CatalogItemWizardEligible(row) {
			continue
		}
		if !in.IncludeNSFW && client.CatalogItemToBrief(ctx, row).ContentLimit == "nsfw" {
			continue
		}
		rows = append(rows, *row)
	}
	summaries, p := s.hydrator.FromRows(ctx, rows)
	if p != nil {
		return nil, p
	}
	items := make([]WorkSubmissionCandidate, 0, len(rows))
	for i := range rows {
		state := rows[i].ClaimState()
		if state == "" {
			state = "none"
		}
		items = append(items, WorkSubmissionCandidate{Object: "work_submission_candidate", WorkSummary: summaries[i], State: state})
	}
	var next *string
	if int64(pageNo)*int64(in.Limit) < res.Total {
		next = wrapUpstreamCursor(scope, fp, strconv.Itoa(pageNo+1))
	}
	return &workSubmissionCandidateListOutput{Body: repr.NewList(items, next)}, nil
}
