package apiv1

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"time"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/resourcevocab"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/utils"

	"gorm.io/gorm"
)

func (s *Service) createWorkResource(ctx context.Context, in *createWorkResourceInput) (*createWorkResourceOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	workID, ok := parseID(in.WorkID)
	if !ok {
		return nil, notFound()
	}
	if _, p := s.lookupWork(ctx, workID); p != nil {
		return nil, p
	}
	banned, err := s.store.IsPublishBanned(workID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if banned {
		return nil, resourcePublishBanned()
	}
	body := in.Body
	ax, errs := validateAxes(body.ResourceType, strs(body.ResourceLanguages), strs(body.ResourcePlatforms), strs(body.ResourceRuntimes), body.Title, true)
	size, serrs := validateSize(body.Size, "/size")
	errs = append(errs, serrs...)
	urls, uerrs := validateURLs(body.DownloadURLs, true)
	errs = append(errs, uerrs...)
	note := markdown.NormalizeStoredContent(body.ContentMarkdown)
	if len(body.ContentMarkdown) > maxNote {
		errs = append(errs, tooLong("/content_markdown", maxNote))
	}
	if len(body.ExtractionCode) > maxCode {
		errs = append(errs, tooLong("/extraction_code", maxCode))
	}
	if len(body.ArchivePassword) > maxCode {
		errs = append(errs, tooLong("/archive_password", maxCode))
	}
	if len(errs) > 0 {
		return nil, validationFailed(errs...)
	}
	ax.VersionLabel = versionStored(body.VersionLabel)
	moderation := gate.ComposeText(append([]string{note}, urls...)...)
	decision, matched, p := s.rejectContent(ctx, moderation, user.ID)
	if p != nil {
		return nil, p
	}
	row := model.GalgameResource{
		Type: ax.Type, Title: ax.Title, VersionLabel: ax.VersionLabel,
		Language: ax.Language, Platform: ax.Platform,
		Languages: ax.Languages, Platforms: ax.Platforms, Runtimes: ax.Runtimes,
		Size: size, Code: body.ExtractionCode, Password: body.ArchivePassword,
		Note: note, WorkID: workID, UserID: user.ID,
	}
	var flipped bool
	err = s.store.InTx(func(tx *gorm.DB) error {
		was, err := s.store.LocalPublished(tx, workID)
		if err != nil {
			return err
		}
		if err := s.store.PublishLocal(tx, workID); err != nil {
			return err
		}
		flipped = !was
		if err := s.store.Create(tx, &row); err != nil {
			return err
		}
		if err := s.store.ReplaceProviders(tx, row.ID, utils.DetectProvidersFromURLs(urls)); err != nil {
			return err
		}
		if err := s.store.ReplaceProviderNames(tx, row.ID, utils.DetectProviderNamesFromURLs(urls)); err != nil {
			return err
		}
		if err := s.store.CreateLinks(tx, row.ID, urls); err != nil {
			return err
		}
		if err := s.store.AdjustResourceCount(tx, workID, 1); err != nil {
			return err
		}
		return s.store.TouchResourceUpdate(tx, workID)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	s.award(user.ID, 3, moemoepoint.ReasonContentApproved,
		moemoepoint.Ref("galgame_resource", row.ID),
		moemoepoint.Key("galgame_resource_create", strconv.Itoa(row.ID)))
	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameResource, "subject_id", row.ID, "author_id", user.ID, "matched", matched)
	}
	s.scan.ScanBg(gate.SubjectKindGalgameResource, strconv.Itoa(row.ID), moderation, int64(user.ID))
	if flipped {
		s.claimAfterCreate(ctx, workID)
	}
	created, err := s.store.Find(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	out, p := s.one(ctx, created, user)
	if p != nil {
		return nil, p
	}
	return &createWorkResourceOutput{
		Location: "/api/v1/galgame-resources/" + strconv.Itoa(created.ID),
		Body:     out,
	}, nil
}

func (s *Service) claimAfterCreate(ctx context.Context, workID int) {
	token := accessToken(ctx)
	if token == "" {
		return
	}
	fn := s.claim()
	if fn == nil {
		return
	}
	if appErr := fn(ctx, token, int64(workID)); appErr != nil {
		if appErr.StatusCode == http.StatusConflict {
			slog.Info("resource: catalog work already claimed, skip silent adopt", "work_id", workID)
			return
		}
		slog.Warn("resource: silent catalog adopt failed", "work_id", workID, "error", appErr)
	}
}

func (s *Service) updateGalgameResource(ctx context.Context, in *patchResourceInput) (*resourceOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleResource(ctx, in.ResourceID)
	if p != nil {
		return nil, p
	}
	if !canEditResource(row.UserID, user) {
		return nil, permissionRequired()
	}
	banned, err := s.store.IsPublishBanned(row.WorkID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if banned {
		return nil, resourcePublishBanned()
	}
	patch := in.Body
	if patch.ResourceType == nil && patch.ResourceLanguages == nil && patch.ResourcePlatforms == nil &&
		patch.ResourceRuntimes == nil && patch.Title == nil && patch.VersionLabel == nil &&
		patch.Size == nil && patch.DownloadURLs == nil && patch.ExtractionCode == nil &&
		patch.ArchivePassword == nil && patch.ContentMarkdown == nil && patch.State == nil {
		out, p := s.one(ctx, row, user)
		if p != nil {
			return nil, p
		}
		return &resourceOutput{Body: out}, nil
	}

	var errs []problem.FieldError
	fields := map[string]any{}
	typ := row.Type
	if patch.ResourceType != nil {
		typ = *patch.ResourceType
		if !slices.Contains(resourcevocab.TypeKeys, typ) {
			errs = append(errs, unknownValue("/resource_type", resourcevocab.TypeKeys))
		} else {
			fields["type"] = typ
		}
	}
	langs := []string(row.Languages)
	if patch.ResourceLanguages != nil {
		cleaned, lerrs := uniqueVocab("/resource_languages", strs(patch.ResourceLanguages), resourcevocab.LanguageKeys, 1)
		errs = append(errs, lerrs...)
		langs = cleaned
		if len(lerrs) == 0 {
			fields["languages"] = resourcevocab.Keys(langs)
			fields["language"] = resourcevocab.CompatLanguage(langs)
		}
	}
	plats := []string(row.Platforms)
	runs := []string(row.Runtimes)
	if patch.ResourcePlatforms != nil {
		cleaned, perrs := uniqueVocab("/resource_platforms", strs(patch.ResourcePlatforms), resourcevocab.PlatformKeys, 0)
		errs = append(errs, perrs...)
		plats = cleaned
	}
	if patch.ResourceRuntimes != nil {
		cleaned, rerrs := uniqueVocab("/resource_runtimes", strs(patch.ResourceRuntimes), resourcevocab.RuntimeKeys, 0)
		errs = append(errs, rerrs...)
		runs = cleaned
	}
	if patch.ResourcePlatforms != nil || patch.ResourceRuntimes != nil {
		if len(plats) == 0 && len(runs) == 0 {
			errs = append(errs, inconsistent("/resource_platforms", "/resource_runtimes"))
		}
		if resourcevocab.HasRuntimeAxis(typ) && len(runs) == 0 {
			errs = append(errs, tooFew("/resource_runtimes", 1))
		}
		if !resourcevocab.HasRuntimeAxis(typ) && len(runs) > 0 {
			errs = append(errs, inconsistent("/resource_runtimes", "/resource_type"))
		}
		if len(errs) == 0 {
			fields["platforms"] = resourcevocab.Keys(plats)
			fields["runtimes"] = resourcevocab.Keys(runs)
			fields["platform"] = resourcevocab.CompatPlatform(plats, runs)
		}
	}
	if patch.Title != nil {
		t, terrs := validateTitle(*patch.Title, "/title")
		errs = append(errs, terrs...)
		if len(terrs) == 0 {
			fields["title"] = t
		}
	}
	if patch.VersionLabel != nil {
		fields["version_label"] = versionStored(patch.VersionLabel)
	}
	if patch.Size != nil {
		size, serrs := validateSize(*patch.Size, "/size")
		errs = append(errs, serrs...)
		if len(serrs) == 0 {
			fields["size"] = size
		}
	}
	var urls []string
	if patch.DownloadURLs != nil {
		cleaned, uerrs := validateURLs(patch.DownloadURLs, true)
		errs = append(errs, uerrs...)
		urls = cleaned
	}
	if patch.ExtractionCode != nil {
		if len(*patch.ExtractionCode) > maxCode {
			errs = append(errs, tooLong("/extraction_code", maxCode))
		} else {
			fields["code"] = *patch.ExtractionCode
		}
	}
	if patch.ArchivePassword != nil {
		if len(*patch.ArchivePassword) > maxCode {
			errs = append(errs, tooLong("/archive_password", maxCode))
		} else {
			fields["password"] = *patch.ArchivePassword
		}
	}
	var note string
	var noteSet bool
	if patch.ContentMarkdown != nil {
		if len(*patch.ContentMarkdown) > maxNote {
			errs = append(errs, tooLong("/content_markdown", maxNote))
		} else {
			note = markdown.NormalizeStoredContent(*patch.ContentMarkdown)
			noteSet = true
			fields["note"] = note
		}
	}
	if patch.State != nil {
		if *patch.State == "expired" {
			errs = append(errs, notAllowed("/state", "expired is reported through expiry-reports"))
		} else if *patch.State != "valid" {
			errs = append(errs, unknownValue("/state", []string{"valid"}))
		} else {
			fields["status"] = 0
		}
	}
	if len(errs) > 0 {
		return nil, validationFailed(errs...)
	}

	var modParts []string
	if noteSet {
		modParts = append(modParts, note)
	}
	if patch.DownloadURLs != nil {
		modParts = append(modParts, urls...)
	}
	moderation := gate.ComposeText(modParts...)
	decision, matched, p := gate.DecisionAllow, []string(nil), (*problem.Problem)(nil)
	if len(modParts) > 0 {
		decision, matched, p = s.rejectContent(ctx, moderation, row.UserID)
		if p != nil {
			return nil, p
		}
	}

	fields["edited"] = time.Now()
	err = s.store.InTx(func(tx *gorm.DB) error {
		if err := s.store.Patch(tx, row.ID, fields); err != nil {
			return err
		}
		if patch.DownloadURLs != nil {
			if err := s.store.DeleteLinks(tx, row.ID); err != nil {
				return err
			}
			if err := s.store.CreateLinks(tx, row.ID, urls); err != nil {
				return err
			}
			if err := s.store.ReplaceProviders(tx, row.ID, utils.DetectProvidersFromURLs(urls)); err != nil {
				return err
			}
			if err := s.store.ReplaceProviderNames(tx, row.ID, utils.DetectProviderNamesFromURLs(urls)); err != nil {
				return err
			}
		}
		return s.store.TouchResourceUpdate(tx, row.WorkID)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	if len(modParts) > 0 {
		if decision == gate.DecisionHold {
			slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameResource, "subject_id", row.ID, "author_id", row.UserID, "matched", matched)
		}
		s.scan.ScanBg(gate.SubjectKindGalgameResource, strconv.Itoa(row.ID), moderation, int64(row.UserID))
	}
	fresh, err := s.store.Find(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	out, p := s.one(ctx, fresh, user)
	if p != nil {
		return nil, p
	}
	return &resourceOutput{Body: out}, nil
}

func (s *Service) deleteGalgameResource(ctx context.Context, in *resourceIDInput) (*noContentOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleResource(ctx, in.ResourceID)
	if p != nil {
		return nil, p
	}
	if !canDeleteResource(row.UserID, user) {
		return nil, permissionRequired()
	}
	err := s.store.InTx(func(tx *gorm.DB) error {
		if err := s.store.Delete(tx, row.ID); err != nil {
			return err
		}
		return s.store.AdjustResourceCount(tx, row.WorkID, -1)
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	s.award(row.UserID, -3, moemoepoint.ReasonContentRemoved,
		moemoepoint.Ref("galgame_resource", row.ID),
		moemoepoint.Key("galgame_resource_delete", strconv.Itoa(row.ID)))
	return &noContentOutput{}, nil
}


