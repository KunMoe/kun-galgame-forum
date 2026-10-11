package apiv1

import (
	"context"
	"time"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/problem"
)

type tagBlockStore interface {
	MyTagBlocks(ctx context.Context, token string) ([]catalogclient.TagBlock, error)
	PutTagBlock(ctx context.Context, token string, tagID int64, mode string) (*catalogclient.TagBlock, error)
	DeleteTagBlock(ctx context.Context, token string, tagID int64) error
	MyTagBlock(ctx context.Context, token string, tagID int64) (*catalogclient.TagBlock, error)
}

type TagBlock struct {
	Object    string         `json:"object" enum:"tag_block" maxLength:"9" doc:"Type discriminant. Always tag_block."`
	ID        repr.DecimalID `json:"id" doc:"The blocked catalog tag's id."`
	Mode      string         `json:"mode" enum:"tree,exact" maxLength:"5" doc:"tree: send the id in exclude_tag_trees, which also drops the works under every tag below it. exact: send it in exclude_tag_ids."`
	CreatedAt repr.DateTime  `json:"created_at" doc:"When the caller blocked this tag."`
}

type listMyTagBlocksOutput struct {
	Body repr.List[TagBlock]
}

type tagBlockPathInput struct {
	TagID repr.DecimalID `path:"tag_id" doc:"Catalog tag id."`
}

type putTagBlockInput struct {
	TagID repr.DecimalID `path:"tag_id" doc:"Catalog tag id."`
	Body  *struct {
		Mode *string `json:"mode,omitempty" enum:"tree,exact" maxLength:"5" doc:"tree (the default) also blocks every tag below this one; exact blocks this tag only."`
	}
}

type tagBlockOutput struct {
	Body TagBlock
}

func tagBlockOf(b catalogclient.TagBlock) TagBlock {
	created, _ := time.Parse(time.RFC3339, b.CreatedAt)
	return TagBlock{Object: "tag_block", ID: repr.ID(int(b.TagID)), Mode: b.Mode, CreatedAt: repr.Timestamp(created)}
}

func (s *Service) tagBlocks(ctx context.Context) (tagBlockStore, string, *problem.Problem) {
	if _, p := s.requireActive(ctx); p != nil {
		return nil, "", p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, "", p
	}
	store, _ := s.catalog.(tagBlockStore)
	if store == nil {
		return nil, "", problem.Unavailable(errUnconfigured)
	}
	return store, token, nil
}

func (s *Service) listMyTagBlocks(ctx context.Context, _ *struct{}) (*listMyTagBlocksOutput, error) {
	store, token, p := s.tagBlocks(ctx)
	if p != nil {
		return nil, p
	}
	rows, err := store.MyTagBlocks(ctx, token)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	items := make([]TagBlock, len(rows))
	for i, b := range rows {
		items[i] = tagBlockOf(b)
	}
	return &listMyTagBlocksOutput{Body: repr.NewList(items, nil)}, nil
}

func (s *Service) getMyTagBlock(ctx context.Context, in *tagBlockPathInput) (*tagBlockOutput, error) {
	store, token, p := s.tagBlocks(ctx)
	if p != nil {
		return nil, p
	}
	tagID, ok := repr.ParseID(in.TagID)
	if !ok {
		return nil, notFound()
	}
	b, err := store.MyTagBlock(ctx, token, int64(tagID))
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	if b == nil {
		return nil, notFound()
	}
	return &tagBlockOutput{Body: tagBlockOf(*b)}, nil
}

func (s *Service) putMyTagBlock(ctx context.Context, in *putTagBlockInput) (*tagBlockOutput, error) {
	store, token, p := s.tagBlocks(ctx)
	if p != nil {
		return nil, p
	}
	tagID, ok := repr.ParseID(in.TagID)
	if !ok {
		return nil, notFound()
	}
	mode := catalogclient.TagBlockTree
	if in.Body != nil && in.Body.Mode != nil {
		mode = *in.Body.Mode
	}
	b, err := store.PutTagBlock(ctx, token, int64(tagID), mode)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	return &tagBlockOutput{Body: tagBlockOf(*b)}, nil
}

func (s *Service) deleteMyTagBlock(ctx context.Context, in *tagBlockPathInput) (*noContentOutput, error) {
	store, token, p := s.tagBlocks(ctx)
	if p != nil {
		return nil, p
	}
	tagID, ok := repr.ParseID(in.TagID)
	if !ok {
		return nil, notFound()
	}
	if err := store.DeleteTagBlock(ctx, token, int64(tagID)); err != nil {
		return nil, mapUserPlane(err, true)
	}
	return &noContentOutput{}, nil
}
