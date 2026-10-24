package apiv1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/section/repository"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

var groupKeys = []any{"galgame", "resource", "technique", "community"}

type SectionGroupKey string

func (SectionGroupKey) Schema(huma.Registry) *huma.Schema {
	n := 9
	return &huma.Schema{Type: huma.TypeString, Enum: groupKeys, MaxLength: &n, Description: "Section group key."}
}

type SectionIconName string

func (SectionIconName) Schema(huma.Registry) *huma.Schema {
	n := 64
	return &huma.Schema{
		Type:        huma.TypeString,
		Pattern:     "^lucide:[a-z0-9]+(-[a-z0-9]+)*$",
		MaxLength:   &n,
		Description: "Lucide icon name with its lucide: prefix.",
	}
}

type SectionColor string

func (SectionColor) Schema(huma.Registry) *huma.Schema {
	n := 9
	return &huma.Schema{
		Type:        huma.TypeString,
		Enum:        []any{"default", "primary", "secondary", "success", "warning", "danger"},
		MaxLength:   &n,
		Description: "Theme color name.",
	}
}

type SectionPatch struct {
	Label       *string          `json:"label,omitempty" minLength:"1" maxLength:"40" doc:"New display name. Trimmed; a blank one is TOO_SHORT. Free text; never use it as a decision input."`
	Description *string          `json:"description,omitempty" maxLength:"500" doc:"New description. Trimmed. Free text; never use it as a decision input."`
	IconName    *SectionIconName `json:"icon_name,omitempty" doc:"New icon."`
	Color       *SectionColor    `json:"color,omitempty" doc:"New color."`
	IsArchived  *bool            `json:"is_archived,omitempty" doc:"Archive or restore the section. Archiving keeps its topics and refuses new ones."`
}

type updateSectionInput struct {
	Section topicapiv1.TopicSectionKey `path:"section" doc:"The section to edit."`
	Body    SectionPatch
}

type SectionGroupPatch struct {
	Label string `json:"label" minLength:"1" maxLength:"40" doc:"New display name. Trimmed; a blank one is TOO_SHORT. Free text; never use it as a decision input."`
}

type updateSectionGroupInput struct {
	Group SectionGroupKey `path:"group" doc:"The group to rename."`
	Body  SectionGroupPatch
}

type SectionGroupOrder struct {
	Group    SectionGroupKey          `json:"group" doc:"A group."`
	Sections []topicapiv1.SectionSlug `json:"sections" minItems:"1" maxItems:"20" uniqueItems:"true" doc:"Every section of the group, in the new order. Current keys only."`
}

type SectionOrder struct {
	Groups []SectionGroupOrder `json:"groups" minItems:"4" maxItems:"4" doc:"Every group, in the new order."`
}

type putSectionOrderInput struct {
	Body SectionOrder
}

type noContentOutput struct{}

func (s *Service) admin(ctx context.Context) (*middleware.UserInfo, *problem.Problem) {
	if s == nil || s.repo == nil || s.users == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, problem.New(problem.CodeMissingCredential, "The request has no credentials.")
	}
	if !user.Can(perm.TopicManageSections) {
		return nil, problem.New(problem.CodePermissionRequired, "This write needs "+string(perm.TopicManageSections)+".")
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

func tooShort(pointer string) problem.FieldError {
	n := 1
	return problem.AtPointer(pointer, problem.ReasonTooShort, "must contain at least 1 character after trimming whitespace", &problem.FieldParams{MinLength: &n})
}

func writeProblem(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
	}
	return problem.Internal(err)
}

func (s *Service) updateSection(ctx context.Context, in *updateSectionInput) (*noContentOutput, error) {
	if _, p := s.admin(ctx); p != nil {
		return nil, p
	}
	change := repository.SectionChange{IsArchived: in.Body.IsArchived}
	if in.Body.Label != nil {
		label := strings.TrimSpace(*in.Body.Label)
		if label == "" {
			return nil, problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", tooShort("/label"))
		}
		change.Label = &label
	}
	if in.Body.Description != nil {
		d := strings.TrimSpace(*in.Body.Description)
		change.Description = &d
	}
	if in.Body.IconName != nil {
		icon := string(*in.Body.IconName)
		change.Icon = &icon
	}
	if in.Body.Color != nil {
		color := string(*in.Body.Color)
		change.Color = &color
	}
	if err := s.repo.UpdateSection(string(in.Section), change); err != nil {
		return nil, writeProblem(err)
	}
	s.invalidate()
	return &noContentOutput{}, nil
}

