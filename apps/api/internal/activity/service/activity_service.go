package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kun-galgame-api/internal/activity/dto"
	"kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/userclient"

	"github.com/redis/go-redis/v9"
)

var contentImageTokenRe = regexp.MustCompile(`/image/[0-9a-f]{64}`)

var solutionTopicLinkRe = regexp.MustCompile(`^/topic/(\d+)`)

const activityCacheTTL = 30 * time.Second

const (
	activityMaxRounds = 5
	nextMoeBatchChunk = 100
)

type ActivityService struct {
	repo          *repository.ActivityRepository
	galgameClient *client.GalgameClient
	userClient    *userclient.Client
	rdb           *redis.Client
}

func NewActivityService(
	repo *repository.ActivityRepository,
	gc *client.GalgameClient,
	userClient *userclient.Client,
	rdb *redis.Client,
) *ActivityService {
	return &ActivityService{repo: repo, galgameClient: gc, userClient: userClient, rdb: rdb}
}

type Result struct {
	Items      []dto.ActivityItem `json:"items"`
	NextCursor string             `json:"next_cursor"`
}

func (s *ActivityService) GetActivity(ctx context.Context, typeStr, cursor string, limit int, isSFW, showNoResource bool) (*Result, *errors.AppError) {
	if typeStr == "all" {
		return s.GetTimeline(ctx, cursor, limit, isSFW, showNoResource)
	}
	if !s.repo.IsKnownType(typeStr) {
		return &Result{Items: []dto.ActivityItem{}, NextCursor: ""}, nil
	}
	cacheKey := fmt.Sprintf("activity:v2:%s:%s:%d:%t:%t", typeStr, cursor, limit, isSFW, showNoResource)
	return s.cachedKeyset(ctx, cacheKey, []string{typeStr}, cursor, limit, isSFW, showNoResource, "normal")
}

func (s *ActivityService) getCachedResult(ctx context.Context, key string) (*Result, bool) {
	if s.rdb == nil {
		return nil, false
	}
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var result Result
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, false
	}
	return &result, true
}

func (s *ActivityService) cacheResult(ctx context.Context, key string, result *Result) {
	if s.rdb == nil || result == nil {
		return
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, key, raw, activityCacheTTL).Err()
}

var homeTabTypes = map[string][]string{
	"topic": {
		"TOPIC_CREATION", "TOPIC_REPLY_CREATION", "TOPIC_COMMENT_CREATION",
		"TOPIC_UPVOTE", "MESSAGE_SOLUTION",
	},
	"galgame": {
		"GALGAME_CREATION", "GALGAME_EDIT", "GALGAME_PR_CREATION",
		"GALGAME_COMMENT_CREATION", "GALGAME_RATING_CREATION",
		"GALGAME_RATING_COMMENT_CREATION", "GALGAME_WEBSITE_CREATION",
		"GALGAME_WEBSITE_COMMENT_CREATION", "TOOLSET_CREATION",
		"TOOLSET_RESOURCE_CREATION", "TOOLSET_COMMENT_CREATION",
		"GALGAME_QUIZ_CREATION", "GALGAME_QUIZ_COMMENT_CREATION",
	},
	"resource": {
		"GALGAME_RESOURCE_CREATION", "GALGAME_RESOURCE_COMMENT_CREATION",
		"TOPIC_CREATION",
	},
	"others": {"TODO_CREATION", "UPDATE_LOG_CREATION"},
}

func homeTabSourceTypes(tab string) []string {
	if tab == "all" {
		out := make([]string, 0, 18)
		out = append(out, homeTabTypes["topic"]...)
		for _, t := range homeTabTypes["galgame"] {
			if t == "GALGAME_COMMENT_CREATION" {
				continue
			}
			out = append(out, t)
		}
		out = append(out, homeTabTypes["others"]...)
		return out
	}
	return homeTabTypes[tab]
}

func (s *ActivityService) GetTab(ctx context.Context, tab, cursor string, limit int, isSFW, showNoResource bool) (*Result, *errors.AppError) {
	types := homeTabSourceTypes(tab)
	if len(types) == 0 {
		return &Result{Items: []dto.ActivityItem{}, NextCursor: ""}, nil
	}
	sectionMode := "normal"
	if tab == "resource" {
		sectionMode = "help"
	}
	cacheKey := fmt.Sprintf("activity:v2:tab:%s:%s:%d:%t:%t", tab, cursor, limit, isSFW, showNoResource)
	return s.cachedKeyset(ctx, cacheKey, types, cursor, limit, isSFW, showNoResource, sectionMode)
}

