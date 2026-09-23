package apiv1

import (
	"context"
	"strconv"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/website/model"
	"kun-galgame-api/internal/website/repository"
	"kun-galgame-api/pkg/problem"
)

const (
	tableCategory = "galgame_website_category"
	tableTag      = "galgame_website_tag"
	tableTagGroup = "galgame_website_tag_group"

	orderSort = "sort_order_asc"
	idSort    = "id_asc"
)

func categoryNotEmpty(count int) *problem.Problem {
	p := problem.New(problem.CodeWebsiteCategoryNotEmpty, "Websites are still listed under this category; move them first.")
	p.SetExtension("website_count", count)
	return p
}

func parseOrderPos(keys []string) (*repository.OrderPos, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 2 {
		return nil, invalidCursor()
	}
	order, err1 := strconv.Atoi(keys[0])
	id, err2 := strconv.Atoi(keys[1])
	if err1 != nil || err2 != nil || id <= 0 {
		return nil, invalidCursor()
	}
	return &repository.OrderPos{SortOrder: order, ID: id}, nil
}

func orderCursor(fp string, order, id int) *string {
	cur := collect.EncodeCursor(orderSort, fp, strconv.Itoa(order), strconv.Itoa(id))
	return &cur
}

func blankLabel(label *string) []problem.FieldError {
	if label != nil && strings.TrimSpace(*label) == "" {
		return []problem.FieldError{tooShort("/label", 1)}
	}
	return nil
}

func (s *Service) slugFree(table string, slug *string, excludeID int) *problem.Problem {
	if slug == nil {
		return nil
	}
	used, err := s.store.SlugTaken(table, *slug, excludeID)
	if err != nil {
		return problem.Internal(err)
	}
	if used {
		return taken("/slug")
	}
	return nil
}

func (s *Service) listWebsiteCategories(_ context.Context, in *listVocabularyInput) (*listCategoriesOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	const fp = "website_categories"
	keys, p := collect.DecodeCursor(in.Cursor, orderSort, fp)
	if p != nil {
		return nil, p
	}
	pos, p := parseOrderPos(keys)
	if p != nil {
		return nil, p
	}
	limit := limitOf(in.Limit)
	rows, err := s.store.ListCategories(pos, limit+1)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		next = orderCursor(fp, rows[limit-1].SortOrder, rows[limit-1].ID)
	}
	items := make([]WebsiteCategory, 0, len(rows))
	for _, r := range rows {
		items = append(items, renderCategory(r))
	}
	return &listCategoriesOutput{Body: repr.NewList(items, next)}, nil
}

func (s *Service) getWebsiteCategory(_ context.Context, in *categorySlugInput) (*categoryOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	row, err := s.store.FindCategoryBySlug(in.WebsiteCategorySlug)
	if err != nil {
		return nil, storeProblem(err)
	}
	return &categoryOutput{Body: renderCategory(*row)}, nil
}

func adminCategory(c model.GalgameWebsiteCategory) AdminWebsiteCategory {
	return AdminWebsiteCategory{
		Object: "admin_website_category", ID: repr.ID(c.ID), Slug: c.Name, Label: c.Label,
		Description: c.Description, SortOrder: c.SortOrder,
		CreatedAt: repr.Timestamp(c.CreatedAt), UpdatedAt: repr.Timestamp(c.UpdatedAt),
	}
}

func (s *Service) adminCategoryByID(raw string) (*AdminWebsiteCategory, *problem.Problem) {
	id, ok := parseID(raw)
	if !ok {
		return nil, notFound()
	}
	row, err := s.store.FindCategory(id)
	if err != nil {
		return nil, storeProblem(err)
	}
	out := adminCategory(row.GalgameWebsiteCategory)
	return &out, nil
}

func (s *Service) getAdminWebsiteCategory(ctx context.Context, in *categoryIDInput) (*adminCategoryOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permEdit); p != nil {
		return nil, p
	}
	out, p := s.adminCategoryByID(in.WebsiteCategoryID)
	if p != nil {
		return nil, p
	}
	return &adminCategoryOutput{Body: *out}, nil
}

