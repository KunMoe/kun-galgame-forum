package apiv1

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"kun-galgame-api/internal/activity/repository"
	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/apiv1/repr"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	"kun-galgame-api/pkg/problem"
	"kun-galgame-api/pkg/userclient"
)

var (
	topicLinkRe   = regexp.MustCompile(`^/topic/(\d+)`)
	quotedReplyRe = regexp.MustCompile(`\[#[^\]]*\]\(kungal-reply:(\d+)\)`)
	releaseRe     = regexp.MustCompile(`^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$`)
)

var todoStates = map[int]string{0: "pending", 1: "in_progress", 2: "done", 3: "discarded"}

type ids map[int]struct{}

func (s ids) add(id int) {
	if id > 0 {
		s[id] = struct{}{}
	}
}

func (s ids) list() []int {
	out := make([]int, 0, len(s))
	for id := range s {
		out = append(out, id)
	}
	return out
}

type batch struct {
	topicOf       map[int]int
	topics        map[int]repository.TopicCardRow
	sections      map[int][]string
	miniApps      map[int][]string
	topReplies    map[int]repository.ReplyExcerptRow
	bestAnswers   map[int]repository.ReplyExcerptRow
	latest        map[int]repository.LatestRow
	upvotes       map[int]repository.UpvoteRow
	reactions     map[int][]repository.ReactionRow
	replyCtx      map[int]repository.ReplyContext
	commentCtx    map[int]repository.CommentContext
	bestCtx       map[int]repository.BestAnswerContext
	quoted        map[int]repository.QuotedRow
	works         map[int]client.CatalogWorkListItem
	published     map[int]repository.WorkRow
	revisions     map[int]repository.EditRevision
	ratings       map[int]repository.RatingActivity
	resources     map[int]repository.GalgameResourceRow
	quizzes       map[int]repository.QuizActivity
	toolsets      map[int]repository.ToolsetParent
	todos         map[int]int
	versions      map[int]string
	users         map[int]userclient.User
	replyDocs     map[int]content.ContentDocument
	replyQuotedID map[int]int
}

// assemble returns one entry per row, in order; nil marks a row left out.
func (s *Service) assemble(ctx context.Context, rows []repository.FeedRow, includeNSFW bool) ([]*Activity, *problem.Problem) {
	b, prob := s.load(ctx, rows, includeNSFW)
	if prob != nil {
		return nil, prob
	}
	out := make([]*Activity, len(rows))
	for i, row := range rows {
		out[i] = s.build(ctx, row, b)
	}
	return out, nil
}

