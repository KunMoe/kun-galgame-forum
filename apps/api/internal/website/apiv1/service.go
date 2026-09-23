package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/website/model"
	"kun-galgame-api/internal/website/repository"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

var errUnconfigured = errors.New("apiv1 website: service is not configured")

type ImageMeta func(hashes []string) map[string]imageclient.ImageMeta

type Service struct {
	store *repository.Store
	users *userclient.Client
	meta  ImageMeta
	cdn   string
}

func New(store *repository.Store, users *userclient.Client, meta ImageMeta, cdn string) *Service {
	return &Service{store: store, users: users, meta: meta, cdn: cdn}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || !s.store.Ready() || s.users == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

// requireActive checks the caller against OAuth's current record. A session
// only learns of a ban when its token is refreshed, so without this a banned
// user keeps writing until then.
func (s *Service) requireActive(ctx context.Context) (*middleware.UserInfo, *problem.Problem) {
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	users, err := s.users.Users(ctx, []int{user.ID})
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if u, ok := users[user.ID]; ok && !userclient.IsRenderable(u) {
		return nil, problem.New(problem.CodeAccountBanned, "The signed-in user's account is banned.")
	}
	return user, nil
}

func (s *Service) staff(ctx context.Context, p perm.Permission) (*middleware.UserInfo, *problem.Problem) {
	user, prob := s.requireActive(ctx)
	if prob != nil {
		return nil, prob
	}
	if !user.Can(p) {
		return nil, problem.New(problem.CodePermissionRequired, "This write needs "+string(p)+".")
	}
	return user, nil
}

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func storeProblem(err error) *problem.Problem {
	if errors.Is(err, repository.ErrNotFound) {
		return notFound()
	}
	return problem.Internal(err)
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
}

func taken(pointer string) *problem.Problem {
	return problem.New(problem.CodeAlreadyExists, "Another record already uses this value.",
		problem.AtPointer(pointer, problem.ReasonNotAllowedValue, "already used by another record", nil))
}

func tooShort(pointer string, min int) problem.FieldError {
	return problem.AtPointer(pointer, problem.ReasonTooShort, "too short once surrounding whitespace is removed", &problem.FieldParams{MinLength: &min})
}

func invalidCursor() *problem.Problem {
	return problem.New(
		problem.CodeInvalidCursor,
		"The cursor cannot be parsed or is no longer valid.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil),
	)
}

func parseID(raw string) (int, bool) {
	return repr.ParseID(repr.DecimalID(raw))
}

func optionalID(raw string) (*int, *problem.Problem) {
	if raw == "" {
		return nil, nil
	}
	id, ok := parseID(raw)
	if !ok {
		return nil, notFound()
	}
	return &id, nil
}

func idPtr(id *int) *repr.DecimalID {
	if id == nil {
		return nil
	}
	v := repr.ID(*id)
	return &v
}

func (s *Service) images(hashes []string) map[string]imageclient.ImageMeta {
	if s.meta == nil || len(hashes) == 0 {
		return nil
	}
	return s.meta(hashes)
}

func (s *Service) icon(w *model.GalgameWebsite, metas map[string]imageclient.ImageMeta) (*repr.Image, *string) {
	if w.IconImageHash != "" {
		var meta *imageclient.ImageMeta
		if m, ok := metas[w.IconImageHash]; ok {
			meta = &m
		}
		if img := repr.NewImage(s.cdn, w.IconImageHash, meta); img != nil {
			return img, nil
		}
	}
	if u := strings.TrimSpace(w.Icon); u != "" {
		return nil, &u
	}
	return nil, nil
}

func urlsOf(raw json.RawMessage) []SiteURL {
	out := []SiteURL{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []SiteURL{}
	}
	return out
}

func plainURLs(urls []SiteURL) []string {
	if urls == nil {
		return nil
	}
	out := make([]string, len(urls))
	for i, u := range urls {
		out[i] = string(u)
	}
	return out
}

func categoryRef(c model.GalgameWebsiteCategory) WebsiteCategoryRef {
	return WebsiteCategoryRef{Object: "website_category", ID: repr.ID(c.ID), Slug: c.Name, Label: c.Label}
}

func renderTag(t model.GalgameWebsiteTag) WebsiteTag {
	return WebsiteTag{
		Object: "website_tag", ID: repr.ID(t.ID), Slug: t.Name, Label: t.Label,
		Description: t.Description, Level: t.Level, WebsiteTagGroupID: idPtr(t.GroupID),
	}
}

func renderCategory(c repository.CategoryRow) WebsiteCategory {
	return WebsiteCategory{
		Object: "website_category", ID: repr.ID(c.ID), Slug: c.Name, Label: c.Label,
		Description: c.Description, SortOrder: c.SortOrder, WebsiteCount: c.WebsiteCount,
	}
}

func renderTagGroup(g model.GalgameWebsiteTagGroup) WebsiteTagGroup {
	return WebsiteTagGroup{
		Object: "website_tag_group", ID: repr.ID(g.ID), Slug: g.Name, Label: g.Label,
		Description: g.Description, SortOrder: g.SortOrder, IsMultiSelect: g.MultiSelect,
	}
}
