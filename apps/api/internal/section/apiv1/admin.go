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
	Label             *string                 `json:"label,omitempty" minLength:"1" maxLength:"40" doc:"New display name. Trimmed; a blank one is TOO_SHORT. Free text; never use it as a decision input."`
	Description       *string                 `json:"description,omitempty" maxLength:"500" doc:"New description. Trimmed. Free text; never use it as a decision input."`
	IconName          *SectionIconName        `json:"icon_name,omitempty" doc:"New icon."`
	Color             *SectionColor           `json:"color,omitempty" doc:"New color."`
	IsArchived        *bool                   `json:"is_archived,omitempty" doc:"Archive or restore the section. Archiving keeps its topics and refuses new ones."`
	BannerImageHash   *BannerImageHash        `json:"banner_image_hash,omitempty" doc:"New header banner by image-service hash. An empty string removes it."`
	MascotImageHashes *[]topicapiv1.ImageHash `json:"mascot_image_hashes,omitempty" maxItems:"12" uniqueItems:"true" doc:"New mascot images, replacing the stored ones; the section header shows one of them at random. An empty array removes them all."`
}

type BannerImageHash string

func (BannerImageHash) Schema(huma.Registry) *huma.Schema {
	n := 64
	return &huma.Schema{
		Type:        huma.TypeString,
		Pattern:     `^([0-9a-f]{64})?$`,
		MaxLength:   &n,
		Description: "Image-service content hash of the banner, or an empty string for no banner.",
	}
}

type updateSectionInput struct {
	Section topicapiv1.TopicSectionKey `path:"section" doc:"The section to edit."`
	Body    SectionPatch
}

type SectionOrder struct {
	Sections []topicapiv1.SectionSlug `json:"sections" minItems:"1" maxItems:"40" uniqueItems:"true" doc:"Every section, in the new order. Current keys only."`
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
	if in.Body.BannerImageHash != nil {
		banner := string(*in.Body.BannerImageHash)
		change.Banner = &banner
	}
	if in.Body.MascotImageHashes != nil {
		mascots := make([]string, len(*in.Body.MascotImageHashes))
		for i, h := range *in.Body.MascotImageHashes {
			mascots[i] = string(h)
		}
		change.Mascots = &mascots
	}
	if err := s.repo.UpdateSection(string(in.Section), change); err != nil {
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
	want := make([]string, 0, len(rows))
	for _, r := range rows {
		want = append(want, r.Name)
	}
	names := make([]string, len(in.Body.Sections))
	var fields []problem.FieldError
	for i, sec := range in.Body.Sections {
		names[i] = string(sec)
		if !topicapiv1.IsSectionKey(names[i]) {
			fields = append(fields, problem.AtPointer(fmt.Sprintf("/sections/%d", i), problem.ReasonNotAllowedValue, "a retired key", nil))
		}
	}
	got := slices.Clone(names)
	slices.Sort(want)
	slices.Sort(got)
	if len(fields) == 0 && !slices.Equal(want, got) {
		fields = append(fields, problem.AtPointer("/sections", problem.ReasonInconsistentWith, "every section exactly once", nil))
	}
	if len(fields) > 0 {
		return nil, problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
	}
	if err := s.repo.Reorder(names); err != nil {
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
		Description:   "Changes the fields that are sent: display name, description, icon, color, header banner, mascot images, archived. Which topics the section holds and its moemoepoint rule are not editable. " + needs + "An empty object writes nothing.",
		Tags:          []string{"topics"},
		Responses:     errs(map[int]string{403: denied, 404: "NOT_FOUND when the section does not exist.", 422: "VALIDATION_FAILED for a blank label or a field outside its format."}),
	}), s.updateSection)
	huma.Register(api, v1.Required(huma.Operation{
		OperationID:   "putAdminSectionOrder",
		Method:        http.MethodPut,
		Path:          "/admin/section-order",
		Summary:       "Reorder topic sections",
		DefaultStatus: http.StatusNoContent,
		Description:   "Sets the order sections are listed in. Every section appears exactly once. " + needs,
		Tags:          []string{"topics"},
		Responses:     errs(map[int]string{403: denied, 422: "VALIDATION_FAILED unless the list holds every current section exactly once."}),
	}), s.putSectionOrder)
}