func (s *Service) load(ctx context.Context, rows []repository.FeedRow, includeNSFW bool) (*batch, *problem.Problem) {
	b := &batch{topicOf: map[int]int{}, replyQuotedID: map[int]int{}, replyDocs: map[int]content.ContentDocument{}}
	topicIDs, upvoteIDs, replyIDs, commentIDs, solutionTopics := ids{}, ids{}, ids{}, ids{}, ids{}
	workIDs, creationWorks, editIDs, ratingIDs, resourceIDs, quizIDs := ids{}, ids{}, ids{}, ids{}, ids{}, ids{}
	toolsetResIDs, todoIDs, logIDs, userIDs := ids{}, ids{}, ids{}, ids{}
	var docSources []string
	var docKeys []int
	for _, r := range rows {
		userIDs.add(r.UserID)
		if r.WorkID > 0 {
			workIDs.add(r.WorkID)
		}
		switch r.TypeStr {
		case "TOPIC_CREATION":
			topicIDs.add(r.SourceID)
		case "TOPIC_UPVOTE":
			upvoteIDs.add(r.SourceID)
		case "TOPIC_REPLY_CREATION":
			replyIDs.add(r.SourceID)
			if m := quotedReplyRe.FindStringSubmatch(r.Content); m != nil {
				id, _ := strconv.Atoi(m[1])
				b.replyQuotedID[r.SourceID] = id
			}
			docSources = append(docSources, r.Content)
			docKeys = append(docKeys, int(r.RowID))
		case "MESSAGE_SOLUTION":
			if m := topicLinkRe.FindStringSubmatch(r.Link); m != nil {
				id, _ := strconv.Atoi(m[1])
				solutionTopics.add(id)
			}
			docSources = append(docSources, r.Content)
			docKeys = append(docKeys, int(r.RowID))
		case "TOPIC_COMMENT_CREATION":
			commentIDs.add(r.SourceID)
		case "GALGAME_CREATION":
			creationWorks.add(r.WorkID)
		case "GALGAME_EDIT":
			editIDs.add(r.SourceID)
		case "GALGAME_RATING_CREATION":
			ratingIDs.add(r.SourceID)
		case "GALGAME_RESOURCE_CREATION":
			resourceIDs.add(r.SourceID)
		case "GALGAME_QUIZ_CREATION":
			quizIDs.add(r.SourceID)
		case "TOOLSET_RESOURCE_CREATION":
			toolsetResIDs.add(r.SourceID)
		case "TODO_CREATION":
			todoIDs.add(r.SourceID)
		case "UPDATE_LOG_CREATION":
			logIDs.add(r.SourceID)
		}
	}

	var err error
	fail := func(e error) *problem.Problem { return problem.Internal(e) }
	if b.topicOf, err = s.repo.FetchUpvoteTopics(upvoteIDs.list()); err != nil {
		return nil, fail(err)
	}
	for _, tid := range b.topicOf {
		topicIDs.add(tid)
	}
	tids := topicIDs.list()
	if b.topics, err = s.repo.TopicCards(tids); err != nil {
		return nil, fail(err)
	}
	if b.sections, err = s.repo.FetchTopicSections(tids); err != nil {
		return nil, fail(err)
	}
	b.miniApps = s.repo.FetchTopicMiniApps(tids)
	if b.topReplies, err = s.repo.TopReplies(tids); err != nil {
		return nil, fail(err)
	}
	if b.bestAnswers, err = s.repo.BestAnswers(tids); err != nil {
		return nil, fail(err)
	}
	if b.latest, err = s.repo.LatestReplyOrComment(tids); err != nil {
		return nil, fail(err)
	}
	if b.upvotes, err = s.repo.LatestUpvotes(tids); err != nil {
		return nil, fail(err)
	}
	samples, err := s.repo.ReactionSamples(tids)
	if err != nil {
		return nil, fail(err)
	}
	b.reactions = map[int][]repository.ReactionRow{}
	for _, row := range samples {
		b.reactions[row.TopicID] = append(b.reactions[row.TopicID], row)
		userIDs.add(row.UserID)
	}
	for _, m := range []map[int]repository.ReplyExcerptRow{b.topReplies, b.bestAnswers} {
		for _, r := range m {
			userIDs.add(r.UserID)
		}
	}
	for _, r := range b.latest {
		userIDs.add(r.UserID)
	}
	for _, r := range b.upvotes {
		userIDs.add(r.UserID)
	}
	for _, t := range b.topics {
		userIDs.add(t.UserID)
	}

	if b.replyCtx, err = s.repo.ReplyContexts(replyIDs.list()); err != nil {
		return nil, fail(err)
	}
	if b.commentCtx, err = s.repo.CommentContexts(commentIDs.list()); err != nil {
		return nil, fail(err)
	}
	if b.bestCtx, err = s.repo.BestAnswerContexts(solutionTopics.list()); err != nil {
		return nil, fail(err)
	}
	quotedIDs := ids{}
	for _, id := range b.replyQuotedID {
		quotedIDs.add(id)
	}
	for _, c := range b.commentCtx {
		quotedIDs.add(c.ReplyID)
	}
	if b.quoted, err = s.repo.VisibleReplies(quotedIDs.list()); err != nil {
		return nil, fail(err)
	}

	if b.published, err = s.repo.PublishedWorks(creationWorks.list()); err != nil {
		return nil, fail(err)
	}
	for _, w := range b.published {
		if w.CreatorUserID != nil {
			userIDs.add(*w.CreatorUserID)
		}
	}
	if b.revisions, err = s.repo.FetchEditRevisions(editIDs.list()); err != nil {
		return nil, fail(err)
	}
	if b.ratings, err = s.repo.FetchRatingActivityData(ratingIDs.list()); err != nil {
		return nil, fail(err)
	}
	if b.resources, err = s.repo.FetchGalgameResourceDetails(resourceIDs.list()); err != nil {
		return nil, fail(err)
	}
	if b.quizzes, err = s.repo.FetchQuizActivityData(quizIDs.list()); err != nil {
		return nil, fail(err)
	}
	if b.toolsets, err = s.repo.ToolsetParents(toolsetResIDs.list()); err != nil {
		return nil, fail(err)
	}
	if b.todos, err = s.repo.FetchTodoStatuses(todoIDs.list()); err != nil {
		return nil, fail(err)
	}
	if b.versions, err = s.repo.FetchUpdateLogVersions(logIDs.list()); err != nil {
		return nil, fail(err)
	}

	var prob *problem.Problem
	if b.works, prob = s.catalogRows(ctx, workIDs.list(), !includeNSFW); prob != nil {
		return nil, prob
	}
	if b.users, prob = s.lookupUsers(ctx, userIDs.list()); prob != nil {
		return nil, prob
	}
	if len(docSources) > 0 {
		docs, err := s.convert.Convert(ctx, docSources)
		if err != nil {
			return nil, problem.Unavailable(err)
		}
		for i, key := range docKeys {
			b.replyDocs[key] = docs[i]
		}
	}
	return b, nil
}

