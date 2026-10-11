package client

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kun-galgame-api/pkg/errors"
)

const (
	v2CacheKeyPrefix    = "nmcache:v2:"
	v2CacheMaxBody      = 512 << 10
	v2CacheDetailTTL    = 15 * time.Second
	v2CacheListTTL      = 60 * time.Second
	v2CacheCalendarTTL  = time.Hour
	v2CacheExclusionTTL = 10 * time.Minute
)

func v2CacheIdentity(v2Path string, q url.Values) string {
	if len(q) > 0 {
		return v2Path + "?" + q.Encode()
	}
	return v2Path
}

func v2CacheKey(v2Path string, q url.Values) string {
	return v2CacheKeyPrefix + v2CacheIdentity(v2Path, q)
}

func v2CacheTTL(v2Path string) time.Duration {
	if v2Path == tagExclusionPath {
		return v2CacheExclusionTTL
	}
	if strings.HasPrefix(v2Path, "/v2/catalog/calendar") {
		return v2CacheCalendarTTL
	}
	for _, seg := range strings.Split(v2Path, "/") {
		if seg != "" && v2SegmentIsNumericID(seg) {
			return v2CacheDetailTTL
		}
	}
	return v2CacheListTTL
}

func v2SegmentIsNumericID(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

func (c *GalgameClient) getV2(ctx context.Context, v2Path string, q url.Values) (int, []byte, *errors.AppError) {
	if body, ok := c.v2CacheGet(ctx, v2Path, q); ok {
		return http.StatusOK, body, nil
	}
	status, body, appErr := c.doV2HTTP(ctx, http.MethodGet, v2Path, q, nil)
	if appErr != nil {
		return status, body, appErr
	}
	if status == http.StatusOK {
		c.v2CacheSet(ctx, v2Path, q, body)
	}
	return status, body, nil
}

func (c *GalgameClient) v2CacheGet(ctx context.Context, v2Path string, q url.Values) ([]byte, bool) {
	if c.rdb == nil {
		return nil, false
	}
	body, err := c.rdb.Get(ctx, v2CacheKey(v2Path, q)).Bytes()
	if err != nil {
		return nil, false
	}
	return body, true
}

func (c *GalgameClient) v2CacheSet(ctx context.Context, v2Path string, q url.Values, body []byte) {
	if c.rdb == nil || len(body) > v2CacheMaxBody {
		return
	}
	if err := c.rdb.Set(ctx, v2CacheKey(v2Path, q), body, v2CacheTTL(v2Path)).Err(); err != nil {
		return
	}
}
