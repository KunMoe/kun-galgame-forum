package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
)

type fakeMergeRepo struct {
	local map[int]bool
	folds [][2]int
	chain map[int]int
}

func newFakeMerge(ids ...int) *fakeMergeRepo {
	local := make(map[int]bool, len(ids))
	for _, id := range ids {
		local[id] = true
	}
	return &fakeMergeRepo{local: local}
}

func (f *fakeMergeRepo) LocalIDsIn(ids []int) []int {
	if len(ids) == 0 {
		return nil
	}
	var out []int
	for _, id := range ids {
		if f.local[id] {
			out = append(out, id)
		}
	}
	return out
}

func (f *fakeMergeRepo) Fold(oldGID, newGID int) (repository.MergeCounts, error) {
	f.folds = append(f.folds, [2]int{oldGID, newGID})
	delete(f.local, oldGID)
	return repository.MergeCounts{}, nil
}

func (f *fakeMergeRepo) RedirectTarget(oldGID int) (int, bool) {
	t, ok := f.chain[oldGID]
	return t, ok
}

func claimedWork(id int64, gid int, refs string) string {
	if refs == "" {
		refs = "[]"
	}
	return fmt.Sprintf(
		`{"id":%d,"medium":"galgame","display_name":"x",`+
			`"claim":{"site":"kungal","site_work_id":%d,"state":"live"},"refs":%s}`,
		id, gid, refs)
}

