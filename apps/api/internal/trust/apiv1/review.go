package apiv1

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"unicode/utf8"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/trustclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
)

type accessTokenKey struct{}

var reviewStates = []string{"pending", "claimed", "actioned", "dismissed"}

var reviewOrigins = []string{"reports", "ai_text", "ai_image", "community_forward", "mislabel", "manual", "ai_sample"}

var dispositionActions = map[string]int16{
	"none": 0, "hide": 1, "remove": 2, "warn_user": 3, "restrict": 4, "escalate_idp": 5,
}

type listReviewItemsInput struct {
	collect.PageNumber
	State string `query:"state" enum:"pending,claimed,actioned,dismissed" maxLength:"9" doc:"Only items in this state. Absent means every state."`
}

type listReviewItemsOutput struct {
	Body repr.PageList[ReviewItemSummary]
}

type reviewItemInput struct {
	ReviewItemID string `path:"review_item_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Review item id."`
}

type updateReviewItemInput struct {
	ReviewItemID string `path:"review_item_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Review item id."`
	Body         ReviewItemPatch
}

type reviewItemOutput struct {
	Body ReviewItem
}

// withAccessToken hands the caller's OAuth access token to the handler: the
// trust admin API authorizes the moderator, not this forum's client.
func withAccessToken(ctx huma.Context, next func(huma.Context)) {
	fc := humafiber.Unwrap(ctx)
	next(huma.WithValue(ctx, accessTokenKey{}, middleware.GetAccessToken(fc)))
}

func accessToken(ctx context.Context) string {
	s, _ := ctx.Value(accessTokenKey{}).(string)
	return s
}

func (s *Service) requireReviewer(ctx context.Context) *problem.Problem {
	if prob := s.ready(); prob != nil {
		return prob
	}
	if !v1.User(ctx).Can(perm.TrustReview) {
		return permissionRequired()
	}
	return nil
}

func (s *Service) listReviewItems(ctx context.Context, in *listReviewItemsInput) (*listReviewItemsOutput, error) {
	if prob := s.requireReviewer(ctx); prob != nil {
		return nil, prob
	}
	if prob := in.CheckDepth(); prob != nil {
		return nil, prob
	}
	q := trustclient.ReviewQuery{Site: s.site, Page: in.Page, Limit: in.Limit}
	if in.State != "" {
		status := int16(slices.Index(reviewStates, in.State))
		q.Status = &status
	}
	page, err := s.trust.ListReviewItems(ctx, accessToken(ctx), q)
	if err != nil {
		return nil, adminProblem(err)
	}
	users, prob := s.lookupUsers(ctx, reviewUserIDs(page.Items, nil))
	if prob != nil {
		return nil, prob
	}
	items := make([]ReviewItemSummary, 0, len(page.Items))
	for _, it := range page.Items {
		summary, err := s.reviewSummary(it, users)
		if err != nil {
			return nil, problem.Internal(err)
		}
		items = append(items, summary)
	}
	total, relation := collect.ClampTotal(int(page.Total))
	return &listReviewItemsOutput{Body: repr.NewPageList(items, total, relation)}, nil
}

func (s *Service) getReviewItem(ctx context.Context, in *reviewItemInput) (*reviewItemOutput, error) {
	if prob := s.requireReviewer(ctx); prob != nil {
		return nil, prob
	}
	id, ok := parseItemID(in.ReviewItemID)
	if !ok {
		return nil, notFound()
	}
	item, prob := s.readItem(ctx, id)
	if prob != nil {
		return nil, prob
	}
	return &reviewItemOutput{Body: *item}, nil
}

func (s *Service) updateReviewItem(ctx context.Context, in *updateReviewItemInput) (*reviewItemOutput, error) {
	if prob := s.requireReviewer(ctx); prob != nil {
		return nil, prob
	}
	id, ok := parseItemID(in.ReviewItemID)
	if !ok {
		return nil, notFound()
	}
	patch := in.Body
	if prob := checkPatch(patch); prob != nil {
		return nil, prob
	}
	token := accessToken(ctx)
	var err error
	if patch.State == "claimed" {
		err = s.trust.ClaimReviewItem(ctx, token, id)
	} else {
		req := trustclient.DecideRequest{Decision: patch.State, Statement: patch.Statement}
		if patch.Action != nil {
			action := dispositionActions[*patch.Action]
			req.Action = &action
		}
		if patch.ReasonCode != nil {
			req.ReasonCode = *patch.ReasonCode
		}
		err = s.trust.DecideReviewItem(ctx, token, id, req)
	}
	if err != nil {
		prob := adminProblem(err)
		if prob.Code == problem.CodeInvalidStateTransition {
			if current, again := s.readItem(ctx, id); again == nil {
				prob.Detail = fmt.Sprintf("The current state is %s; legal targets: %s.", current.State, legalTargets(current.State))
			}
		}
		return nil, prob
	}
	item, prob := s.readItem(ctx, id)
	if prob != nil {
		return nil, prob
	}
	return &reviewItemOutput{Body: *item}, nil
}

