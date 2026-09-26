package apiv1

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/problem"
)

const mentionCap = 10

func trimTitle(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
}

func tooShort(pointer string) problem.FieldError {
	min := 1
	return problem.AtPointer(pointer, problem.ReasonTooShort, "must contain at least 1 character after trimming whitespace", &problem.FieldParams{MinLength: &min})
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
}

func permissionRequired() *problem.Problem {
	return problem.New(problem.CodePermissionRequired, "The token lacks the permission this decision needs.")
}

func contentRejected() *problem.Problem {
	return problem.New(problem.CodeContentRejected, "The trust-and-safety check refused the submitted text. Nothing was written.")
}

func dailyLimitReached(limit int) *problem.Problem {
	p := problem.New(problem.CodeTopicDailyLimitReached, "The caller has created as many topics in the last 24 hours as their moemoepoint balance allows.")
	p.SetExtension("limit", limit)
	return p
}

func moemoepointInsufficient(required int) *problem.Problem {
	p := problem.New(problem.CodeMoemoepointInsufficient, "The caller's moemoepoint balance, as this forum last cached it, is below what the operation costs.")
	p.SetExtension("required", required)
	return p
}

func sectionCategory(slug string) string {
	switch {
	case strings.HasPrefix(slug, "g-"):
		return "galgame"
	case strings.HasPrefix(slug, "t-"):
		return "technique"
	case strings.HasPrefix(slug, "o-"):
		return "others"
	default:
		return ""
	}
}

func sectionMismatch(sections []string, category string) []int {
	var bad []int
	for i, slug := range sections {
		if sectionCategory(slug) != category {
			bad = append(bad, i)
		}
	}
	return bad
}

func sectionFields(sections []string, category string) []problem.FieldError {
	var fields []problem.FieldError
	for _, i := range sectionMismatch(sections, category) {
		fields = append(fields, problem.AtPointer(
			fmt.Sprintf("/sections/%d", i),
			problem.ReasonInconsistentWith,
			"/category",
			nil,
		))
	}
	return fields
}

func categorySectionField(sections []string, category string) []problem.FieldError {
	if len(sectionMismatch(sections, category)) == 0 {
		return nil
	}
	return []problem.FieldError{problem.AtPointer("/category", problem.ReasonInconsistentWith, "/sections", nil)}
}

func coversFromHashes(hashes []ImageHash) model.ImageTokens {
	if len(hashes) == 0 {
		return model.ImageTokens{}
	}
	out := make(model.ImageTokens, len(hashes))
	for i, h := range hashes {
		out[i] = "/image/" + string(h)
	}
	return out
}

func deriveCovers(body string) model.ImageTokens {
	tokens := markdown.ExtractCoverImages(body, 9)
	if len(tokens) == 0 {
		return model.ImageTokens{}
	}
	return model.ImageTokens(tokens)
}

func anyConsumeSection(sections []string) bool {
	for _, sec := range sections {
		if constants.TopicSectionConsume[sec] {
			return true
		}
	}
	return false
}

func topicSectionFootprint(consume bool) int {
	if consume {
		return -constants.CostConsumeSection
	}
	return constants.RewardCreateTopic
}

func topicModerationText(title, content string) string {
	return title + "\n\n" + content
}

type accessFields struct {
	scope string
	roles *[]AccessRole
	users *[]repr.DecimalID
	// A users-scoped topic granted only to its author stores no grant row at
	// all, so "the scope needs grants" cannot be answered from the stored rows:
	// every PATCH of such a topic answered 422 REQUIRED.
	keepStored bool
}

func accessErrors(a accessFields) []problem.FieldError {
	var fields []problem.FieldError
	if a.keepStored {
		return nil
	}
	switch a.scope {
	case "public", "login":
		if a.roles != nil {
			fields = append(fields, problem.AtPointer("/access_roles", problem.ReasonInconsistentWith, "/access_scope", nil))
		}
		if a.users != nil {
			fields = append(fields, problem.AtPointer("/access_user_ids", problem.ReasonInconsistentWith, "/access_scope", nil))
		}
	case "role":
		if a.roles == nil {
			fields = append(fields, problem.AtPointer("/access_roles", problem.ReasonRequired, "required when access_scope is role", nil))
		}
		if a.users != nil {
			fields = append(fields, problem.AtPointer("/access_user_ids", problem.ReasonInconsistentWith, "/access_scope", nil))
		}
	case "users":
		if a.users == nil {
			fields = append(fields, problem.AtPointer("/access_user_ids", problem.ReasonRequired, "required when access_scope is users", nil))
		}
		if a.roles != nil {
			fields = append(fields, problem.AtPointer("/access_roles", problem.ReasonInconsistentWith, "/access_scope", nil))
		}
	}
	return fields
}

func grantsFromAccess(a accessFields, authorID int) []model.TopicAccessGrant {
	switch a.scope {
	case "role":
		if a.roles == nil {
			return nil
		}
		seen := map[string]bool{}
		out := make([]model.TopicAccessGrant, 0, len(*a.roles))
		for _, role := range *a.roles {
			v := string(role)
			if seen[v] {
				continue
			}
			seen[v] = true
			out = append(out, model.TopicAccessGrant{SubjectType: "role", SubjectValue: v})
		}
		return out
	case "users":
		if a.users == nil {
			return nil
		}
		seen := map[string]bool{}
		out := make([]model.TopicAccessGrant, 0, len(*a.users))
		for _, raw := range *a.users {
			id, ok := repr.ParseID(raw)
			if !ok || id == authorID {
				continue
			}
			v := strconv.Itoa(id)
			if seen[v] {
				continue
			}
			seen[v] = true
			out = append(out, model.TopicAccessGrant{SubjectType: "user", SubjectValue: v})
		}
		return out
	default:
		return nil
	}
}

func splitGrants(rows []model.TopicAccessGrant) (roles []AccessRole, users []repr.DecimalID) {
	roles = []AccessRole{}
	users = []repr.DecimalID{}
	for _, g := range rows {
		switch g.SubjectType {
		case "role":
			roles = append(roles, AccessRole(g.SubjectValue))
		case "user":
			users = append(users, repr.DecimalID(g.SubjectValue))
		}
	}
	return roles, users
}

func mergeAccess(storedScope string, stored []model.TopicAccessGrant, patch TopicPatch) accessFields {
	out := accessFields{scope: storedScope}
	storedRoles, storedUsers := splitGrants(stored)
	scopeChanged := patch.AccessScope != nil && *patch.AccessScope != storedScope
	if patch.AccessScope != nil {
		out.scope = *patch.AccessScope
	}
	if scopeChanged {
		out.roles = patch.AccessRoles
		out.users = patch.AccessUserIDs
		return out
	}
	if patch.AccessRoles != nil {
		out.roles = patch.AccessRoles
	} else if storedScope == "role" {
		cp := append([]AccessRole(nil), storedRoles...)
		out.roles = &cp
		out.keepStored = true
	}
	if patch.AccessUserIDs != nil {
		out.users = patch.AccessUserIDs
	} else if storedScope == "users" {
		cp := append([]repr.DecimalID(nil), storedUsers...)
		out.users = &cp
		out.keepStored = true
	}
	if patch.AccessRoles != nil || patch.AccessUserIDs != nil {
		out.keepStored = false
	}
	return out
}