func mergeCatalogServer(t *testing.T, byID map[int64]string, byRef map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := req.URL.Query()
		var rows []string
		if refs := q.Get("refs"); refs != "" {
			seen := map[string]bool{}
			for _, token := range strings.Split(refs, ",") {
				ext := token
				if i := strings.LastIndex(token, ":"); i >= 0 {
					ext = token[i+1:]
				}
				if seen[ext] {
					continue
				}
				seen[ext] = true
				if frag, ok := byRef[ext]; ok {
					rows = append(rows, frag)
				}
			}
			_, _ = w.Write([]byte(`{"object":"list","items":[` + strings.Join(rows, ",") + `],"missing":[]}`))
			return
		}
		for _, raw := range strings.Split(q.Get("ids"), ",") {
			id, _ := strconv.ParseInt(raw, 10, 64)
			if frag, ok := byID[id]; ok {
				rows = append(rows, frag)
			}
		}
		_, _ = w.Write([]byte(`{"items":[` + strings.Join(rows, ",") + `],"next_cursor":null}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newMergeSync(t *testing.T, srv *httptest.Server, local ...int) (*GalgameMergeSync, *fakeMergeRepo) {
	t.Helper()
	repo := newFakeMerge(local...)
	s := &GalgameMergeSync{
		galgameClient: client.New(srv.URL, "nm_test_key", ""),
		mergeRepo:     repo,
	}
	return s, repo
}

func assertFolds(t *testing.T, repo *fakeMergeRepo, want ...[2]int) {
	t.Helper()
	if len(repo.folds) != len(want) {
		t.Fatalf("folds = %v, want %v", repo.folds, want)
	}
	for i, p := range want {
		if repo.folds[i] != p {
			t.Errorf("fold[%d] = %v, want %v", i, repo.folds[i], p)
		}
	}
}

func yan6Survivor() (byID map[int64]string, byRef map[string]string) {
	row := claimedWork(61101, 61295, `[{"source":"curated","external_id":"5904"}]`)
	return map[int64]string{61101: row}, map[string]string{
		"5904":  row,
		"61295": row,
	}
}

func TestFold_StaleGIDThatResolvesToTheSurvivor(t *testing.T) {
	byID, byRef := yan6Survivor()
	srv := mergeCatalogServer(t, byID, byRef)

	t.Run("wiki-era old catalog id is not a local gid", func(t *testing.T) {
		s, repo := newMergeSync(t, srv, 5904, 61295)
		folded, deferred, err := s.fold(t.Context(), map[int]int64{1: 61101})
		if err != nil {
			t.Fatalf("fold: %v", err)
		}
		if folded != 1 || deferred != 0 {
			t.Errorf("folded=%d deferred=%d, want 1/0", folded, deferred)
		}
		assertFolds(t, repo, [2]int{5904, 61295})
	})

	t.Run("stale gid is the redirect old id and still resolves", func(t *testing.T) {
		s, repo := newMergeSync(t, srv, 5904, 61295)
		folded, deferred, err := s.fold(t.Context(), map[int]int64{5904: 61101})
		if err != nil {
			t.Fatalf("fold: %v", err)
		}
		if folded != 1 || deferred != 0 {
			t.Errorf("folded=%d deferred=%d, want 1/0", folded, deferred)
		}
		assertFolds(t, repo, [2]int{5904, 61295})
	})
}

func TestFold_CoincidenceGidDoesNotFold(t *testing.T) {
	survivor := claimedWork(61101, 61295, `[{"source":"curated","external_id":"5000"}]`)
	other := claimedWork(5000, 5000, `[]`)
	srv := mergeCatalogServer(t,
		map[int64]string{61101: survivor, 5000: other},
		map[string]string{
			"5000":  other,
			"61295": survivor,
		},
	)
	s, repo := newMergeSync(t, srv, 5000, 61295)
	folded, deferred, err := s.fold(t.Context(), map[int]int64{1: 61101})
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	if folded != 0 || deferred != 0 {
		t.Errorf("folded=%d deferred=%d, want 0/0 — 5000 is another work's catalog id", folded, deferred)
	}
	assertFolds(t, repo)
}

func TestFold_StaleGIDLeftAloneWhenCanonicalHasNoRow(t *testing.T) {
	byID, byRef := yan6Survivor()
	srv := mergeCatalogServer(t, byID, byRef)
	s, repo := newMergeSync(t, srv, 5904)
	folded, deferred, err := s.fold(t.Context(), map[int]int64{1: 61101})
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	if folded != 0 || deferred != 0 {
		t.Errorf("folded=%d deferred=%d, want 0/0 — folding 5904 into a gid with no row discards the busy page", folded, deferred)
	}
	if !repo.local[5904] {
		t.Error("stale gid 5904 was removed without a fold")
	}
	assertFolds(t, repo)
}

func TestFold_WorkIDShapedStaleCandidate(t *testing.T) {
	idsRow := claimedWork(61101, 61295, `[]`)
	refsRow := claimedWork(61101, 61295, `[{"source":"curated","external_id":"61101"}]`)
	srv := mergeCatalogServer(t,
		map[int64]string{61101: idsRow},
		map[string]string{
			"61101": refsRow,
			"61295": idsRow,
		},
	)
	s, repo := newMergeSync(t, srv, 61101, 61295)
	folded, deferred, err := s.fold(t.Context(), map[int]int64{1: 61101})
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	if folded != 1 || deferred != 0 {
		t.Errorf("folded=%d deferred=%d, want 1/0", folded, deferred)
	}
	assertFolds(t, repo, [2]int{61101, 61295})
}

func TestFold_OrdinaryDeadGidStillFolds(t *testing.T) {
	survivor := claimedWork(200, 300, `[]`)
	srv := mergeCatalogServer(t,
		map[int64]string{200: survivor},
		map[string]string{"300": survivor},
	)
	s, repo := newMergeSync(t, srv, 100, 300)
	folded, deferred, err := s.fold(t.Context(), map[int]int64{100: 200})
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	if folded != 1 || deferred != 0 {
		t.Errorf("folded=%d deferred=%d, want 1/0", folded, deferred)
	}
	assertFolds(t, repo, [2]int{100, 300})
}

// The stale candidate list is built from the survivor's leftover refs, so it
// names gids the forum may never have materialised: 61295 itself is one on
// production. Folding a gid with no row writes a redirect for a page nobody
// ever opened and counts it as work done.
func TestFold_StaleGIDWithNoLocalRowIsNotFolded(t *testing.T) {
	byID, byRef := yan6Survivor()
	srv := mergeCatalogServer(t, byID, byRef)
	s, repo := newMergeSync(t, srv, 61295)
	folded, deferred, err := s.fold(t.Context(), map[int]int64{1: 61101})
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	if folded != 0 || deferred != 0 {
		t.Errorf("folded=%d deferred=%d, want 0/0 — 5904 has no forum row, there is nothing to move", folded, deferred)
	}
	assertFolds(t, repo)
}
