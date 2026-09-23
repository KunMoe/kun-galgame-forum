package apiv1

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/doc/repository"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"

	"gorm.io/gorm"
)

var errUnconfigured = errors.New("apiv1 docs: service is not configured")

type Service struct {
	repo    *repository.DocRepository
	users   *userclient.Client
	convert *content.Converter
	images  func(hashes []string) map[string]imageclient.ImageMeta
	cdn     string
}

func New(
	repo *repository.DocRepository,
	users *userclient.Client,
	convert *content.Converter,
	images func(hashes []string) map[string]imageclient.ImageMeta,
	cdn string,
) *Service {
	return &Service{repo: repo, users: users, convert: convert, images: images, cdn: cdn}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.repo == nil || s.users == nil || s.convert == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func (s *Service) require(ctx context.Context, p perm.Permission) *problem.Problem {
	if prob := s.ready(); prob != nil {
		return prob
	}
	if !v1.User(ctx).Can(p) {
		return problem.New(problem.CodePermissionRequired, "The token lacks the permission this decision needs.")
	}
	return nil
}

func notFound() *problem.Problem {
	return problem.New(problem.CodeNotFound, "Nothing visible exists at this URL.")
}

func validationFailed(fields ...problem.FieldError) *problem.Problem {
	return problem.New(problem.CodeValidationFailed, "The request is syntactically valid but semantically not.", fields...)
}

func tooShort(pointer string) problem.FieldError {
	min := 1
	return problem.AtPointer(pointer, problem.ReasonTooShort, "must contain at least 1 character after trimming whitespace", &problem.FieldParams{MinLength: &min})
}

func slugTaken() *problem.Problem {
	return problem.New(problem.CodeAlreadyExists, "Another doc already uses this slug.")
}

func trimText(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
}

// categoryToken refuses a slug outside the closed enum instead of filing the
// doc under one of the four: a fifth doc_category row is a schema change the
// contract has not made.
func categoryToken(slug string) (string, error) {
	if slices.Contains(docCategories, slug) {
		return slug, nil
	}
	return "", fmt.Errorf("doc_category slug %q is outside the closed enum", slug)
}

func (s *Service) banners(rows []repository.DocRow) map[string]imageclient.ImageMeta {
	if s.images == nil {
		return nil
	}
	var hashes []string
	for _, r := range rows {
		if r.BannerImageHash != "" {
			hashes = append(hashes, r.BannerImageHash)
		}
	}
	if len(hashes) == 0 {
		return nil
	}
	return s.images(hashes)
}

func (s *Service) banner(hash string, metas map[string]imageclient.ImageMeta) *repr.Image {
	if hash == "" {
		return nil
	}
	var meta *imageclient.ImageMeta
	if m, ok := metas[hash]; ok {
		meta = &m
	}
	return repr.NewImage(s.cdn, hash, meta)
}

func (s *Service) summary(row repository.DocRow, metas map[string]imageclient.ImageMeta) (DocSummary, error) {
	category, err := categoryToken(row.CategorySlug)
	if err != nil {
		return DocSummary{}, err
	}
	return DocSummary{
		Object:      "doc",
		ID:          repr.ID(row.ID),
		Slug:        row.Slug,
		Title:       row.Title,
		Description: row.Description,
		DocCategory: category,
		Banner:      s.banner(row.BannerImageHash, metas),
		IsPinned:    row.IsPin,
		ViewCount:   row.View,
		PublishedAt: repr.Timestamp(row.PublishedTime),
		EditedAt:    repr.TimestampPtr(row.EditedTime),
	}, nil
}

func (s *Service) adminDoc(row *repository.DocRow) (AdminDoc, *problem.Problem) {
	sum, err := s.summary(*row, s.banners([]repository.DocRow{*row}))
	if err != nil {
		return AdminDoc{}, problem.Internal(err)
	}
	return AdminDoc{
		Object:          "admin_doc",
		ID:              sum.ID,
		Slug:            sum.Slug,
		Title:           sum.Title,
		Description:     sum.Description,
		DocCategory:     sum.DocCategory,
		Banner:          sum.Banner,
		IsPinned:        sum.IsPinned,
		ViewCount:       sum.ViewCount,
		PublishedAt:     sum.PublishedAt,
		EditedAt:        sum.EditedAt,
		ContentMarkdown: row.ContentMarkdown,
	}, nil
}

type docSortSpec struct {
	Token string
	Sort  repository.DocSort
}

var docSorts = []docSortSpec{
	{Token: "position_asc", Sort: repository.DocSortPosition},
	{Token: "published_desc", Sort: repository.DocSortPublished},
	{Token: "views_desc", Sort: repository.DocSortViews},
}

const defaultDocSort = "position_asc"

func lookupDocSort(token string) (docSortSpec, bool) {
	if token == "" {
		token = defaultDocSort
	}
	for _, s := range docSorts {
		if s.Token == token {
			return s, true
		}
	}
	return docSortSpec{}, false
}

func listFingerprint(sort, category string, pinned *bool) string {
	p := ""
	if pinned != nil {
		p = strconv.FormatBool(*pinned)
	}
	return collect.Fingerprint(sort, category, p)
}

func encodeDocPos(row repository.DocRow, sort repository.DocSort) []string {
	id := strconv.Itoa(row.ID)
	switch sort {
	case repository.DocSortPublished:
		return []string{row.PublishedTime.UTC().Format(time.RFC3339Nano), id}
	case repository.DocSortViews:
		return []string{strconv.Itoa(row.View), id}
	default:
		return []string{strconv.Itoa(row.SortOrder), id}
	}
}

func decodeDocPos(keys []string, sort repository.DocSort) (*repository.DocPos, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	bad := problem.New(problem.CodeInvalidCursor, "The cursor is malformed or was issued for different parameters.",
		problem.AtParameter("cursor", problem.ReasonInvalidFormat, "cursor does not decode for this collection", nil))
	if len(keys) != 2 {
		return nil, bad
	}
	id, err := strconv.Atoi(keys[1])
	if err != nil || id <= 0 {
		return nil, bad
	}
	pos := &repository.DocPos{ID: id}
	if sort == repository.DocSortPublished {
		t, err := time.Parse(time.RFC3339Nano, keys[0])
		if err != nil {
			return nil, bad
		}
		pos.Time = t
		return pos, nil
	}
	n, err := strconv.ParseInt(keys[0], 10, 64)
	if err != nil {
		return nil, bad
	}
	pos.Int = n
	return pos, nil
}

func (s *Service) listDocs(_ context.Context, in *listDocsInput) (*listDocsOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	spec, ok := lookupDocSort(string(in.Sort))
	if !ok {
		return nil, problem.New(problem.CodeUnknownSort, "The sort token is not in this collection's vocabulary.",
			problem.AtParameter("sort", problem.ReasonUnknownValue, "use a sort token declared by this operation", nil))
	}
	pinned := in.IsPinned.ptr()
	fp := listFingerprint(spec.Token, in.DocCategory, pinned)
	keys, curErr := collect.DecodeCursor(in.Cursor, spec.Token, fp)
	if curErr != nil {
		return nil, curErr
	}
	pos, prob := decodeDocPos(keys, spec.Sort)
	if prob != nil {
		return nil, prob
	}
	rows, err := s.repo.FindKeyset(repository.DocKeysetQuery{
		Sort: spec.Sort, Category: in.DocCategory, Pinned: pinned, After: pos, Limit: in.Limit,
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	more := len(rows) > in.Limit
	if more {
		rows = rows[:in.Limit]
	}
	metas := s.banners(rows)
	items := make([]DocSummary, 0, len(rows))
	for _, row := range rows {
		item, err := s.summary(row, metas)
		if err != nil {
			return nil, problem.Internal(err)
		}
		items = append(items, item)
	}
	var next *string
	if more {
		cur := collect.EncodeCursor(spec.Token, fp, encodeDocPos(rows[len(rows)-1], spec.Sort)...)
		next = &cur
	}
	return &listDocsOutput{Body: repr.NewList(items, next)}, nil
}

func (s *Service) getDoc(ctx context.Context, in *getDocInput) (*getDocOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	row, err := s.repo.FindBySlug(in.DocSlug)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	if view, err := s.repo.CountView(row.ID); err != nil {
		slog.Warn("doc view count failed", "doc_id", row.ID, "error", err)
	} else {
		row.View = view
	}
	sum, err := s.summary(*row, s.banners([]repository.DocRow{*row}))
	if err != nil {
		return nil, problem.Internal(err)
	}
	users, err := s.users.Users(ctx, []int{row.AuthorID})
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	author := repr.DeletedUserRef(row.AuthorID)
	if u, ok := users[row.AuthorID]; ok && userclient.IsRenderable(u) {
		author = repr.NewUserRef(s.cdn, u)
	}
	docs, err := s.convert.Convert(ctx, []string{row.ContentMarkdown})
	if err != nil {
		return nil, problem.Unavailable(err)
	}
	return &getDocOutput{Body: Doc{
		Object:      "doc",
		ID:          sum.ID,
		Slug:        sum.Slug,
		Title:       sum.Title,
		Description: sum.Description,
		DocCategory: sum.DocCategory,
		Banner:      sum.Banner,
		IsPinned:    sum.IsPinned,
		ViewCount:   sum.ViewCount,
		PublishedAt: sum.PublishedAt,
		EditedAt:    sum.EditedAt,
		Author:      author,
		Content:     docs[0],
	}}, nil
}
