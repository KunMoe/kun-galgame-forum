package apiv1

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/topic/model"
	"kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

const MaxTopicDraftsPerUser = 30

type Drafts struct {
	drafts *repository.TopicDraftRepository
}

func NewDrafts(drafts *repository.TopicDraftRepository) *Drafts {
	return &Drafts{drafts: drafts}
}

func (d *Drafts) ready() *problem.Problem {
	if d == nil || d.drafts == nil || d.drafts.DB() == nil {
		return problem.Internal(errUnconfigured)
	}
	return nil
}

func draftLimitReached(limit int) *problem.Problem {
	p := problem.New(problem.CodeDraftLimitReached,
		"The caller already holds the maximum number of drafts. A draft is never overwritten, so the only way to make room is to delete one.")
	p.SetExtension("limit", limit)
	return p
}

type listTopicDraftsInput struct {
	collect.Page
}

type listTopicDraftsOutput struct {
	Body repr.List[TopicDraftSummary]
}

type topicDraftInput struct {
	DraftID string `path:"draft_id" pattern:"^[1-9][0-9]{0,18}$" maxLength:"19" doc:"Draft id."`
}

type topicDraftOutput struct {
	Body TopicDraft
}

type createTopicDraftInput struct {
	Body TopicDraftCreate
}

type createTopicDraftOutput struct {
	Location string `header:"Location" format:"uri-reference" maxLength:"64" doc:"Absolute path of the new draft, such as /api/v1/me/topic-drafts/412."`
	Body     TopicDraft
}

type deleteTopicDraftOutput struct{}

const draftSort = "updated_desc"

func draftFingerprint(userID int) string {
	return collect.Fingerprint("topic_drafts", strconv.Itoa(userID))
}

func parseDraftPos(keys []string) (*repository.DraftKeysetPos, *problem.Problem) {
	if keys == nil {
		return nil, nil
	}
	if len(keys) != 2 {
		return nil, invalidCursor()
	}
	updated, err := time.Parse(time.RFC3339Nano, keys[0])
	if err != nil {
		return nil, invalidCursor()
	}
	id, err := strconv.Atoi(keys[1])
	if err != nil || id <= 0 {
		return nil, invalidCursor()
	}
	return &repository.DraftKeysetPos{Updated: updated, ID: id}, nil
}

func (d *Drafts) listTopicDrafts(ctx context.Context, in *listTopicDraftsInput) (*listTopicDraftsOutput, error) {
	if prob := d.ready(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, notFound()
	}
	fp := draftFingerprint(user.ID)
	keys, curErr := collect.DecodeCursor(in.Cursor, draftSort, fp)
	if curErr != nil {
		return nil, curErr
	}
	pos, posErr := parseDraftPos(keys)
	if posErr != nil {
		return nil, posErr
	}
	limit := in.Limit
	if limit <= 0 {
		limit = collect.DefaultLimit
	}

	rows, err := d.drafts.FindKeysetForUser(user.ID, limit+1, pos)
	if err != nil {
		return nil, problem.Internal(err)
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	items := make([]TopicDraftSummary, len(rows))
	for i, row := range rows {
		items[i] = TopicDraftSummary{
			Object:    "topic_draft_summary",
			ID:        repr.ID(row.ID),
			Title:     row.Title,
			Summary:   row.Summary,
			CreatedAt: repr.Timestamp(row.CreatedAt),
			UpdatedAt: repr.Timestamp(row.UpdatedAt),
		}
	}
	var next *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		cur := collect.EncodeCursor(draftSort, fp,
			last.UpdatedAt.UTC().Format(time.RFC3339Nano), strconv.Itoa(last.ID))
		next = &cur
	}
	return &listTopicDraftsOutput{Body: repr.NewList(items, next)}, nil
}

func (d *Drafts) getTopicDraft(ctx context.Context, in *topicDraftInput) (*topicDraftOutput, error) {
	draft, prob := d.ownDraft(ctx, in.DraftID)
	if prob != nil {
		return nil, prob
	}
	body, err := mapDraft(draft)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &topicDraftOutput{Body: body}, nil
}

