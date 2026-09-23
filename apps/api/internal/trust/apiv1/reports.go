package apiv1

import (
	"context"
	"errors"
	"slices"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/trustclient"
	"kun-galgame-api/pkg/userclient"
)

type listReportReasonsOutput struct {
	Body repr.List[ReportReason]
}

type createReportInput struct {
	Body ReportCreate
}

func (s *Service) listReportReasons(ctx context.Context, _ *struct{}) (*listReportReasonsOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	views, prob := s.currentReasons(ctx)
	if prob != nil {
		return nil, prob
	}
	items := make([]ReportReason, 0, len(views))
	for _, v := range views {
		items = append(items, reportReason(v))
	}
	return &listReportReasonsOutput{Body: repr.NewList(items, nil)}, nil
}

func reportReason(v trustclient.ReasonView) ReportReason {
	return ReportReason{Object: "report_reason", Key: ReportReasonKey(v.Key), DisplayName: v.NameCN}
}

func onSite(raw string) bool {
	return strings.HasPrefix(raw, v1.SiteOrigin+"/")
}

func (s *Service) createReport(ctx context.Context, in *createReportInput) (*struct{}, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	viewer := v1.User(ctx)
	if prob := s.requireActive(ctx, viewer); prob != nil {
		return nil, prob
	}
	body := in.Body
	if !slices.Contains(gate.CanonicalSubjectKinds, string(body.SubjectKind)) {
		return nil, validationFailed(problem.AtPointer("/subject_kind", problem.ReasonUnknownValue,
			"subject_kind is not a kind this forum registers with the trust service", nil))
	}
	if body.SubjectURL != nil && !onSite(*body.SubjectURL) {
		return nil, validationFailed(problem.AtPointer("/subject_url", problem.ReasonNotAllowedValue,
			"subject_url must start with "+v1.SiteOrigin+"/", nil))
	}
	reasons, prob := s.currentReasons(ctx)
	if prob != nil {
		return nil, prob
	}
	if !slices.ContainsFunc(reasons, func(r trustclient.ReasonView) bool { return r.Key == string(body.ReasonKey) }) {
		return nil, validationFailed(problem.AtPointer("/reason_key", problem.ReasonUnknownValue,
			"reason_key is not one of the reasons listReportReasons returns", nil))
	}

	req := trustclient.ReportRequest{
		SubjectKind: string(body.SubjectKind),
		SubjectID:   string(body.SubjectID),
		ReasonKey:   string(body.ReasonKey),
		ReporterID:  int64(viewer.ID),
	}
	if body.Note != nil {
		req.Note = *body.Note
	}
	if body.Snapshot != nil {
		req.Snapshot = *body.Snapshot
	}
	if body.SubjectURL != nil {
		req.SubjectURL = *body.SubjectURL
	}
	if _, err := s.trust.SubmitReport(ctx, req); err != nil {
		if errors.Is(err, trustclient.ErrRateLimited) {
			return nil, problem.New(problem.CodeRateLimited, "The trust service's per-reporter limit was exceeded.")
		}
		return nil, problem.Unavailable(err)
	}
	return nil, nil
}

// A session only learns of a ban when its token is refreshed; OAuth's current
// record is what decides whether this reporter may still write.
func (s *Service) requireActive(ctx context.Context, viewer *middleware.UserInfo) *problem.Problem {
	users, prob := s.lookupUsers(ctx, []int{viewer.ID})
	if prob != nil {
		return prob
	}
	if u, ok := users[viewer.ID]; ok && !userclient.IsRenderable(u) {
		return problem.New(problem.CodeAccountBanned, "The signed-in user's account is banned.")
	}
	return nil
}