func (s *ActivityService) GetFeedByTypes(ctx context.Context, kinds []string, cursor string, limit int, isSFW, showNoResource bool) (*Result, *errors.AppError) {
	types, sectionMode := s.resolveKinds(kinds)
	if len(types) == 0 {
		return &Result{Items: []dto.ActivityItem{}, NextCursor: ""}, nil
	}
	if len(types) == 1 && types[0] == "TOPIC_CREATION" {
		cacheKey := fmt.Sprintf("activity:v2:topicfeed:%s:%s:%d:%t", sectionMode, cursor, limit, isSFW)
		return s.cachedTopicKeyset(ctx, cacheKey, cursor, limit, isSFW, sectionMode)
	}
	cacheKey := fmt.Sprintf("activity:v2:custom:%s|%s:%s:%d:%t:%t",
		strings.Join(types, ","), sectionMode, cursor, limit, isSFW, showNoResource)
	return s.cachedKeyset(ctx, cacheKey, types, cursor, limit, isSFW, showNoResource, sectionMode)
}

func (s *ActivityService) resolveKinds(kinds []string) (types []string, sectionMode string) {
	seen := map[string]bool{}
	addType := func(t string) {
		if !seen[t] {
			types = append(types, t)
			seen[t] = true
		}
	}
	wantNormal, wantHelp := false, false
	for _, k := range kinds {
		switch k {
		case "TOPIC_NORMAL":
			wantNormal = true
			addType("TOPIC_CREATION")
		case "TOPIC_RESOURCE_HELP":
			wantHelp = true
			addType("TOPIC_CREATION")
		default:
			if s.repo.IsKnownType(k) {
				addType(k)
			}
		}
	}
	switch {
	case wantHelp && !wantNormal:
		sectionMode = "help"
	case wantNormal && !wantHelp:
		sectionMode = "normal"
	default:
		sectionMode = "all"
	}
	return types, sectionMode
}

func (s *ActivityService) GetTimeline(ctx context.Context, cursor string, limit int, isSFW, showNoResource bool) (*Result, *errors.AppError) {
	cacheKey := fmt.Sprintf("activity:v2:all:%s:%d:%t:%t", cursor, limit, isSFW, showNoResource)
	return s.cachedKeyset(ctx, cacheKey, nil, cursor, limit, isSFW, showNoResource, "normal")
}

func (s *ActivityService) cachedKeyset(ctx context.Context, cacheKey string, types []string, cursor string, limit int, isSFW, showNoResource bool, sectionMode string) (*Result, *errors.AppError) {
	if cached, ok := s.getCachedResult(ctx, cacheKey); ok {
		return cached, nil
	}
	result, appErr := s.serveKeyset(ctx, types, cursor, limit, isSFW, showNoResource, sectionMode)
	if appErr != nil {
		return nil, appErr
	}
	s.cacheResult(ctx, cacheKey, result)
	return result, nil
}

func (s *ActivityService) serveKeyset(ctx context.Context, types []string, cursor string, limit int, isSFW, showNoResource bool, sectionMode string) (*Result, *errors.AppError) {
	cur := decodeCursor(cursor)
	collected := make([]dto.ActivityItem, 0, limit)
	exhausted := false

	for round := 0; len(collected) < limit && round < activityMaxRounds; round++ {
		rows, err := s.repo.FetchFeed(types, limit, cur, isSFW, showNoResource, sectionMode)
		if err != nil {
			return nil, errors.ErrInternal("查询活动数据失败")
		}
		if len(rows) == 0 {
			exhausted = true
			break
		}
		for _, it := range s.enrichAndHydrate(ctx, rows, isSFW) {
			collected = append(collected, it)
			if len(collected) == limit {
				break
			}
		}
		if len(collected) >= limit {
			break
		}
		last := rows[len(rows)-1]
		cur = &repository.Cursor{Created: last.Created, TypeStr: last.TypeStr, ID: last.ID}
		if len(rows) < limit {
			exhausted = true
			break
		}
	}

	next := ""
	switch {
	case len(collected) == limit:
		last := collected[len(collected)-1]
		next = encodeCursor(last.Timestamp, last.Type, last.ID)
	case !exhausted && cur != nil:
		next = encodeCursor(cur.Created, cur.TypeStr, cur.ID)
	}
	return &Result{Items: collected, NextCursor: next}, nil
}

