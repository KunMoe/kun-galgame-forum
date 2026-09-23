package apiv1

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/doc/repository"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func (s *Service) loadAdminDoc(id int) (*AdminDoc, *problem.Problem) {
	row, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound()
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	doc, prob := s.adminDoc(row)
	if prob != nil {
		return nil, prob
	}
	return &doc, nil
}

func parseDocID(raw string) (int, *problem.Problem) {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, notFound()
	}
	return id, nil
}

func (s *Service) getAdminDoc(ctx context.Context, in *adminDocInput) (*adminDocOutput, error) {
	if prob := s.require(ctx, perm.DocEdit); prob != nil {
		return nil, prob
	}
	id, prob := parseDocID(in.DocID)
	if prob != nil {
		return nil, prob
	}
	doc, prob := s.loadAdminDoc(id)
	if prob != nil {
		return nil, prob
	}
	return &adminDocOutput{Body: *doc}, nil
}

func bannerValue(h *BannerImageHash) string {
	if h == nil {
		return ""
	}
	return string(*h)
}

func (s *Service) createDoc(ctx context.Context, in *createDocInput) (*createDocOutput, error) {
	if prob := s.require(ctx, perm.DocCreate); prob != nil {
		return nil, prob
	}
	body := in.Body
	title := trimText(body.Title)
	source := markdown.NormalizeStoredContent(body.ContentMarkdown)
	var fields []problem.FieldError
	if title == "" {
		fields = append(fields, tooShort("/title"))
	}
	if strings.TrimSpace(source) == "" {
		fields = append(fields, tooShort("/content_markdown"))
	}
	if len(fields) > 0 {
		return nil, validationFailed(fields...)
	}
	description := ""
	if body.Description != nil {
		description = trimText(*body.Description)
	}
	id, err := s.repo.Create(repository.DocWrite{
		Slug:            body.Slug,
		Title:           title,
		Description:     description,
		Category:        body.DocCategory,
		BannerImageHash: bannerValue(body.BannerImageHash),
		IsPin:           body.IsPinned != nil && *body.IsPinned,
		ContentMarkdown: source,
		AuthorID:        v1.User(ctx).ID,
	})
	if errors.Is(err, repository.ErrSlugTaken) {
		return nil, slugTaken()
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	doc, prob := s.loadAdminDoc(id)
	if prob != nil {
		return nil, prob
	}
	return &createDocOutput{
		Location: "/api/v1/admin/docs/" + strconv.Itoa(id),
		Body:     *doc,
	}, nil
}

func (s *Service) updateDoc(ctx context.Context, in *updateDocInput) (*adminDocOutput, error) {
	if prob := s.require(ctx, perm.DocEdit); prob != nil {
		return nil, prob
	}
	id, prob := parseDocID(in.DocID)
	if prob != nil {
		return nil, prob
	}
	p := in.Body
	if p.Slug == nil && p.Title == nil && p.Description == nil && p.DocCategory == nil &&
		p.BannerImageHash == nil && p.IsPinned == nil && p.ContentMarkdown == nil {
		return nil, validationFailed(problem.AtPointer("", problem.ReasonRequired, "send at least one field to change", nil))
	}
	ch := repository.DocChanges{Slug: p.Slug, Category: p.DocCategory, IsPin: p.IsPinned}
	var fields []problem.FieldError
	if p.Title != nil {
		t := trimText(*p.Title)
		if t == "" {
			fields = append(fields, tooShort("/title"))
		}
		ch.Title = &t
	}
	if p.ContentMarkdown != nil {
		src := markdown.NormalizeStoredContent(*p.ContentMarkdown)
		if strings.TrimSpace(src) == "" {
			fields = append(fields, tooShort("/content_markdown"))
		}
		ch.ContentMarkdown = &src
	}
	if len(fields) > 0 {
		return nil, validationFailed(fields...)
	}
	if p.Description != nil {
		d := trimText(*p.Description)
		ch.Description = &d
	}
	if p.BannerImageHash != nil {
		h := string(*p.BannerImageHash)
		ch.BannerImageHash = &h
	}
	err := s.repo.Update(id, ch)
	if errors.Is(err, repository.ErrSlugTaken) {
		return nil, slugTaken()
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	doc, prob := s.loadAdminDoc(id)
	if prob != nil {
		return nil, prob
	}
	return &adminDocOutput{Body: *doc}, nil
}

func (s *Service) deleteDoc(ctx context.Context, in *adminDocInput) (*struct{}, error) {
	if prob := s.require(ctx, perm.DocDelete); prob != nil {
		return nil, prob
	}
	id, prob := parseDocID(in.DocID)
	if prob != nil {
		return nil, prob
	}
	deleted, err := s.repo.Delete(id)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if !deleted {
		return nil, notFound()
	}
	return &struct{}{}, nil
}

func (s *Service) putDocOrder(ctx context.Context, in *putDocOrderInput) (*struct{}, error) {
	if prob := s.require(ctx, perm.DocEdit); prob != nil {
		return nil, prob
	}
	ids := make([]int, len(in.Body.DocIDs))
	for i, raw := range in.Body.DocIDs {
		id, ok := repr.ParseID(raw)
		if !ok {
			return nil, validationFailed(problem.AtPointer(fmt.Sprintf("/doc_ids/%d", i), problem.ReasonUnknownReference, "no doc has this id", nil))
		}
		ids[i] = id
	}
	res, applied, err := s.repo.Reorder(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if res.UnknownAt >= 0 {
		return nil, validationFailed(problem.AtPointer(fmt.Sprintf("/doc_ids/%d", res.UnknownAt), problem.ReasonUnknownReference, "no doc has this id", nil))
	}
	if !applied {
		n := res.TotalCount
		return nil, validationFailed(problem.AtPointer("/doc_ids", problem.ReasonTooFewItems, "list every doc exactly once", &problem.FieldParams{MinItems: &n}))
	}
	return &struct{}{}, nil
}