func (s *Service) userRef(b *batch, id int) (*repr.UserRef, bool) {
	u, ok := b.users[id]
	if !ok {
		ref := repr.DeletedUserRef(id)
		return &ref, true
	}
	if !userclient.IsRenderable(u) {
		return nil, false
	}
	ref := repr.NewUserRef(s.cdn, u)
	return &ref, true
}

func (s *Service) build(ctx context.Context, r repository.FeedRow, b *batch) *Activity {
	kind := kindByFeed[r.TypeStr]
	if kind == "" {
		return nil
	}
	a := &Activity{
		Object:          "activity",
		ID:              repr.DecimalID(strconv.FormatInt(r.RowID, 10)),
		ActivityType:    ActivityType(kind),
		OccurredAt:      repr.Timestamp(r.Created),
		Path:            r.Link,
		ExcerptMarkdown: excerpt(r.Content, 1000),
	}
	if !strings.HasPrefix(a.Path, "/") {
		a.Path = "/" + a.Path
	}
	actorID := r.UserID
	if r.TypeStr == "GALGAME_CREATION" && actorID == 0 {
		if w, ok := b.published[r.WorkID]; ok && w.CreatorUserID != nil {
			actorID = *w.CreatorUserID
		}
	}
	if actorID > 0 {
		ref, ok := s.userRef(b, actorID)
		if !ok {
			return nil
		}
		a.Performer = ref
	}
	if r.WorkID > 0 {
		row, ok := b.works[r.WorkID]
		if !ok {
			return nil
		}
		ref := galgameapiv1.WorkRefOf(ctx, &row, s.cdn)
		a.Work = &ref
	}

	switch r.TypeStr {
	case "TOPIC_CREATION":
		if !s.fillTopic(a, b, r.SourceID) {
			return nil
		}
	case "TOPIC_UPVOTE":
		if !s.fillTopic(a, b, b.topicOf[r.SourceID]) {
			return nil
		}
	case "TOPIC_REPLY_CREATION":
		c, ok := b.replyCtx[r.SourceID]
		if !ok {
			return nil
		}
		a.Reply = &ActivityReply{
			ReplyID: repr.ID(r.SourceID), TopicID: repr.ID(c.TopicID), TopicTitle: c.TopicTitle,
			Floor: max(c.Floor, 1), Content: b.replyDocs[int(r.RowID)],
		}
		if q, ok := b.quoted[b.replyQuotedID[r.SourceID]]; ok {
			a.Reply.Quoted = &QuotedReply{Floor: max(q.Floor, 1), ExcerptMarkdown: excerpt(q.Content, 200)}
		}
	case "MESSAGE_SOLUTION":
		m := topicLinkRe.FindStringSubmatch(r.Link)
		if m == nil {
			return nil
		}
		tid, _ := strconv.Atoi(m[1])
		c, ok := b.bestCtx[tid]
		if !ok || c.ReplyID == nil || c.Floor == nil {
			return nil
		}
		a.Reply = &ActivityReply{
			ReplyID: repr.ID(*c.ReplyID), TopicID: repr.ID(tid), TopicTitle: c.TopicTitle,
			Floor: max(*c.Floor, 1), Content: b.replyDocs[int(r.RowID)],
		}
	case "TOPIC_COMMENT_CREATION":
		c, ok := b.commentCtx[r.SourceID]
		if !ok {
			return nil
		}
		a.Comment = &ActivityComment{CommentID: repr.ID(r.SourceID), TopicID: repr.ID(c.TopicID), TopicTitle: c.TopicTitle}
		if q, ok := b.quoted[c.ReplyID]; ok {
			a.Comment.Quoted = &QuotedReply{Floor: max(q.Floor, 1), ExcerptMarkdown: excerpt(q.Content, 200)}
		}
	case "GALGAME_CREATION", "GALGAME_EDIT", "GALGAME_PR_CREATION":
		if a.Work != nil {
			row := b.works[r.WorkID]
			d := client.CatalogItemToDetailBrief(ctx, &row)
			names := make([]DeveloperName, 0, min(len(d.Officials), 10))
			for _, n := range d.Officials[:min(len(d.Officials), 10)] {
				names = append(names, DeveloperName(excerpt(n, 256)))
			}
			digest := &WorkDigest{DeveloperNames: names}
			if len(d.Intros) > 0 {
				digest.IntroExcerpt = excerptPtr(d.Intros[0].Intro, 300)
			}
			if row.ReleaseDate != nil && releaseRe.MatchString(*row.ReleaseDate) {
				rel := *row.ReleaseDate
				digest.Release = &rel
			}
			a.WorkDigest = digest
		}
		switch r.TypeStr {
		case "GALGAME_CREATION":
			if w, ok := b.published[r.WorkID]; ok {
				a.WorkStats = &WorkStats{ResourceCount: w.ResourceCount, LikeCount: w.LikeCount, FavoriteCount: w.FavoriteCount}
			}
		case "GALGAME_EDIT":
			if rv, ok := b.revisions[r.SourceID]; ok && (rv.RevisionID > 0 || rv.RevisionNumber > 0) {
				wr := &WorkRevision{}
				if rv.RevisionNumber > 0 {
					wr.RevisionNumber = &rv.RevisionNumber
				}
				if rv.RevisionID > 0 {
					id := repr.ID(rv.RevisionID)
					wr.LegacyRevisionID = &id
				}
				a.WorkRevision = wr
			}
		}
	case "GALGAME_RATING_CREATION":
		rt, ok := b.ratings[r.SourceID]
		if !ok {
			return nil
		}
		a.Rating = &ActivityRating{
			RatingID: repr.ID(r.SourceID), Overall: rt.Overall, PlayStatus: rt.PlayStatus, Recommend: rt.Recommend,
			SpoilerLevel: rt.SpoilerLevel, ShortSummary: excerptPtr(rt.ShortSummary, 1314), LikeCount: rt.LikeCount,
		}
	case "GALGAME_RESOURCE_CREATION":
		rs, ok := b.resources[r.SourceID]
		if !ok {
			return nil
		}
		a.Resource = &ActivityResource{
			ResourceID: repr.ID(r.SourceID), ResourceType: rs.Type, Language: OpenToken(rs.Language),
			Platform: OpenToken(rs.Platform), Size: excerpt(rs.Size, 64), Note: excerptPtr(rs.Note, 300), LikeCount: rs.LikeCount,
		}
	case "GALGAME_QUIZ_CREATION":
		q, ok := b.quizzes[r.SourceID]
		if !ok {
			return nil
		}
		a.Quiz = &ActivityQuiz{
			QuizID: repr.ID(r.SourceID), Category: OpenToken(q.Category), QuestionType: OpenToken(q.Type),
			Difficulty: min(max(q.Difficulty, 1), 10), AnswerCount: q.AnswerCount, CorrectCount: q.CorrectCount,
			FavoriteCount: q.FavoriteCount, DescriptionExcerpt: excerpt(q.Description, 256),
		}
	case "TOOLSET_RESOURCE_CREATION":
		if p, ok := b.toolsets[r.SourceID]; ok {
			a.Toolset = &ActivityToolset{ToolsetID: repr.ID(p.ToolsetID), Title: excerpt(p.Name, 256)}
		}
	case "TODO_CREATION":
		if st, ok := b.todos[r.SourceID]; ok && todoStates[st] != "" {
			a.Todo = &ActivityTodo{TodoID: repr.ID(r.SourceID), State: todoStates[st]}
		}
	case "UPDATE_LOG_CREATION":
		if v, ok := b.versions[r.SourceID]; ok {
			a.UpdateLog = &ActivityUpdateLog{UpdateLogID: repr.ID(r.SourceID), ReleaseVersion: excerpt(v, 20)}
		}
	}
	return a
}