func (s *ActivityService) cachedTopicKeyset(ctx context.Context, cacheKey, cursor string, limit int, isSFW bool, sectionMode string) (*Result, *errors.AppError) {
	if cached, ok := s.getCachedResult(ctx, cacheKey); ok {
		return cached, nil
	}
	result, appErr := s.serveTopicKeyset(ctx, cursor, limit, isSFW, sectionMode)
	if appErr != nil {
		return nil, appErr
	}
	s.cacheResult(ctx, cacheKey, result)
	return result, nil
}

func (s *ActivityService) serveTopicKeyset(ctx context.Context, cursor string, limit int, isSFW bool, sectionMode string) (*Result, *errors.AppError) {
	rows, err := s.repo.FetchTopicFeed(limit, decodeCursor(cursor), isSFW, sectionMode)
	if err != nil {
		return nil, errors.ErrInternal("查询话题失败")
	}
	items := s.enrichAndHydrate(ctx, rows, isSFW)
	next := ""
	if len(rows) == limit {
		last := rows[len(rows)-1]
		next = encodeCursor(last.Created, last.TypeStr, last.ID)
	}
	return &Result{Items: items, NextCursor: next}, nil
}

type cursorPayload struct {
	C time.Time `json:"c"`
	T string    `json:"t"`
	I int       `json:"i"`
}

