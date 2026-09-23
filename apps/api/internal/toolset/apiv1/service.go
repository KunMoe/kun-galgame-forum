package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/toolset/model"
	"kun-galgame-api/internal/toolset/repository"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/artifactclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

var errUnconfigured = errors.New("apiv1 toolset: service is not configured")

type blobStore interface {
	Delete(ctx context.Context, key string) error
}

type artifactAPI interface {
	InitUpload(ctx context.Context, req artifactclient.InitUploadRequest) (*artifactclient.InitUploadResponse, error)
	CompleteUpload(ctx context.Context, uuid string, req artifactclient.CompleteUploadRequest) (*artifactclient.ArtifactResponse, error)
	Resume(ctx context.Context, uuid string) (*artifactclient.ResumeUploadResponse, error)
	Download(ctx context.Context, uuid string) (*artifactclient.DownloadResponse, error)
	Delete(ctx context.Context, uuid string) error
}

type AwardFunc func(userID, delta int, reason, ref, idempotencyKey string)

type Service struct {
	store    *repository.Store
	users    *userclient.Client
	convert  *content.Converter
	check    *gate.CheckService
	scan     *gate.ScanService
	artifact artifactAPI
	blobs    blobStore
	award    AwardFunc
	cdn      string
}

func New(
	store *repository.Store,
	users *userclient.Client,
	convert *content.Converter,
	check *gate.CheckService,
	scan *gate.ScanService,
	artifact artifactAPI,
	blobs blobStore,
	award AwardFunc,
	cdn string,
) *Service {
	if check == nil {
		check = gate.NewCheckService(nil)
	}
	if scan == nil {
		scan = gate.NewScanService(nil)
	}
	if award == nil {
		award = moemoepoint.Award
	}
	return &Service{
		store: store, users: users, convert: convert,
		check: check, scan: scan, artifact: artifact, blobs: blobs,
		award: award, cdn: cdn,
	}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.store == nil || !s.store.Ready() || s.users == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

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

func (s *Service) lookupUsers(ctx context.Context, ids []int) (map[int]userclient.User, *problem.Problem) {
	if s.users == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	users, err := s.users.Users(ctx, ids)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	if users == nil {
		users = map[int]userclient.User{}
	}
	return users, nil
}

func renderable(users map[int]userclient.User, id int) bool {
	u, ok := users[id]
	return ok && userclient.IsRenderable(u)
}

func (s *Service) convertBody(ctx context.Context, source string) (content.ContentDocument, *problem.Problem) {
	if s.convert == nil {
		return content.ContentDocument{}, problem.Internal(errUnconfigured)
	}
	docs, err := s.convert.Convert(ctx, []string{source})
	if err != nil {
		return content.ContentDocument{}, problem.Unavailable(err)
	}
	return docs[0], nil
}

func homepageURLs(raw json.RawMessage) ([]string, error) {
	out := []string{}
	if len(raw) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

func (s *Service) summaries(ctx context.Context, rows []model.GalgameToolset) ([]ToolsetSummary, *problem.Problem) {
	if len(rows) == 0 {
		return []ToolsetSummary{}, nil
	}
	ids := make([]int, len(rows))
	authorIDs := make([]int, 0, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
		authorIDs = append(authorIDs, r.UserID)
	}
	users, p := s.lookupUsers(ctx, authorIDs)
	if p != nil {
		return nil, p
	}
	aliases, err := s.store.Aliases(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	prac, err := s.store.Practicality(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	dls, err := s.store.DownloadSums(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	out := make([]ToolsetSummary, 0, len(rows))
	for _, r := range rows {
		item, p := s.summary(r, users, aliases[r.ID], prac[r.ID], dls[r.ID])
		if p != nil {
			return nil, p
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) summary(
	r model.GalgameToolset,
	users map[int]userclient.User,
	aliases []string,
	prac repository.PracticalityAgg,
	downloads int,
) (ToolsetSummary, *problem.Problem) {
	homes, err := homepageURLs(r.Homepage)
	if err != nil {
		return ToolsetSummary{}, problem.Internal(err)
	}
	if aliases == nil {
		aliases = []string{}
	}
	dist := distOf(prac)
	if prac.Count == 0 {
		dist = emptyDist()
	}
	author, ok := users[r.UserID]
	if !ok {
		author = userclient.User{ID: r.UserID}
	}
	return ToolsetSummary{
		Object: "toolset", ID: repr.ID(r.ID), Name: r.Name, Aliases: aliases,
		Type: r.Type, Language: r.Language, Platform: r.Platform, Version: r.Version,
		HomepageURLs: homes, Author: repr.NewUserRef(s.cdn, author),
		ViewCount: r.View, DownloadCount: downloads, CommentCount: r.CommentCount,
		PracticalityAverage: prac.Average, PracticalityCount: prac.Count, PracticalityDistribution: dist,
		CreatedAt: repr.Timestamp(r.CreatedAt), UpdatedAt: repr.Timestamp(r.UpdatedAt),
		EditedAt: repr.TimestampPtr(r.Edited), ResourceUpdatedAt: repr.Timestamp(r.ResourceUpdateTime),
	}, nil
}

func (s *Service) visibleToolset(ctx context.Context, rawID string) (*model.GalgameToolset, map[int]userclient.User, *problem.Problem) {
	id, ok := parseID(rawID)
	if !ok {
		return nil, nil, notFound()
	}
	row, err := s.store.Find(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, notFound()
		}
		return nil, nil, problem.Internal(err)
	}
	users, p := s.lookupUsers(ctx, []int{row.UserID})
	if p != nil {
		return nil, nil, p
	}
	if !renderable(users, row.UserID) {
		return nil, nil, notFound()
	}
	return row, users, nil
}

func (s *Service) detail(ctx context.Context, row *model.GalgameToolset, users map[int]userclient.User, viewer *middleware.UserInfo) (*Toolset, *problem.Problem) {
	sum, p := s.summaries(ctx, []model.GalgameToolset{*row})
	if p != nil {
		return nil, p
	}
	doc, p := s.convertBody(ctx, row.Description)
	if p != nil {
		return nil, p
	}
	contribIDs, err := s.store.ContributorIDs(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	resources, err := s.store.Resources(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	extra := append([]int{}, contribIDs...)
	for _, r := range resources {
		extra = append(extra, r.UserID)
	}
	more, p := s.lookupUsers(ctx, extra)
	if p != nil {
		return nil, p
	}
	for id, u := range more {
		users[id] = u
	}
	contributors := make([]repr.UserRef, 0, len(contribIDs))
	for _, id := range contribIDs {
		if !renderable(users, id) {
			continue
		}
		contributors = append(contributors, repr.NewUserRef(s.cdn, users[id]))
	}
	resOut, p := s.resourceSummaries(resources, users, viewer)
	if p != nil {
		return nil, p
	}
	item := sum[0]
	out := Toolset{
		Object: item.Object, ID: item.ID, Name: item.Name, Aliases: item.Aliases,
		Type: item.Type, Language: item.Language, Platform: item.Platform, Version: item.Version,
		HomepageURLs: item.HomepageURLs, Author: item.Author,
		ViewCount: item.ViewCount, DownloadCount: item.DownloadCount, CommentCount: item.CommentCount,
		PracticalityAverage: item.PracticalityAverage, PracticalityCount: item.PracticalityCount,
		PracticalityDistribution: item.PracticalityDistribution,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, EditedAt: item.EditedAt,
		ResourceUpdatedAt: item.ResourceUpdatedAt,
		Content: doc, Contributors: contributors, Resources: resOut,
	}
	if viewer != nil {
		var rating *int
		r, err := s.store.UserRating(row.ID, viewer.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		rating = r
		out.Viewer = &ToolsetViewer{
			CanEdit: canEditToolset(row.UserID, viewer),
			CanDelete: canDeleteToolset(row.UserID, viewer),
			PracticalityRating: rating,
		}
	}
	return &out, nil
}

func (s *Service) resourceSummaries(
	rows []model.GalgameToolsetResource,
	users map[int]userclient.User,
	viewer *middleware.UserInfo,
) ([]ToolsetResourceSummary, *problem.Problem) {
	uuids := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.ArtifactUUID != "" {
			uuids = append(uuids, r.ArtifactUUID)
		}
	}
	uploads, err := s.store.UploadsByUUIDs(uuids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	out := make([]ToolsetResourceSummary, 0, len(rows))
	for _, r := range rows {
		if !renderable(users, r.UserID) {
			continue
		}
		out = append(out, s.resourceSummary(r, users, uploads, viewer))
	}
	return out, nil
}

func (s *Service) resourceSummary(
	r model.GalgameToolsetResource,
	users map[int]userclient.User,
	uploads map[string]model.ToolsetUpload,
	viewer *middleware.UserInfo,
) ToolsetResourceSummary {
	kind := "link"
	if r.Type == "s3" {
		kind = "file"
	}
	item := ToolsetResourceSummary{
		Object: "toolset_resource", ID: repr.ID(r.ID), ResourceType: kind,
		FileSize: fileSizeOf(r, uploads), SizeLabel: sizeLabelOf(r),
		Note: r.Note, DownloadCount: r.Download,
		Poster: repr.NewUserRef(s.cdn, users[r.UserID]),
		CreatedAt: repr.Timestamp(r.CreatedAt),
	}
	if viewer != nil {
		item.Viewer = &ResourceViewer{
			CanEdit:   canEditResource(r.UserID, viewer),
			CanDelete: canDeleteResource(r.UserID, viewer),
		}
	}
	return item
}

func (s *Service) rejectContent(ctx context.Context, text string, authorID int) (decision string, matched []string, p *problem.Problem) {
	if stringsTrimEmpty(text) {
		return gate.DecisionAllow, nil, nil
	}
	aid := int64(authorID)
	decision, matched = s.check.Decision(ctx, text, &aid)
	if decision == gate.DecisionDeny {
		return decision, matched, contentRejected()
	}
	return decision, matched, nil
}

func stringsTrimEmpty(s string) bool {
	return trimName(s) == ""
}

func (s *Service) pushPoints(userID, id, delta int, reason, keyPrefix, refKind string) {
	s.award(userID, delta, reason, moemoepoint.Ref(refKind, id), moemoepoint.Key(keyPrefix, strconv.Itoa(id)))
}
