package apiv1

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func (w *Writes) createTopic(ctx context.Context, in *createTopicInput) (*createTopicOutput, error) {
	if w == nil || w.reads == nil || w.db() == nil || w.state == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	user := w.caller(ctx)
	title := trimTitle(in.Body.Title)
	body := markdown.NormalizeStoredContent(in.Body.ContentMarkdown)
	var fields []problem.FieldError
	if title == "" {
		fields = append(fields, tooShort("/title"))
	}
	if strings.TrimSpace(in.Body.ContentMarkdown) == "" {
		fields = append(fields, tooShort("/content_markdown"))
	}
	sections := make([]string, len(in.Body.Sections))
	for i, s := range in.Body.Sections {
		sections[i] = string(s)
	}
	fields = append(fields, sectionFields(sections, in.Body.Category)...)
	access := accessFields{scope: in.Body.AccessScope, roles: in.Body.AccessRoles, users: in.Body.AccessUserIDs}
	fields = append(fields, accessErrors(access)...)
	if len(fields) > 0 {
		return nil, validationFailed(fields...)
	}

	var covers model.ImageTokens
	if in.Body.CoverImageHashes == nil {
		covers = deriveCovers(body)
	} else {
		covers = coversFromHashes(*in.Body.CoverImageHashes)
	}
	grants := grantsFromAccess(access, user.ID)
	consume := anyConsumeSection(sections)
	moderation := topicModerationText(title, body)
	decision, matched, p := w.rejectContent(ctx, moderation, user.ID)
	if p != nil {
		return nil, p
	}

	var awards []pendingAward
	var topicID int
	err := w.db().Transaction(func(tx *gorm.DB) error {
		state, err := w.state.LockForUpdate(tx, user.ID)
		if err != nil {
			return err
		}
		todayCount, err := w.reads.topics.CountTodayTopicsByUser(tx, user.ID)
		if err != nil {
			return err
		}
		limit := state.Moemoepoint/constants.DailyTopicPerMoemoepoint + 1
		if todayCount >= int64(limit) {
			return txFail{p: dailyLimitReached(limit)}
		}
		if consume && state.Moemoepoint < constants.CostConsumeSection {
			return txFail{p: moemoepointInsufficient(constants.CostConsumeSection)}
		}
		topic := &model.Topic{
			AccessScope: in.Body.AccessScope,
			Title:       title,
			Content:     body,
			Category:    in.Body.Category,
			IsNSFW:      in.Body.IsNSFW,
			UserID:      user.ID,
			CoverImages: covers,
		}
		if err := w.reads.topics.CreateTopic(tx, topic); err != nil {
			return err
		}
		if err := w.reads.topics.ReplaceAccessGrants(tx, topic.ID, grants); err != nil {
			return err
		}
		if err := w.writeSections(tx, topic.ID, sections); err != nil {
			return err
		}
		if err := w.notifyMentions(tx, user.ID, topic.ID, 0, body); err != nil {
			return err
		}
		topicID = topic.ID
		awards = append(awards, topicCreatedAward(user.ID, topic.ID, consume))
		return nil
	})
	if p := txProblem(err); p != nil {
		return nil, p
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	w.flushAwards(awards)
	w.scanTopic(decision, matched, topicID, user.ID, moderation)
	w.notifyFollowersOfTopic(user.ID, topicID, in.Body.AccessScope)

	topic, p := w.loadTopic(topicID)
	if p != nil {
		return nil, p
	}
	out, p := w.reads.buildTopic(ctx, topic, user)
	if p != nil {
		return nil, p
	}
	return &createTopicOutput{
		Location: "/api/v1/topics/" + strconv.Itoa(topicID),
		Body:     *out,
	}, nil
}

func (w *Writes) writeSections(tx *gorm.DB, topicID int, names []string) error {
	if len(names) == 0 {
		return w.reads.taxonomy.ReplaceSectionRelations(tx, topicID, nil)
	}
	found, err := w.reads.taxonomy.FindSectionsByNamesTx(tx, names)
	if err != nil {
		return err
	}
	byName := make(map[string]int, len(found))
	for _, s := range found {
		byName[s.Name] = s.ID
	}
	ids := make([]int, 0, len(names))
	for _, name := range names {
		id, ok := byName[name]
		if !ok {
			return fmt.Errorf("topic section %q is missing", name)
		}
		ids = append(ids, id)
	}
	return w.reads.taxonomy.ReplaceSectionRelations(tx, topicID, ids)
}
