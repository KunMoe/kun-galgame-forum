package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/website/model"
	"kun-galgame-api/internal/website/repository"
	"kun-galgame-api/pkg/problem"
)

const websiteSort = "created_at_desc"

func parseTimePos(keys []string) (*repository.TimePos, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 2 {
		return nil, invalidCursor()
	}
	created, err := time.Parse(time.RFC3339Nano, keys[0])
	if err != nil {
		return nil, invalidCursor()
	}
	id, err := strconv.Atoi(keys[1])
	if err != nil || id <= 0 {
		return nil, invalidCursor()
	}
	return &repository.TimePos{Created: created, ID: id}, nil
}

func limitOf(n int) int {
	if n <= 0 {
		return collect.DefaultLimit
	}
	return n
}

func (s *Service) summaries(rows []repository.WebsiteRow) ([]WebsiteSummary, *problem.Problem) {
	catIDs := make([]int, 0, len(rows))
	hashes := make([]string, 0, len(rows))
	for _, r := range rows {
		catIDs = append(catIDs, r.CategoryID)
		if r.IconImageHash != "" {
			hashes = append(hashes, r.IconImageHash)
		}
	}
	cats, err := s.store.CategoriesByIDs(catIDs)
	if err != nil {
		return nil, problem.Internal(err)
	}
	metas := s.images(hashes)
	out := make([]WebsiteSummary, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		icon, external := s.icon(&r.GalgameWebsite, metas)
		out = append(out, WebsiteSummary{
			Object: "website", ID: repr.ID(r.ID), Host: r.URL, Title: r.Name, Description: r.Description,
			Icon: icon, ExternalIconURL: external, WebsiteCategory: categoryRef(cats[r.CategoryID]),
			IsNSFW: r.AgeLimit != "all", State: r.Status, Score: r.Score,
		})
	}
	return out, nil
}

func (s *Service) listWebsites(ctx context.Context, in *listWebsitesInput) (*listWebsitesOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	categoryID, p := optionalID(in.WebsiteCategoryID)
	if p != nil {
		return nil, p
	}
	tagID, p := optionalID(in.WebsiteTagID)
	if p != nil {
		return nil, p
	}
	filter := repository.WebsiteFilter{IncludeNSFW: in.IncludeNSFW, CategoryID: categoryID, TagID: tagID}
	fp := collect.Fingerprint("websites", strconv.FormatBool(in.IncludeNSFW), in.WebsiteCategoryID, in.WebsiteTagID)
	keys, p := collect.DecodeCursor(in.Cursor, websiteSort, fp)
	if p != nil {
		return nil, p
	}
	pos, p := parseTimePos(keys)
	if p != nil {
		return nil, p
	}
	limit := limitOf(in.Limit)
	rows, err := s.store.ListWebsites(filter, pos, limit+1)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		cur := collect.EncodeCursor(websiteSort, fp, last.CreatedAt.UTC().Format(time.RFC3339Nano), strconv.Itoa(last.ID))
		next = &cur
	}
	var total *int
	if in.IncludeTotal {
		n, err := s.store.CountWebsites(filter)
		if err != nil {
			return nil, problem.Internal(err)
		}
		total = &n
	}
	items, p := s.summaries(rows)
	if p != nil {
		return nil, p
	}
	return &listWebsitesOutput{Body: repr.NewCountedList(items, next, total)}, nil
}

func (s *Service) byHost(host string) (*repository.WebsiteRow, *problem.Problem) {
	row, err := s.store.FindWebsiteByHost(host)
	if err != nil {
		return nil, storeProblem(err)
	}
	return row, nil
}

