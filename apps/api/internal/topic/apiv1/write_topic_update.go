package apiv1

import (
	"context"
	"fmt"
	"time"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func (w *Writes) updateTopic(ctx context.Context, in *updateTopicInput) (*updateTopicOutput, error) {
	if w == nil || w.reads == nil || w.db() == nil {
		return nil, problem.Internal(errUnconfigured)
	}
	topic, user, p := w.reads.visibleTopic(ctx, in.TopicID)
	if p != nil {
		return nil, p
	}
	caps := capsForTopic(topic, user)
	patch := in.Body
	needEdit := patch.Title != nil || patch.ContentMarkdown != nil || patch.Category != nil ||
		patch.Sections != nil || patch.IsNSFW != nil || patch.CoverImageHashes != nil ||
		patch.AccessScope != nil || patch.AccessRoles != nil || patch.AccessUserIDs != nil
	needHide := false
	needUnhide := false
	if patch.State != nil && *patch.State != topicStateName(topic.Status) {
		if *patch.State == "hidden" {
			needHide = true
		} else {
			needUnhide = true
		}
	}
	if (needEdit && !caps.Edit) || (needHide && !caps.Hide) || (needUnhide && !caps.Unhide) {
		return nil, permissionRequired()
	}

	storedSections, err := w.reads.taxonomy.FindSectionNamesByTopicID(topic.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	storedGrants, err := w.reads.topics.FindAccessGrants(topic.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}

	title := topic.Title
	if patch.Title != nil {
		title = trimTitle(*patch.Title)
	}
	body := topic.Content
	if patch.ContentMarkdown != nil {
		body = markdown.NormalizeStoredContent(*patch.ContentMarkdown)
	}
	section := ""
	if len(storedSections) > 0 {
		section = storedSections[0]
	}
	storedSection := section
	if patch.Sections != nil {
		section, _ = NormalizeSectionKey(string((*patch.Sections)[0]))
	}
	if section == "" {
		return nil, validationFailed(problem.AtPointer("/sections", problem.ReasonRequired, "the topic has no section yet", nil))
	}
	rules, err := w.reads.taxonomy.FindSectionRules(nil, []string{storedSection, section})
	if err != nil {
		return nil, problem.Internal(err)
	}
	rule, ok := rules[section]
	if !ok {
		return nil, problem.Internal(fmt.Errorf("topic section %q is missing", section))
	}
	category := rule.LegacyCategory
	nsfw := topic.IsNSFW
	if patch.IsNSFW != nil {
		nsfw = *patch.IsNSFW
	}
	covers := topic.CoverImages
	if patch.CoverImageHashes != nil {
		covers = coversFromHashes(*patch.CoverImageHashes)
	}
	access := mergeAccess(topic.AccessScope, storedGrants, patch)

	var fields []problem.FieldError
	if patch.Title != nil && title == "" {
		fields = append(fields, tooShort("/title"))
	}
	if patch.ContentMarkdown != nil && stringsTrimEmpty(*patch.ContentMarkdown) {
		fields = append(fields, tooShort("/content_markdown"))
	}
	if section != storedSection && w.ai.ClassifyEnabled() && !user.Can(perm.TopicEditAny) {
		fields = append(fields, problem.AtPointer("/sections", problem.ReasonNotPermitted, "the model decides the section; ask staff to move the topic", nil))
	}
	if section != storedSection && rule.IsArchived {
		fields = append(fields, sectionArchived("/sections/0"))
	}
	fields = append(fields, accessErrors(access)...)
	if len(fields) > 0 {
		return nil, validationFailed(fields...)
	}

	grants := grantsFromAccess(access, topic.UserID)
	titleChanged := patch.Title != nil && title != topic.Title
	bodyChanged := patch.ContentMarkdown != nil && body != topic.Content
	textChanged := titleChanged || bodyChanged
	categoryChanged := category != topic.Category
	nsfwChanged := patch.IsNSFW != nil && nsfw != topic.IsNSFW
	coversChanged := patch.CoverImageHashes != nil && !coversEqual(covers, topic.CoverImages)
	scopeChanged := access.scope != topic.AccessScope
	grantsTouched := patch.AccessScope != nil || patch.AccessRoles != nil || patch.AccessUserIDs != nil
	sectionChanged := section != storedSection
	stateChanged := needHide || needUnhide
	oldConsume := rules[storedSection].CostsMoemoepoint
	newConsume := rule.CostsMoemoepoint
	footprintDelta := topicSectionFootprint(newConsume) - topicSectionFootprint(oldConsume)

	if !titleChanged && !bodyChanged && !categoryChanged && !nsfwChanged && !coversChanged &&
		!scopeChanged && !grantsTouched && !sectionChanged && !stateChanged {
		out, p := w.reads.buildTopic(ctx, topic, user)
		if p != nil {
			return nil, p
		}
		return &updateTopicOutput{Body: *out}, nil
	}

	var verdict gate.Verdict
	moderation := topicModerationText(title, body)
	if textChanged {
		var rp *problem.Problem
		verdict, rp = w.rejectContent(ctx, moderation, topic.UserID)
		if rp != nil {
			return nil, rp
		}
	}

	drift := 0
	if bodyChanged {
		drift = editDrift(topic.Content, body)
	}
	var awards []pendingAward
	err = w.db().Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{}
		if titleChanged {
			updates["title"] = title
		}
		if bodyChanged {
			updates["content"] = body
		}
		if categoryChanged {
			updates["category"] = category
		}
		if nsfwChanged {
			updates["is_nsfw"] = nsfw
		}
		if coversChanged {
			updates["cover_images"] = covers
		}
		if scopeChanged {
			updates["access_scope"] = access.scope
		}
		if textChanged {
			now := time.Now()
			updates["edited"] = &now
		}
		if needHide {
			updates["status"] = 1
			hiddenBy := "moderator"
			if user.ID == topic.UserID {
				hiddenBy = "author"
			}
			updates["hidden_by"] = hiddenBy
		}
		if needUnhide {
			updates["status"] = 0
			updates["hidden_by"] = ""
		}
		if len(updates) > 0 {
			if err := w.reads.topics.UpdateTopicFields(tx, topic.ID, updates); err != nil {
				return err
			}
		}
		if textChanged {
			if err := w.reads.topics.TouchStatusUpdateTime(tx, topic.ID, time.Now()); err != nil {
				return err
			}
		}
		if grantsTouched {
			if err := w.reads.topics.ReplaceAccessGrants(tx, topic.ID, grants); err != nil {
				return err
			}
		}
		if sectionChanged {
			if err := w.reads.taxonomy.ReplaceSectionRelations(tx, topic.ID, []int{rule.ID}); err != nil {
				return err
			}
		}
		if drift > 0 && w.ai != nil {
			if err := w.ai.repo.AddSummaryDrift(tx, topic.ID, drift); err != nil {
				return err
			}
		}
		if textChanged {
			if err := w.notifyTopicMentions(tx, topic.UserID, topic.ID, body); err != nil {
				return err
			}
		}
		if footprintDelta != 0 {
			awards = append(awards, topicCostChangedAward(topic.UserID, topic.ID, footprintDelta))
		}
		return nil
	})
	if p := txProblem(err); p != nil {
		return nil, p
	}
	if err != nil {
		return nil, problem.Internal(err)
	}
	w.flushAwards(awards)
	if textChanged {
		w.scanTopic(verdict, topic.ID, topic.UserID, moderation)
	}
	if bodyChanged {
		w.ai.AfterEdit(topic, drift, title, body)
	}

	fresh, p := w.loadTopic(topic.ID)
	if p != nil {
		return nil, p
	}
	out, p := w.reads.buildTopic(ctx, fresh, user)
	if p != nil {
		return nil, p
	}
	return &updateTopicOutput{Body: *out}, nil
}

func topicStateName(status int) string {
	if status == 1 {
		return "hidden"
	}
	return "published"
}

func stringsTrimEmpty(s string) bool {
	return trimTitle(s) == ""
}

func coversEqual(a, b model.ImageTokens) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
