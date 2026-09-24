package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/galgame/workrepr"
	"kun-galgame-api/internal/infrastructure/storelink"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/catalogclient"
	legacyErrors "kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/moyuclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

const (
	moyuCacheKey = "moyu:patches:v1:"
	// The face's own s-maxage: it declares its answers shareable for this long.
	moyuCacheTTL = 30 * time.Minute
)

var errUnconfigured = errors.New("apiv1 galgames: service is not configured")

type workCatalog interface {
	CatalogWorkExists(ctx context.Context, workID int) (bool, *legacyErrors.AppError)
	CatalogRowsByWorkIDs(ctx context.Context, ids []int, include, contentLimit string) (map[int]client.CatalogWorkListItem, *legacyErrors.AppError)
}

type workDetailCatalog interface {
	CatalogWorkDetail(ctx context.Context, workID int) (*client.CatalogWorkDetail, bool, int64, *legacyErrors.AppError)
	CatalogLabel(ctx context.Context, id string) (*client.CatalogLabelDetail, bool, int64, *legacyErrors.AppError)
	CatalogEngine(ctx context.Context, id string) (*client.CatalogEngineDetail, bool, *legacyErrors.AppError)
	CatalogSeries(ctx context.Context, id string) (*client.CatalogSeriesDetail, bool, *legacyErrors.AppError)
	CatalogWorksSearch(ctx context.Context, q url.Values) (*client.CatalogWorksPage, *legacyErrors.AppError)
}

type CatalogUser interface {
	MyFoldersContaining(ctx context.Context, token string, workID int64) ([]catalogclient.Folder, error)
	MyFolderHoldings(ctx context.Context, token string, workIDs []int64) ([]catalogclient.FolderHolding, error)
	WorkCoversUser(ctx context.Context, token string, workID int64) ([]catalogclient.CoverTally, error)
	WorkCoverVotes(ctx context.Context, workID int64) ([]catalogclient.CoverTally, error)
	MyPlaytime(ctx context.Context, token string, workID int64) (*catalogclient.PlaytimeSelf, error)
	MyWorkState(ctx context.Context, token string, workID int64) (*catalogclient.WorkStateRecord, error)
	VoteCover(ctx context.Context, token string, workID, coverID int64) (*catalogclient.CoverVoteResult, error)
	UnvoteCover(ctx context.Context, token string, workID, coverID int64) (*catalogclient.CoverVoteResult, error)
	ReportPlaytime(ctx context.Context, token string, workID int64, report catalogclient.PlaytimeReport) (*catalogclient.PlaytimeRecord, error)
	DeleteMyPlaytime(ctx context.Context, token string, workID int64) error
	ListMyPlaytime(ctx context.Context, token, cursor string, limit int) ([]catalogclient.PlaytimeRecord, string, error)
	PutWorkState(ctx context.Context, token string, workID int64, state string, completion *string) (*catalogclient.WorkStateRecord, error)
	DeleteWorkState(ctx context.Context, token string, workID int64) error
	ListMyWorkStates(ctx context.Context, token, cursor string, limit int) ([]catalogclient.WorkStateRecord, string, error)
	MyFolders(ctx context.Context, token string) ([]catalogclient.Folder, error)
	MyFolder(ctx context.Context, token string, folderID int64) (*catalogclient.Folder, error)
	FolderPreviewItems(ctx context.Context, token string, folderID int64, n int) ([]catalogclient.FolderItem, error)
	MyFolderItems(ctx context.Context, token string, folderID int64) ([]catalogclient.FolderItem, error)
	PublicFolders(ctx context.Context, ownerUID int64) ([]catalogclient.Folder, error)
	PublicFolder(ctx context.Context, folderID int64) (*catalogclient.Folder, error)
	PublicFolderItems(ctx context.Context, folderID int64) ([]catalogclient.FolderItem, error)
	CreateFolderKeyed(ctx context.Context, token string, in catalogclient.FolderWrite, idempotencyKey string) (*catalogclient.Folder, error)
	PatchFolder(ctx context.Context, token string, folderID int64, in catalogclient.FolderWrite) (*catalogclient.Folder, error)
	DeleteFolder(ctx context.Context, token string, folderID int64) error
	PutFolderItem(ctx context.Context, token string, folderID, workID int64) error
	DeleteFolderItem(ctx context.Context, token string, folderID, workID int64) error
	ModeratePatchFolder(ctx context.Context, token string, folderID int64, in catalogclient.FolderWrite) (*catalogclient.Folder, error)
	ModerateDeleteFolder(ctx context.Context, token string, folderID int64) error
}

