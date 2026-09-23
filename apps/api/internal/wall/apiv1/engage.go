package apiv1

import (
	"context"
	"errors"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/problem"
)

func selfLikeForbidden() *problem.Problem {
	return problem.New(problem.CodeSelfLikeForbidden, "Users cannot like their own comments.")
}

func (s *Service) likeWallComment(ctx context.Context, in *commentInput) (*commentOutput, error) {
	return s.setLike(ctx, in.WallCommentID, true)
}

func (s *Service) unlikeWallComment(ctx context.Context, in *commentInput) (*commentOutput, error) {
	return s.setLike(ctx, in.WallCommentID, false)
}

// The community service only toggles. The forum's own like table decides
// whether a toggle is needed at all, so replaying PUT or DELETE changes
// nothing; when the table has drifted from upstream, the toggle lands the wrong
// way round and is sent once more.
func (s *Service) setLike(ctx context.Context, rawID string, want bool) (*commentOutput, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	loc, p := s.locate(ctx, rawID, viewer)
	if p != nil {
		return nil, p
	}
	post := loc.post
	if post.Status == communityclient.PostDeleted {
		return nil, tombstoned()
	}
	if post.AuthorID == int64(viewer.ID) {
		return nil, selfLikeForbidden()
	}
	liked, err := s.store.LikedSet(viewer.ID, []int64{post.ID})
	if err != nil {
		return nil, problem.Internal(err)
	}
	if liked[post.ID] != want {
		toggle := communityclient.ReactionToggleRequest{UserID: int64(viewer.ID), Kind: communityclient.ReactionLike}
		res, err := s.community.ToggleReaction(ctx, post.ID, toggle)
		if err != nil {
			return nil, upstreamProblem(err)
		}
		matched := res.Added == want
		if !matched {
			if res, err = s.community.ToggleReaction(ctx, post.ID, toggle); err != nil {
				return nil, upstreamProblem(err)
			}
			if res.Added != want {
				return nil, problem.Internal(errors.New("community like toggle did not settle"))
			}
		}
		if want {
			err = s.store.EnsureLike(post.ID, viewer.ID)
		} else {
			err = s.store.RemoveLike(post.ID, viewer.ID)
		}
		if err != nil {
			return nil, problem.Internal(err)
		}
		if matched && s.award != nil {
			delta := 1
			if !want {
				delta = -1
			}
			ref := moemoepoint.Ref("galgame_post", int(post.ID))
			s.award(int(post.AuthorID), delta, moemoepoint.ReasonLiked, ref, moemoepoint.KeyNonce(moemoepoint.ReasonLiked, ref))
		}
	}
	item, p := s.renderOne(ctx, loc.sub, post, viewer)
	if p != nil {
		return nil, p
	}
	return &commentOutput{Body: *item}, nil
}

var flagReasons = map[string]int32{
	"spam":          communityclient.FlagReasonSpam,
	"abuse":         communityclient.FlagReasonAbuse,
	"off_topic":     communityclient.FlagReasonOffTopic,
	"other":         communityclient.FlagReasonOther,
	"nsfw_mislabel": communityclient.FlagReasonNsfwMislabel,
}

func (s *Service) flagWallComment(ctx context.Context, in *flagInput) (*struct{}, error) {
	if !s.ready() {
		return nil, problem.Internal(errUnconfigured)
	}
	viewer := v1.User(ctx)
	if p := s.requireActive(ctx, viewer); p != nil {
		return nil, p
	}
	loc, p := s.locate(ctx, in.WallCommentID, viewer)
	if p != nil {
		return nil, p
	}
	if loc.post.Status == communityclient.PostDeleted {
		return nil, tombstoned()
	}
	if loc.post.AuthorID == int64(viewer.ID) {
		return nil, permissionRequired()
	}
	reason, ok := flagReasons[in.Body.FlagReason]
	if !ok {
		return nil, validationFailed(problem.AtPointer("/flag_reason", problem.ReasonUnknownValue, "use a declared flag reason", nil))
	}
	req := communityclient.FlagRequest{FlaggerID: int64(viewer.ID), Reason: reason}
	if in.Body.Note != nil {
		req.Note = *in.Body.Note
	}
	if err := s.community.SubmitFlag(ctx, loc.post.ID, req); err != nil {
		return nil, upstreamProblem(err)
	}
	return nil, nil
}
