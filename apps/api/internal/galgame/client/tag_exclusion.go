package client

import (
	"context"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"kun-galgame-api/pkg/errors"
)

const tagExclusionPath = "/v2/catalog/tag-exclusion"

type TagExclusion struct {
	Exact []int64
	Trees []int64
}

func NewTagExclusion(exact, trees []int64) TagExclusion {
	return TagExclusion{Exact: normalizeTagIDs(exact), Trees: normalizeTagIDs(trees)}
}

func normalizeTagIDs(in []int64) []int64 {
	out := make([]int64, 0, len(in))
	for _, id := range in {
		if id > 0 {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func (e TagExclusion) Empty() bool { return len(e.Exact) == 0 && len(e.Trees) == 0 }

func (e TagExclusion) Apply(q url.Values) url.Values {
	if len(e.Exact) > 0 {
		q.Set("exclude_tag_id", joinTagIDs(e.Exact))
	}
	if len(e.Trees) > 0 {
		q.Set("exclude_tag_tree", joinTagIDs(e.Trees))
	}
	return q
}

func joinTagIDs(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
}

func (c *GalgameClient) ExcludedWorkIDs(ctx context.Context, e TagExclusion) ([]int, *errors.AppError) {
	if e.Empty() {
		return nil, nil
	}
	var body struct {
		WorkIDs []string `json:"work_ids"`
	}
	if appErr := c.getV2Native(ctx, tagExclusionPath, e.Apply(url.Values{}), &body); appErr != nil {
		return nil, appErr
	}
	out := make([]int, 0, len(body.WorkIDs))
	for _, s := range body.WorkIDs {
		if n, err := strconv.Atoi(s); err == nil {
			out = append(out, n)
		}
	}
	return out, nil
}
