package communityclient

import (
	"context"
	"net/http"
)

// GetComments is a pure read: an anchor nobody has commented on yet has no
// thread, so Thread comes back nil and Posts empty. The face it replaced,
// POST /comments/resolve, was get-or-create, so the three downstream sites
// calling it to render a page had minted 110,918 empty threads by 2026-09-15
// — 97.2% of every thread in the community database.
func (c *Client) GetComments(ctx context.Context, anchorKind int32, anchorID, after, limit string) (*CommentsPage, error) {
	var out CommentsPage
	q := query(map[string]string{
		"anchor_kind": itoa(int64(anchorKind)), "anchor_id": anchorID, "after": after, "limit": limit,
	})
	err := c.do(ctx, http.MethodGet, "/comments"+q, nil, &out)
	return &out, err
}

// CommentOnAnchor posts to an anchor's comment wall; the thread is created in
// the same transaction when this is its first comment.
func (c *Client) CommentOnAnchor(ctx context.Context, req CommentRequest) (*ThreadWithPost, error) {
	var out ThreadWithPost
	err := c.do(ctx, http.MethodPost, "/comments", req, &out)
	return &out, err
}

func (c *Client) EditPost(ctx context.Context, postID int64, req EditPostRequest) (*PostView, error) {
	var out struct {
		Post PostView `json:"post"`
	}
	err := c.do(ctx, http.MethodPatch, "/posts/"+itoa(postID), req, &out)
	return &out.Post, err
}

func (c *Client) DeletePost(ctx context.Context, postID, authorID int64, asModerator bool) error {
	q := map[string]string{"author_id": itoa(authorID)}
	if asModerator {
		q["as_moderator"] = "true"
	}
	return c.do(ctx, http.MethodDelete, "/posts/"+itoa(postID)+query(q), nil, nil)
}

func (c *Client) ToggleReaction(ctx context.Context, postID int64, req ReactionToggleRequest) (*ReactionToggleResult, error) {
	var out ReactionToggleResult
	err := c.do(ctx, http.MethodPost, "/posts/"+itoa(postID)+"/reaction", req, &out)
	return &out, err
}

func (c *Client) SubmitFlag(ctx context.Context, postID int64, req FlagRequest) error {
	return c.do(ctx, http.MethodPost, "/posts/"+itoa(postID)+"/flag", req, nil)
}

func (c *Client) Boost(ctx context.Context, req SetBoostRequest) (*TrustView, error) {
	var out TrustView
	err := c.do(ctx, http.MethodPost, "/trust/boost", req, &out)
	return &out, err
}

func (c *Client) AuthorPosts(ctx context.Context, authorID int64, after string, limit, anchorKind int) (*AuthorPostsResponse, error) {
	var out AuthorPostsResponse
	q := map[string]string{"after": after}
	if limit > 0 {
		q["limit"] = itoa(int64(limit))
	}
	if anchorKind >= 0 {
		q["anchor_kind"] = itoa(int64(anchorKind))
	}
	err := c.do(ctx, http.MethodGet, "/authors/"+itoa(authorID)+"/posts"+query(q), nil, &out)
	return &out, err
}

func (c *Client) AuthorStats(ctx context.Context, ids []int64) (*AuthorStatsResponse, error) {
	if len(ids) == 0 {
		return &AuthorStatsResponse{Stats: []AuthorStat{}}, nil
	}
	var out AuthorStatsResponse
	err := c.do(ctx, http.MethodGet, "/authors/stats"+query(map[string]string{"ids": joinInt64(ids)}), nil, &out)
	return &out, err
}

func (c *Client) AuthorPurge(ctx context.Context, authorID int64) (*PurgeResult, error) {
	var out PurgeResult
	err := c.do(ctx, http.MethodPost, "/authors/"+itoa(authorID)+"/purge", nil, &out)
	return &out, err
}

// RestoreAuthorPurge answers 404 when nothing is left to restore: the author
// was never purged on this site, the purge was already undone, or it is older
// than 30 days.
func (c *Client) RestoreAuthorPurge(ctx context.Context, authorID int64) (*RestoreResult, error) {
	var out RestoreResult
	err := c.do(ctx, http.MethodPost, "/authors/"+itoa(authorID)+"/purge/restore", nil, &out)
	return &out, err
}

func (c *Client) ResolvePosts(ctx context.Context, ids []int64) (*PostsResolveResponse, error) {
	if len(ids) == 0 {
		return &PostsResolveResponse{Posts: []AuthorPostView{}}, nil
	}
	var out PostsResolveResponse
	err := c.do(ctx, http.MethodPost, "/posts/resolve", PostsResolveRequest{IDs: ids}, &out)
	return &out, err
}

