package apiv1

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/trustclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

var errUnconfigured = errors.New("trust v1 is not configured")

const reasonTTL = 5 * time.Minute

type Upstream interface {
	ListReportReasons(ctx context.Context) ([]trustclient.ReasonView, error)
	SubmitReport(ctx context.Context, req trustclient.ReportRequest) (*trustclient.ReportResult, error)
	ListReviewItems(ctx context.Context, token string, q trustclient.ReviewQuery) (*trustclient.ReviewItemPage, error)
	GetReviewItem(ctx context.Context, token string, id int64) (*trustclient.ReviewItemDetail, error)
	ClaimReviewItem(ctx context.Context, token string, id int64) error
	DecideReviewItem(ctx context.Context, token string, id int64, req trustclient.DecideRequest) error
}

type Users interface {
	Users(ctx context.Context, ids []int) (map[int]userclient.User, error)
}

type Service struct {
	trust Upstream
	users Users
	site  string
	cdn   string

	mu        sync.Mutex
	reasons   []trustclient.ReasonView
	fetchedAt time.Time
}

func New(trust Upstream, users Users, site, cdn string) *Service {
	return &Service{trust: trust, users: users, site: site, cdn: cdn}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.trust == nil || s.users == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Service) currentReasons(ctx context.Context) ([]trustclient.ReasonView, *problem.Problem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reasons != nil && time.Since(s.fetchedAt) < reasonTTL {
		return s.reasons, nil
	}
	views, err := s.trust.ListReportReasons(ctx)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if views == nil {
		views = []trustclient.ReasonView{}
	}
	s.reasons, s.fetchedAt = views, time.Now()
	return views, nil
}

func (s *Service) lookupUsers(ctx context.Context, ids []int) (map[int]userclient.User, *problem.Problem) {
	users, err := s.users.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if users == nil {
		users = map[int]userclient.User{}
	}
	return users, nil
}

func problemResponses(byStatus map[int]string) map[string]*huma.Response {
	out := make(map[string]*huma.Response, len(byStatus))
	for status, desc := range byStatus {
		out[strconv.Itoa(status)] = &huma.Response{
			Description: desc,
			Content: map[string]*huma.MediaType{
				problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}},
			},
		}
	}
	return out
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
}

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func permissionRequired() *problem.Problem {
	return problem.New(problem.CodePermissionRequired, "The caller lacks the trust.review permission.")
}

func adminProblem(err error) *problem.Problem {
	var ae *trustclient.AdminError
	if errors.As(err, &ae) {
		switch ae.Status {
		case http.StatusUnauthorized:
			return problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
		case http.StatusForbidden:
			return permissionRequired()
		case http.StatusNotFound:
			return notFound()
		case http.StatusConflict:
			return problem.New(problem.CodeInvalidStateTransition, "The current state does not allow this transition.")
		}
	}
	return problem.Unavailable(err)
}
