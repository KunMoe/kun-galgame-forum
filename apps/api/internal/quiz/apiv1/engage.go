package apiv1

import (
	"context"
	"strconv"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/problem"

	"gorm.io/gorm"
)

func (s *Service) putQuizFavorite(ctx context.Context, in *quizFavoriteInput) (*engagementOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleQuiz(ctx, in.QuizID)
	if p != nil {
		return nil, p
	}
	var jobs []pendingAward
	err := s.store.InTx(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		id, inserted, err := s.store.InsertFavorite(tx, row.ID, user.ID)
		if err != nil || !inserted {
			return err
		}
		if err := s.store.AdjustFavoriteCount(tx, row.ID, 1); err != nil {
			return err
		}
		if user.ID == row.UserID {
			return nil
		}
		jobs = append(jobs, pendingAward{
			userID: row.UserID,
			delta:  1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("galgame_quiz", row.ID),
			key:    moemoepoint.Key("favorited", "quiz_favorite_"+strconv.FormatInt(id, 10)),
		})
		return nil
	})
	if err := s.afterCommit(err, jobs); err != nil {
		return nil, problem.Internal(err)
	}
	return s.engagementOut(row.ID, user.ID)
}

func (s *Service) deleteQuizFavorite(ctx context.Context, in *quizFavoriteInput) (*engagementOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleQuiz(ctx, in.QuizID)
	if p != nil {
		return nil, p
	}
	var jobs []pendingAward
	err := s.store.InTx(func(tx *gorm.DB) error {
		jobs = jobs[:0]
		id, deleted, err := s.store.DeleteFavorite(tx, row.ID, user.ID)
		if err != nil || !deleted {
			return err
		}
		if err := s.store.AdjustFavoriteCount(tx, row.ID, -1); err != nil {
			return err
		}
		if user.ID == row.UserID {
			return nil
		}
		jobs = append(jobs, pendingAward{
			userID: row.UserID,
			delta:  -1,
			reason: moemoepoint.ReasonLiked,
			ref:    moemoepoint.Ref("galgame_quiz", row.ID),
			key:    moemoepoint.Key("unfavorited", "quiz_favorite_"+strconv.FormatInt(id, 10)),
		})
		return nil
	})
	if err := s.afterCommit(err, jobs); err != nil {
		return nil, problem.Internal(err)
	}
	return s.engagementOut(row.ID, user.ID)
}

func (s *Service) engagementOut(quizID, userID int) (*engagementOutput, error) {
	n, err := s.store.FavoriteCount(quizID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	fav, err := s.store.HasFavorited(quizID, userID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	return &engagementOutput{Body: QuizEngagement{
		Object: "quiz_engagement", QuizID: repr.ID(quizID), FavoriteCount: n,
		Viewer: &EngagementViewer{HasFavorited: fav},
	}}, nil
}

func (s *Service) putQuizQualityRating(ctx context.Context, in *putQualityInput) (*qualityOutput, error) {
	if p := s.ready(); p != nil {
		return nil, p
	}
	user, p := s.requireActive(ctx)
	if p != nil {
		return nil, p
	}
	row, _, p := s.visibleQuiz(ctx, in.QuizID)
	if p != nil {
		return nil, p
	}
	ans, err := s.store.FindAnswer(row.ID, user.ID)
	if err != nil {
		return nil, problem.Internal(err)
	}
	if ans == nil || ans.Role != "answerer" {
		return nil, quizAnswerRequired()
	}
	sumDelta, countDelta := in.Body.Rating, 1
	if ans.QualityRating != nil {
		sumDelta, countDelta = in.Body.Rating-*ans.QualityRating, 0
	}
	var sum, count int
	err = s.store.InTx(func(tx *gorm.DB) error {
		if err := s.store.SetAnswerQuality(tx, ans.ID, in.Body.Rating); err != nil {
			return err
		}
		if err := s.store.AdjustQuality(tx, row.ID, sumDelta, countDelta); err != nil {
			return err
		}
		var e error
		sum, count, e = s.store.ReadQuality(tx, row.ID)
		return e
	})
	if err != nil {
		return nil, problem.Internal(err)
	}
	rating := in.Body.Rating
	return &qualityOutput{Body: QuizQuality{
		Object: "quiz_quality", QuizID: repr.ID(row.ID),
		QualityAverage: qualityAverage(sum, count), QualityCount: count,
		Viewer: &QualityViewer{QualityRating: &rating},
	}}, nil
}