func (s *Service) createAdminWebsiteCategory(ctx context.Context, in *createCategoryInput) (*createCategoryOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permCreate); p != nil {
		return nil, p
	}
	b := in.Body
	if errs := blankLabel(&b.Label); errs != nil {
		return nil, validationFailed(errs...)
	}
	if p := s.slugFree(tableCategory, &b.Slug, 0); p != nil {
		return nil, p
	}
	row := model.GalgameWebsiteCategory{Name: b.Slug, Label: b.Label, Description: b.Description, SortOrder: b.SortOrder}
	if err := s.store.Create(&row); err != nil {
		return nil, problem.Internal(err)
	}
	return &createCategoryOutput{
		Location: v1.Prefix + "/admin/website-categories/" + strconv.Itoa(row.ID),
		Body:     adminCategory(row),
	}, nil
}

func (s *Service) updateAdminWebsiteCategory(ctx context.Context, in *patchCategoryInput) (*adminCategoryOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permEdit); p != nil {
		return nil, p
	}
	id, ok := parseID(in.WebsiteCategoryID)
	if !ok {
		return nil, notFound()
	}
	if _, err := s.store.FindCategory(id); err != nil {
		return nil, storeProblem(err)
	}
	b := in.Body
	if errs := blankLabel(b.Label); errs != nil {
		return nil, validationFailed(errs...)
	}
	if p := s.slugFree(tableCategory, b.Slug, id); p != nil {
		return nil, p
	}
	fields := map[string]any{}
	setIf(fields, "name", b.Slug)
	setIf(fields, "label", b.Label)
	setIf(fields, "description", b.Description)
	setIf(fields, "sort_order", b.SortOrder)
	if len(fields) > 0 {
		if err := s.store.Patch(tableCategory, id, fields); err != nil {
			return nil, storeProblem(err)
		}
	}
	out, p := s.adminCategoryByID(in.WebsiteCategoryID)
	if p != nil {
		return nil, p
	}
	return &adminCategoryOutput{Body: *out}, nil
}

func (s *Service) deleteAdminWebsiteCategory(ctx context.Context, in *categoryIDInput) (*noContentOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permDelete); p != nil {
		return nil, p
	}
	id, ok := parseID(in.WebsiteCategoryID)
	if !ok {
		return nil, notFound()
	}
	if _, err := s.store.FindCategory(id); err != nil {
		return nil, storeProblem(err)
	}
	count, err := s.store.CountCategoryWebsites(id)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if count > 0 {
		return nil, categoryNotEmpty(count)
	}
	if err := s.store.Delete(tableCategory, id); err != nil {
		return nil, storeProblem(err)
	}
	return &noContentOutput{}, nil
}

func setIf[T any](fields map[string]any, column string, v *T) {
	if v != nil {
		fields[column] = *v
	}
}

func (s *Service) listWebsiteTags(_ context.Context, in *listVocabularyInput) (*listTagsOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	const fp = "website_tags"
	keys, p := collect.DecodeCursor(in.Cursor, idSort, fp)
	if p != nil {
		return nil, p
	}
	after := 0
	if keys != nil {
		n, err := strconv.Atoi(keys[0])
		if len(keys) != 1 || err != nil || n <= 0 {
			return nil, invalidCursor()
		}
		after = n
	}
	limit := limitOf(in.Limit)
	rows, err := s.store.ListTags(after, limit+1)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		cur := collect.EncodeCursor(idSort, fp, strconv.Itoa(rows[limit-1].ID))
		next = &cur
	}
	items := make([]WebsiteTag, 0, len(rows))
	for _, t := range rows {
		items = append(items, renderTag(t))
	}
	return &listTagsOutput{Body: repr.NewList(items, next)}, nil
}

func (s *Service) getWebsiteTag(_ context.Context, in *tagSlugInput) (*tagOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	row, err := s.store.FindTagBySlug(in.WebsiteTagSlug)
	if err != nil {
		return nil, storeProblem(err)
	}
	return &tagOutput{Body: renderTag(*row)}, nil
}

func adminTag(t model.GalgameWebsiteTag) AdminWebsiteTag {
	return AdminWebsiteTag{
		Object: "admin_website_tag", ID: repr.ID(t.ID), Slug: t.Name, Label: t.Label, Description: t.Description,
		Level: t.Level, WebsiteTagGroupID: idPtr(t.GroupID),
		CreatedAt: repr.Timestamp(t.CreatedAt), UpdatedAt: repr.Timestamp(t.UpdatedAt),
	}
}

func (s *Service) adminTagByID(id int) (*AdminWebsiteTag, *problem.Problem) {
	row, err := s.store.FindTag(id)
	if err != nil {
		return nil, storeProblem(err)
	}
	out := adminTag(*row)
	return &out, nil
}