// SearchPosts matches the markdown source, not the cooked HTML, and answers a
// keyset page: there is no total to count and no relevance to rank by.
func (c *Client) SearchPosts(ctx context.Context, q string, kind int32, cursor string, limit int) (*PostFeedResponse, error) {
	var out PostFeedResponse
	q2 := map[string]string{"q": q, "kind": itoa(int64(kind)), "cursor": cursor}
	if limit > 0 {
		q2["limit"] = itoa(int64(limit))
	}
	err := c.do(ctx, http.MethodGet, "/search/posts"+query(q2), nil, &out)
	return &out, err
}

// MarkThreadRead advances the reader's high-water mark; it is clamped upstream
// to the thread's highest post number, so MaxInt32 means "all of it".
func (c *Client) MarkThreadRead(ctx context.Context, threadID, userID int64, lastRead int32) (*ThreadUserView, error) {
	var out ThreadUserView
	req := ThreadReadRequest{UserID: userID, LastReadPostNumber: lastRead}
	err := c.do(ctx, http.MethodPost, "/threads/"+itoa(threadID)+"/read", req, &out)
	return &out, err
}

func (c *Client) SetThreadNotification(ctx context.Context, threadID, userID int64, level int32) (*ThreadUserView, error) {
	var out ThreadUserView
	req := ThreadNotificationRequest{UserID: userID, Level: level}
	err := c.do(ctx, http.MethodPost, "/threads/"+itoa(threadID)+"/notification", req, &out)
	return &out, err
}

// ThreadStates reports only the threads the user has actually interacted with:
// a thread they never opened carries no row and is simply absent.
func (c *Client) ThreadStates(ctx context.Context, userID int64, threadIDs []int64) (*ThreadStatesResponse, error) {
	if len(threadIDs) == 0 {
		return &ThreadStatesResponse{States: []ThreadUserView{}}, nil
	}
	var out ThreadStatesResponse
	err := c.do(ctx, http.MethodPost, "/threads/states", ThreadStatesRequest{UserID: userID, ThreadIDs: threadIDs}, &out)
	return &out, err
}

func (c *Client) SetAnchorNotification(ctx context.Context, userID int64, kind int32, id string, level int32) (*AnchorSubscriptionView, error) {
	var out AnchorSubscriptionView
	req := AnchorNotificationRequest{UserID: userID, AnchorKind: kind, AnchorID: id, Level: level}
	err := c.do(ctx, http.MethodPost, "/anchors/notification", req, &out)
	return &out, err
}

func (c *Client) AnchorStates(ctx context.Context, userID int64, anchors []AnchorRef) (*AnchorStatesResponse, error) {
	if len(anchors) == 0 {
		return &AnchorStatesResponse{States: []AnchorSubscriptionView{}}, nil
	}
	var out AnchorStatesResponse
	err := c.do(ctx, http.MethodPost, "/anchors/states", AnchorStatesRequest{UserID: userID, Anchors: anchors}, &out)
	return &out, err
}

func (c *Client) ListAnchorSubscriptions(ctx context.Context, userID int64, anchorKind int32, cursor string, limit int) (*AnchorSubscriptionListResponse, error) {
	var out AnchorSubscriptionListResponse
	q := map[string]string{"anchor_kind": itoa(int64(anchorKind)), "cursor": cursor}
	if limit > 0 {
		q["limit"] = itoa(int64(limit))
	}
	err := c.do(ctx, http.MethodGet, "/users/"+itoa(userID)+"/anchor-subscriptions"+query(q), nil, &out)
	return &out, err
}

func (c *Client) NotificationFeed(ctx context.Context, after int64, limit int) (*NotificationFeedResponse, error) {
	var out NotificationFeedResponse
	q := map[string]string{"after": itoa(after)}
	if limit > 0 {
		q["limit"] = itoa(int64(limit))
	}
	err := c.do(ctx, http.MethodGet, "/notifications/feed"+query(q), nil, &out)
	return &out, err
}

func (c *Client) MarkNotificationsRead(ctx context.Context, userID int64, ids []int64) (*MarkNotificationsReadResponse, error) {
	if len(ids) == 0 {
		return &MarkNotificationsReadResponse{}, nil
	}
	var out MarkNotificationsReadResponse
	err := c.do(ctx, http.MethodPost, "/users/"+itoa(userID)+"/notifications/read", MarkNotificationsReadRequest{IDs: ids}, &out)
	return &out, err
}
