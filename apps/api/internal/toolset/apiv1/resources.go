package apiv1

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

func (s *Service) visibleResource(ctx context.Context, toolsetRaw, resourceRaw string) (*model.GalgameToolset, *model.GalgameToolsetResource, map[int]userclient.User, *problem.Problem) {
	row, users, p := s.visibleToolset(ctx, toolsetRaw)
	if p != nil {
		return nil, nil, nil, p
	}
	rid, ok := parseID(resourceRaw)
	if !ok {
		return nil, nil, nil, notFound()
	}
	res, err := s.store.FindResource(rid)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, nil, notFound()
		}
		return nil, nil, nil, problem.Internal(err)
	}
	if res.ToolsetID != row.ID {
		return nil, nil, nil, notFound()
	}
	more, p := s.lookupUsers(ctx, []int{res.UserID})
	if p != nil {
		return nil, nil, nil, p
	}
	for id, u := range more {
		users[id] = u
	}
	if !renderable(users, res.UserID) {
		return nil, nil, nil, notFound()
	}
	return row, res, users, nil
}

func (s *Service) getToolsetResource(ctx context.Context, in *resourceIDInput) (*resourceOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	_, res, users, p := s.visibleResource(ctx, in.ToolsetID, in.ResourceID)
	if p != nil {
		return nil, p
	}
	items, p := s.resourceSummaries([]model.GalgameToolsetResource{*res}, users, v1.User(ctx))
	if p != nil {
		return nil, p
	}
	if len(items) == 0 {
		return nil, notFound()
	}
	return &resourceOutput{Body: items[0]}, nil
}