func (d *Drafts) createTopicDraft(ctx context.Context, in *createTopicDraftInput) (*createTopicDraftOutput, error) {
	if prob := d.ready(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, notFound()
	}

	content := markdown.NormalizeStoredContent(in.Body.ContentMarkdown)
	if strings.TrimSpace(in.Body.Title) == "" && strings.TrimSpace(content) == "" {
		min := 1
		return nil, validationFailed(problem.AtPointer("/content_markdown", problem.ReasonRequired,
			"a draft needs a title or a body; both were blank after trimming whitespace",
			&problem.FieldParams{MinLength: &min}))
	}

	count, err := d.drafts.CountByUser(user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if count >= MaxTopicDraftsPerUser {
		return nil, draftLimitReached(MaxTopicDraftsPerUser)
	}

	category := ""
	if in.Body.Category != nil {
		category = *in.Body.Category
	}
	sections := make(model.StringSlice, 0, len(in.Body.Sections))
	for _, s := range in.Body.Sections {
		sections = append(sections, string(s))
	}
	miniApps, err := encodeDraftMiniApps(in.Body.Polls, in.Body.Lotteries)
	if err != nil {
		return nil, problem.Internal(err)
	}
	draft := &model.TopicDraft{
		UserID:           user.ID,
		Title:            in.Body.Title,
		Content:          content,
		Category:         category,
		Sections:         sections,
		CoverImages:      coversFromHashes(in.Body.CoverImageHashes),
		IsNSFW:           in.Body.IsNSFW,
		Polls:            miniApps.polls,
		Lotteries:        miniApps.lotteries,
		PrizeImageHashes: miniApps.prizeImages,
	}
	if err := d.drafts.Create(draft); err != nil {
		return nil, problem.Internal(err)
	}
	body, err := mapDraft(draft)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &createTopicDraftOutput{
		Location: "/api/v1/me/topic-drafts/" + strconv.Itoa(draft.ID),
		Body:     body,
	}, nil
}

func (d *Drafts) deleteTopicDraft(ctx context.Context, in *topicDraftInput) (*deleteTopicDraftOutput, error) {
	if prob := d.ready(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, notFound()
	}
	id, ok := parsePositiveID(in.DraftID)
	if !ok {
		return nil, notFound()
	}
	n, err := d.drafts.DeleteForUser(id, user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if n == 0 {
		return nil, notFound()
	}
	return &deleteTopicDraftOutput{}, nil
}

// Another user's draft is NOT_FOUND, not PERMISSION_REQUIRED: a private
// resource must not tell a stranger that its id exists.
func (d *Drafts) ownDraft(ctx context.Context, idStr string) (*model.TopicDraft, *problem.Problem) {
	if prob := d.ready(); prob != nil {
		return nil, prob
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, notFound()
	}
	id, ok := parsePositiveID(idStr)
	if !ok {
		return nil, notFound()
	}
	draft, err := d.drafts.GetByIDForUser(id, user.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound()
		}
		return nil, problem.Internal(err)
	}
	return draft, nil
}

func mapDraft(d *model.TopicDraft) (TopicDraft, error) {
	var category *string
	if d.Category != "" {
		c := d.Category
		category = &c
	}
	sections := make([]SectionSlug, 0, len(d.Sections))
	for _, s := range d.Sections {
		sections = append(sections, SectionSlug(s))
	}
	hashes := make([]ImageHash, 0, len(d.CoverImages))
	for _, token := range d.CoverImages {
		hash, _, ok := markdown.ParseContentImageRef(token)
		if !ok {
			continue
		}
		hashes = append(hashes, ImageHash(hash))
	}
	polls, lotteries, err := decodeDraftMiniApps(d)
	if err != nil {
		return TopicDraft{}, err
	}
	return TopicDraft{
		Object:           "topic_draft",
		ID:               repr.ID(d.ID),
		Title:            d.Title,
		ContentMarkdown:  d.Content,
		Category:         category,
		Sections:         sections,
		IsNSFW:           d.IsNSFW,
		CoverImageHashes: hashes,
		Polls:            polls,
		Lotteries:        lotteries,
		CreatedAt:        repr.Timestamp(d.CreatedAt),
		UpdatedAt:        repr.Timestamp(d.UpdatedAt),
	}, nil
}
