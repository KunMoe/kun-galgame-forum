package apiv1

import (
	"context"
	"strconv"
	"strings"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/collect"
	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/pkg/problem"
)

func (s *Service) listMyAnsweredQuizzes(ctx context.Context, in *listMyAnsweredInput) (*listQuizzesOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	pg := pageOf(in.Page, in.Limit)
	if p := pg.CheckDepth(); p != nil {
		return nil, p
	}
	authors, err := s.store.DistinctAnsweredAuthors(user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	keep, p := s.keepAuthors(ctx, authors)
	if p != nil {
		return nil, p
	}
	total, err := s.store.CountAnswered(user.ID, keep)
	if err != nil {
		return nil, problem.Internal(err)
	}
	n, rel := collect.ClampTotal(total)
	rows, err := s.store.ListAnswered(user.ID, keep, pg.Offset(), pg.Limit)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items, p := s.summaries(ctx, rows, user)
	if p != nil {
		return nil, p
	}
	return &listQuizzesOutput{Body: repr.NewPageList(items, n, rel)}, nil
}

func (s *Service) listMyQuizStates(ctx context.Context, in *listMyQuizStatesInput) (*listMyQuizStatesOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user := v1.User(ctx)
	if user == nil {
		return nil, notFound()
	}
	requested, p := parseQuizIDs(in.QuizIDs)
	if p != nil {
		return nil, p
	}
	unique := make([]int, 0, len(requested))
	seen := map[int]bool{}
	for _, id := range requested {
		if seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	readable, p := s.readableQuizzes(ctx, unique)
	if p != nil {
		return nil, p
	}
	favorited, err := s.store.FavoritedSet(user.ID, unique)
	if err != nil {
		return nil, problem.Internal(err)
	}
	items := make([]QuizState, 0, len(unique))
	missing := []repr.DecimalID{}
	emitted := map[int]bool{}
	for _, id := range requested {
		if !readable[id] {
			if !emitted[id] {
				emitted[id] = true
				missing = append(missing, repr.ID(id))
			}
			continue
		}
		if emitted[id] {
			continue
		}
		emitted[id] = true
		items = append(items, QuizState{
			Object: "quiz_state", QuizID: repr.ID(id), HasFavorited: favorited[id],
		})
	}
	return &listMyQuizStatesOutput{Body: repr.NewBatchList(items, missing)}, nil
}

func parseQuizIDs(parts []repr.DecimalID) ([]int, *problem.Problem) {
	out := make([]int, 0, len(parts))
	for i, part := range parts {
		id, ok := repr.ParseID(repr.DecimalID(strings.TrimSpace(string(part))))
		if !ok {
			return nil, validationFailed(problem.AtParameter("quiz_ids", problem.ReasonInvalidFormat,
				"every id must be a positive decimal integer; item "+strconv.Itoa(i)+" is not", nil))
		}
		out = append(out, id)
	}
	return out, nil
}

func (s *Service) readableQuizzes(ctx context.Context, ids []int) (map[int]bool, *problem.Problem) {
	out := map[int]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.store.FindMany(ids)
	if err != nil {
		return nil, problem.Internal(err)
	}
	authors := make([]int, 0, len(rows))
	for i := range rows {
		authors = append(authors, rows[i].UserID)
	}
	users, p := s.lookupUsers(ctx, authors)
	if p != nil {
		return nil, p
	}
	for i := range rows {
		if renderable(users, rows[i].UserID) {
			out[rows[i].ID] = true
		}
	}
	return out, nil
}
