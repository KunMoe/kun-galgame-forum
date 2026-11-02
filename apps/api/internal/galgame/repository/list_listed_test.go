package repository

import (
	"sync"
	"testing"
	"time"

	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/testdb"

	"gorm.io/gorm"
)

func TestListedWorkIDsUsesTheCachedSet(t *testing.T) {
	cached := map[int]struct{}{7: {}}
	repo := &GalgameListRepository{db: &gorm.DB{}}
	repo.listed.ids = cached
	repo.listed.ok = true
	repo.listed.at = time.Now()

	var wg sync.WaitGroup
	got := make([]map[int]struct{}, 8)
	for i := range got {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got[i] = repo.ListedWorkIDs()
		}(i)
	}
	wg.Wait()
	got[0][8] = struct{}{}
	for i := range got {
		if _, ok := got[i][7]; !ok {
			t.Errorf("call %d lost the cached id", i)
		}
		if _, ok := got[i][8]; !ok {
			t.Errorf("call %d did not share the cached map", i)
		}
	}
	delete(got[0], 8)
}

func TestListedWorkIDs(t *testing.T) {
	db := testdb.Open(t)

	const base = 2_000_400_000
	listed, bare, hidden, banned := base, base+1, base+2, base+3
	all := []int{listed, bare, hidden, banned}

	cleanup := func() {
		db.Exec("DELETE FROM galgame_resource WHERE work_id = ANY(?::int[])", intArrayLit(all))
		db.Exec("DELETE FROM galgame WHERE id = ANY(?::int[])", intArrayLit(all))
	}
	cleanup()
	defer cleanup()

	seed(t, db, listed, nil)
	if err := db.Create(&model.GalgameLocal{ID: bare, Published: true}).Error; err != nil {
		t.Fatalf("seed bare: %v", err)
	}
	seed(t, db, hidden, nil)
	db.Exec("UPDATE galgame SET catalog_rendered = false WHERE id = ?", hidden)
	seed(t, db, banned, nil)
	db.Exec("UPDATE galgame SET published = false WHERE id = ?", banned)

	repo := NewGalgameListRepository(db)
	set := repo.ListedWorkIDs()
	if _, ok := set[listed]; !ok {
		t.Error("a published, rendered work with a resource is listed")
	}
	for _, id := range []int{bare, hidden, banned} {
		if _, ok := set[id]; ok {
			t.Errorf("work %d is listed", id)
		}
	}

	again := repo.ListedWorkIDs()
	if _, ok := again[listed]; !ok {
		t.Error("cached set lost the listed work")
	}

	if err := db.Create(&model.GalgameResource{
		WorkID: bare, UserID: 1, Platform: "windows", Language: "ja", Type: "game",
	}).Error; err != nil {
		t.Fatalf("add resource: %v", err)
	}
	if _, ok := repo.ListedWorkIDs()[bare]; ok {
		t.Error("a resource published inside the ttl showed up immediately")
	}

	repo.listed.mu.Lock()
	repo.listed.at = time.Now().Add(-listedWorkTTL - time.Second)
	repo.listed.mu.Unlock()
	if _, ok := repo.ListedWorkIDs()[bare]; !ok {
		t.Error("the set did not reload after the ttl")
	}
}
