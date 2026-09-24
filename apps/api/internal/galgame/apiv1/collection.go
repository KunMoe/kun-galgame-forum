package apiv1

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
)

type collectionIDInput struct {
	CollectionID string `path:"collection_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog folder id."`
	IncludeNSFW  bool   `query:"include_nsfw" default:"false" doc:"When true, adult works are included in preview_covers. Default false."`
}

type collectionOutput struct {
	Body Collection
}

type createCollectionInput struct {
	Body collectionCreate
}

type collectionCreate struct {
	Title       string `json:"title" minLength:"1" maxLength:"100" doc:"Display name. 1–100 characters after trimming. Free text; never use it as a decision input."`
	Description string `json:"description" required:"false" maxLength:"500" doc:"Owner's note. Omitted is an empty string. Free text; never use it as a decision input."`
	Visibility  string `json:"visibility" enum:"private,public" required:"true" maxLength:"7" doc:"Must be sent. There is no default."`
	IsDefault   *bool  `json:"is_default,omitempty" doc:"When true, this collection becomes the owner's default and the previous default loses the flag. Omitted is false."`
}

type createCollectionOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new collection, /api/v1/collections/{collection_id}."`
	Body     Collection
}

type updateCollectionInput struct {
	CollectionID string `path:"collection_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog folder id."`
	Body         collectionPatch
}

type collectionPatch struct {
	Title       *string `json:"title,omitempty" minLength:"1" maxLength:"100" doc:"New display name. Free text; never use it as a decision input."`
	Description *string `json:"description,omitempty" maxLength:"500" doc:"New note; an empty string clears it. Free text; never use it as a decision input."`
	Visibility  *string `json:"visibility,omitempty" enum:"private,public" maxLength:"7" doc:"New visibility."`
	IsDefault   *bool   `json:"is_default,omitempty" doc:"Only true is accepted. false is refused."`
}

type noContentOutput struct{}

func (s *Service) createCollection(ctx context.Context, in *createCollectionInput) (*createCollectionOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	title := strings.TrimSpace(in.Body.Title)
	if title == "" {
		return nil, validationFailed(titleTooShort())
	}
	desc := in.Body.Description
	// Only what the request actually changes is scanned. The unchanged half
	// was scanned when it was set, and reading it back to recompose the pair
	// would need a folder read this path does not otherwise make.
	if p := s.checkCollectionText(ctx, gate.ComposeText(title, desc), user.ID); p != nil {
		return nil, p
	}
	write := catalogclient.FolderWrite{Name: &title, Description: &desc, Visibility: &in.Body.Visibility, IsDefault: in.Body.IsDefault}
	folder, err := s.catalog.CreateFolderKeyed(ctx, token, write, idempotencyKey(ctx))
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	if s.scan != nil {
		s.scan.ScanBg(gate.SubjectKindGalgameCollection, string(repr.ID(int(folder.ID))), gate.ComposeText(title, desc), int64(user.ID))
	}
	col, ok, p := s.toCollection(ctx, *folder, user, token, false, false)
	if p != nil {
		return nil, p
	}
	if !ok {
		return nil, notFound()
	}
	return &createCollectionOutput{
		Location: "/api/v1/collections/" + string(col.ID),
		Body:     *col,
	}, nil
}

func (s *Service) getCollection(ctx context.Context, in *collectionIDInput) (*collectionOutput, error) {
	folder, token, user, err := s.visibleFolder(ctx, in.CollectionID)
	if err != nil {
		return nil, err
	}
	col, ok, p := s.toCollection(ctx, *folder, user, token, false, in.IncludeNSFW)
	if p != nil {
		return nil, p
	}
	if !ok {
		return nil, notFound()
	}
	return &collectionOutput{Body: *col}, nil
}

