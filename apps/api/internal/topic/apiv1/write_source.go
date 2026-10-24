package apiv1

import (
	"context"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

func (w *Writes) getTopicSource(ctx context.Context, in *getTopicSourceInput) (*getTopicSourceOutput, error) {
	if w == nil || w.reads == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	topic, user, p := w.reads.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	if !capsForTopic(topic, user).Edit {
		return nil, permissionRequired()
	}
	sections, err := w.reads.taxonomy.FindSectionNamesByTopicID(topic.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if sections == nil {
		sections = []string{}
	}
	grants, err := repository.ListAccessGrants(w.db(), topic.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	roles, userIDs := splitGrants(grants)
	access := AccessGrants{Roles: []AccessRole{}, Users: []repr.UserRef{}}
	if topic.AccessScope == "role" {
		access.Roles = roles
	}
	if topic.AccessScope == "users" {
		ids := make([]int, 0, len(userIDs))
		for _, raw := range userIDs {
			if id, ok := repr.ParseID(raw); ok {
				ids = append(ids, id)
			}
		}
		users, p := w.reads.lookupUsers(ctx, ids)
		if p != nil {
			return nil, p
		}
		refs := make([]repr.UserRef, 0, len(ids))
		for _, id := range ids {
			if u, ok := users[id]; ok && userclient.IsRenderable(u) {
				refs = append(refs, repr.NewUserRef(w.reads.cdn, u))
				continue
			}
			refs = append(refs, repr.DeletedUserRef(id))
		}
		access.Users = refs
	}

	coverMeta := map[string]imageclient.ImageMeta{}
	if hashes := coverHashList(topic.CoverImages); w.reads.convert != nil && w.reads.convert.Images != nil && len(hashes) > 0 {
		coverMeta = w.reads.convert.Images(hashes)
		if coverMeta == nil {
			coverMeta = map[string]imageclient.ImageMeta{}
		}
	}

	return &getTopicSourceOutput{Body: TopicSource{
		Object:          "topic_source",
		TopicID:         repr.ID(topic.ID),
		Title:           topic.Title,
		ContentMarkdown: topic.Content,
		Category:        topic.Category,
		Section:         firstSection(sections),
		Sections:        toSectionSlugs(sections),
		IsNSFW:          topic.IsNSFW,
		CoverImages:     coverImagesWithMeta(w.reads.cdn, topic.CoverImages, coverMeta),
		AccessScope:     topic.AccessScope,
		AccessGrants:    access,
	}}, nil
}

func (w *Writes) getReplySource(ctx context.Context, in *getReplySourceInput) (*getReplySourceOutput, error) {
	if w == nil || w.reads == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	topic, row, user, p := w.reads.visibleReply(ctx, in.ReplyID)
	if p != nil {
		return nil, p
	}
	if !capsForReply(topic, row, user).Edit {
		return nil, permissionRequired()
	}
	return &getReplySourceOutput{Body: ReplySource{
		Object:          "reply_source",
		ReplyID:         repr.ID(row.ID),
		TopicID:         repr.ID(row.TopicID),
		ContentMarkdown: row.Content,
	}}, nil
}