func (s *Service) getWebsite(ctx context.Context, in *hostInput) (*websiteOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	row, p := s.byHost(in.WebsiteHost)
	if p != nil {
		return nil, p
	}
	if err := s.store.IncrementView(row.ID); err != nil {
		return nil, problem.Internal(err)
	}
	row.View++
	summary, p := s.summaries([]repository.WebsiteRow{*row})
	if p != nil {
		return nil, p
	}
	tags, err := s.store.WebsiteTags(row.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	sum := summary[0]
	out := Website{
		Object: sum.Object, ID: sum.ID, Host: sum.Host, Title: sum.Title, Description: sum.Description,
		Icon: sum.Icon, ExternalIconURL: sum.ExternalIconURL, WebsiteCategory: sum.WebsiteCategory,
		IsNSFW: sum.IsNSFW, State: sum.State, Score: sum.Score,
		Language: Language(row.Language), URLs: urlsOf(row.Domain), Founded: row.CreateTime,
		WebsiteTags: make([]WebsiteTag, 0, len(tags)), ViewCount: row.View, LikeCount: row.LikeCount,
		FavoriteCount: row.FavoriteCount, CommentCount: row.CommentCount,
		CreatedAt: repr.Timestamp(row.CreatedAt), UpdatedAt: repr.Timestamp(row.UpdatedAt),
	}
	for _, t := range tags {
		out.WebsiteTags = append(out.WebsiteTags, renderTag(t))
	}
	if viewer := v1.User(ctx); viewer != nil {
		e, err := s.store.Engagement(row.ID, viewer.ID)
		if err != nil {
			return nil, problem.Internal(err)
		}
		out.Viewer = &WebsiteViewer{
			HasLiked: e.HasLiked, HasFavorited: e.HasFavorited,
			CanEdit: viewer.Can(permEdit), CanDelete: viewer.Can(permDelete),
		}
	}
	return &websiteOutput{Body: out}, nil
}

func (s *Service) setSlot(ctx context.Context, host string, slot repository.Slot, on bool) (*engagementOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, p := s.byHost(host)
	if p != nil {
		return nil, p
	}
	if err := s.store.SetSlot(slot, row.ID, user.ID, on); err != nil {
		return nil, problem.Internal(err)
	}
	e, err := s.store.Engagement(row.ID, user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &engagementOutput{Body: WebsiteEngagement{
		Object: "website_engagement", WebsiteID: repr.ID(row.ID),
		LikeCount: e.LikeCount, FavoriteCount: e.FavoriteCount,
		Viewer: &WebsiteEngagementViewer{HasLiked: e.HasLiked, HasFavorited: e.HasFavorited},
	}}, nil
}

func (s *Service) likeWebsite(ctx context.Context, in *hostInput) (*engagementOutput, error) {
	return s.setSlot(ctx, in.WebsiteHost, repository.SlotLike, true)
}

func (s *Service) unlikeWebsite(ctx context.Context, in *hostInput) (*engagementOutput, error) {
	return s.setSlot(ctx, in.WebsiteHost, repository.SlotLike, false)
}

func (s *Service) favoriteWebsite(ctx context.Context, in *hostInput) (*engagementOutput, error) {
	return s.setSlot(ctx, in.WebsiteHost, repository.SlotFavorite, true)
}

func (s *Service) unfavoriteWebsite(ctx context.Context, in *hostInput) (*engagementOutput, error) {
	return s.setSlot(ctx, in.WebsiteHost, repository.SlotFavorite, false)
}

func (s *Service) adminWebsite(id int) (*AdminWebsite, *problem.Problem) {
	row, err := s.store.FindWebsite(id)
	if err != nil {
		return nil, storeProblem(err)
	}
	tagIDs, err := s.store.WebsiteTagIDs(id)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var metas = s.images([]string{row.IconImageHash})
	icon, external := s.icon(&row.GalgameWebsite, metas)
	ids := make([]repr.DecimalID, 0, len(tagIDs))
	for _, t := range tagIDs {
		ids = append(ids, repr.ID(t))
	}
	return &AdminWebsite{
		Object: "admin_website", ID: repr.ID(row.ID), Host: row.URL, Title: row.Name, Description: row.Description,
		Icon: icon, ExternalIconURL: external, WebsiteCategoryID: repr.ID(row.CategoryID), WebsiteTagIDs: ids,
		IsNSFW: row.AgeLimit != "all", State: row.Status, Language: Language(row.Language), URLs: urlsOf(row.Domain),
		Founded: row.CreateTime, CreatedAt: repr.Timestamp(row.CreatedAt), UpdatedAt: repr.Timestamp(row.UpdatedAt),
	}, nil
}

func (s *Service) getAdminWebsite(ctx context.Context, in *websiteIDInput) (*adminWebsiteOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permEdit); p != nil {
		return nil, p
	}
	id, ok := parseID(in.WebsiteID)
	if !ok {
		return nil, notFound()
	}
	out, p := s.adminWebsite(id)
	if p != nil {
		return nil, p
	}
	return &adminWebsiteOutput{Body: *out}, nil
}

// websiteFields validates what a create or patch sends and turns it into
// column values; nil pointers are fields the caller did not send.
type websiteFields struct {
	host, title, description, iconHash, state, language, founded *string
	categoryID                                                   *repr.DecimalID
	tagIDs                                                       []repr.DecimalID
	urls                                                         []string
	isNSFW                                                       *bool
	tagsSent, urlsSent                                           bool
}

func (s *Service) validate(f websiteFields) (map[string]any, []int, *problem.Problem) {
	cols := map[string]any{}
	var errs []problem.FieldError
	if f.host != nil {
		cols["url"] = strings.ToLower(*f.host)
	}
	if f.title != nil {
		if strings.TrimSpace(*f.title) == "" {
			errs = append(errs, tooShort("/title", 1))
		}
		cols["name"] = *f.title
	}
	if f.description != nil {
		if len([]rune(strings.TrimSpace(*f.description))) < 10 {
			errs = append(errs, tooShort("/description", 10))
		}
		cols["description"] = *f.description
	}
	if f.iconHash != nil {
		cols["icon_image_hash"] = *f.iconHash
	}
	if f.state != nil {
		cols["status"] = *f.state
	}
	if f.language != nil {
		cols["language"] = strings.ToLower(*f.language)
	}
	if f.founded != nil {
		cols["create_time"] = strings.TrimSpace(*f.founded)
	}
	if f.isNSFW != nil {
		cols["age_limit"] = "all"
		if *f.isNSFW {
			cols["age_limit"] = "r18"
		}
	}
	if f.urlsSent {
		for i, u := range f.urls {
			if !validURL(u) {
				errs = append(errs, problem.AtPointer("/urls/"+strconv.Itoa(i), problem.ReasonInvalidFormat, "must be an http or https URL of at most 100 characters", nil))
			}
		}
		raw, _ := jsonArray(f.urls)
		cols["domain"] = raw
	}
	if f.categoryID != nil {
		id, ok := parseID(string(*f.categoryID))
		exists := false
		if ok {
			var err error
			if exists, err = s.store.ExistingCategory(id); err != nil {
				return nil, nil, problem.Internal(err)
			}
		}
		if !exists {
			errs = append(errs, problem.AtPointer("/website_category_id", problem.ReasonUnknownReference, "no such category", nil))
		}
		cols["category_id"] = id
	}
	var tagIDs []int
	if f.tagsSent {
		ids, tagErrs, p := s.validateTags(f.tagIDs)
		if p != nil {
			return nil, nil, p
		}
		errs = append(errs, tagErrs...)
		tagIDs = ids
	}
	if len(errs) > 0 {
		return nil, nil, validationFailed(errs...)
	}
	return cols, tagIDs, nil
}

func (s *Service) validateTags(raw []repr.DecimalID) ([]int, []problem.FieldError, *problem.Problem) {
	var errs []problem.FieldError
	ids := make([]int, 0, len(raw))
	seen := map[int]bool{}
	for i, r := range raw {
		id, ok := parseID(string(r))
		if ok && seen[id] {
			errs = append(errs, problem.AtPointer("/website_tag_ids/"+strconv.Itoa(i), problem.ReasonDuplicateItem, "listed twice", nil))
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	tags, err := s.store.TagsByIDs(ids)
	if err != nil {
		return nil, nil, problem.Internal(err)
	}
	byID := map[int]model.GalgameWebsiteTag{}
	groupIDs := []int{}
	for _, t := range tags {
		byID[t.ID] = t
		if t.GroupID != nil {
			groupIDs = append(groupIDs, *t.GroupID)
		}
	}
	groups, err := s.store.GroupsByIDs(groupIDs)
	if err != nil {
		return nil, nil, problem.Internal(err)
	}
	multi := map[int]bool{}
	for _, g := range groups {
		multi[g.ID] = g.MultiSelect
	}
	used := map[int]bool{}
	for i, r := range raw {
		id, _ := parseID(string(r))
		t, ok := byID[id]
		if !ok {
			errs = append(errs, problem.AtPointer("/website_tag_ids/"+strconv.Itoa(i), problem.ReasonUnknownReference, "no such tag", nil))
			continue
		}
		if t.GroupID == nil || multi[*t.GroupID] {
			continue
		}
		if used[*t.GroupID] {
			errs = append(errs, problem.AtPointer("/website_tag_ids/"+strconv.Itoa(i), problem.ReasonInconsistentWith, "its group allows one tag per website", nil))
			continue
		}
		used[*t.GroupID] = true
	}
	return ids, errs, nil
}

func (s *Service) checkConflict(cols map[string]any, excludeID int) *problem.Problem {
	host, _ := cols["url"].(string)
	title, _ := cols["name"].(string)
	if host == "" && title == "" {
		return nil
	}
	c, err := s.store.WebsiteConflict(host, title, excludeID)
	if err != nil {
		return problem.Internal(err)
	}
	if c.Host {
		return taken("/host")
	}
	if c.Title {
		return taken("/title")
	}
	return nil
}

func (s *Service) createAdminWebsite(ctx context.Context, in *createWebsiteInput) (*createWebsiteOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.staff(ctx, permCreate)
	if p != nil {
		return nil, p
	}
	b := in.Body
	state := "normal"
	if b.State != nil {
		state = *b.State
	}
	founded, iconHash := "", ""
	if b.Founded != nil {
		founded = *b.Founded
	}
	if b.IconImageHash != nil {
		iconHash = *b.IconImageHash
	}
	cols, tagIDs, p := s.validate(websiteFields{
		host: &b.Host, title: &b.Title, description: &b.Description, iconHash: &iconHash, state: &state,
		language: &b.Language, founded: &founded, categoryID: &b.WebsiteCategoryID, tagIDs: b.WebsiteTagIDs,
		urls: plainURLs(b.URLs), isNSFW: &b.IsNSFW, tagsSent: true, urlsSent: true,
	})
	if p != nil {
		return nil, p
	}
	if p := s.checkConflict(cols, 0); p != nil {
		return nil, p
	}
	row := model.GalgameWebsite{
		Name: cols["name"].(string), URL: cols["url"].(string), CreateTime: cols["create_time"].(string),
		Description: cols["description"].(string), IconImageHash: cols["icon_image_hash"].(string),
		Language: cols["language"].(string), AgeLimit: cols["age_limit"].(string), Status: cols["status"].(string),
		Domain: json.RawMessage(cols["domain"].(string)), CategoryID: cols["category_id"].(int), UserID: user.ID,
	}
	if err := s.store.CreateWebsite(&row, tagIDs); err != nil {
		return nil, problem.Internal(err)
	}
	out, p := s.adminWebsite(row.ID)
	if p != nil {
		return nil, p
	}
	return &createWebsiteOutput{Location: v1.Prefix + "/admin/websites/" + strconv.Itoa(row.ID), Body: *out}, nil
}

func (s *Service) updateAdminWebsite(ctx context.Context, in *patchWebsiteInput) (*adminWebsiteOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permEdit); p != nil {
		return nil, p
	}
	id, ok := parseID(in.WebsiteID)
	if !ok {
		return nil, notFound()
	}
	if _, err := s.store.FindWebsite(id); err != nil {
		return nil, storeProblem(err)
	}
	b := in.Body
	cols, tagIDs, p := s.validate(websiteFields{
		host: b.Host, title: b.Title, description: b.Description, iconHash: b.IconImageHash, state: b.State,
		language: b.Language, founded: b.Founded, categoryID: b.WebsiteCategoryID, tagIDs: b.WebsiteTagIDs,
		urls: plainURLs(b.URLs), isNSFW: b.IsNSFW, tagsSent: b.WebsiteTagIDs != nil, urlsSent: b.URLs != nil,
	})
	if p != nil {
		return nil, p
	}
	if p := s.checkConflict(cols, id); p != nil {
		return nil, p
	}
	if b.WebsiteTagIDs != nil && tagIDs == nil {
		tagIDs = []int{}
	}
	if err := s.store.PatchWebsite(id, cols, tagIDs); err != nil {
		return nil, storeProblem(err)
	}
	out, p := s.adminWebsite(id)
	if p != nil {
		return nil, p
	}
	return &adminWebsiteOutput{Body: *out}, nil
}

func (s *Service) deleteAdminWebsite(ctx context.Context, in *websiteIDInput) (*noContentOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permDelete); p != nil {
		return nil, p
	}
	id, ok := parseID(in.WebsiteID)
	if !ok {
		return nil, notFound()
	}
	if err := s.store.DeleteWebsite(id); err != nil {
		return nil, storeProblem(err)
	}
	return &noContentOutput{}, nil
}

func (s *Service) SummariesByIDs(ids []int) (map[int]WebsiteSummary, error) {
	rows := make([]repository.WebsiteRow, 0, len(ids))
	for _, id := range ids {
		row, err := s.store.FindWebsite(id)
		if errors.Is(err, repository.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, *row)
	}
	summaries, p := s.summaries(rows)
	if p != nil {
		return nil, p
	}
	out := make(map[int]WebsiteSummary, len(summaries))
	for i, sum := range summaries {
		out[rows[i].ID] = sum
	}
	return out, nil
}