func (s *Service) updateCollection(ctx context.Context, in *updateCollectionInput) (*collectionOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if in.Body.Title == nil && in.Body.Description == nil && in.Body.Visibility == nil && in.Body.IsDefault == nil {
		return nil, validationFailed(problem.AtPointer("", problem.ReasonRequired, "send at least one field to change", nil))
	}
	if in.Body.Title != nil {
		title := strings.TrimSpace(*in.Body.Title)
		if title == "" {
			return nil, validationFailed(titleTooShort())
		}
		in.Body.Title = &title
	}
	if in.Body.IsDefault != nil && !*in.Body.IsDefault {
		return nil, validationFailed(problem.AtPointer("/is_default", problem.ReasonNotAllowedValue, "false is not accepted", nil))
	}
	folderID, ok := parseCollectionID(in.CollectionID)
	if !ok {
		return nil, notFound()
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	moderationText := gate.ComposeText(derefOr(in.Body.Title, ""), derefOr(in.Body.Description, ""))
	if p := s.checkCollectionText(ctx, moderationText, user.ID); p != nil {
		return nil, p
	}
	write := catalogclient.FolderWrite{
		Name: in.Body.Title, Description: in.Body.Description,
		Visibility: in.Body.Visibility, IsDefault: in.Body.IsDefault,
	}
	mine, err := s.catalog.MyFolder(ctx, token, folderID)
	if err == nil {
		updated, pErr := s.catalog.PatchFolder(ctx, token, folderID, write)
		if pErr != nil {
			return nil, mapUserPlane(pErr, true)
		}
		if s.scan != nil && moderationText != "" {
			s.scan.ScanBg(gate.SubjectKindGalgameCollection, in.CollectionID, moderationText, int64(mine.OwnerUID))
		}
		col, ok, p := s.toCollection(ctx, *updated, user, token, false, false)
		if p != nil {
			return nil, p
		}
		if !ok {
			return nil, notFound()
		}
		return &collectionOutput{Body: *col}, nil
	}
	if errors.Is(err, catalogclient.ErrInsufficientScope) {
		return nil, mapUserPlane(err, false)
	}
	if !errors.Is(err, catalogclient.ErrNotFound) {
		return nil, mapUserPlane(err, false)
	}
	if !user.Can(perm.CollectionEditAny) {
		return nil, notFound()
	}
	if in.Body.IsDefault != nil {
		return nil, notFound()
	}
	updated, pErr := s.catalog.ModeratePatchFolder(ctx, token, folderID, catalogclient.FolderWrite{
		Name: in.Body.Title, Description: in.Body.Description, Visibility: in.Body.Visibility,
	})
	if pErr != nil {
		return nil, mapUserPlane(pErr, false)
	}
	if s.scan != nil && moderationText != "" {
		s.scan.ScanBg(gate.SubjectKindGalgameCollection, in.CollectionID, moderationText, int64(updated.OwnerUID))
	}
	col, ok, p := s.toCollection(ctx, *updated, user, token, false, false)
	if p != nil {
		return nil, p
	}
	if !ok {
		return nil, notFound()
	}
	return &collectionOutput{Body: *col}, nil
}

func (s *Service) deleteCollection(ctx context.Context, in *collectionIDInput) (*noContentOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	folderID, ok := parseCollectionID(in.CollectionID)
	if !ok {
		return nil, notFound()
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	mine, err := s.catalog.MyFolder(ctx, token, folderID)
	if err == nil {
		if mine.IsDefault {
			return nil, problem.New(problem.CodeInvalidStateTransition, "The default collection cannot be deleted.")
		}
		orphaned, lErr := s.worksLeavingTheLibrary(ctx, token, folderID)
		if lErr != nil {
			return nil, mapUserPlane(lErr, true)
		}
		if dErr := s.catalog.DeleteFolder(ctx, token, folderID); dErr != nil {
			return nil, mapUserPlane(dErr, true)
		}
		s.folderDeleteSideEffects(orphaned)
		return &noContentOutput{}, nil
	}
	if !errors.Is(err, catalogclient.ErrNotFound) {
		return nil, mapUserPlane(err, false)
	}
	if !user.Can(perm.CollectionDeleteAny) {
		return nil, notFound()
	}
	if dErr := s.catalog.ModerateDeleteFolder(ctx, token, folderID); dErr != nil {
		return nil, mapUserPlane(dErr, false)
	}
	return &noContentOutput{}, nil
}

// visibleFolder tries the public lane first even when the viewer owns the
// folder, and hands back no token for a public folder. A public folder is
// public to its owner too, and reading it with their token is what walked
// 3,560 items through /v2/me/folders/{id}/items on every page of
// /galgame/collection/4535 and spent user 90769's 10k/day quota (2026-09-20).
func (s *Service) visibleFolder(ctx context.Context, rawID string) (*catalogclient.Folder, string, *middleware.UserInfo, error) {
	folderID, ok := parseCollectionID(rawID)
	if !ok {
		return nil, "", nil, notFound()
	}
	if s.catalog == nil {
		return nil, "", nil, problem.Unavailable(errUnconfigured)
	}
	user := callerUser(ctx)
	token := accessToken(ctx)
	folder, err := s.catalog.PublicFolder(ctx, folderID)
	if err == nil {
		return folder, "", user, nil
	}
	if !errors.Is(err, catalogclient.ErrNotFound) {
		return nil, "", nil, mapUserPlane(err, false)
	}
	if user == nil {
		return nil, "", nil, notFound()
	}
	if token == "" {
		return nil, "", nil, problem.New(problem.CodeInvalidCredential, "The credential is invalid, expired, or revoked.")
	}
	folder, err = s.catalog.MyFolder(ctx, token, folderID)
	if err != nil {
		return nil, "", nil, mapUserPlane(err, true)
	}
	return folder, token, user, nil
}

func (s *Service) checkCollectionText(ctx context.Context, text string, authorID int) *problem.Problem {
	if text == "" || s.check == nil {
		return nil
	}
	id := int64(authorID)
	decision, matched := s.check.Decision(ctx, text, &id)
	if decision == gate.DecisionDeny {
		return problem.New(problem.CodeContentRejected, "The trust-and-safety check refused the submitted text. Nothing was written.")
	}
	if decision == gate.DecisionHold {
		slog.Info("trust check hold", "subject_kind", gate.SubjectKindGalgameCollection, "author_id", authorID, "matched", matched)
	}
	return nil
}

func derefOr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	return *p
}

func titleTooShort() problem.FieldError {
	min := 1
	return problem.AtPointer("/title", problem.ReasonTooShort, "at least 1 character once surrounding whitespace is removed",
		&problem.FieldParams{MinLength: &min})
}
