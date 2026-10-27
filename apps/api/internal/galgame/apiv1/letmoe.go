package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/letmoeclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"github.com/danielgtaylor/huma/v2"
)

const (
	letmoeCacheKey = "letmoe:resources:v1:"
	// The face's own s-maxage: it declares its answers shareable for this long.
	letmoeCacheTTL = 30 * time.Minute
)

func (s *Service) WithLetmoe(c *letmoeclient.Client) *Service {
	s.letmoe = c
	return s
}

type LetmoeVocabulary string

func (LetmoeVocabulary) Schema(huma.Registry) *huma.Schema {
	s := repr.OpenEnum("letmoe_resource_vocabulary", 32)
	s.Pattern = "^[A-Za-z0-9_-]+$"
	s.Description = "A token of www.letmoe.com's own vocabulary, such as game, zh-cn, win or baidu. That site may add tokens; show an unknown one as it is."
	return s
}

type LetmoeResource struct {
	Object            string             `json:"object" enum:"letmoe_resource" maxLength:"15" doc:"Type discriminant. Always letmoe_resource."`
	ID                repr.DecimalID     `json:"id" doc:"The resource's id on www.letmoe.com. Not a forum resource id."`
	Title             string             `json:"title" maxLength:"500" doc:"The title letmoe shows, never empty. Free text; never use it as a decision input."`
	LetmoeType        LetmoeVocabulary   `json:"letmoe_type" doc:"Resource kind in www.letmoe.com's vocabulary, such as game, translation or asmr. A superset of this forum's resource_type."`
	VersionText       *string            `json:"version_text" maxLength:"200" doc:"The version its publisher labelled it with. null when none. Free text; never use it as a decision input."`
	Languages         []LetmoeVocabulary `json:"languages" doc:"Languages, such as zh-cn or ja-jp. Empty array, never null."`
	Platforms         []LetmoeVocabulary `json:"platforms" doc:"Catalog platform keys, such as win or and. Empty array, never null."`
	Runtimes          []LetmoeVocabulary `json:"runtimes" doc:"What it runs on, such as native-win or kirikiroid2. Empty array, never null."`
	Size              string             `json:"size" maxLength:"64" doc:"Size as its publisher gave it, such as 1.2 GB. Empty when unknown. Free text; never use it as a decision input."`
	NoteMarkdown      *string            `json:"note_markdown" maxLength:"20000" doc:"Its publisher's description as Markdown source written on www.letmoe.com. Not a forum body, so it is not a content document. null when there is none. Free text; never use it as a decision input."`
	ContentWarning    *string            `json:"content_warning" maxLength:"2000" doc:"A warning its publisher attached. null when none. Free text; never use it as a decision input."`
	Providers         []LetmoeVocabulary `json:"providers" doc:"Where its mirrors live; direct is letmoe's own hosted file. Empty array, never null."`
	HasDirectDownload bool               `json:"has_direct_download" doc:"A mirror or a merged download downloads without a netdisk."`
	CompositeCount    int                `json:"composite_count" minimum:"0" doc:"Merged downloads (this resource plus a www.moyu.moe patch) offered alongside."`
	DownloadCount     int                `json:"download_count" minimum:"0" doc:"Downloads counted on www.letmoe.com."`
	WebURL            string             `json:"web_url" format:"uri" maxLength:"512" doc:"The resource on www.letmoe.com."`
	UpdatedAt         repr.DateTime      `json:"updated_at" doc:"When the resource last changed."`
	Publisher         repr.UserRef       `json:"publisher" doc:"Who published it. The account is the same one on this forum."`
}

type LetmoeMirror struct {
	Object          string            `json:"object" enum:"letmoe_mirror" maxLength:"13" doc:"Type discriminant. Always letmoe_mirror."`
	ID              repr.DecimalID    `json:"id" doc:"The mirror's id on www.letmoe.com."`
	IsHosted        bool              `json:"is_hosted" doc:"true for letmoe's own file, whose url downloads it directly; false for a link its publisher hosts elsewhere."`
	Provider        *LetmoeVocabulary `json:"provider" doc:"Netdisk of an external_link mirror, such as baidu or quark. null on a hosted mirror."`
	URL             string            `json:"url" format:"uri" maxLength:"2048" doc:"The link. A hosted url stops working when the file is replaced or taken down; do not store it."`
	ExtractionCode  string            `json:"extraction_code" maxLength:"1007" doc:"Netdisk share code. Empty string when none. Free text; never use it as a decision input."`
	ArchivePassword string            `json:"archive_password" maxLength:"1007" doc:"Archive password. Empty string when none. Free text; never use it as a decision input."`
	Note            *string           `json:"note" maxLength:"2000" doc:"Its publisher's note on this mirror. null when none. Free text; never use it as a decision input."`
	IsPrimary       bool              `json:"is_primary" doc:"The publisher's primary mirror. It comes first."`
}

