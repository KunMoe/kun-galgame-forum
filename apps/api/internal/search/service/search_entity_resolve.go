package service

import (
	"context"
	"strconv"
	"strings"

	galgameDto "kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/errors"
)

// Catalog answers 400 past ten tag ids.
const maxSearchTagIDs = 10

func parseIDList(raw string, cap int) []string {
	out := make([]string, 0, cap)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if n, err := strconv.Atoi(part); err != nil || n <= 0 {
			continue
		}
		out = append(out, part)
		if len(out) == cap {
			break
		}
	}
	return out
}

func (s *SearchService) ResolveEntities(
	ctx context.Context,
	family, rawIDs string,
	isSFW bool,
) ([]galgameDto.EntitySearchItem, *errors.AppError) {
	if s.entityService == nil {
		return nil, errors.ErrInternal("Galgame 资料库搜索未启用")
	}
	raw := parseIDList(rawIDs, maxSearchTagIDs)
	ids := make([]int, 0, len(raw))
	for _, id := range raw {
		n, _ := strconv.Atoi(id)
		ids = append(ids, n)
	}
	return s.entityService.Resolve(ctx, family, ids, isSFW)
}
