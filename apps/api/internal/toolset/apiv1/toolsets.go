package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) listToolsets(ctx context.Context, in *listToolsetsInput) (*listToolsetsOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	pg := pageOf(in.Page, in.Limit)
	if p := pg.CheckDepth(); p != nil {
		return nil, p
	}
	sort, ok := sortSpecs[in.Sort]
	if !ok {
		return nil, problem.New(problem.CodeUnknownSort, "The sort token is not in this collection's vocabulary.",
			problem.AtParameter("sort", problem.ReasonUnknownValue, "use a sort token declared by this operation", nil))
	}
	filter := repository.ListFilter{
		Type: in.Type, Language: in.Language, Platform: in.Platform, Version: in.Version, Q: in.Q,
	}
	authors, err := s.store.DistinctAuthors(filter)
	if err != nil {
		return nil, problem.Internal(err)
	}
	users, p := s.lookupUsers(ctx, authors)
	if p != nil {
		return nil, p
	}
	keep := make([]int, 0, len(authors))
	for _, id := range authors {
		if renderable(users, id) {
			keep = append(keep, id)
		}
	}
	filter.AuthorIDs = keep
	total, err := s.store.Count(filter)
	if err != nil {
		return nil, problem.Internal(err)
	}
	n, rel := collect.ClampTotal(total)
	rows, err := s.store.List(filter, sort, pg.Offset(), pg.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.summaries(ctx, rows)
	if p != nil {
		return nil, p
	}
	return &listToolsetsOutput{Body: repr.NewPageList(items, n, rel)}, nil
}

func (s *Service) listUserToolsets(ctx context.Context, in *listUserToolsetsInput) (*listToolsetsOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	pg := pageOf(in.Page, in.Limit)
	if p := pg.CheckDepth(); p != nil {
		return nil, p
	}
	userID, ok := parseID(in.UserID)
	if !ok {
		return nil, problem.New(problem.CodeInvalidParameter, "user_id must be a positive integer.",
			problem.AtParameter("user_id", problem.ReasonInvalidFormat, "user_id must be a positive integer", nil))
	}
	users, p := s.lookupUsers(ctx, []int{userID})
	if p != nil {
		return nil, p
	}
	if !renderable(users, userID) {
		return nil, notFound()
	}
	filter := repository.ListFilter{UserID: userID}
	total, err := s.store.Count(filter)
	if err != nil {
		return nil, problem.Internal(err)
	}
	n, rel := collect.ClampTotal(total)
	rows, err := s.store.List(filter, repository.SortSpec{Column: "created", Desc: true}, pg.Offset(), pg.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.summaries(ctx, rows)
	if p != nil {
		return nil, p
	}
	return &listToolsetsOutput{Body: repr.NewPageList(items, n, rel)}, nil
}

func (s *Service) getToolset(ctx context.Context, in *toolsetIDInput) (*toolsetOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	row, users, p := s.visibleToolset(ctx, in.ToolsetID)
	if p != nil {
		return nil, p
	}
	if err := s.store.IncrementView(row.ID); err != nil {
		return nil, problem.Internal(err)
	}
	row.View++
	out, p := s.detail(ctx, row, users, v1.User(ctx))
	if p != nil {
		return nil, p
	}
	return &toolsetOutput{Body: *out}, nil
}

func (s *Service) getToolsetSource(ctx context.Context, in *toolsetIDInput) (*sourceOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleToolset(ctx, in.ToolsetID)
	if p != nil {
		return nil, p
	}
	if !canEditToolset(row.UserID, user) {
		return nil, permissionRequired()
	}
	return &sourceOutput{Body: ToolsetSource{
		Object: "toolset_source", ToolsetID: repr.ID(row.ID), ContentMarkdown: row.Description,
	}}, nil
}

func (s *Service) createToolset(ctx context.Context, in *createToolsetInput) (*createToolsetOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	name, aliases, homes, markdownSrc, fields, p := validateToolsetWrite(in.Body.Name, in.Body.ContentMarkdown, in.Body.Aliases, in.Body.HomepageURLs, true)
	if p != nil {
		return nil, p
	}
	if len(fields) > 0 {
		return nil, validationFailed(fields...)
	}
	moderation := gate.ComposeText(append([]string{name, markdownSrc}, aliases...)...)
	if _, _, p := s.rejectContent(ctx, moderation, user.ID); p != nil {
		return nil, p
	}
	raw, err := json.Marshal(homes)
	if err != nil {
		return nil, problem.Internal(err)
	}
	row := model.GalgameToolset{
		Name: name, Description: markdownSrc, Type: in.Body.Type, Language: in.Body.Language,
		Platform: in.Body.Platform, Homepage: raw, Version: in.Body.Version, UserID: user.ID,
	}
	if err := s.store.CreateToolset(&row, aliases); err != nil {
		if repository.UniqueViolation(err) {
			return nil, alreadyExists("/aliases")
		}
		return nil, problem.Internal(err)
	}
	s.pushPoints(user.ID, row.ID, 3, moemoepoint.ReasonContentApproved, "toolset_create", "toolset")
	s.scan.ScanBg(gate.SubjectKindToolset, strconv.Itoa(row.ID), moderation, int64(user.ID))
	created, err := s.store.Find(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	users, p := s.lookupUsers(ctx, []int{created.UserID})
	if p != nil {
		return nil, p
	}
	out, p := s.detail(ctx, created, users, user)
	if p != nil {
		return nil, p
	}
	return &createToolsetOutput{Location: "/api/v1/toolsets/" + strconv.Itoa(created.ID), Body: *out}, nil
}

func (s *Service) updateToolset(ctx context.Context, in *patchToolsetInput) (*toolsetOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, users, p := s.visibleToolset(ctx, in.ToolsetID)
	if p != nil {
		return nil, p
	}
	if !canEditToolset(row.UserID, user) {
		return nil, permissionRequired()
	}
	patch := in.Body
	if patch.Name == nil && patch.ContentMarkdown == nil && patch.Type == nil && patch.Language == nil &&
		patch.Platform == nil && patch.Version == nil && patch.Aliases == nil && patch.HomepageURLs == nil {
		out, p := s.detail(ctx, row, users, user)
		if p != nil {
			return nil, p
		}
		return &toolsetOutput{Body: *out}, nil
	}

	fields := map[string]any{}
	var aliases []string
	var modParts []string
	var errs []problem.FieldError

	if patch.Name != nil {
		if len(*patch.Name) > maxName {
			errs = append(errs, tooLong("/name", maxName))
		} else {
			name := trimName(*patch.Name)
			if name == "" {
				errs = append(errs, tooShort("/name", 1))
			} else {
				fields["name"] = name
				modParts = append(modParts, name)
			}
		}
	}
	if patch.ContentMarkdown != nil {
		src := markdown.NormalizeStoredContent(*patch.ContentMarkdown)
		if len(*patch.ContentMarkdown) > maxMarkdown {
			errs = append(errs, tooLong("/content_markdown", maxMarkdown))
		} else {
			fields["description"] = src
			modParts = append(modParts, src)
		}
	}
	if patch.Type != nil {
		fields["type"] = *patch.Type
	}
	if patch.Language != nil {
		fields["language"] = *patch.Language
	}
	if patch.Platform != nil {
		fields["platform"] = *patch.Platform
	}
	if patch.Version != nil {
		fields["version"] = *patch.Version
	}
	if patch.Aliases != nil {
		cleaned, ferrs := validateAliases(patch.Aliases)
		errs = append(errs, ferrs...)
		aliases = cleaned
		modParts = append(modParts, cleaned...)
	}
	if patch.HomepageURLs != nil {
		cleaned, ferrs := validateHomepages(patch.HomepageURLs)
		errs = append(errs, ferrs...)
		if len(ferrs) == 0 {
			raw, err := json.Marshal(cleaned)
			if err != nil {
				return nil, problem.Internal(err)
			}
			fields["homepage"] = raw
		}
	}
	if len(errs) > 0 {
		return nil, validationFailed(errs...)
	}

	moderation := gate.ComposeText(modParts...)
	if len(modParts) > 0 {
		if _, _, p := s.rejectContent(ctx, moderation, row.UserID); p != nil {
			return nil, p
		}
	}

	now := time.Now()
	fields["edited"] = now
	fields["updated"] = now
	if err := s.store.PatchToolset(row.ID, fields, aliases); err != nil {
		if repository.UniqueViolation(err) {
			return nil, alreadyExists("/aliases")
		}
		if errors.Is(err, repository.ErrNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	if len(modParts) > 0 {
		s.scan.ScanBg(gate.SubjectKindToolset, strconv.Itoa(row.ID), moderation, int64(row.UserID))
	}
	fresh, err := s.store.Find(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	out, p := s.detail(ctx, fresh, users, user)
	if p != nil {
		return nil, p
	}
	return &toolsetOutput{Body: *out}, nil
}

func (s *Service) deleteToolset(ctx context.Context, in *toolsetIDInput) (*noContentOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleToolset(ctx, in.ToolsetID)
	if p != nil {
		return nil, p
	}
	if !canDeleteToolset(row.UserID, user) {
		return nil, permissionRequired()
	}
	files, err := s.store.FileResources(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if p := s.deleteFileObjects(ctx, files); p != nil {
		return nil, p
	}
	if err := s.store.DeleteToolset(row.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	s.pushPoints(row.UserID, row.ID, -3, moemoepoint.ReasonContentRemoved, "toolset_delete", "toolset")
	return &noContentOutput{}, nil
}

func (s *Service) deleteFileObjects(ctx context.Context, files []model.GalgameToolsetResource) *problem.Problem {
	for _, r := range files {
		if r.ArtifactUUID != "" {
			if nilIface(s.artifact) {
				return problem.Unavailable(errUnconfigured)
			}
			if err := s.artifact.Delete(ctx, r.ArtifactUUID); err != nil {
				return problem.Unavailable(err)
			}
			continue
		}
		if r.Code == "" {
			continue
		}
		if nilIface(s.blobs) {
			return problem.Unavailable(errUnconfigured)
		}
		if err := s.blobs.Delete(ctx, r.Code); err != nil {
			return problem.Unavailable(err)
		}
	}
	return nil
}

func validateToolsetWrite(name, body string, aliases, homes []string, creating bool) (string, []string, []string, string, []problem.FieldError, *problem.Problem) {
	var errs []problem.FieldError
	if creating {
		if len(name) > maxName {
			errs = append(errs, tooLong("/name", maxName))
		}
		trimmed := trimName(name)
		if trimmed == "" {
			errs = append(errs, tooShort("/name", 1))
		}
		name = trimmed
	}
	src := markdown.NormalizeStoredContent(body)
	cleanedAliases, aerr := validateAliases(aliases)
	errs = append(errs, aerr...)
	cleanedHomes, herr := validateHomepages(homes)
	errs = append(errs, herr...)
	if len(errs) > 0 {
		return name, cleanedAliases, cleanedHomes, src, errs, nil
	}
	return name, cleanedAliases, cleanedHomes, src, nil, nil
}

func validateAliases(in []string) ([]string, []problem.FieldError) {
	if in == nil {
		return []string{}, nil
	}
	if len(in) > maxAliases {
		return nil, []problem.FieldError{tooMany("/aliases", maxAliases)}
	}
	var errs []problem.FieldError
	out := make([]string, 0, len(in))
	seen := map[string]int{}
	for i, raw := range in {
		ptr := "/aliases/" + strconv.Itoa(i)
		if len(raw) > maxName {
			errs = append(errs, tooLong(ptr, maxName))
			continue
		}
		name := trimName(raw)
		if name == "" {
			errs = append(errs, tooShort(ptr, 1))
			continue
		}
		if prev, ok := seen[name]; ok {
			errs = append(errs, duplicateItem(ptr))
			_ = prev
			continue
		}
		seen[name] = i
		out = append(out, name)
	}
	return out, errs
}

func validateHomepages(in []string) ([]string, []problem.FieldError) {
	if in == nil {
		return []string{}, nil
	}
	if len(in) > maxHomepages {
		return nil, []problem.FieldError{tooMany("/homepage_urls", maxHomepages)}
	}
	var errs []problem.FieldError
	out := make([]string, 0, len(in))
	for i, raw := range in {
		ptr := "/homepage_urls/" + strconv.Itoa(i)
		if len(raw) > maxHomepage {
			errs = append(errs, tooLong(ptr, maxHomepage))
			continue
		}
		if !validHomepage(raw) {
			errs = append(errs, invalidFormat(ptr, "must be an http or https URL"))
			continue
		}
		out = append(out, raw)
	}
	return out, errs
}
