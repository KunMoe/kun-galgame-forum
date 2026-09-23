package service

import (
	"testing"

	"kun-galgame-api/internal/galgame/repository"
)

type fakeMergeRepo struct {
	local map[int]bool
	folds [][2]int
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

func (f *fakeMergeRepo) Fold(oldWorkID, newWorkID int) (repository.MergeCounts, error) {
	f.folds = append(f.folds, [2]int{oldWorkID, newWorkID})
	delete(f.local, oldWorkID)
	return repository.MergeCounts{}, nil
}

func TestFold_LocalRetiredIDMovesOntoSurvivor(t *testing.T) {
	repo := newFakeMerge(5904, 61101)
	s := &GalgameMergeSync{mergeRepo: repo}

	folded, deferred := s.fold(t.Context(), map[int]int64{5904: 61101})
	if folded != 1 || deferred != 0 {
		t.Errorf("folded=%d deferred=%d, want 1/0", folded, deferred)
	}
	if len(repo.folds) != 1 || repo.folds[0] != [2]int{5904, 61101} {
		t.Errorf("folds = %v, want [5904 61101]", repo.folds)
	}
}

func TestFold_AbsentLocalRowIsANoOp(t *testing.T) {
	repo := newFakeMerge(61101)
	s := &GalgameMergeSync{mergeRepo: repo}

	folded, deferred := s.fold(t.Context(), map[int]int64{5904: 61101})
	if folded != 0 || deferred != 0 {
		t.Errorf("folded=%d deferred=%d, want 0/0", folded, deferred)
	}
	if len(repo.folds) != 0 {
		t.Errorf("folds = %v, want none", repo.folds)
	}
}

func TestFold_IdentityRedirectIsSkipped(t *testing.T) {
	repo := newFakeMerge(61101)
	s := &GalgameMergeSync{mergeRepo: repo}

	folded, deferred := s.fold(t.Context(), map[int]int64{61101: 61101})
	if folded != 0 || deferred != 0 {
		t.Errorf("folded=%d deferred=%d, want 0/0", folded, deferred)
	}
	if len(repo.folds) != 0 {
		t.Errorf("folds = %v, want none", repo.folds)
	}
}