type LetmoeComposite struct {
	Object         string           `json:"object" enum:"letmoe_composite" maxLength:"16" doc:"Type discriminant. Always letmoe_composite."`
	ID             repr.DecimalID   `json:"id" doc:"The merged download's id on www.letmoe.com."`
	URL            string           `json:"url" format:"uri" maxLength:"2048" doc:"A direct link to the game and the patch spliced into one download."`
	Filename       string           `json:"filename" maxLength:"500" doc:"The downloaded file's name. Free text; never use it as a decision input."`
	Bytes          int64            `json:"bytes" minimum:"0" doc:"The merged file's size in bytes."`
	PatchName      string           `json:"patch_name" maxLength:"300" doc:"The www.moyu.moe patch it carries. Free text; never use it as a decision input."`
	PatchPublisher string           `json:"patch_publisher" maxLength:"200" doc:"The patch author's display name. Free text; never use it as a decision input."`
	PatchTypes     []MoyuVocabulary `json:"patch_types" doc:"The patch's kinds in www.moyu.moe's vocabulary, such as manual, machine or fix. Empty array, never null."`
	PatchLanguages []MoyuVocabulary `json:"patch_languages" doc:"The patch's languages in www.moyu.moe's vocabulary, such as zh-Hans or zh-Hant. Empty array, never null."`
	MoyuResourceID repr.DecimalID   `json:"moyu_resource_id" doc:"The patch resource's id on www.moyu.moe."`
}

type LetmoeResourceDownload struct {
	Object     string            `json:"object" enum:"letmoe_resource_download" maxLength:"24" doc:"Type discriminant. Always letmoe_resource_download."`
	ResourceID repr.DecimalID    `json:"resource_id" doc:"The resource's id on www.letmoe.com."`
	Mirrors    []LetmoeMirror    `json:"mirrors" doc:"Download options, the primary mirror first. Empty array, never null."`
	Composites []LetmoeComposite `json:"composites" doc:"Merged downloads. Empty array, never null."`
}

type listWorkLetmoeResourcesInput struct {
	WorkID string `path:"work_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Catalog work id, which is also the forum galgame page id."`
}

type listWorkLetmoeResourcesOutput struct {
	Body repr.List[LetmoeResource]
}

type letmoeResourceIDInput struct {
	ResourceID string `path:"resource_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The resource's id on www.letmoe.com."`
}

type createLetmoeResourceDownloadOutput struct {
	Body LetmoeResourceDownload
}

func (s *Service) listWorkLetmoeResources(ctx context.Context, in *listWorkLetmoeResourcesInput) (*listWorkLetmoeResourcesOutput, error) {
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
	resources, err := s.letmoeResourcesFor(ctx, int64(workID))
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	users, err := s.letmoePublishers(ctx, resources)
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	items := make([]LetmoeResource, 0, len(resources))
	for _, r := range resources {
		items = append(items, s.letmoeResource(r, users))
	}
	return &listWorkLetmoeResourcesOutput{Body: repr.NewList(items, nil)}, nil
}

type getLetmoeResourceOutput struct {
	Body LetmoeResource
}

