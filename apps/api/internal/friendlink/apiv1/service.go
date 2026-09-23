package apiv1

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/friendlink/repository"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

var errUnconfigured = errors.New("apiv1 friend links: service is not configured")

type Service struct {
	repo   *repository.FriendLinkRepository
	images func(hashes []string) map[string]imageclient.ImageMeta
	cdn    string
}

func New(repo *repository.FriendLinkRepository, images func(hashes []string) map[string]imageclient.ImageMeta, cdn string) *Service {
	return &Service{repo: repo, images: images, cdn: cdn}
}

func (s *Service) ready() *problem.Problem {
	if s == nil || s.repo == nil {
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

func trimText(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
}

var linkStates = []string{"normal", "down"}

func (s *Service) metas(rows []repository.LinkRow) map[string]imageclient.ImageMeta {
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

// friendLink refuses a stored status outside the closed enum rather than
// passing it off as normal: essential was dropped from the contract because no
// row ever held it, so seeing one means the contract is wrong.
func (s *Service) friendLink(row repository.LinkRow, metas map[string]imageclient.ImageMeta) (FriendLink, error) {
	if !slices.Contains(linkStates, row.Status) {
		return FriendLink{}, fmt.Errorf("friend_link %d has status %q outside the closed enum", row.ID, row.Status)
	}
	if !slices.Contains(repository.Categories, row.Category) {
		return FriendLink{}, fmt.Errorf("friend_link %d has category %q outside the closed enum", row.ID, row.Category)
	}
	var banner *repr.Image
	if row.BannerImageHash != "" {
		var meta *imageclient.ImageMeta
		if m, ok := metas[row.BannerImageHash]; ok {
			meta = &m
		}
		banner = repr.NewImage(s.cdn, row.BannerImageHash, meta)
	}
	return FriendLink{
		Object:             "friend_link",
		ID:                 repr.ID(row.ID),
		FriendLinkCategory: row.Category,
		Title:              row.Name,
		URL:                row.Link,
		Description:        row.Description,
		Banner:             banner,
		State:              row.Status,
	}, nil
}

func (s *Service) listFriendLinks(_ context.Context, in *listFriendLinksInput) (*listFriendLinksOutput, error) {
	if prob := s.ready(); prob != nil {
		return nil, prob
	}
	const sortToken = "position_asc"
	fp := collect.Fingerprint(sortToken, in.FriendLinkCategory)
	keys, curErr := collect.DecodeCursor(in.Cursor, sortToken, fp)
	if curErr != nil {
		return nil, curErr
	}
	var after *repository.LinkPos
	if keys != nil {
		pos, ok := decodePos(keys)
		if !ok {
			return nil, problem.New(problem.CodeInvalidCursor, "The cursor is malformed or was issued for different parameters.",
				problem.AtParameter("cursor", problem.ReasonInvalidFormat, "cursor does not decode for this collection", nil))
		}
		after = pos
	}
	rows, err := s.repo.ListKeyset(in.FriendLinkCategory, after, in.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	more := len(rows) > in.Limit
	if more {
		rows = rows[:in.Limit]
	}
	metas := s.metas(rows)
	items := make([]FriendLink, 0, len(rows))
	for _, row := range rows {
		item, err := s.friendLink(row, metas)
		if err != nil {
			return nil, problem.Internal(err)
		}
		items = append(items, item)
	}
	var next *string
	if more {
		last := rows[len(rows)-1]
		cur := collect.EncodeCursor(sortToken, fp, strconv.Itoa(last.Rank), strconv.Itoa(last.SortOrder), strconv.Itoa(last.ID))
		next = &cur
	}
	return &listFriendLinksOutput{Body: repr.NewList(items, next)}, nil
}

func decodePos(keys []string) (*repository.LinkPos, bool) {
	if len(keys) != 3 {
		return nil, false
	}
	var n [3]int
	for i, k := range keys {
		v, err := strconv.Atoi(k)
		if err != nil {
			return nil, false
		}
		n[i] = v
	}
	if n[2] <= 0 {
		return nil, false
	}
	return &repository.LinkPos{Rank: n[0], SortOrder: n[1], ID: n[2]}, true
}

func parseID(raw string) (int, *problem.Problem) {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, notFound()
	}
	return id, nil
}

func (s *Service) load(id int) (*FriendLink, *problem.Problem) {
	row, err := s.repo.FindRow(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	link, err := s.friendLink(*row, s.metas([]repository.LinkRow{*row}))
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &link, nil
}

func (s *Service) getFriendLink(ctx context.Context, in *friendLinkInput) (*friendLinkOutput, error) {
	if prob := s.require(ctx, perm.FriendLinkEdit); prob != nil {
		return nil, prob
	}
	id, prob := parseID(in.FriendLinkID)
	if prob != nil {
		return nil, prob
	}
	link, prob := s.load(id)
	if prob != nil {
		return nil, prob
	}
	return &friendLinkOutput{Body: *link}, nil
}

func tooShort(pointer string) problem.FieldError {
	min := 1
	return problem.AtPointer(pointer, problem.ReasonTooShort, "must contain at least 1 character after trimming whitespace", &problem.FieldParams{MinLength: &min})
}

func (s *Service) createFriendLink(ctx context.Context, in *createFriendLinkInput) (*createFriendLinkOutput, error) {
	if prob := s.require(ctx, perm.FriendLinkCreate); prob != nil {
		return nil, prob
	}
	b := in.Body
	name := trimText(b.Title)
	if name == "" {
		return nil, validationFailed(tooShort("/title"))
	}
	w := repository.LinkWrite{
		Category: b.FriendLinkCategory,
		Name:     name,
		Link:     b.URL,
		Status:   "normal",
	}
	if b.Description != nil {
		w.Description = trimText(*b.Description)
	}
	if b.BannerImageHash != nil {
		w.BannerImageHash = string(*b.BannerImageHash)
	}
	if b.State != nil {
		w.Status = *b.State
	}
	id, err := s.repo.CreateLink(w)
	if err != nil {
		return nil, problem.Internal(err)
	}
	link, prob := s.load(id)
	if prob != nil {
		return nil, prob
	}
	return &createFriendLinkOutput{Location: "/api/v1/admin/friend-links/" + strconv.Itoa(id), Body: *link}, nil
}

func (s *Service) updateFriendLink(ctx context.Context, in *updateFriendLinkInput) (*friendLinkOutput, error) {
	if prob := s.require(ctx, perm.FriendLinkEdit); prob != nil {
		return nil, prob
	}
	id, prob := parseID(in.FriendLinkID)
	if prob != nil {
		return nil, prob
	}
	p := in.Body
	if p.FriendLinkCategory == nil && p.Title == nil && p.URL == nil && p.Description == nil &&
		p.BannerImageHash == nil && p.State == nil {
		return nil, validationFailed(problem.AtPointer("", problem.ReasonRequired, "send at least one field to change", nil))
	}
	ch := repository.LinkChanges{Category: p.FriendLinkCategory, Link: p.URL, Status: p.State}
	if p.Title != nil {
		name := trimText(*p.Title)
		if name == "" {
			return nil, validationFailed(tooShort("/title"))
		}
		ch.Name = &name
	}
	if p.Description != nil {
		d := trimText(*p.Description)
		ch.Description = &d
	}
	if p.BannerImageHash != nil {
		h := string(*p.BannerImageHash)
		ch.BannerImageHash = &h
	}
	if err := s.repo.UpdateLink(id, ch); err != nil {
		return nil, problem.Internal(err)
	}
	link, prob := s.load(id)
	if prob != nil {
		return nil, prob
	}
	return &friendLinkOutput{Body: *link}, nil
}

func (s *Service) deleteFriendLink(ctx context.Context, in *friendLinkInput) (*struct{}, error) {
	if prob := s.require(ctx, perm.FriendLinkDelete); prob != nil {
		return nil, prob
	}
	id, prob := parseID(in.FriendLinkID)
	if prob != nil {
		return nil, prob
	}
	deleted, err := s.repo.DeleteLink(id)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if !deleted {
		return nil, notFound()
	}
	return &struct{}{}, nil
}

func (s *Service) putFriendLinkOrder(ctx context.Context, in *putFriendLinkOrderInput) (*struct{}, error) {
	if prob := s.require(ctx, perm.FriendLinkEdit); prob != nil {
		return nil, prob
	}
	unknown := func(i int) *problem.Problem {
		return validationFailed(problem.AtPointer(fmt.Sprintf("/friend_link_ids/%d", i), problem.ReasonUnknownReference, "no link on this shelf has this id", nil))
	}
	ids := make([]int, len(in.Body.FriendLinkIDs))
	for i, raw := range in.Body.FriendLinkIDs {
		id, ok := repr.ParseID(raw)
		if !ok {
			return nil, unknown(i)
		}
		ids[i] = id
	}
	res, applied, err := s.repo.ReorderShelf(in.Body.FriendLinkCategory, ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if res.UnknownAt >= 0 {
		return nil, unknown(res.UnknownAt)
	}
	if !applied {
		n := res.Count
		return nil, validationFailed(problem.AtPointer("/friend_link_ids", problem.ReasonTooFewItems, "list every link on the shelf exactly once", &problem.FieldParams{MinItems: &n}))
	}
	return &struct{}{}, nil
}