func (s *Service) getAdminWebsiteTag(ctx context.Context, in *tagIDInput) (*adminTagOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permEdit); p != nil {
		return nil, p
	}
	id, ok := parseID(in.WebsiteTagID)
	if !ok {
		return nil, notFound()
	}
	out, p := s.adminTagByID(id)
	if p != nil {
		return nil, p
	}
	return &adminTagOutput{Body: *out}, nil
}

// groupRef resolves a group id from a request body; ok=false means the caller
// named a group that does not exist.
func (s *Service) groupRef(raw *repr.DecimalID) (*int, bool, *problem.Problem) {
	if raw == nil {
		return nil, true, nil
	}
	id, ok := parseID(string(*raw))
	if !ok {
		return nil, false, nil
	}
	if _, err := s.store.FindTagGroup(id); err != nil {
		p := storeProblem(err)
		if p.Code == problem.CodeNotFound {
			return nil, false, nil
		}
		return nil, false, p
	}
	return &id, true, nil
}

func unknownGroup() *problem.Problem {
	return validationFailed(problem.AtPointer("/website_tag_group_id", problem.ReasonUnknownReference, "no such group", nil))
}

func (s *Service) createAdminWebsiteTag(ctx context.Context, in *createTagInput) (*createTagOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permCreate); p != nil {
		return nil, p
	}
	b := in.Body
	if errs := blankLabel(&b.Label); errs != nil {
		return nil, validationFailed(errs...)
	}
	group, ok, p := s.groupRef(b.WebsiteTagGroupID.Value)
	if p != nil {
		return nil, p
	}
	if !ok {
		return nil, unknownGroup()
	}
	if p := s.slugFree(tableTag, &b.Slug, 0); p != nil {
		return nil, p
	}
	row := model.GalgameWebsiteTag{Name: b.Slug, Label: b.Label, Description: b.Description, Level: b.Level, GroupID: group}
	if err := s.store.Create(&row); err != nil {
		return nil, problem.Internal(err)
	}
	return &createTagOutput{Location: v1.Prefix + "/admin/website-tags/" + strconv.Itoa(row.ID), Body: adminTag(row)}, nil
}

func (s *Service) updateAdminWebsiteTag(ctx context.Context, in *patchTagInput) (*adminTagOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permEdit); p != nil {
		return nil, p
	}
	id, ok := parseID(in.WebsiteTagID)
	if !ok {
		return nil, notFound()
	}
	if _, err := s.store.FindTag(id); err != nil {
		return nil, storeProblem(err)
	}
	b := in.Body
	if errs := blankLabel(b.Label); errs != nil {
		return nil, validationFailed(errs...)
	}
	fields := map[string]any{}
	if b.WebsiteTagGroupID.Present {
		group, ok, p := s.groupRef(b.WebsiteTagGroupID.Value)
		if p != nil {
			return nil, p
		}
		if !ok {
			return nil, unknownGroup()
		}
		fields["group_id"] = group
	}
	if p := s.slugFree(tableTag, b.Slug, id); p != nil {
		return nil, p
	}
	setIf(fields, "name", b.Slug)
	setIf(fields, "label", b.Label)
	setIf(fields, "description", b.Description)
	setIf(fields, "level", b.Level)
	if len(fields) > 0 {
		if err := s.store.Patch(tableTag, id, fields); err != nil {
			return nil, storeProblem(err)
		}
	}
	out, p := s.adminTagByID(id)
	if p != nil {
		return nil, p
	}
	return &adminTagOutput{Body: *out}, nil
}

func (s *Service) deleteAdminWebsiteTag(ctx context.Context, in *tagIDInput) (*noContentOutput, error) {
	return s.deleteVocabulary(ctx, tableTag, in.WebsiteTagID)
}

func (s *Service) deleteVocabulary(ctx context.Context, table, raw string) (*noContentOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permDelete); p != nil {
		return nil, p
	}
	id, ok := parseID(raw)
	if !ok {
		return nil, notFound()
	}
	if err := s.store.Delete(table, id); err != nil {
		return nil, storeProblem(err)
	}
	return &noContentOutput{}, nil
}

