package apiv1

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/community/anchor"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/search/repository"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

var errUnconfigured = errors.New("search v1 is not configured")

type Users interface {
	Users(ctx context.Context, ids []int) (map[int]userclient.User, error)
	SearchUsers(ctx context.Context, q string, limit int) ([]userclient.User, error)
}

type Service struct {
	repo      *repository.SearchRepository
	topics    *topicapiv1.Service
	users     Users
	galgame   *client.GalgameClient
	community *communityclient.Client
	anchors   *anchor.Resolver
	cdn       string
}

type Deps struct {
	Repo      *repository.SearchRepository
	Topics    *topicapiv1.Service
	Users     Users
	Galgame   *client.GalgameClient
	Community *communityclient.Client
	Anchors   *anchor.Resolver
	CDN       string
}

func New(d Deps) *Service {
	return &Service{
		repo: d.Repo, topics: d.Topics, users: d.Users, galgame: d.Galgame,
		community: d.Community, anchors: d.Anchors, cdn: d.CDN,
	}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.repo == nil || s.users == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func keywordsOf(q string) ([]string, *problem.Problem) {
	keywords := strings.Fields(q)
	if len(keywords) == 0 {
		minLen := 1
		return nil, problem.New(problem.CodeInvalidParameter, "q is blank.",
			problem.AtParameter("q", problem.ReasonTooShort, "q must contain a non-space character",
				&problem.FieldParams{MinLength: &minLen}))
	}
	return keywords, nil
}

func authenticated(ctx context.Context) bool {
	return v1.User(ctx) != nil
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

const (
	excerptLen  = 233
	excerptLead = 30
)

// Plain text first, then the window: a window cut out of the Markdown source
// showed ** and ![](/image/…) to the reader and could split a token in half.
func excerpt(source string, keywords []string) string {
	text := markdown.ToPlainText(source, utf8.RuneCountInString(source))
	runes := []rune(text)
	if len(runes) <= excerptLen {
		return text
	}
	lower := strings.ToLower(text)
	first := -1
	for _, kw := range keywords {
		if i := strings.Index(lower, strings.ToLower(kw)); i >= 0 && (first < 0 || i < first) {
			first = i
		}
	}
	at := 0
	if first > 0 {
		at = utf8.RuneCountInString(lower[:first])
	}
	if at <= excerptLead {
		return string(runes[:excerptLen])
	}
	start := at - excerptLead
	end := min(start+excerptLen, len(runes))
	return "…" + string(runes[start:end])
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
