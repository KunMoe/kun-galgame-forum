package apiv1

import (
	"context"
	"strings"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

type listCollectionTagsInput struct {
	IncludeNSFW bool `query:"include_nsfw" default:"false" doc:"When true, collections holding an adult work are counted. Default false, the same rule as the directory."`
	Limit       int  `query:"limit" minimum:"1" maximum:"100" default:"30" doc:"How many tags, most used first. 1–100, default 30."`
}

type CollectionTagCount struct {
	Object          string        `json:"object" enum:"collection_tag" maxLength:"14" doc:"Type discriminant. Always collection_tag."`
	Tag             CollectionTag `json:"tag" doc:"The tag; pass it back as tag= on listCollections. Free text; never use it as a decision input."`
	CollectionCount int           `json:"collection_count" minimum:"0" doc:"Directory collections carrying it."`
}

type collectionTagListOutput struct {
	Body repr.List[CollectionTagCount]
}

func (s *Service) listCollectionTags(ctx context.Context, in *listCollectionTagsInput) (*collectionTagListOutput, error) {
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	rows, err := s.catalog.FolderTags(ctx, in.Limit, in.IncludeNSFW)
	if err != nil {
		return nil, mapUserPlane(err, false)
	}
	items := make([]CollectionTagCount, 0, len(rows))
	for _, r := range rows {
		items = append(items, CollectionTagCount{Object: "collection_tag", Tag: CollectionTag(r.Tag), CollectionCount: max(r.FolderCount, 0)})
	}
	return &collectionTagListOutput{Body: repr.NewList(items, nil)}, nil
}

type copyCollectionInput struct {
	Body collectionCopyCreate
}

type collectionCopyCreate struct {
	CollectionID string  `json:"collection_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"The collection to copy: a public or unlisted one, or one of the caller's own."`
	Title        *string `json:"title,omitempty" maxLength:"100" doc:"Title of the copy. Absent or blank: the source's title. Free text; never use it as a decision input."`
}

func (s *Service) copyCollection(ctx context.Context, in *copyCollectionInput) (*createCollectionOutput, error) {
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	sourceID, ok := parseCollectionID(in.Body.CollectionID)
	if !ok {
		return nil, notFound()
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	title := strings.TrimSpace(derefOr(in.Body.Title, ""))
	if title != "" {
		if _, p := s.checkCollectionText(ctx, title, user.ID); p != nil {
			return nil, p
		}
	}
	folder, err := s.catalog.CopyFolder(ctx, token, sourceID, title, idempotencyKey(ctx))
	if err != nil {
		return nil, mapUserPlaneAt(err, true, copyPointer)
	}
	col, ok, p := s.collectionOf(ctx, *folder, user, token, viewerState{}, false, 0)
	if p != nil {
		return nil, p
	}
	if !ok {
		return nil, notFound()
	}
	return &createCollectionOutput{Location: "/api/v1/collections/" + string(col.ID), Body: *col}, nil
}

func copyPointer(upstream string) (string, bool) {
	switch upstream {
	case "/folder_id":
		return "/collection_id", true
	case "/name":
		return "/title", true
	}
	return "", false
}

type FollowedCollectionUpdates struct {
	Object       string `json:"object" enum:"followed_collection_updates" maxLength:"27" doc:"Type discriminant. Always followed_collection_updates."`
	UpdatedCount int    `json:"updated_count" minimum:"0" maximum:"100" doc:"Followed collections that gained a work since the caller last opened them. 100 means 100 or more."`
}

type followedUpdatesOutput struct {
	Body FollowedCollectionUpdates
}

func (s *Service) getFollowedCollectionUpdates(ctx context.Context, _ *struct{}) (*followedUpdatesOutput, error) {
	if _, p := s.requireActive(ctx); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	n, err := s.catalog.FollowedUpdates(ctx, token)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	return &followedUpdatesOutput{Body: FollowedCollectionUpdates{Object: "followed_collection_updates", UpdatedCount: min(max(n, 0), 100)}}, nil
}

func (s *Service) putFollowedCollectionReadMarker(ctx context.Context, in *collectionFollowInput) (*noContentOutput, error) {
	_, token, folderID, p := s.followPrelude(ctx, in.CollectionID)
	if p != nil {
		return nil, p
	}
	if err := s.catalog.MarkFollowedSeen(ctx, token, folderID); err != nil {
		return nil, mapUserPlane(err, true)
	}
	return &noContentOutput{}, nil
}

type listCollectionProgressInput struct {
	CollectionIDs []repr.DecimalID `query:"collection_ids" required:"true" minItems:"1" maxItems:"50" doc:"Collection ids, comma-separated. 1–50."`
}

type CollectionProgress struct {
	Object         string         `json:"object" enum:"collection_progress" maxLength:"19" doc:"Type discriminant. Always collection_progress."`
	CollectionID   repr.DecimalID `json:"collection_id" doc:"Catalog folder id."`
	ItemCount      int            `json:"item_count" minimum:"0" doc:"Live works in the collection."`
	CollectedCount int            `json:"collected_count" minimum:"0" doc:"Of those, how many the caller keeps in another collection of their own."`
	WishCount      int            `json:"wish_count" minimum:"0" doc:"Of those, how many carry the caller's play state wish."`
	DoingCount     int            `json:"doing_count" minimum:"0" doc:"The same for doing."`
	DoneCount      int            `json:"done_count" minimum:"0" doc:"The same for any finished state."`
	OnHoldCount    int            `json:"on_hold_count" minimum:"0" doc:"The same for on_hold."`
	DroppedCount   int            `json:"dropped_count" minimum:"0" doc:"The same for dropped."`
}

type collectionProgressOutput struct {
	Body repr.BatchList[CollectionProgress]
}

func (s *Service) listCollectionProgress(ctx context.Context, in *listCollectionProgressInput) (*collectionProgressOutput, error) {
	if _, p := s.requireActive(ctx); p != nil {
		return nil, p
	}
	token, p := requireToken(ctx)
	if p != nil {
		return nil, p
	}
	if s.catalog == nil {
		return nil, problem.Unavailable(errUnconfigured)
	}
	ids := make([]int64, 0, len(in.CollectionIDs))
	for _, raw := range in.CollectionIDs {
		id, ok := parseCollectionID(string(raw))
		if !ok {
			return nil, invalidParameter(problem.AtParameter("collection_ids", problem.ReasonInvalidFormat, "each id must be a positive decimal integer", nil))
		}
		ids = append(ids, id)
	}
	rows, err := s.catalog.FolderProgress(ctx, token, ids)
	if err != nil {
		return nil, mapUserPlane(err, true)
	}
	got := map[int64]bool{}
	items := make([]CollectionProgress, 0, len(rows))
	for _, r := range rows {
		got[r.FolderID] = true
		items = append(items, CollectionProgress{
			Object: "collection_progress", CollectionID: repr.ID(int(r.FolderID)), ItemCount: max(r.ItemCount, 0),
			CollectedCount: max(r.Collected, 0), WishCount: max(r.Wish, 0), DoingCount: max(r.Doing, 0),
			DoneCount: max(r.Done, 0), OnHoldCount: max(r.OnHold, 0), DroppedCount: max(r.Dropped, 0),
		})
	}
	missing := []repr.DecimalID{}
	for _, id := range ids {
		if !got[id] {
			missing = append(missing, repr.ID(int(id)))
		}
	}
	return &collectionProgressOutput{Body: repr.NewBatchList(items, missing)}, nil
}