func (s *Service) createToolsetResource(ctx context.Context, in *createResourceInput) (*createResourceOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	toolset, _, p := s.visibleToolset(ctx, in.ToolsetID)
	if p != nil {
		return nil, p
	}
	body := in.Body
	if fields := resourceCreateErrors(body); len(fields) > 0 {
		return nil, validationFailed(fields...)
	}

	row := model.GalgameToolsetResource{
		ToolsetID: toolset.ID,
		UserID:    user.ID,
		Note:      markdown.NormalizeStoredContent(deref(body.Note)),
		Code:      deref(body.ExtractionCode),
		Password:  deref(body.ArchivePassword),
	}
	var modParts []string
	if body.ResourceType == "file" {
		up, p := s.ownedCompletedUpload(toolset.ID, user.ID, deref(body.ArtifactID))
		if p != nil {
			return nil, p
		}
		row.Type = "s3"
		row.ArtifactUUID = up.ArtifactUUID
		row.Size = strconv.FormatInt(up.FileSize, 10)
		row.Content = ""
	} else {
		url := strings.TrimSpace(deref(body.URL))
		row.Type = "user"
		row.Content = url
		row.Size = deref(body.SizeLabel)
		modParts = append(modParts, url)
	}
	if row.Note != "" {
		modParts = append(modParts, row.Note)
	}
	moderation := gate.ComposeText(modParts...)
	if _, _, p := s.rejectContent(ctx, moderation, user.ID); p != nil {
		return nil, p
	}
	if err := s.store.CreateResource(&row); err != nil {
		if repository.UniqueViolation(err) {
			if body.ResourceType == "file" {
				return nil, alreadyExists("/artifact_id")
			}
			return nil, alreadyExists("/link_url")
		}
		return nil, problem.Internal(err)
	}
	s.pushPoints(user.ID, row.ID, 3, moemoepoint.ReasonContentApproved, "resource_create", "toolset_resource")
	s.scan.ScanBg(gate.SubjectKindToolsetResource, strconv.Itoa(row.ID), moderation, int64(user.ID))
	fresh, err := s.store.FindResource(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	users, p := s.lookupUsers(ctx, []int{fresh.UserID})
	if p != nil {
		return nil, p
	}
	items, p := s.resourceSummaries([]model.GalgameToolsetResource{*fresh}, users, user)
	if p != nil {
		return nil, p
	}
	loc := "/api/v1/toolsets/" + strconv.Itoa(toolset.ID) + "/resources/" + strconv.Itoa(fresh.ID)
	return &createResourceOutput{Location: loc, Body: items[0]}, nil
}

func (s *Service) updateToolsetResource(ctx context.Context, in *patchResourceInput) (*resourceOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	_, res, users, p := s.visibleResource(ctx, in.ToolsetID, in.ResourceID)
	if p != nil {
		return nil, p
	}
	if !canEditResource(res.UserID, user) {
		return nil, permissionRequired()
	}
	patch := in.Body
	if patch.URL == nil && patch.SizeLabel == nil && patch.ExtractionCode == nil && patch.ArchivePassword == nil && patch.Note == nil {
		items, p := s.resourceSummaries([]model.GalgameToolsetResource{*res}, users, user)
		if p != nil {
			return nil, p
		}
		return &resourceOutput{Body: items[0]}, nil
	}
	var errs []problem.FieldError
	file := res.Type == "s3"
	if file {
		if patch.URL != nil {
			errs = append(errs, immutable("/link_url"))
		}
		if patch.ExtractionCode != nil {
			errs = append(errs, immutable("/extraction_code"))
		}
		if patch.SizeLabel != nil {
			errs = append(errs, immutable("/size_label"))
		}
	}
	fields := map[string]any{}
	var modParts []string
	if !file && patch.URL != nil {
		url := strings.TrimSpace(*patch.URL)
		if url == "" {
			errs = append(errs, tooShort("/link_url", 1))
		} else if len(url) > maxURL {
			errs = append(errs, tooLong("/link_url", maxURL))
		} else if !validDownloadLink(url) {
			errs = append(errs, invalidFormat("/link_url", "must be a download link"))
		} else {
			fields["content"] = url
			modParts = append(modParts, url)
		}
	}
	if !file && patch.ExtractionCode != nil {
		if len(*patch.ExtractionCode) > maxNote {
			errs = append(errs, tooLong("/extraction_code", maxNote))
		} else {
			fields["code"] = *patch.ExtractionCode
		}
	}
	if !file && patch.SizeLabel != nil {
		if len(*patch.SizeLabel) > maxSizeLabel {
			errs = append(errs, tooLong("/size_label", maxSizeLabel))
		} else {
			fields["size"] = *patch.SizeLabel
		}
	}
	if patch.ArchivePassword != nil {
		if len(*patch.ArchivePassword) > maxNote {
			errs = append(errs, tooLong("/archive_password", maxNote))
		} else {
			fields["password"] = *patch.ArchivePassword
		}
	}
	if patch.Note != nil {
		if len(*patch.Note) > maxNote {
			errs = append(errs, tooLong("/note", maxNote))
		} else {
			note := markdown.NormalizeStoredContent(*patch.Note)
			fields["note"] = note
			modParts = append(modParts, note)
		}
	}
	if len(errs) > 0 {
		return nil, validationFailed(errs...)
	}
	moderation := gate.ComposeText(modParts...)
	if len(modParts) > 0 {
		if _, _, p := s.rejectContent(ctx, moderation, res.UserID); p != nil {
			return nil, p
		}
	}
	now := time.Now()
	fields["edited"] = now
	fields["updated"] = now
	if err := s.store.PatchResource(res.ID, fields); err != nil {
		if repository.UniqueViolation(err) {
			return nil, alreadyExists("/link_url")
		}
		return nil, problem.Internal(err)
	}
	if len(modParts) > 0 {
		s.scan.ScanBg(gate.SubjectKindToolsetResource, strconv.Itoa(res.ID), moderation, int64(res.UserID))
	}
	fresh, err := s.store.FindResource(res.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.resourceSummaries([]model.GalgameToolsetResource{*fresh}, users, user)
	if p != nil {
		return nil, p
	}
	return &resourceOutput{Body: items[0]}, nil
}

func (s *Service) deleteToolsetResource(ctx context.Context, in *resourceIDInput) (*noContentOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	_, res, _, p := s.visibleResource(ctx, in.ToolsetID, in.ResourceID)
	if p != nil {
		return nil, p
	}
	if !canDeleteResource(res.UserID, user) {
		return nil, permissionRequired()
	}
	if p := s.deleteFileObjects(ctx, []model.GalgameToolsetResource{*res}); p != nil {
		return nil, p
	}
	if err := s.store.DeleteResource(res.ID); err != nil {
		return nil, problem.Internal(err)
	}
	s.pushPoints(res.UserID, res.ID, -3, moemoepoint.ReasonContentRemoved, "resource_delete", "toolset_resource")
	return &noContentOutput{}, nil
}

func (s *Service) createToolsetDownload(ctx context.Context, in *resourceIDInput) (*downloadOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	_, res, _, p := s.visibleResource(ctx, in.ToolsetID, in.ResourceID)
	if p != nil {
		return nil, p
	}
	out := ToolsetDownload{
		Object:          "toolset_download",
		ExtractionCode:  res.Code,
		ArchivePassword: res.Password,
	}
	if res.Type == "s3" {
		if res.ArtifactUUID == "" || nilIface(s.artifact) {
			return nil, problem.Unavailable(errUnconfigured)
		}
		dl, err := s.artifact.Download(ctx, res.ArtifactUUID)
		if err != nil {
			return nil, problem.Unavailable(err)
		}
		out.URL = dl.Url
		if dl.ExpiresAt != nil {
			out.ExpiresAt = parseInstant(*dl.ExpiresAt)
		}
	} else {
		out.URL = res.Content
	}
	if err := s.store.IncrementDownload(res.ID); err != nil {
		return nil, problem.Internal(err)
	}
	return &downloadOutput{Body: out}, nil
}

func (s *Service) ownedCompletedUpload(toolsetID, userID int, uuid string) (*model.ToolsetUpload, *problem.Problem) {
	if uuid == "" {
		return nil, validationFailed(requiredField("/artifact_id"))
	}
	up, err := s.store.FindUpload(toolsetID, uuid)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, validationFailed(unknownRef("/artifact_id"))
		}
		return nil, problem.Internal(err)
	}
	if up.UserID != userID || up.CompletedAt == nil {
		return nil, validationFailed(unknownRef("/artifact_id"))
	}
	return up, nil
}