func checkPatch(p ReviewItemPatch) *problem.Problem {
	var fields []problem.FieldError
	if p.State == "actioned" {
		if p.Action == nil {
			fields = append(fields, problem.AtPointer("/action", problem.ReasonRequired, "actioned needs an action", nil))
		}
		if p.ReasonCode == nil {
			fields = append(fields, problem.AtPointer("/reason_code", problem.ReasonRequired, "actioned needs a reason_code", nil))
		}
	} else {
		if p.Action != nil {
			fields = append(fields, problem.AtPointer("/action", problem.ReasonInconsistentWith, "/state", nil))
		}
		if p.ReasonCode != nil {
			fields = append(fields, problem.AtPointer("/reason_code", problem.ReasonInconsistentWith, "/state", nil))
		}
		if p.Statement != nil {
			fields = append(fields, problem.AtPointer("/statement", problem.ReasonInconsistentWith, "/state", nil))
		}
	}
	if len(fields) > 0 {
		return validationFailed(fields...)
	}
	return nil
}

func legalTargets(state string) string {
	switch state {
	case "pending":
		return "claimed, actioned, dismissed"
	case "claimed":
		return "actioned, dismissed"
	}
	return "none"
}

func (s *Service) readItem(ctx context.Context, id int64) (*ReviewItem, *problem.Problem) {
	detail, err := s.trust.GetReviewItem(ctx, accessToken(ctx), id)
	if err != nil {
		return nil, adminProblem(err)
	}
	if detail.Item.Site != s.site {
		return nil, notFound()
	}
	users, prob := s.lookupUsers(ctx, reviewUserIDs([]trustclient.ReviewItem{detail.Item}, detail.Reports))
	if prob != nil {
		return nil, prob
	}
	reasons, prob := s.currentReasons(ctx)
	if prob != nil {
		return nil, prob
	}
	summary, err := s.reviewSummary(detail.Item, users)
	if err != nil {
		return nil, problem.Internal(err)
	}
	reports := make([]ReviewReport, 0, len(detail.Reports))
	for _, r := range detail.Reports {
		reports = append(reports, s.reviewReport(r, users, reasons))
	}
	return &ReviewItem{ReviewItemSummary: summary, Reports: reports}, nil
}

func (s *Service) reviewSummary(it trustclient.ReviewItem, users map[int]userclient.User) (ReviewItemSummary, error) {
	if it.Status < 0 || int(it.Status) >= len(reviewStates) {
		return ReviewItemSummary{}, fmt.Errorf("review item %d: unknown status %d", it.ID, it.Status)
	}
	origin := "unknown"
	if it.Source >= 0 && int(it.Source) < len(reviewOrigins) {
		origin = reviewOrigins[it.Source]
	}
	out := ReviewItemSummary{
		Object:          "review_item",
		ID:              repr.DecimalID(fmt.Sprint(it.ID)),
		SubjectKind:     TrustSubjectKind(it.SubjectKind),
		SubjectID:       repr.DecimalID(it.SubjectID),
		OpenedBy:        ReviewItemOrigin(origin),
		State:           reviewStates[it.Status],
		Priority:        max(it.Priority, 0),
		ClassifierScore: it.ClassifierScore,
		ReportWeightSum: it.ReportWeightSum,
		ReachCount:      it.SubjectReach,
		ContextNote:     truncate(it.ContextNote, 4000),
		Claimant:        s.userRef(it.ClaimedBy, users),
		ClaimedAt:       repr.TimestampPtr(it.ClaimedAt),
		Decider:         s.userRef(it.DecidedBy, users),
		DecidedAt:       repr.TimestampPtr(it.DecidedAt),
		CreatedAt:       repr.Timestamp(it.CreatedAt),
	}
	if it.Severity != nil {
		severity := int(*it.Severity)
		out.Severity = &severity
	}
	return out, nil
}

func (s *Service) reviewReport(r trustclient.Report, users map[int]userclient.User, reasons []trustclient.ReasonView) ReviewReport {
	out := ReviewReport{
		Object:    "report",
		ID:        repr.DecimalID(fmt.Sprint(r.ID)),
		Reporter:  *s.userRef(&r.ReporterID, users),
		Note:      truncate(r.Note, 1000),
		Snapshot:  truncate(r.SubjectSnapshot, 2000),
		Weight:    max(r.Weight, 0),
		CreatedAt: repr.Timestamp(r.CreatedAt),
	}
	for _, v := range reasons {
		if v.ID == r.ReasonID {
			reason := reportReason(v)
			out.ReportReason = &reason
			break
		}
	}
	if r.SubjectURL != nil && onSite(*r.SubjectURL) && len(*r.SubjectURL) <= 512 {
		out.SubjectURL = r.SubjectURL
	}
	return out
}

func (s *Service) userRef(id *int64, users map[int]userclient.User) *repr.UserRef {
	if id == nil {
		return nil
	}
	ref := repr.DeletedUserRef(int(*id))
	if u, ok := users[int(*id)]; ok {
		ref = repr.NewUserRef(s.cdn, u)
	}
	return &ref
}

func reviewUserIDs(items []trustclient.ReviewItem, reports []trustclient.Report) []int {
	seen := map[int]bool{}
	var ids []int
	add := func(id *int64) {
		if id != nil && !seen[int(*id)] {
			seen[int(*id)] = true
			ids = append(ids, int(*id))
		}
	}
	for _, it := range items {
		add(it.ClaimedBy)
		add(it.DecidedBy)
	}
	for _, r := range reports {
		add(&r.ReporterID)
	}
	return ids
}

func truncate(s *string, maxRunes int) *string {
	if s == nil || utf8.RuneCountInString(*s) <= maxRunes {
		return s
	}
	cut := string([]rune(*s)[:maxRunes])
	slog.Warn("trust v1: upstream text truncated", "max_runes", maxRunes)
	return &cut
}

func parseItemID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}
