package apiv1

import (
	"context"
	"log/slog"
	"strings"

	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

// Must equal the forum OAuth client's oauth_clients.catalog_site binding.
const catalogSiteKungal = "kungal"

func permissionRequired() *problem.Problem {
	return problem.New(problem.CodePermissionRequired, "The caller cannot perform this operation.")
}

// A bare catalog cursor passed through is not bound to the v1 filter it was
// minted under; wrapped, a cursor reused across filters is INVALID_CURSOR.
func wrapUpstreamCursor(scope, fingerprint, upstream string) *string {
	if upstream == "" {
		return nil
	}
	cur := collect.EncodeCursor(scope, fingerprint, upstream)
	return &cur
}

func unwrapUpstreamCursor(cur, scope, fingerprint string) (string, *problem.Problem) {
	keys, p := collect.DecodeCursor(cur, scope, fingerprint)
	if p != nil {
		return "", p
	}
	if len(keys) == 0 {
		return "", nil
	}
	if len(keys) != 1 || keys[0] == "" {
		return "", problem.New(problem.CodeInvalidCursor, "The cursor cannot be parsed or is no longer valid.",
			problem.AtParameter("cursor", problem.ReasonInvalidFormat, "pass the next_cursor from a previous page of this collection", nil))
	}
	return keys[0], nil
}

func (s *Service) lookupUserRefs(ctx context.Context, ids []int) (map[int]repr.UserRef, *problem.Problem) {
	unique := make([]int, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	out := make(map[int]repr.UserRef, len(unique))
	if len(unique) == 0 {
		return out, nil
	}
	users, p := s.lookupUsers(ctx, unique)
	if p != nil {
		return nil, p
	}
	for _, id := range unique {
		u, ok := users[id]
		if !ok || !userclient.IsRenderable(u) {
			out[id] = repr.DeletedUserRef(id)
			continue
		}
		out[id] = repr.NewUserRef(s.cdn, u)
	}
	return out, nil
}

func userRefOf(refs map[int]repr.UserRef, id int) repr.UserRef {
	if r, ok := refs[id]; ok {
		return r
	}
	return repr.DeletedUserRef(id)
}

func userRefPtr(refs map[int]repr.UserRef, id int) *repr.UserRef {
	if id <= 0 {
		return nil
	}
	r := userRefOf(refs, id)
	return &r
}

func optionalNote(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func trimmedNote(note *string) string {
	if note == nil {
		return ""
	}
	return strings.TrimSpace(*note)
}

func noteRequired() *problem.Problem {
	return validationFailed(problem.AtPointer("/note", problem.ReasonRequired, "declining needs a note that is not blank", nil))
}

// ifMatchOrAny forwards the client's validator, or * when it sent none: catalog
// answers 428 to a write without If-Match.
func ifMatchOrAny(v string) string {
	if v == "" {
		return "*"
	}
	return v
}

func invalidTransition(object, current string, allowed []string) *problem.Problem {
	targets := "none"
	if len(allowed) > 0 {
		targets = strings.Join(allowed, ", ")
	}
	return problem.New(problem.CodeInvalidStateTransition,
		"The "+object+" is in state "+current+"; allowed targets are "+targets+".")
}

// Hydrated under all and gated by the client on is_nsfw: G6.1 hydrated under
// sfw and every NSFW work fell out of the page as "catalog did not render".
func (s *Service) workSummaries(ctx context.Context, ids []int) map[int]workrepr.WorkSummary {
	out := make(map[int]workrepr.WorkSummary, len(ids))
	if s.hydrator == nil || len(ids) == 0 {
		return out
	}
	items, p := s.hydrator.ByIDs(ctx, ids, true)
	if p != nil {
		slog.Warn("galgame contribution: work summary hydration failed", "work_ids", ids, "err", p)
		return out
	}
	for _, it := range items {
		if id, ok := repr.ParseID(it.ID); ok {
			out[id] = it
		}
	}
	return out
}

func emptyWorkSummary(workID int, displayName string) workrepr.WorkSummary {
	return workrepr.WorkSummary{
		WorkRef:           repr.NewWorkRef(workID, repr.NewCatalogName(displayName, "", nil), nil, false),
		ResourcePlatforms: []workrepr.ResourcePlatform{},
		ResourceLanguages: []workrepr.ResourceLanguage{},
	}
}

func (s *Service) checkContributionText(ctx context.Context, text string, authorID int) *problem.Problem {
	if strings.TrimSpace(text) == "" || s.check == nil {
		return nil
	}
	id := int64(authorID)
	decision, matched := s.check.Decision(ctx, text, &id)
	switch decision {
	case gate.DecisionDeny:
		return problem.New(problem.CodeContentRejected, "The trust-and-safety check refused the submitted text. Nothing was written.")
	case gate.DecisionHold:
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgame, "author_id", authorID, "matched", matched)
	}
	return nil
}