func encodeCursor(created time.Time, typeStr string, id int) string {
	b, err := json.Marshal(cursorPayload{C: created, T: typeStr, I: id})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(cursor string) *repository.Cursor {
	if cursor == "" {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil
	}
	var p cursorPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.T == "" || p.I <= 0 {
		return nil
	}
	return &repository.Cursor{Created: p.C, TypeStr: p.T, ID: p.I}
}

func (s *ActivityService) enrichAndHydrate(ctx context.Context, rows []repository.ActivityRow, isSFW bool) []dto.ActivityItem {
	items := rowsToItems(rows)
	items = s.enrichGalgameItems(ctx, rows, items, isSFW)
	s.enrichGalgameResourceDetails(items)
	s.enrichTopicItems(ctx, items)
	s.enrichTopicCommentItems(ctx, items)
	s.enrichReplyItems(ctx, items)
	s.enrichNoteItems(items)
	s.enrichQuizItems(items)
	s.enrichEntityRefItems(items)
	s.enrichSolutionItems(items)
	s.renderMarkdownBodies(items)
	items = s.hydrateActors(ctx, items)
	return items
}

func (s *ActivityService) enrichSolutionItems(items []dto.ActivityItem) {
	topicIDByIdx := map[int]int{}
	idSet := map[int]struct{}{}
	for i, it := range items {
		if it.Type != "MESSAGE_SOLUTION" {
			continue
		}
		m := solutionTopicLinkRe.FindStringSubmatch(it.Link)
		if m == nil {
			continue
		}
		tid, _ := strconv.Atoi(m[1])
		if tid > 0 {
			topicIDByIdx[i] = tid
			idSet[tid] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return
	}
	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	titles, err := s.repo.FetchTopicTitles(ids)
	if err != nil {
		return
	}
	bestAnswers, _ := s.repo.FetchTopicBestAnswers(ids)
	for i, tid := range topicIDByIdx {
		if title, ok := titles[tid]; ok {
			items[i].Data = dto.SolutionActivityData{TopicTitle: title, Floor: bestAnswers[tid].Floor}
		}
	}
}

func (s *ActivityService) enrichEntityRefItems(items []dto.ActivityItem) {
	var resIDs []int
	for _, it := range items {
		if it.Type == "TOOLSET_RESOURCE_CREATION" {
			resIDs = append(resIDs, it.ID)
		}
	}
	if len(resIDs) == 0 {
		return
	}
	resParents, _ := s.repo.FetchToolsetResourceParents(resIDs)
	for i := range items {
		if items[i].Type == "TOOLSET_RESOURCE_CREATION" {
			if name := resParents[items[i].ID]; name != "" {
				items[i].Data = dto.EntityRefActivityData{ParentName: name}
			}
		}
	}
}

func (s *ActivityService) renderMarkdownBodies(items []dto.ActivityItem) {
	for i := range items {
		switch items[i].Type {
		case "TOPIC_REPLY_CREATION", "GALGAME_COMMENT_CREATION":
			if items[i].Content != "" {
				items[i].Content = markdown.Render(items[i].Content)
			}
		}
	}
}

func (s *ActivityService) enrichNoteItems(items []dto.ActivityItem) {
	todoIdx := map[int][]int{}
	logIdx := map[int][]int{}
	for i, it := range items {
		switch it.Type {
		case "TODO_CREATION":
			todoIdx[it.ID] = append(todoIdx[it.ID], i)
		case "UPDATE_LOG_CREATION":
			logIdx[it.ID] = append(logIdx[it.ID], i)
		}
	}
	if len(todoIdx) > 0 {
		ids := make([]int, 0, len(todoIdx))
		for id := range todoIdx {
			ids = append(ids, id)
		}
		if m, err := s.repo.FetchTodoStatuses(ids); err == nil {
			for id, st := range m {
				status := st
				for _, i := range todoIdx[id] {
					items[i].Data = dto.NoteActivityData{Status: &status}
				}
			}
		}
	}
	if len(logIdx) > 0 {
		ids := make([]int, 0, len(logIdx))
		for id := range logIdx {
			ids = append(ids, id)
		}
		if m, err := s.repo.FetchUpdateLogVersions(ids); err == nil {
			for id, v := range m {
				for _, i := range logIdx[id] {
					items[i].Data = dto.NoteActivityData{Version: v}
				}
			}
		}
	}
}

func (s *ActivityService) enrichQuizItems(items []dto.ActivityItem) {
	idToIdx := map[int][]int{}
	for i, it := range items {
		if it.Type == "GALGAME_QUIZ_CREATION" {
			idToIdx[it.ID] = append(idToIdx[it.ID], i)
		}
	}
	if len(idToIdx) == 0 {
		return
	}
	ids := make([]int, 0, len(idToIdx))
	for id := range idToIdx {
		ids = append(ids, id)
	}
	metaMap, err := s.repo.FetchQuizActivityData(ids)
	if err != nil {
		return
	}
	for id, idxs := range idToIdx {
		m, ok := metaMap[id]
		if !ok {
			continue
		}
		payload := dto.QuizActivityData{
			Category:      m.Category,
			Type:          m.Type,
			Difficulty:    m.Difficulty,
			AnswerCount:   m.AnswerCount,
			CorrectCount:  m.CorrectCount,
			FavoriteCount: m.FavoriteCount,
			Description:   m.Description,
		}
		for _, i := range idxs {
			items[i].Data = payload
		}
	}
}

func (s *ActivityService) enrichTopicCommentItems(ctx context.Context, items []dto.ActivityItem) {
	idToIdx := map[int][]int{}
	for i, it := range items {
		if it.Type == "TOPIC_COMMENT_CREATION" {
			idToIdx[it.ID] = append(idToIdx[it.ID], i)
		}
	}
	if len(idToIdx) == 0 {
		return
	}
	ids := make([]int, 0, len(idToIdx))
	for id := range idToIdx {
		ids = append(ids, id)
	}
	ctxMap, err := s.repo.FetchTopicCommentContext(ids)
	if err != nil {
		return
	}

	mentionSet := map[int]struct{}{}
	for id, idxs := range idToIdx {
		for _, mid := range collectReplyMentionIDs(items[idxs[0]].Content) {
			mentionSet[mid] = struct{}{}
		}
		if c, ok := ctxMap[id]; ok {
			for _, mid := range collectReplyMentionIDs(c.ReplyContent) {
				mentionSet[mid] = struct{}{}
			}
		}
	}
	names := map[int]string{}
	if len(mentionSet) > 0 {
		mids := make([]int, 0, len(mentionSet))
		for mid := range mentionSet {
			mids = append(mids, mid)
		}
		for id, u := range s.userClient.Hydrate(ctx, mids) {
			names[id] = u.Name
		}
	}

	for id, idxs := range idToIdx {
		c, ok := ctxMap[id]
		if !ok {
			continue
		}
		payload := dto.TopicCommentActivityData{
			TopicTitle: c.TopicTitle,
			CommentId:  id,
			QuotedReply: &dto.QuotedReply{
				Floor:   c.ReplyFloor,
				Content: renderReplyTokens(c.ReplyContent, names),
			},
		}
		for _, i := range idxs {
			items[i].Content = renderReplyTokens(items[i].Content, names)
			items[i].Data = payload
		}
	}
}

func (s *ActivityService) enrichGalgameResourceDetails(items []dto.ActivityItem) {
	ids := []int{}
	for _, it := range items {
		if it.Type == "GALGAME_RESOURCE_CREATION" {
			ids = append(ids, it.ID)
		}
	}
	details, err := s.repo.FetchGalgameResourceDetails(ids)
	if err != nil {
		return
	}
	for i := range items {
		if items[i].Type != "GALGAME_RESOURCE_CREATION" {
			continue
		}
		d, ok := details[items[i].ID]
		if !ok {
			continue
		}
		if ga, ok := items[i].Data.(dto.GalgameActivityData); ok {
			ga.Resource = &dto.GalgameResourceDetails{
				Type:      d.Type,
				Language:  d.Language,
				Platform:  d.Platform,
				Size:      d.Size,
				Note:      d.Note,
				LikeCount: d.LikeCount,
			}
			items[i].Data = ga
		}
	}
}

var (
	replyMentionRe = regexp.MustCompile(`\[[^\]]*\]\(kungal-user:(\d+)\)`)
	replyQuoteRe   = regexp.MustCompile(`\[(#[^\]]*)\]\(kungal-reply:(\d+)\)`)
)

func collectReplyMentionIDs(content string) []int {
	matches := replyMentionRe.FindAllStringSubmatch(content, -1)
	ids := make([]int, 0, len(matches))
	for _, m := range matches {
		if id, err := strconv.Atoi(m[1]); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func firstQuotedReplyID(content string) int {
	m := replyQuoteRe.FindStringSubmatch(content)
	if m == nil {
		return 0
	}
	id, _ := strconv.Atoi(m[2])
	return id
}

func renderReplyTokens(content string, names map[int]string) string {
	content = replyMentionRe.ReplaceAllStringFunc(content, func(tok string) string {
		m := replyMentionRe.FindStringSubmatch(tok)
		id, _ := strconv.Atoi(m[1])
		if name := names[id]; name != "" {
			return "@" + name
		}
		return "@用户"
	})
	return replyQuoteRe.ReplaceAllString(content, "$1")
}

func (s *ActivityService) enrichReplyItems(ctx context.Context, items []dto.ActivityItem) {
	idToIdx := map[int][]int{}
	for i, it := range items {
		if it.Type == "TOPIC_REPLY_CREATION" {
			idToIdx[it.ID] = append(idToIdx[it.ID], i)
		}
	}
	if len(idToIdx) == 0 {
		return
	}
	ids := make([]int, 0, len(idToIdx))
	for id := range idToIdx {
		ids = append(ids, id)
	}
	titles, _ := s.repo.FetchReplyTopicTitles(ids)
	floors, _ := s.repo.FetchReplyFloors(ids)

	quotedIDByReply := map[int]int{}
	quotedIDSet := map[int]struct{}{}
	for id, idxs := range idToIdx {
		if qid := firstQuotedReplyID(items[idxs[0]].Content); qid > 0 {
			quotedIDByReply[id] = qid
			quotedIDSet[qid] = struct{}{}
		}
	}
	quotedIDs := make([]int, 0, len(quotedIDSet))
	for qid := range quotedIDSet {
		quotedIDs = append(quotedIDs, qid)
	}
	quotedContents, _ := s.repo.FetchReplyContents(quotedIDs)

	mentionSet := map[int]struct{}{}
	for _, idxs := range idToIdx {
		for _, mid := range collectReplyMentionIDs(items[idxs[0]].Content) {
			mentionSet[mid] = struct{}{}
		}
	}
	for _, qc := range quotedContents {
		for _, mid := range collectReplyMentionIDs(qc.Content) {
			mentionSet[mid] = struct{}{}
		}
	}
	names := map[int]string{}
	if len(mentionSet) > 0 {
		mids := make([]int, 0, len(mentionSet))
		for mid := range mentionSet {
			mids = append(mids, mid)
		}
		for id, u := range s.userClient.Hydrate(ctx, mids) {
			names[id] = u.Name
		}
	}

	for id, idxs := range idToIdx {
		var quoted *dto.QuotedReply
		if qc, ok := quotedContents[quotedIDByReply[id]]; ok {
			quoted = &dto.QuotedReply{
				Floor:   qc.Floor,
				Content: renderReplyTokens(qc.Content, names),
			}
		}
		data := dto.ReplyActivityData{TopicTitle: titles[id], Floor: floors[id], QuotedReply: quoted}
		for _, i := range idxs {
			items[i].Content = renderReplyTokens(items[i].Content, names)
			items[i].Data = data
		}
	}
}

func (s *ActivityService) enrichTopicItems(ctx context.Context, items []dto.ActivityItem) {
	idToIdx := map[int][]int{}
	for i, it := range items {
		if it.Type == "TOPIC_CREATION" {
			idToIdx[it.ID] = append(idToIdx[it.ID], i)
		}
	}
	upvoteIdx := map[int][]int{}
	for i, it := range items {
		if it.Type == "TOPIC_UPVOTE" {
			upvoteIdx[it.ID] = append(upvoteIdx[it.ID], i)
		}
	}
	if len(upvoteIdx) > 0 {
		upvoteIDs := make([]int, 0, len(upvoteIdx))
		for id := range upvoteIdx {
			upvoteIDs = append(upvoteIDs, id)
		}
		if topicByUpvote, err := s.repo.FetchUpvoteTopics(upvoteIDs); err == nil {
			for upvoteID, idxs := range upvoteIdx {
				if tid := topicByUpvote[upvoteID]; tid > 0 {
					idToIdx[tid] = append(idToIdx[tid], idxs...)
				}
			}
		}
	}
	if len(idToIdx) == 0 {
		return
	}
	ids := make([]int, 0, len(idToIdx))
	for id := range idToIdx {
		ids = append(ids, id)
	}
	core, err := s.repo.FetchTopicActivityData(ids)
	if err != nil {
		return
	}
	sections, _ := s.repo.FetchTopicSections(ids)
	miniApps := s.repo.FetchTopicMiniApps(ids)
	topReplies, _ := s.repo.FetchTopicTopReply(ids)
	bestAnswers, _ := s.repo.FetchTopicBestAnswers(ids)
	upvotes, _ := s.repo.FetchTopicUpvotesBatch(ids)
	latest, _ := s.repo.FetchTopicLatestActivity(ids)
	reactionRows, _ := s.repo.FetchTopicsReactions(ids)

	extraIDs := make([]int, 0, len(topReplies)+len(reactionRows))
	for _, tr := range topReplies {
		if tr.UserID > 0 {
			extraIDs = append(extraIDs, tr.UserID)
		}
	}
	for _, ba := range bestAnswers {
		if ba.UserID > 0 {
			extraIDs = append(extraIDs, ba.UserID)
		}
	}
	for _, ups := range upvotes {
		for _, up := range ups {
			if up.UserID > 0 {
				extraIDs = append(extraIDs, up.UserID)
			}
		}
	}
	for _, la := range latest {
		if la.UserID > 0 {
			extraIDs = append(extraIDs, la.UserID)
		}
	}
	for _, row := range reactionRows {
		if row.UserID > 0 {
			extraIDs = append(extraIDs, row.UserID)
		}
	}
	extraUsers := s.userClient.Hydrate(ctx, extraIDs)

	type rkey struct {
		tid int
		r   string
	}
	racc := map[rkey]*dto.TopicReactionCount{}
	rorder := map[int][]rkey{}
	for _, row := range reactionRows {
		k := rkey{row.TopicID, row.Reaction}
		rc, ok := racc[k]
		if !ok {
			rc = &dto.TopicReactionCount{Reaction: row.Reaction, Count: row.Count}
			racc[k] = rc
			rorder[row.TopicID] = append(rorder[row.TopicID], k)
		}
		if u, ok := extraUsers[row.UserID]; ok {
			rc.Reactors = append(rc.Reactors,
				dto.Actor{ID: u.ID, Name: u.Name, Avatar: u.Avatar})
		}
	}
	reactionsByTopic := map[int][]dto.TopicReactionCount{}
	for tid, keys := range rorder {
		for _, k := range keys {
			reactionsByTopic[tid] = append(reactionsByTopic[tid], *racc[k])
		}
	}

	for id, idxs := range idToIdx {
		c, ok := core[id]
		if !ok {
			continue
		}
		covers := []string(c.CoverImages)
		if covers == nil {
			covers = []string{}
		}
		if len(covers) == 0 {
			if img := contentImageTokenRe.FindString(c.Excerpt); img != "" {
				covers = []string{img}
			}
		}
		reactions := reactionsByTopic[id]
		if reactions == nil {
			reactions = []dto.TopicReactionCount{}
		}
		sec := sections[id]
		if sec == nil {
			sec = []string{}
		}
		var topReply *dto.TopReply
		if tr, ok := topReplies[id]; ok {
			topReply = &dto.TopReply{ReplyID: tr.ID, Floor: tr.Floor, Content: tr.Content, LikeCount: tr.LikeCount}
			if u, ok := extraUsers[tr.UserID]; ok {
				topReply.User = dto.Actor{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
			}
		}
		var bestAnswer *dto.TopReply
		if ba, ok := bestAnswers[id]; ok {
			bestAnswer = &dto.TopReply{ReplyID: ba.ReplyID, Floor: ba.Floor, Content: ba.Content, LikeCount: ba.LikeCount}
			if u, ok := extraUsers[ba.UserID]; ok {
				bestAnswer.User = dto.Actor{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
			}
		}
		var upvoteList []dto.TopicUpvote
		for _, up := range upvotes[id] {
			tu := dto.TopicUpvote{ID: up.ID, Description: up.Description, Created: up.Created}
			if u, ok := extraUsers[up.UserID]; ok {
				tu.User = dto.Actor{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
			}
			upvoteList = append(upvoteList, tu)
		}
		var latestActivity *dto.LatestActivity
		if la, ok := latest[id]; ok {
			latestActivity = &dto.LatestActivity{Kind: la.Kind, Content: la.Content, Created: la.Created}
			if la.Kind == "reply" {
				latestActivity.ReplyID = la.ID
				latestActivity.Floor = la.Floor
			} else {
				latestActivity.CommentId = la.ID
			}
			if u, ok := extraUsers[la.UserID]; ok {
				latestActivity.User = dto.Actor{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
			}
		}
		payload := dto.TopicActivityData{
			TopicID:        id,
			Title:          c.Title,
			AuthorID:       c.AuthorID,
			Excerpt:        c.Excerpt,
			Sections:       sec,
			CoverImages:    covers,
			CoverImageMeta: markdown.ResolveContentImageMeta(covers),
			View:           c.View,
			LikeCount:      c.LikeCount,
			FavoriteCount:  c.FavoriteCount,
			ReplyCount:     c.ReplyCount,
			CommentCount:   c.CommentCount,
			UpvoteTime:     c.UpvoteTime,
			Edited:         c.Edited,
			HasBestAnswer:  c.BestAnswerID != nil,
			MiniApps:       miniApps[id],
			IsNSFW:         c.IsNSFW,
			TopReply:       topReply,
			BestAnswer:     bestAnswer,
			Upvotes:        upvoteList,
			LatestActivity: latestActivity,
			Reactions:      reactions,
		}
		for _, i := range idxs {
			items[i].Data = payload
		}
	}
}

func rowsToItems(rows []repository.ActivityRow) []dto.ActivityItem {
	items := make([]dto.ActivityItem, len(rows))
	for i, r := range rows {
		items[i] = dto.ActivityItem{
			ID:        r.ID,
			UniqueID:  fmt.Sprintf("%s-%d", r.TypeStr, r.ID),
			Type:      r.TypeStr,
			Content:   r.Content,
			Link:      r.Link,
			Timestamp: r.Created,
			Actor: dto.Actor{
				ID: r.UserID,
			},
		}
	}
	return items
}

func (s *ActivityService) enrichGalgameItems(
	ctx context.Context,
	rows []repository.ActivityRow,
	items []dto.ActivityItem,
	isSFW bool,
) []dto.ActivityItem {
	idSet := map[int]struct{}{}
	for _, r := range rows {
		if r.WorkID > 0 {
			idSet[r.WorkID] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return items
	}
	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	briefMap := make(map[int]client.GalgameBrief, len(ids))
	for start := 0; start < len(ids); start += nextMoeBatchChunk {
		end := min(start+nextMoeBatchChunk, len(ids))
		m, appErr := s.galgameClient.GetBatchPublic(ctx, ids[start:end], isSFW)
		if appErr != nil {
			return items
		}
		maps.Copy(briefMap, m)
	}

	briefName := func(b client.GalgameBrief) string {
		if b.Name != "" {
			return b.Name
		}
		return fmt.Sprintf("galgame#%d", b.ID)
	}

	preferredIntro := func(d client.GalgameDetailBrief) string {
		if len(d.Intros) == 0 {
			return ""
		}
		return d.Intros[0].Intro
	}

	creationWorkIDs := make([]int, 0)
	editIDs := make([]int, 0)
	editWorkIDs := make([]int, 0)
	prWorkIDs := make([]int, 0)
	ratingIDs := make([]int, 0)
	for _, r := range rows {
		switch {
		case r.TypeStr == "GALGAME_CREATION" && r.WorkID > 0:
			creationWorkIDs = append(creationWorkIDs, r.WorkID)
		case r.TypeStr == "GALGAME_EDIT":
			editIDs = append(editIDs, r.ID)
			if r.WorkID > 0 {
				editWorkIDs = append(editWorkIDs, r.WorkID)
			}
		case r.TypeStr == "GALGAME_PR_CREATION" && r.WorkID > 0:
			prWorkIDs = append(prWorkIDs, r.WorkID)
		case r.TypeStr == "GALGAME_RATING_CREATION":
			ratingIDs = append(ratingIDs, r.ID)
		}
	}
	countsMap, _ := s.repo.FetchGalgameCounts(creationWorkIDs)
	revMap, _ := s.repo.FetchEditRevisions(editIDs)
	ratingMap, _ := s.repo.FetchRatingActivityData(ratingIDs)

	detailMap := map[int]client.GalgameDetailBrief{}
	if detailWorkIDs := append(append(append([]int{}, creationWorkIDs...), editWorkIDs...), prWorkIDs...); len(detailWorkIDs) > 0 {
		if m, appErr := s.galgameClient.GetBatchDetailPublic(ctx, detailWorkIDs, isSFW); appErr == nil {
			detailMap = m
		}
	}

	kept := make([]dto.ActivityItem, 0, len(items))
	for i, r := range rows {
		if r.WorkID == 0 {
			kept = append(kept, items[i])
			continue
		}
		b, ok := briefMap[r.WorkID]
		if !ok {
			continue
		}
		name := briefName(b)
		ga := dto.GalgameActivityData{
			Name:        name,
			CoverHash:   b.EffectiveBannerHash,
			Language:    b.OriginalLanguage,
			AgeLimit:    b.AgeLimit,
			ReleaseDate: b.ReleaseDate,
			WorkID:      r.WorkID,
		}
		if d, ok := detailMap[r.WorkID]; ok &&
			(r.TypeStr == "GALGAME_CREATION" || r.TypeStr == "GALGAME_EDIT" ||
				r.TypeStr == "GALGAME_PR_CREATION") {
			ga.Developer = strings.Join(d.Officials, "、")
			ga.ReleaseDate = d.ReleaseDate
			if intro := []rune(preferredIntro(d)); len(intro) > 0 {
				if len(intro) > 300 {
					intro = intro[:300]
				}
				ga.Intro = string(intro)
			}
		}
		if r.TypeStr == "GALGAME_CREATION" {
			c := countsMap[r.WorkID]
			ga.ResourceCount = c.ResourceCount
			ga.LikeCount = c.LikeCount
			ga.FavoriteCount = c.FavoriteCount
		}
		if r.TypeStr == "GALGAME_EDIT" {
			e := revMap[r.ID]
			ga.RevisionID = e.RevisionID
			ga.RevisionNumber = e.RevisionNumber
		}
		if r.TypeStr == "GALGAME_RATING_CREATION" {
			if rt, ok := ratingMap[r.ID]; ok {
				ga.Rating = &dto.RatingInfo{
					RatingID:     r.ID,
					Overall:      rt.Overall,
					PlayStatus:   rt.PlayStatus,
					Recommend:    rt.Recommend,
					ShortSummary: rt.ShortSummary,
					SpoilerLevel: rt.SpoilerLevel,
					LikeCount:    rt.LikeCount,
					AuthorID:     rt.AuthorID,
				}
			}
		}
		items[i].Data = ga
		switch r.TypeStr {
		case "GALGAME_CREATION":
			items[i].Content = name
			if items[i].Actor.ID == 0 {
				items[i].Actor.ID = userclient.DerefID(countsMap[r.WorkID].CreatorUserID)
			}
		case "GALGAME_RESOURCE_CREATION":
			items[i].Content = fmt.Sprintf("在《%s》发布了下载资源", name)
		case "GALGAME_EDIT":
			items[i].Content = fmt.Sprintf("编辑了《%s》", name)
		case "GALGAME_PR_CREATION":
			items[i].Content = fmt.Sprintf("对《%s》提出了更新请求", name)
		case "GALGAME_RATING_CREATION":
			if r.Content != "" {
				items[i].Content = fmt.Sprintf("%s · %s", name, r.Content)
			} else {
				items[i].Content = name
			}
		}
		kept = append(kept, items[i])
	}
	return kept
}

func (s *ActivityService) hydrateActors(ctx context.Context, items []dto.ActivityItem) []dto.ActivityItem {
	uids := userclient.CollectIDs(items, func(it dto.ActivityItem) int { return it.Actor.ID })
	if len(uids) == 0 {
		return items
	}
	userMap := s.userClient.Hydrate(ctx, uids)
	kept := make([]dto.ActivityItem, 0, len(items))
	for i := range items {
		u := userMap[items[i].Actor.ID]
		if items[i].Actor.ID > 0 && !userclient.IsRenderable(u) {
			continue
		}
		items[i].Actor.Name = u.Name
		items[i].Actor.Avatar = u.Avatar
		kept = append(kept, items[i])
	}
	return kept
}