func (s *Service) fillTopic(a *Activity, b *batch, tid int) bool {
	t, ok := b.topics[tid]
	if !ok || t.Status != 0 {
		return false
	}
	author, ok := s.userRef(b, t.UserID)
	if !ok {
		return false
	}
	summary, err := topicapiv1.MapSummary(s.cdn, t.TopicKeysetRow, *author, b.sections[tid], b.miniApps[tid])
	if err != nil {
		return false
	}
	a.Topic = &summary
	digest := &TopicDigest{
		ExcerptMarkdown: excerpt(t.Excerpt, 1000),
		FavoriteCount:   t.FavoriteCount,
		EditedAt:        repr.TimestampPtr(t.Edited),
		TopReply:        s.replyExcerpt(b, b.topReplies, tid),
		BestAnswer:      s.replyExcerpt(b, b.bestAnswers, tid),
		Reactions:       s.reactionSummaries(b, tid),
	}
	if up, ok := b.upvotes[tid]; ok {
		if ref, ok := s.userRef(b, up.UserID); ok {
			digest.LatestUpvote = &UpvoteExcerpt{Upvoter: *ref, Note: excerptPtr(up.Description, 30), UpvotedAt: repr.TimestampPtr(&up.Created)}
		}
	}
	if l, ok := b.latest[tid]; ok {
		if ref, ok := s.userRef(b, l.UserID); ok {
			if l.Kind == "reply" {
				digest.LatestReply = &ReplyExcerpt{
					ReplyID: repr.ID(l.ID), Floor: max(l.Floor, 1), Author: *ref,
					ExcerptMarkdown: excerpt(l.Content, 200), LikeCount: l.LikeCnt, CreatedAt: repr.Timestamp(l.Created),
				}
			} else {
				digest.LatestComment = &CommentExcerpt{
					CommentID: repr.ID(l.ID), Author: *ref, ExcerptMarkdown: excerpt(l.Content, 200), CreatedAt: repr.Timestamp(l.Created),
				}
			}
		}
	}
	a.TopicDigest = digest
	return true
}

