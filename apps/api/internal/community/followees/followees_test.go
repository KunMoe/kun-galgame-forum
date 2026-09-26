package followees

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"

	"kun-galgame-api/pkg/communityclient"
)

type pagedSource struct {
	ids   []int64
	calls atomic.Int32
	err   error
}

func (s *pagedSource) ListFollowing(_ context.Context, _ int64, cursor string, limit int) (*communityclient.FollowListResponse, error) {
	s.calls.Add(1)
	if s.err != nil {
		return nil, s.err
	}
	start, _ := strconv.Atoi(cursor)
	end := min(start+limit, len(s.ids))
	out := &communityclient.FollowListResponse{}
	for _, id := range s.ids[start:end] {
		out.Users = append(out.Users, communityclient.FollowListUser{UserID: id})
	}
	if end < len(s.ids) {
		out.NextCursor = strconv.Itoa(end)
	}
	return out, nil
}

func TestIDsWalksEveryPageAndCaches(t *testing.T) {
	src := &pagedSource{}
	for i := range 250 {
		src.ids = append(src.ids, int64(i+1))
	}
	c := New(src)
	ids, err := c.IDs(context.Background(), 7)
	if err != nil || len(ids) != 250 || ids[0] != 1 || ids[249] != 250 {
		t.Fatalf("ids %d %v", len(ids), err)
	}
	if n := src.calls.Load(); n != 3 {
		t.Fatalf("pages fetched %d, want 3", n)
	}
	if _, err := c.IDs(context.Background(), 7); err != nil || src.calls.Load() != 3 {
		t.Fatalf("second read went upstream: calls %d, err %v", src.calls.Load(), err)
	}

	src.ids = append(src.ids, 999)
	c.Forget(7)
	ids, err = c.IDs(context.Background(), 7)
	if err != nil || !slices.Contains(ids, 999) {
		t.Fatalf("after Forget: %d ids, err %v", len(ids), err)
	}
}

func TestIDsFollowsNobody(t *testing.T) {
	ids, err := New(&pagedSource{}).IDs(context.Background(), 7)
	if err != nil || ids == nil || len(ids) != 0 {
		t.Fatalf("ids %#v, err %v; want an empty, non-nil list", ids, err)
	}
}

func TestIDsUpstreamErrorIsNotCached(t *testing.T) {
	src := &pagedSource{err: errors.New("down")}
	c := New(src)
	if _, err := c.IDs(context.Background(), 7); err == nil {
		t.Fatal("want the upstream error")
	}
	src.err = nil
	src.ids = []int64{3}
	if ids, err := c.IDs(context.Background(), 7); err != nil || len(ids) != 1 {
		t.Fatalf("after recovery: %v %v", ids, err)
	}
}

func TestUnconfigured(t *testing.T) {
	if _, err := NewFromClient(nil).IDs(context.Background(), 7); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err %v, want ErrUnavailable", err)
	}
	var c *Cache
	c.Forget(7)
	if _, err := c.IDs(context.Background(), 7); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil cache err %v", err)
	}
}

type gateSource struct {
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
	ids     []int64
}

func (s *gateSource) ListFollowing(_ context.Context, _ int64, _ string, _ int) (*communityclient.FollowListResponse, error) {
	s.calls.Add(1)
	if s.entered != nil {
		select {
		case s.entered <- struct{}{}:
		default:
		}
	}
	if s.release != nil {
		<-s.release
	}
	out := &communityclient.FollowListResponse{}
	for _, id := range s.ids {
		out.Users = append(out.Users, communityclient.FollowListUser{UserID: id})
	}
	return out, nil
}

func TestForgetDiscardsInFlight(t *testing.T) {
	for _, forgetID := range []int{7, 9} {
		t.Run(strconv.Itoa(forgetID), func(t *testing.T) {
			src := &gateSource{
				entered: make(chan struct{}, 1),
				release: make(chan struct{}),
				ids:     []int64{1, 2, 3},
			}
			c := New(src)
			done := make(chan struct{})
			var got []int
			var gotErr error
			go func() {
				got, gotErr = c.IDs(context.Background(), 7)
				close(done)
			}()
			<-src.entered
			c.Forget(forgetID)
			close(src.release)
			<-done
			if gotErr != nil || len(got) != 3 {
				t.Fatalf("in-flight: ids %v err %v", got, gotErr)
			}
			calls := src.calls.Load()
			if _, err := c.IDs(context.Background(), 7); err != nil {
				t.Fatal(err)
			}
			if src.calls.Load() == calls {
				t.Fatal("second IDs served from cache after Forget during fetch")
			}
		})
	}
}

type endlessSource struct {
	calls atomic.Int32
}

func (s *endlessSource) ListFollowing(_ context.Context, _ int64, _ string, limit int) (*communityclient.FollowListResponse, error) {
	n := int(s.calls.Add(1))
	out := &communityclient.FollowListResponse{NextCursor: "more"}
	base := (n - 1) * limit
	for i := range limit {
		out.Users = append(out.Users, communityclient.FollowListUser{UserID: int64(base + i + 1)})
	}
	return out, nil
}

func TestIDsTruncatesAtMaxPages(t *testing.T) {
	src := &endlessSource{}
	ids, err := New(src).IDs(context.Background(), 7)
	want := maxPages * pageSize
	if err != nil || len(ids) != want {
		t.Fatalf("ids %d err %v, want %d and no error", len(ids), err, want)
	}
	if n := src.calls.Load(); n != int32(maxPages) {
		t.Fatalf("pages fetched %d, want %d", n, maxPages)
	}
}
