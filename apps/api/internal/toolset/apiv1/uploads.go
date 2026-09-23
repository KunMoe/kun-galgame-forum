package apiv1

import (
	"context"
	"errors"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/pkg/artifactclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) createToolsetUpload(ctx context.Context, in *createUploadInput) (*createUploadOutput, error) {
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
	var errs []problem.FieldError
	if !archiveFilename(body.Filename) {
		errs = append(errs, invalidFormat("/filename", "filename must end in .7z, .zip, or .rar"))
	}
	if body.FileSize < minFileSize || body.FileSize > maxFileSize {
		errs = append(errs, outOfRange("/file_size", minFileSize, maxFileSize))
	}
	if len(errs) > 0 {
		return nil, validationFailed(errs...)
	}
	if !user.Can(perm.ToolsetUploadBypass) {
		used, moe, err := s.store.DailyUploadState(user.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		budget := int64(dailyBase) + int64(moe)*bytesPerMB
		if used+body.FileSize > budget {
			return nil, quotaExceeded()
		}
	}
	if nilIface(s.artifact) {
		return nil, problem.Unavailable(errUnconfigured)
	}
	public := true
	req := artifactclient.InitUploadRequest{Name: body.Filename, FileSize: body.FileSize, Public: &public}
	if body.ContentType != nil && *body.ContentType != "" {
		mime := *body.ContentType
		req.MimeType = &mime
	}
	init, err := s.artifact.InitUpload(ctx, req)
	if err != nil {
		return nil, mapArtifactProblem(err)
	}
	row := model.ToolsetUpload{
		ArtifactUUID: init.Uuid,
		ToolsetID:    toolset.ID,
		UserID:       user.ID,
		Filename:     body.Filename,
		FileSize:     body.FileSize,
		CreatedAt:    time.Now(),
	}
	if err := s.store.InsertUpload(&row); err != nil {
		return nil, problem.Internal(err)
	}
	fresh, err := s.store.FindUpload(toolset.ID, init.Uuid)
	if err != nil {
		return nil, problem.Internal(err)
	}
	out := uploadFromInit(*fresh, init)
	loc := "/api/v1/toolsets/" + strconv.Itoa(toolset.ID) + "/uploads/" + fresh.ArtifactUUID
	return &createUploadOutput{Location: loc, Body: out}, nil
}

func (s *Service) getToolsetUpload(ctx context.Context, in *uploadIDInput) (*uploadOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, p := s.ownedUpload(user.ID, in.ToolsetID, in.UploadID)
	if p != nil {
		return nil, p
	}
	if row.CompletedAt != nil {
		return &uploadOutput{Body: uploadCompleted(*row)}, nil
	}
	if nilIface(s.artifact) {
		return nil, problem.Unavailable(errUnconfigured)
	}
	resume, err := s.artifact.Resume(ctx, row.ArtifactUUID)
	if err != nil {
		return nil, mapArtifactProblem(err)
	}
	return &uploadOutput{Body: uploadFromResume(*row, resume)}, nil
}

func (s *Service) updateToolsetUpload(ctx context.Context, in *patchUploadInput) (*uploadOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, p := s.ownedUpload(user.ID, in.ToolsetID, in.UploadID)
	if p != nil {
		return nil, p
	}
	if in.Body.State != "completed" {
		return nil, invalidTransition("The upload is pending and can only move to completed.")
	}
	if row.CompletedAt != nil {
		return &uploadOutput{Body: uploadCompleted(*row)}, nil
	}
	if nilIface(s.artifact) {
		return nil, problem.Unavailable(errUnconfigured)
	}
	var parts *[]artifactclient.CompletedPart
	if len(in.Body.Parts) > 0 {
		cps := make([]artifactclient.CompletedPart, len(in.Body.Parts))
		for i, p := range in.Body.Parts {
			cps[i] = artifactclient.CompletedPart{Etag: p.Etag, PartNumber: int32(p.PartNumber)}
		}
		parts = &cps
	}
	art, err := s.artifact.CompleteUpload(ctx, row.ArtifactUUID, artifactclient.CompleteUploadRequest{Parts: parts})
	if err != nil {
		return nil, mapArtifactProblem(err)
	}
	size := row.FileSize
	if art != nil && art.FileSize > 0 {
		size = art.FileSize
	}
	if _, err := s.store.CompleteUpload(row.ArtifactUUID, user.ID, size); err != nil {
		return nil, problem.Internal(err)
	}
	fresh, err := s.store.FindUpload(row.ToolsetID, row.ArtifactUUID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &uploadOutput{Body: uploadCompleted(*fresh)}, nil
}

func (s *Service) deleteToolsetUpload(ctx context.Context, in *uploadIDInput) (*noContentOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, p := s.ownedUpload(user.ID, in.ToolsetID, in.UploadID)
	if p != nil {
		return nil, p
	}
	if row.CompletedAt != nil {
		return nil, invalidTransition("The upload is completed and cannot be aborted.")
	}
	if nilIface(s.artifact) {
		return nil, problem.Unavailable(errUnconfigured)
	}
	if err := s.artifact.Delete(ctx, row.ArtifactUUID); err != nil {
		return nil, problem.Unavailable(err)
	}
	if err := s.store.DeleteUpload(row.ArtifactUUID); err != nil {
		return nil, problem.Internal(err)
	}
	return &noContentOutput{}, nil
}

func (s *Service) ownedUpload(userID int, toolsetRaw, uploadID string) (*model.ToolsetUpload, *problem.Problem) {
	tid, ok := parseID(toolsetRaw)
	if !ok {
		return nil, notFound()
	}
	if _, err := s.store.Find(tid); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	row, err := s.store.FindUpload(tid, uploadID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	if row.UserID != userID {
		return nil, notFound()
	}
	return row, nil
}

func mapArtifactProblem(err error) error {
	switch {
	case errors.Is(err, artifactclient.ErrQuotaExceeded):
		return quotaExceeded()
	case errors.Is(err, artifactclient.ErrTooBig), errors.Is(err, artifactclient.ErrSizeMismatch):
		return validationFailed(outOfRange("/file_size", minFileSize, maxFileSize))
	case errors.Is(err, artifactclient.ErrMIMEDenied):
		return validationFailed(invalidFormat("/filename", "filename must end in .7z, .zip, or .rar"))
	case errors.Is(err, artifactclient.ErrNotFound):
		return notFound()
	default:
		return problem.Unavailable(err)
	}
}

func uploadCompleted(row model.ToolsetUpload) ToolsetUpload {
	state := "pending"
	if row.CompletedAt != nil {
		state = "completed"
	}
	return ToolsetUpload{
		Object: "toolset_upload", ID: row.ArtifactUUID, ToolsetID: repr.ID(row.ToolsetID),
		Filename: row.Filename, FileSize: row.FileSize, State: state,
		Parts: []UploadPart{}, UploadedParts: []UploadedPart{},
		CreatedAt: repr.Timestamp(row.CreatedAt), CompletedAt: repr.TimestampPtr(row.CompletedAt),
	}
}

func uploadFromInit(row model.ToolsetUpload, init *artifactclient.InitUploadResponse) ToolsetUpload {
	out := uploadCompleted(row)
	out.State = "pending"
	out.Multipart = init.Multipart
	out.ExpiresAt = parseInstant(init.ExpiresAt)
	out.PartSize = init.PartSize
	out.UploadURL = init.UploadUrl
	if init.PartUrls != nil {
		for _, p := range *init.PartUrls {
			out.Parts = append(out.Parts, UploadPart{PartNumber: int(p.PartNumber), URL: p.Url})
		}
	}
	if out.Parts == nil {
		out.Parts = []UploadPart{}
	}
	return out
}

func uploadFromResume(row model.ToolsetUpload, resume *artifactclient.ResumeUploadResponse) ToolsetUpload {
	out := uploadCompleted(row)
	out.State = "pending"
	out.Multipart = resume.Multipart
	out.ExpiresAt = parseInstant(resume.ExpiresAt)
	out.PartSize = resume.PartSize
	out.UploadURL = resume.UploadUrl
	if resume.PartUrls != nil {
		for _, p := range *resume.PartUrls {
			out.Parts = append(out.Parts, UploadPart{PartNumber: int(p.PartNumber), URL: p.Url})
		}
	}
	if out.Parts == nil {
		out.Parts = []UploadPart{}
	}
	if resume.UploadedParts != nil {
		for _, p := range *resume.UploadedParts {
			out.UploadedParts = append(out.UploadedParts, UploadedPart{PartNumber: int(p.PartNumber), Etag: p.Etag, Size: p.Size})
		}
	}
	if out.UploadedParts == nil {
		out.UploadedParts = []UploadedPart{}
	}
	return out
}