func (s *Service) updateSectionGroup(ctx context.Context, in *updateSectionGroupInput) (*noContentOutput, error) {
	if _, p := s.admin(ctx); p != nil {
		return nil, p
	}
	label := strings.TrimSpace(in.Body.Label)
	if label == "" {
		return nil, problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", tooShort("/label"))
	}
	if err := s.repo.UpdateGroupLabel(string(in.Group), label); err != nil {
		return nil, writeProblem(err)
	}
	s.invalidate()
	return &noContentOutput{}, nil
}

func (s *Service) putSectionOrder(ctx context.Context, in *putSectionOrderInput) (*noContentOutput, error) {
	if _, p := s.admin(ctx); p != nil {
		return nil, p
	}
	rows, err := s.repo.List("")
	if err != nil {
		return nil, problem.Internal(err)
	}
	current := map[string][]string{}
	for _, r := range rows {
		current[r.GroupKey] = append(current[r.GroupKey], r.Name)
	}
	var fields []problem.FieldError
	seen := map[SectionGroupKey]bool{}
	order := make([]repository.GroupOrder, 0, len(in.Body.Groups))
	for gi, g := range in.Body.Groups {
		if seen[g.Group] {
			fields = append(fields, problem.AtPointer(fmt.Sprintf("/groups/%d/group", gi), problem.ReasonDuplicateItem, "", nil))
			continue
		}
		seen[g.Group] = true
		names := make([]string, len(g.Sections))
		for i, sec := range g.Sections {
			names[i] = string(sec)
			if !topicapiv1.IsSectionKey(names[i]) {
				fields = append(fields, problem.AtPointer(fmt.Sprintf("/groups/%d/sections/%d", gi, i), problem.ReasonNotAllowedValue, "a retired key", nil))
			}
		}
		want := slices.Clone(current[string(g.Group)])
		got := slices.Clone(names)
		slices.Sort(want)
		slices.Sort(got)
		if !slices.Equal(want, got) {
			fields = append(fields, problem.AtPointer(fmt.Sprintf("/groups/%d/sections", gi), problem.ReasonInconsistentWith, "/groups/"+fmt.Sprint(gi)+"/group", nil))
		}
		order = append(order, repository.GroupOrder{Key: string(g.Group), Sections: names})
	}
	if len(fields) > 0 {
		return nil, problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
	}
	if err := s.repo.Reorder(order); err != nil {
		return nil, problem.Internal(err)
	}
	s.invalidate()
	return &noContentOutput{}, nil
}

func registerAdmin(api huma.API, s *Service) {
	const (
		needs  = "Needs topic.manage_sections, which a Bearer request never carries. The caller is checked against the account service's current record first: a banned account is ACCOUNT_BANNED. "
		denied = "PERMISSION_REQUIRED without topic.manage_sections; ACCOUNT_BANNED."
	)
	errs := func(m map[int]string) map[string]*huma.Response {
		out := map[string]*huma.Response{}
		for status, desc := range m {
			out[fmt.Sprint(status)] = &huma.Response{
				Description: desc,
				Content:     map[string]*huma.MediaType{problem.ContentType: {Schema: &huma.Schema{Ref: v1.ProblemRef}}},
			}
		}
		return out
	}
	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "updateAdminSection",
		Method:        http.MethodPatch,
		Path:          "/admin/sections/{section}",
		Summary:       "Edit a topic section's display",
		DefaultStatus: http.StatusNoContent,
		Description:   "Changes the fields that are sent: display name, description, icon, color, archived. Which topics the section holds, its group and its moemoepoint rule are not editable. " + needs + "An empty object writes nothing.",
		Tags:          []string{"topics"},
		Responses:     errs(map[int]string{403: denied, 404: "NOT_FOUND when the section does not exist.", 422: "VALIDATION_FAILED for a blank label or a field outside its format."}),
	}), s.updateSection)
	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "updateAdminSectionGroup",
		Method:        http.MethodPatch,
		Path:          "/admin/section-groups/{group}",
		Summary:       "Rename a topic section group",
		DefaultStatus: http.StatusNoContent,
		Description:   "Changes the group's display name. " + needs,
		Tags:          []string{"topics"},
		Responses:     errs(map[int]string{403: denied, 404: "NOT_FOUND when the group does not exist.", 422: "VALIDATION_FAILED for a blank label."}),
	}), s.updateSectionGroup)
	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "putAdminSectionOrder",
		Method:        http.MethodPut,
		Path:          "/admin/section-order",
		Summary:       "Reorder topic sections",
		DefaultStatus: http.StatusNoContent,
		Description:   "Sets the order of the groups and of the sections inside each group. Every group appears once, listing exactly the sections it holds: a section cannot move to another group. " + needs,
		Tags:          []string{"topics"},
		Responses:     errs(map[int]string{403: denied, 422: "VALIDATION_FAILED for a group listed twice, or a group whose sections are not exactly the ones it holds."}),
	}), s.putSectionOrder)
}