type userLookup interface {
	Users(ctx context.Context, ids []int) (map[int]userclient.User, error)
}

type AwardFunc func(userID, delta int, reason, ref, idempotencyKey string)

type pendingAward struct {
	userID int
	delta  int
	reason string
	ref    string
	key    string
}

type Service struct {
	works           workCatalog
	moyu            *moyuclient.Client
	users           userLookup
	rdb             *redis.Client
	cdn             string
	store           *repository.WorkV1Store
	hydrator        *workrepr.Hydrator
	lists           *repository.GalgameListRepository
	collectedMonths func(bool) ([]repository.CollectedMonth, error)
	catalog         CatalogUser
	storeLinks      *storelink.Resolver
	award           AwardFunc
	check           *gate.CheckService
	scan            *gate.ScanService
	aliases         *repository.GalgameCollectionRepository
	popFlight       singleflight.Group
}

func New(works workCatalog, moyu *moyuclient.Client, users userLookup, rdb *redis.Client, cdn string) *Service {
	return &Service{works: works, moyu: moyu, users: users, rdb: rdb, cdn: cdn, award: moemoepoint.Award}
}

func (s *Service) WithWork(db *gorm.DB, catalog CatalogUser, storeLinks *storelink.Resolver, award AwardFunc) *Service {
	if db != nil {
		s.store = repository.NewWorkV1Store(db)
		s.hydrator = workrepr.NewHydrator(s.works, db, s.cdn)
		s.lists = repository.NewGalgameListRepository(db)
		s.collectedMonths = s.lists.ListCollectedCalendar
	}
	s.catalog = catalog
	s.storeLinks = storeLinks
	if award != nil {
		s.award = award
	}
	return s
}

func (s *Service) WithUserPlane(check *gate.CheckService, scan *gate.ScanService, aliases *repository.GalgameCollectionRepository) *Service {
	s.check = check
	s.scan = scan
	s.aliases = aliases
	return s
}

func (s *Service) detailCatalog() workDetailCatalog {
	d, _ := s.works.(workDetailCatalog)
	return d
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

func (s *Service) flushAwards(jobs []pendingAward) {
	if s.award == nil {
		return
	}
	for _, j := range jobs {
		s.award(j.userID, j.delta, j.reason, j.ref, j.key)
	}
}

type MoyuVocabulary string

func (MoyuVocabulary) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("moyu_patch_vocabulary", 32)
	s.Pattern = "^[A-Za-z0-9_-]+$"
	s.Description = "A token of www.moyu.moe's own vocabulary, such as machine_polishing or zh-Hans. That site may add tokens; show an unknown one as it is."
	return s
}

type MoyuPatch struct {
	Object    string              `json:"object" enum:"moyu_patch" maxLength:"10" doc:"Type discriminant. Always moyu_patch."`
	ID        repr.DecimalID      `json:"id" doc:"The page's id on www.moyu.moe. Neither a galgame id nor a catalog work id."`
	WebURL    string              `json:"web_url" format:"uri" maxLength:"512" doc:"The page on www.moyu.moe."`
	Resources []MoyuPatchResource `json:"resources" doc:"The page's live resources, newest change first. Empty array, never null."`
}

type MoyuPatchResource struct {
	Object        string           `json:"object" enum:"moyu_patch_resource" maxLength:"19" doc:"Type discriminant. Always moyu_patch_resource."`
	ID            repr.DecimalID   `json:"id" doc:"The resource's id on www.moyu.moe."`
	Name          *string          `json:"name" maxLength:"300" doc:"The name its publisher gave it. null when left empty. Free text; never use it as a decision input."`
	Storage       MoyuVocabulary   `json:"storage" doc:"Where the file lives: s3 is www.moyu.moe's own object store, user a link its publisher hosts elsewhere."`
	Size          string           `json:"size" maxLength:"64" doc:"Size as its publisher wrote it, such as 0.571 MB; not a byte count. Free text; never use it as a decision input."`
	ModelName     *string          `json:"model_name" maxLength:"1007" doc:"For an AI-translated patch, the model as its publisher typed it. null when not given. Free text; never use it as a decision input."`
	NoteMarkdown  *string          `json:"note_markdown" maxLength:"10007" doc:"Its publisher's note as Markdown source written on www.moyu.moe, image tokens resolved to absolute URLs. Not a forum body, so it is not a content document. null when there is none. Free text; never use it as a decision input."`
	Types         []MoyuVocabulary `json:"types" doc:"Patch kinds, such as manual, ai, machine or save. Empty array, never null."`
	Languages     []MoyuVocabulary `json:"languages" doc:"Languages, such as zh-Hans, zh-Hant, ja or en. Empty array, never null."`
	Platforms     []MoyuVocabulary `json:"platforms" doc:"Platforms, such as windows, android or linux. Empty array, never null."`
	DownloadCount int              `json:"download_count" minimum:"0" doc:"Downloads counted on www.moyu.moe."`
	WebURL        string           `json:"web_url" format:"uri" maxLength:"512" doc:"The resource on www.moyu.moe. The only way to its file: no download link, share code or password is carried."`
	UpdatedAt     repr.DateTime    `json:"updated_at" doc:"When the resource last changed."`
	Publisher     repr.UserRef     `json:"publisher" doc:"Who published it. The account is the same one on this forum."`
}

type listGalgameMoyuPatchesInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog work id, which is also the forum galgame page id."`
}

type listGalgameMoyuPatchesOutput struct {
	Body repr.List[MoyuPatch]
}

func (s *Service) listGalgameMoyuPatches(ctx context.Context, in *listGalgameMoyuPatchesInput) (*listGalgameMoyuPatchesOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	workID, ok := repr.ParseID(repr.DecimalID(in.WorkID))
	if !ok {
		return nil, notFound()
	}
	found, appErr := s.works.CatalogWorkExists(ctx, workID)
	if appErr != nil {
		return nil, problem.Unavailable(appErr)
	}
	if !found {
		return nil, notFound()
	}
	patches, err := s.patchesFor(ctx, int64(workID))
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	publishers, err := s.publishers(ctx, patches)
	if err != nil {
		return nil, problem.Unavailable(err)
	}

	items := make([]MoyuPatch, 0, len(patches))
	for _, p := range patches {
		resources := make([]MoyuPatchResource, 0, len(p.Resources))
		for _, r := range p.Resources {
			resources = append(resources, s.resource(r, publishers))
		}
		items = append(items, MoyuPatch{
			Object:    "moyu_patch",
			ID:        repr.DecimalID(p.ID),
			WebURL:    p.WebURL,
			Resources: resources,
		})
	}
	return &listGalgameMoyuPatchesOutput{Body: repr.NewList(items, nil)}, nil
}

func (s *Service) patchesFor(ctx context.Context, catalogID int64) ([]moyuclient.Patch, error) {
	key := moyuCacheKey + strconv.FormatInt(catalogID, 10)
	if raw, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
		var cached []moyuclient.Patch
		if json.Unmarshal(raw, &cached) == nil {
			return cached, nil
		}
	}
	patches, err := s.moyu.PatchesForWork(ctx, catalogID)
	if err != nil {
		return nil, err
	}
	if raw, err := json.Marshal(patches); err == nil {
		_ = s.rdb.Set(ctx, key, raw, moyuCacheTTL).Err()
	}
	return patches, nil
}

func (s *Service) publishers(ctx context.Context, patches []moyuclient.Patch) (map[int]userclient.User, error) {
	var ids []int
	for _, p := range patches {
		for _, r := range p.Resources {
			if id := publisherID(r); id > 0 {
				ids = append(ids, id)
			}
		}
	}
	return s.users.Users(ctx, ids)
}

func (s *Service) resource(r moyuclient.Resource, users map[int]userclient.User) MoyuPatchResource {
	id := publisherID(r)
	publisher := repr.DeletedUserRef(id)
	if u, ok := users[id]; ok {
		publisher = repr.NewUserRef(s.cdn, u)
	}
	return MoyuPatchResource{
		Object:        "moyu_patch_resource",
		ID:            repr.DecimalID(r.ID),
		Name:          nonEmpty(r.Name),
		Storage:       MoyuVocabulary(r.Storage),
		Size:          r.Size,
		ModelName:     nonEmpty(r.ModelName),
		NoteMarkdown:  nonEmpty(r.Note),
		Types:         vocabulary(r.Type),
		Languages:     vocabulary(r.Language),
		Platforms:     vocabulary(r.Platform),
		DownloadCount: r.DownloadCount,
		WebURL:        r.WebURL,
		UpdatedAt:     repr.Timestamp(r.UpdatedAt),
		Publisher:     publisher,
	}
}

func publisherID(r moyuclient.Resource) int {
	if r.Publisher == nil {
		return 0
	}
	id, _ := strconv.Atoi(r.Publisher.ID)
	return id
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func vocabulary(tokens []string) []MoyuVocabulary {
	out := make([]MoyuVocabulary, len(tokens))
	for i, t := range tokens {
		out[i] = MoyuVocabulary(t)
	}
	return out
}
