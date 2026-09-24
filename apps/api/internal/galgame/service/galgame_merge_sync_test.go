package service

import (
	"context"
	"testing"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/pkg/errors"
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

type fakeSurvivors struct {
	rendered map[int]bool
	down     bool
}

func (f fakeSurvivors) MirrorByCatalogIDs(_ context.Context, ids []int64) (map[int]client.CatalogMirror, []int, *errors.AppError) {
	if f.down {
		return nil, nil, errors.ErrInternal("catalog down")
	}
	out := map[int]client.CatalogMirror{}
	var hidden []int
	for _, id := range ids {
		if f.rendered[int(id)] {
			out[int(id)] = client.CatalogMirror{}
		} else {
			hidden = append(hidden, int(id))
		}
	}
	return out, hidden, nil
}

func TestFold_LocalRetiredIDMovesOntoSurvivor(t *testing.T) {
	repo := newFakeMerge(5904, 61101)
	s := &GalgameMergeSync{mergeRepo: repo, survivors: fakeSurvivors{rendered: map[int]bool{61101: true}}}

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

func TestFold_HiddenSurvivorIsParked(t *testing.T) {
	repo := newFakeMerge(206987)
	s := &GalgameMergeSync{mergeRepo: repo, survivors: fakeSurvivors{}}

	folded, deferred := s.fold(t.Context(), map[int]int64{206987: 226964})
	if folded != 0 || deferred != 1 {
		t.Errorf("folded=%d deferred=%d, want 0/1", folded, deferred)
	}
	if len(repo.folds) != 0 {
		t.Errorf("folds = %v, want none: the survivor is a page nobody can open", repo.folds)
	}
}

func TestFold_SurvivorLookupFailureParks(t *testing.T) {
	repo := newFakeMerge(5904)
	s := &GalgameMergeSync{mergeRepo: repo, survivors: fakeSurvivors{down: true}}

	folded, deferred := s.fold(t.Context(), map[int]int64{5904: 61101})
	if folded != 0 || deferred != 1 {
		t.Errorf("folded=%d deferred=%d, want 0/1", folded, deferred)
	}
	if len(repo.folds) != 0 {
		t.Errorf("folds = %v, want none until the survivor can be looked up", repo.folds)
	}
}