func (s *Service) replyExcerpt(b *batch, m map[int]repository.ReplyExcerptRow, tid int) *ReplyExcerpt {
	r, ok := m[tid]
	if !ok {
		return nil
	}
	ref, ok := s.userRef(b, r.UserID)
	if !ok {
		return nil
	}
	return &ReplyExcerpt{
		ReplyID: repr.ID(r.ID), Floor: max(r.Floor, 1), Author: *ref,
		ExcerptMarkdown: excerpt(r.Content, 200), LikeCount: r.LikeCount, CreatedAt: repr.Timestamp(r.Created),
	}
}

func (s *Service) reactionSummaries(b *batch, tid int) []topicapiv1.ReactionSummary {
	out := []topicapiv1.ReactionSummary{}
	index := map[string]int{}
	for _, row := range b.reactions[tid] {
		i, ok := index[row.Reaction]
		if !ok {
			i = len(out)
			index[row.Reaction] = i
			out = append(out, topicapiv1.ReactionSummary{
				Reaction: topicapiv1.ReactionToken(row.Reaction), Count: max(row.Count, 1), Reactors: []repr.UserRef{},
			})
		}
		if len(out[i].Reactors) >= 3 {
			continue
		}
		if u, ok := b.users[row.UserID]; ok && userclient.IsRenderable(u) {
			out[i].Reactors = append(out[i].Reactors, repr.NewUserRef(s.cdn, u))
		}
	}
	return out
}