func (s *Service) listWebsiteTagGroups(_ context.Context, in *listVocabularyInput) (*listTagGroupsOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	const fp = "website_tag_groups"
	keys, p := collect.DecodeCursor(in.Cursor, orderSort, fp)
	if p != nil {
		return nil, p
	}
	pos, p := parseOrderPos(keys)
	if p != nil {
		return nil, p
	}
	limit := limitOf(in.Limit)
	rows, err := s.store.ListTagGroups(pos, limit+1)
	if err != nil {
		return nil, problem.Internal(err)
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		next = orderCursor(fp, rows[limit-1].SortOrder, rows[limit-1].ID)
	}
	items := make([]WebsiteTagGroup, 0, len(rows))
	for _, g := range rows {
		items = append(items, renderTagGroup(g))
	}
	return &listTagGroupsOutput{Body: repr.NewList(items, next)}, nil
}

func adminTagGroup(g model.GalgameWebsiteTagGroup) AdminWebsiteTagGroup {
	return AdminWebsiteTagGroup{
		Object: "admin_website_tag_group", ID: repr.ID(g.ID), Slug: g.Name, Label: g.Label,
		Description: g.Description, SortOrder: g.SortOrder, IsMultiSelect: g.MultiSelect,
		CreatedAt: repr.Timestamp(g.CreatedAt), UpdatedAt: repr.Timestamp(g.UpdatedAt),
	}
}

func (s *Service) adminTagGroupByID(id int) (*AdminWebsiteTagGroup, *problem.Problem) {
	row, err := s.store.FindTagGroup(id)
	if err != nil {
		return nil, storeProblem(err)
	}
	out := adminTagGroup(*row)
	return &out, nil
}

func (s *Service) getAdminWebsiteTagGroup(ctx context.Context, in *tagGroupIDInput) (*adminTagGroupOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permEdit); p != nil {
		return nil, p
	}
	id, ok := parseID(in.WebsiteTagGroupID)
	if !ok {
		return nil, notFound()
	}
	out, p := s.adminTagGroupByID(id)
	if p != nil {
		return nil, p
	}
	return &adminTagGroupOutput{Body: *out}, nil
}

func (s *Service) createAdminWebsiteTagGroup(ctx context.Context, in *createTagGroupInput) (*createTagGroupOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permCreate); p != nil {
		return nil, p
	}
	b := in.Body
	if errs := blankLabel(&b.Label); errs != nil {
		return nil, validationFailed(errs...)
	}
	if p := s.slugFree(tableTagGroup, &b.Slug, 0); p != nil {
		return nil, p
	}
	row := model.GalgameWebsiteTagGroup{Name: b.Slug, Label: b.Label, Description: b.Description, SortOrder: b.SortOrder, MultiSelect: b.IsMultiSelect}
	if err := s.store.Create(&row); err != nil {
		return nil, problem.Internal(err)
	}
	return &createTagGroupOutput{
		Location: v1.Prefix + "/admin/website-tag-groups/" + strconv.Itoa(row.ID),
		Body:     adminTagGroup(row),
	}, nil
}

func (s *Service) updateAdminWebsiteTagGroup(ctx context.Context, in *patchTagGroupInput) (*adminTagGroupOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	if _, p := s.staff(ctx, permEdit); p != nil {
		return nil, p
	}
	id, ok := parseID(in.WebsiteTagGroupID)
	if !ok {
		return nil, notFound()
	}
	if _, err := s.store.FindTagGroup(id); err != nil {
		return nil, storeProblem(err)
	}
	b := in.Body
	if errs := blankLabel(b.Label); errs != nil {
		return nil, validationFailed(errs...)
	}
	if p := s.slugFree(tableTagGroup, b.Slug, id); p != nil {
		return nil, p
	}
	fields := map[string]any{}
	setIf(fields, "name", b.Slug)
	setIf(fields, "label", b.Label)
	setIf(fields, "description", b.Description)
	setIf(fields, "sort_order", b.SortOrder)
	setIf(fields, "multi_select", b.IsMultiSelect)
	if len(fields) > 0 {
		if err := s.store.Patch(tableTagGroup, id, fields); err != nil {
			return nil, storeProblem(err)
		}
	}
	out, p := s.adminTagGroupByID(id)
	if p != nil {
		return nil, p
	}
	return &adminTagGroupOutput{Body: *out}, nil
}

func (s *Service) deleteAdminWebsiteTagGroup(ctx context.Context, in *tagGroupIDInput) (*noContentOutput, error) {
	return s.deleteVocabulary(ctx, tableTagGroup, in.WebsiteTagGroupID)
}