func (s *Service) getLetmoeResource(ctx context.Context, in *letmoeResourceIDInput) (*getLetmoeResourceOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	r, err := s.letmoe.Resource(ctx, in.ResourceID)
	if errors.Is(err, letmoeclient.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	users, err := s.letmoePublishers(ctx, []letmoeclient.Resource{*r})
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	return &getLetmoeResourceOutput{Body: s.letmoeResource(*r, users)}, nil
}

func (s *Service) createLetmoeResourceDownload(ctx context.Context, in *letmoeResourceIDInput) (*createLetmoeResourceDownloadOutput, error) {
	if s == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	files, err := s.letmoe.Files(ctx, in.ResourceID)
	if errors.Is(err, letmoeclient.ErrNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	mirrors := make([]LetmoeMirror, 0, len(files.Mirrors))
	for _, m := range files.Mirrors {
		mirrors = append(mirrors, LetmoeMirror{
			Object:          "letmoe_mirror",
			ID:              repr.DecimalID(m.ID),
			IsHosted:        m.Kind == "hosted",
			Provider:        letmoeProvider(m.Provider),
			URL:             m.URL,
			ExtractionCode:  m.ExtractCode,
			ArchivePassword: m.ArchivePassword,
			Note:            nonEmpty(m.Note),
			IsPrimary:       m.IsPrimary,
		})
	}
	composites := make([]LetmoeComposite, 0, len(files.Composites))
	for _, c := range files.Composites {
		composites = append(composites, LetmoeComposite{
			Object:         "letmoe_composite",
			ID:             repr.DecimalID(c.ID),
			URL:            c.URL,
			Filename:       c.Filename,
			Bytes:          c.Bytes,
			PatchName:      c.Patch.Name,
			PatchPublisher: c.Patch.Publisher,
			PatchTypes:     vocabulary(c.Patch.Type),
			PatchLanguages: vocabulary(c.Patch.Language),
			MoyuResourceID: repr.DecimalID(c.Patch.MoyuResourceID),
		})
	}
	return &createLetmoeResourceDownloadOutput{Body: LetmoeResourceDownload{
		Object:     "letmoe_resource_download",
		ResourceID: repr.DecimalID(in.ResourceID),
		Mirrors:    mirrors,
		Composites: composites,
	}}, nil
}

func (s *Service) letmoeResourcesFor(ctx context.Context, catalogID int64) ([]letmoeclient.Resource, error) {
	key := letmoeCacheKey + strconv.FormatInt(catalogID, 10)
	if raw, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
		var cached []letmoeclient.Resource
		if json.Unmarshal(raw, &cached) == nil {
			return cached, nil
		}
	}
	resources, err := s.letmoe.ResourcesForWork(ctx, catalogID)
	if err != nil {
		return nil, err
	}
	if raw, err := json.Marshal(resources); err == nil {
		_ = s.rdb.Set(ctx, key, raw, letmoeCacheTTL).Err()
	}
	return resources, nil
}

func (s *Service) letmoePublishers(ctx context.Context, resources []letmoeclient.Resource) (map[int]userclient.User, error) {
	var ids []int
	for _, r := range resources {
		if id := letmoePublisherID(r); id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return map[int]userclient.User{}, nil
	}
	return s.users.Users(ctx, ids)
}

func (s *Service) letmoeResource(r letmoeclient.Resource, users map[int]userclient.User) LetmoeResource {
	id := letmoePublisherID(r)
	publisher := repr.DeletedUserRef(id)
	if u, ok := users[id]; ok {
		publisher = repr.NewUserRef(s.cdn, u)
	}
	return LetmoeResource{
		Object:            "letmoe_resource",
		ID:                repr.DecimalID(r.ID),
		Title:             r.Title,
		LetmoeType:        LetmoeVocabulary(r.Type),
		VersionText:       nonEmpty(r.Version),
		Languages:         letmoeVocabulary(r.Language),
		Platforms:         letmoeVocabulary(r.Platform),
		Runtimes:          letmoeVocabulary(r.Runtime),
		Size:              r.Size,
		NoteMarkdown:      nonEmpty(r.Description),
		ContentWarning:    nonEmpty(r.ContentWarning),
		Providers:         letmoeVocabulary(r.Providers),
		HasDirectDownload: r.DirectDownload,
		CompositeCount:    r.CompositeCount,
		DownloadCount:     r.DownloadCount,
		WebURL:            r.WebURL,
		UpdatedAt:         repr.Timestamp(r.UpdatedAt),
		Publisher:         publisher,
	}
}

func letmoePublisherID(r letmoeclient.Resource) int {
	if r.Publisher == nil {
		return 0
	}
	id, _ := strconv.Atoi(r.Publisher.ID)
	return id
}

func letmoeProvider(p string) *LetmoeVocabulary {
	if p == "" {
		return nil
	}
	v := LetmoeVocabulary(p)
	return &v
}

func letmoeVocabulary(tokens []string) []LetmoeVocabulary {
	out := make([]LetmoeVocabulary, len(tokens))
	for i, t := range tokens {
		out[i] = LetmoeVocabulary(t)
	}
	return out
}
