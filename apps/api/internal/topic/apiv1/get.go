package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) getTopic(ctx context.Context, in *getTopicInput) (*getTopicOutput, error) {
	topic, user, p := s.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	body, p := s.buildTopic(ctx, topic, user)
	if p != nil {
		return nil, p
	}
	return &getTopicOutput{Body: *body}, nil
}

func (s *Service) recordTopicView(ctx context.Context, in *recordTopicViewInput) (*struct{}, error) {
	topic, _, p := s.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	if err := s.topics.IncrementView(topic.ID); err != nil {
		return nil, problem.Internal(err)
	}
	return nil, nil
}

func (s *Service) buildTopic(ctx context.Context, topic *model.Topic, viewer *middleware.UserInfo) (*Topic, *problem.Problem) {
	if s.taxonomy == nil || s.topics == nil || s.convert == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	state, hiddenBy, err := topicLifecycle(topic.Status, topic.HiddenBy)
	if err != nil {
		return nil, problem.Internal(err)
	}
	sectionMap, err := s.taxonomy.FindSectionNamesByTopicIDs([]int{topic.ID})
	if err != nil {
		return nil, problem.Internal(err)
	}
	miniApps, err := s.topics.LookupMiniApps([]int{topic.ID})
	if err != nil {
		return nil, problem.Internal(err)
	}
	specials, p := s.loadSpecialReplies(topic)
	if p != nil {
		return nil, p
	}
	extra, p := s.loadReplyExtras(ctx, specials, viewer)
	if p != nil {
		return nil, p
	}
	samples, err := s.topics.SampleTopicReactions([]int{topic.ID})
	if err != nil {
		return nil, problem.Internal(err)
	}
	userIDs := collectReplyUserIDs(specials, extra)
	userIDs = append(userIDs, topic.UserID)
	for _, row := range samples {
		userIDs = append(userIDs, row.UserID)
	}
	users, p := s.lookupUsers(ctx, userIDs)
	if p != nil {
		return nil, p
	}
	author := repr.DeletedUserRef(topic.UserID)
	if u, ok := users[topic.UserID]; ok {
		author = repr.NewUserRef(s.cdn, u)
	}

	sources := append([]string{topic.Content}, replyBodies(specials)...)
	docs, p := s.convertBodies(ctx, sources)
	if p != nil {
		return nil, p
	}

	moeIDs := []int{topic.UserID}
	for _, r := range specials {
		moeIDs = append(moeIDs, r.UserID)
	}
	moe, err := s.topics.LookupMoemoepoints(moeIDs)
	if err != nil {
		return nil, problem.Internal(err)
	}

	tv, mine, p := s.loadTopicViewer(topic, viewer)
	if p != nil {
		return nil, p
	}

	coverMeta := map[string]imageclient.ImageMeta{}
	if hashes := coverHashList(topic.CoverImages); s.convert.Images != nil && len(hashes) > 0 {
		coverMeta = s.convert.Images(hashes)
		if coverMeta == nil {
			coverMeta = map[string]imageclient.ImageMeta{}
		}
	}

	mapped := s.mapReplies(topic, specials, docs[1:], extra, users, moe, viewer)
	pinned, best := pickSpecials(topic, specials, mapped)
	return &Topic{
		Object:            "topic",
		ID:                repr.ID(topic.ID),
		Title:             topic.Title,
		State:             state,
		HiddenBy:          hiddenBy,
		AccessScope:       topic.AccessScope,
		Category:          topic.Category,
		Section:           firstSection(sectionMap[topic.ID]),
		Sections:          toSectionSlugs(sectionMap[topic.ID]),
		AISummary:         topic.AISummary,
		CoverImages:       coverImagesWithMeta(s.cdn, topic.CoverImages, coverMeta),
		IsNSFW:            topic.IsNSFW,
		Author:            author,
		AuthorMoemoepoint: moe[topic.UserID],
		Content:           docs[0],
		ViewCount:         topic.View,
		LikeCount:         topic.LikeCount,
		DislikeCount:      topic.DislikeCount,
		FavoriteCount:     topic.FavoriteCount,
		UpvoteCount:       topic.UpvoteCount,
		ReplyCount:        topic.ReplyCount,
		CommentCount:      topic.CommentCount,
		Reactions:         reactionSummaries(samples, users, s.cdn, mine, viewer),
		MiniApps:          toMiniAppKinds(miniApps[topic.ID]),
		PinnedReply:       pinned,
		BestAnswer:        best,
		CreatedAt:         repr.Timestamp(topic.CreatedAt),
		EditedAt:          repr.TimestampPtr(topic.Edited),
		BumpedAt:          repr.Timestamp(topic.StatusUpdateTime),
		UpvotedAt:         repr.TimestampPtr(topic.UpvoteTime),
		Viewer:            tv,
	}, nil
}

func (s *Service) loadSpecialReplies(topic *model.Topic) ([]model.TopicReply, *problem.Problem) {
	ids := make([]int, 0, 2)
	if topic.PinnedReplyID != nil {
		ids = append(ids, *topic.PinnedReplyID)
	}
	if topic.BestAnswerID != nil && (topic.PinnedReplyID == nil || *topic.BestAnswerID != *topic.PinnedReplyID) {
		ids = append(ids, *topic.BestAnswerID)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if s.replies == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	rows, err := s.replies.FindRepliesByIDs(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	byID := map[int]model.TopicReply{}
	for _, row := range rows {
		if row.TopicID != topic.ID {
			continue
		}
		byID[row.ID] = row.TopicReply
	}
	out := make([]model.TopicReply, 0, len(ids))
	for _, id := range ids {
		if r, ok := byID[id]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func pickSpecials(topic *model.Topic, rows []model.TopicReply, mapped []*Reply) (*Reply, *Reply) {
	byID := map[int]*Reply{}
	for i, row := range rows {
		if i < len(mapped) && mapped[i] != nil {
			byID[row.ID] = mapped[i]
		}
	}
	var pinned, best *Reply
	if topic.PinnedReplyID != nil {
		pinned = byID[*topic.PinnedReplyID]
	}
	if topic.BestAnswerID != nil {
		best = byID[*topic.BestAnswerID]
	}
	return pinned, best
}

func coverHashList(tokens model.ImageTokens) []string {
	var hashes []string
	seen := map[string]struct{}{}
	for _, tok := range tokens {
		h, _, ok := markdown.ParseContentImageRef(tok)
		if !ok {
			continue
		}
		if _, dup := seen[h]; dup {
			continue
		}
		seen[h] = struct{}{}
		hashes = append(hashes, h)
	}
	return hashes
}