func resourceCreateErrors(body ToolsetResourceCreate) []problem.FieldError {
	var errs []problem.FieldError
	switch body.ResourceType {
	case "file":
		if body.URL != nil {
			errs = append(errs, inconsistent("/link_url", "/resource_type"))
		}
		if body.SizeLabel != nil {
			errs = append(errs, inconsistent("/size_label", "/resource_type"))
		}
		if body.ArtifactID == nil || strings.TrimSpace(*body.ArtifactID) == "" {
			errs = append(errs, requiredField("/artifact_id"))
		}
	case "link":
		if body.ArtifactID != nil {
			errs = append(errs, inconsistent("/artifact_id", "/resource_type"))
		}
		if body.URL == nil || strings.TrimSpace(*body.URL) == "" {
			errs = append(errs, requiredField("/link_url"))
		} else if len(*body.URL) > maxURL {
			errs = append(errs, tooLong("/link_url", maxURL))
		} else if !validDownloadLink(strings.TrimSpace(*body.URL)) {
			errs = append(errs, invalidFormat("/link_url", "must be a download link"))
		}
		if body.SizeLabel == nil {
			errs = append(errs, requiredField("/size_label"))
		} else if len(*body.SizeLabel) > maxSizeLabel {
			errs = append(errs, tooLong("/size_label", maxSizeLabel))
		}
	}
	if body.ExtractionCode != nil && len(*body.ExtractionCode) > maxNote {
		errs = append(errs, tooLong("/extraction_code", maxNote))
	}
	if body.ArchivePassword != nil && len(*body.ArchivePassword) > maxNote {
		errs = append(errs, tooLong("/archive_password", maxNote))
	}
	if body.Note != nil && len(*body.Note) > maxNote {
		errs = append(errs, tooLong("/note", maxNote))
	}
	return errs
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
