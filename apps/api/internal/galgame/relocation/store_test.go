package relocation

import (
	"encoding/json"
	"errors"
	"testing"

	"kun-galgame-api/internal/testdb"

	"gorm.io/gorm"
)

const (
	workEnglish  = 2_000_340_001
	workJapanese = 2_000_340_002
	workChinese  = 2_000_340_003
	workUnsynced = 2_000_340_004
	workKorean   = 2_000_340_005

	resEnglishA = 2_000_340_101
	resEnglishB = 2_000_340_102
	resJapanese = 2_000_340_103
	resChinese  = 2_000_340_104
	resUnsynced = 2_000_340_105
	resKorean   = 2_000_340_106

	uploader = 940_340_001
	liker    = 940_340_002
)

func seed(t *testing.T) (*gorm.DB, *Store) {
	t.Helper()
	db := testdb.Open(t)
	cleanup := func() {
		db.Exec("DELETE FROM galgame_resource WHERE id BETWEEN ? AND ?", resEnglishA, resKorean)
		db.Exec("DELETE FROM galgame_resource_relocation WHERE resource_id BETWEEN ? AND ?", resEnglishA, resKorean)
		db.Exec("DELETE FROM galgame WHERE id BETWEEN ? AND ?", workEnglish, workKorean)
	}
	cleanup()
	t.Cleanup(cleanup)

	must := func(q string, args ...any) {
		t.Helper()
		if err := db.Exec(q, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	must(`INSERT INTO galgame (id, published, content_limit, resource_count, original_language, created, updated) VALUES
		(?, true, 'sfw', 2, 'en', now(), now()), (?, true, 'sfw', 1, 'ja', now(), now()),
		(?, true, 'sfw', 1, 'zh-Hans', now(), now()), (?, true, 'sfw', 1, NULL, now(), now()),
		(?, true, 'sfw', 1, 'ko', now(), now())`, workEnglish, workJapanese, workChinese, workUnsynced, workKorean)
	for res, work := range map[int]int{
		resEnglishA: workEnglish, resEnglishB: workEnglish, resJapanese: workJapanese,
		resChinese: workChinese, resUnsynced: workUnsynced, resKorean: workKorean,
	} {
		must(`INSERT INTO galgame_resource (id, work_id, user_id, type, size, code, password, note, download, view,
				languages, platforms, runtimes, updated)
			VALUES (?, ?, ?, 'game', '1.2GB', 'c0de', 'pass', 'a note', 7, 9,
				'["zh-cn"]', '["win"]', '["native-win"]', now())`, res, work, uploader)
	}
	must(`INSERT INTO galgame_resource_link (galgame_resource_id, url, updated) VALUES
		(?, 'https://pan.example/second', now()), (?, 'https://pan.example/first 提取码: ab12', now())`, resEnglishA, resEnglishA)
	must(`INSERT INTO galgame_resource_like (galgame_resource_id, user_id, updated) VALUES (?, ?, now())`, resEnglishA, liker)
	return db, NewStore(db)
}

func count(t *testing.T, db *gorm.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.Raw(q, args...).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSnapshotTakesOnlyWorksWhoseOriginalLanguageIsNeitherJaNorZh(t *testing.T) {
	db, store := seed(t)
	if _, err := store.Snapshot([]int{workKorean}); err != nil {
		t.Fatal(err)
	}
	var ids []int
	db.Raw(`SELECT resource_id FROM galgame_resource_relocation WHERE resource_id BETWEEN ? AND ? ORDER BY 1`,
		resEnglishA, resKorean).Scan(&ids)
	if len(ids) != 2 || ids[0] != resEnglishA || ids[1] != resEnglishB {
		t.Fatalf("snapshots = %v, want the two English-original resources: ja, zh, an unsynced work and an excluded work stay", ids)
	}

	works, err := store.CandidateWorks(nil)
	if err != nil {
		t.Fatal(err)
	}
	found := map[int]string{}
	for _, w := range works {
		if w.WorkID >= workEnglish && w.WorkID <= workKorean {
			found[w.WorkID] = w.Language
		}
	}
	if len(found) != 2 || found[workEnglish] != "en" || found[workKorean] != "ko" {
		t.Errorf("candidate works = %v, want en and ko", found)
	}
}

func TestSnapshotPayloadIsLetMoesImportLine(t *testing.T) {
	_, store := seed(t)
	if _, err := store.Snapshot(nil); err != nil {
		t.Fatal(err)
	}
	rows, err := store.Unpushed(1 << 30)
	if err != nil {
		t.Fatal(err)
	}
	var line struct {
		ForumID   int      `json:"forum_id"`
		WorkID    int      `json:"work_id"`
		UserID    int      `json:"user_id"`
		Type      string   `json:"type"`
		Size      string   `json:"size"`
		Note      string   `json:"note"`
		Code      string   `json:"code"`
		Password  string   `json:"password"`
		Status    *int     `json:"status"`
		Download  int      `json:"download"`
		View      int      `json:"view"`
		Created   string   `json:"created"`
		Languages []string `json:"languages"`
		Links     []struct {
			URL string `json:"url"`
		} `json:"links"`
		Likes []struct {
			UserID int `json:"user_id"`
		} `json:"likes"`
	}
	for _, row := range rows {
		if row.ResourceID == resEnglishA {
			if err := json.Unmarshal(row.Payload, &line); err != nil {
				t.Fatal(err)
			}
		}
	}
	if line.ForumID != resEnglishA || line.WorkID != workEnglish || line.UserID != uploader || line.Type != "game" ||
		line.Size != "1.2GB" || line.Note != "a note" || line.Code != "c0de" || line.Password != "pass" ||
		line.Status == nil || line.Download != 7 || line.View != 9 || line.Created == "" ||
		len(line.Languages) != 1 || line.Languages[0] != "zh-cn" {
		t.Fatalf("line = %+v", line)
	}
	if len(line.Links) != 2 || line.Links[0].URL != "https://pan.example/second" || line.Links[1].URL != "https://pan.example/first 提取码: ab12" {
		t.Errorf("links = %+v, want both in link-id order with the raw url", line.Links)
	}
	if len(line.Likes) != 1 || line.Likes[0].UserID != liker {
		t.Errorf("likes = %+v", line.Likes)
	}
}

func TestSnapshotKeepsThePayloadLetMoeWasSent(t *testing.T) {
	db, store := seed(t)
	if _, err := store.Snapshot(nil); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceipt(resEnglishA, 501, true); err != nil {
		t.Fatal(err)
	}
	db.Exec(`UPDATE galgame_resource SET note = 'edited' WHERE id IN (?, ?)`, resEnglishA, resEnglishB)
	if _, err := store.Snapshot(nil); err != nil {
		t.Fatal(err)
	}
	notes := map[int]string{}
	var rows []struct {
		ResourceID int
		Note       string
	}
	db.Raw(`SELECT resource_id, payload->>'note' AS note FROM galgame_resource_relocation WHERE resource_id IN (?, ?)`,
		resEnglishA, resEnglishB).Scan(&rows)
	for _, r := range rows {
		notes[r.ResourceID] = r.Note
	}
	if notes[resEnglishA] != "a note" || notes[resEnglishB] != "edited" {
		t.Errorf("notes = %v, want the pushed one frozen and the unpushed one refreshed", notes)
	}
}

func TestRetireDeletesOnlyWhatLetMoeConfirmed(t *testing.T) {
	db, store := seed(t)
	if _, err := store.Snapshot(nil); err != nil {
		t.Fatal(err)
	}
	if ids, _ := store.Retirable(); len(ids) != 0 {
		t.Fatalf("retirable before any receipt = %v", ids)
	}
	if err := store.Retire(resEnglishA); !errors.Is(err, ErrChanged) {
		t.Fatalf("retire without a receipt = %v, want a refusal", err)
	}
	if err := store.SaveReceipt(resEnglishA, 501, true); err != nil {
		t.Fatal(err)
	}
	db.Exec(`UPDATE galgame_resource SET download = download + 50, view = view + 50 WHERE id = ?`, resEnglishA)

	if err := store.Retire(resEnglishA); err != nil {
		t.Fatalf("retire: %v", err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM galgame_resource WHERE id = ?`, resEnglishA); n != 0 {
		t.Error("the resource row is still there")
	}
	if n := count(t, db, `SELECT COUNT(*) FROM galgame_resource_link WHERE galgame_resource_id = ?`, resEnglishA) +
		count(t, db, `SELECT COUNT(*) FROM galgame_resource_like WHERE galgame_resource_id = ?`, resEnglishA); n != 0 {
		t.Errorf("%d link and like rows outlived the resource", n)
	}
	if n := count(t, db, `SELECT resource_count FROM galgame WHERE id = ?`, workEnglish); n != 1 {
		t.Errorf("resource_count = %d, want 1", n)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM galgame_resource_relocation
		WHERE resource_id = ? AND retired_at IS NOT NULL AND destination_id = 501 AND destination_public
		  AND jsonb_array_length(payload->'links') = 2`, resEnglishA); n != 1 {
		t.Error("the ledger row lost its receipt or its snapshot")
	}
	if n := count(t, db, `SELECT COUNT(*) FROM galgame_resource WHERE id = ?`, resEnglishB); n != 1 {
		t.Error("a resource with no receipt was deleted")
	}
}

func TestRetireRefusesAResourceEditedAfterItsSnapshot(t *testing.T) {
	db, store := seed(t)
	if _, err := store.Snapshot(nil); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceipt(resEnglishA, 501, true); err != nil {
		t.Fatal(err)
	}
	res := db.Exec(`UPDATE galgame_resource_link SET url = 'https://pan.example/replaced'
		WHERE galgame_resource_id = ? AND url = 'https://pan.example/second'`, resEnglishA)
	if res.Error != nil || res.RowsAffected != 1 {
		t.Fatalf("edit the links: %d rows, %v", res.RowsAffected, res.Error)
	}

	if err := store.Retire(resEnglishA); !errors.Is(err, ErrChanged) {
		t.Fatalf("retire = %v, want a refusal: LetMoe holds the old link", err)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM galgame_resource WHERE id = ?`, resEnglishA); n != 1 {
		t.Error("the edited resource was deleted")
	}
}

func TestCensusCountsCandidatesAndTheLedgerTogether(t *testing.T) {
	_, store := seed(t)
	before, err := store.Census(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(nil); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceipt(resEnglishA, 501, true); err != nil {
		t.Fatal(err)
	}
	after, err := store.Census(nil)
	if err != nil {
		t.Fatal(err)
	}
	if before.Works < 2 || before.Resources < 3 || after.Resources != before.Resources {
		t.Errorf("candidates: before %+v, after %+v, want the seeded en and ko works in both", before, after)
	}
	if after.Snapshots-before.Snapshots != 3 || after.Pushed-before.Pushed != 1 {
		t.Errorf("ledger: before %+v, after %+v, want 3 new snapshots and 1 pushed", before, after)
	}
}

func TestDropLikesTakesDeletedAccountsOutOfUnpushedSnapshotsOnly(t *testing.T) {
	db, store := seed(t)
	const stays = 940_340_004
	if err := db.Exec(`INSERT INTO galgame_resource_like (galgame_resource_id, user_id, updated) VALUES
		(?, ?, now()), (?, ?, now())`, resEnglishA, stays, resEnglishB, liker).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(nil); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceipt(resEnglishB, 7002, true); err != nil {
		t.Fatal(err)
	}

	likers, err := store.UnpushedLikers()
	if err != nil {
		t.Fatal(err)
	}
	mine := map[int]bool{}
	for _, id := range likers {
		mine[id] = id == liker || id == stays
	}
	if !mine[liker] || !mine[stays] {
		t.Fatalf("unpushed likers = %v, want both likers of the unpushed resource", likers)
	}

	if n, err := store.DropLikes(nil); err != nil || n != 0 {
		t.Fatalf("DropLikes(nil) = %d, %v", n, err)
	}
	if _, err := store.DropLikes([]int{liker}); err != nil {
		t.Fatal(err)
	}
	likes := func(res int) string {
		var out string
		db.Raw(`SELECT COALESCE(string_agg(k->>'user_id', ',' ORDER BY ord), '')
			FROM galgame_resource_relocation m, jsonb_array_elements(m.payload->'likes') WITH ORDINALITY AS t(k, ord)
			WHERE m.resource_id = ?`, res).Scan(&out)
		return out
	}
	if got := likes(resEnglishA); got != "940340004" {
		t.Errorf("unpushed snapshot likes = %q, want only the account that still exists", got)
	}
	if got := likes(resEnglishB); got != "940340002" {
		t.Errorf("pushed snapshot likes = %q, want it left as LetMoe was sent it", got)
	}
	if err := store.Retire(resEnglishB); err != nil {
		t.Fatalf("retire after a like was dropped elsewhere: %v", err)
	}
}

func TestUnsyncedWorksAreTheOnesHoldingResourcesWithNoLanguageYet(t *testing.T) {
	db, store := seed(t)
	if err := db.Exec(`UPDATE galgame SET original_language = NULL WHERE id = ?`, workKorean).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`DELETE FROM galgame_resource WHERE id = ?`, resKorean).Error; err != nil {
		t.Fatal(err)
	}
	ids, err := store.UnsyncedWorks()
	if err != nil {
		t.Fatal(err)
	}
	var mine []int
	for _, id := range ids {
		if id >= workEnglish && id <= workKorean {
			mine = append(mine, id)
		}
	}
	if len(mine) != 1 || mine[0] != workUnsynced {
		t.Errorf("unsynced works = %v, want only the one that holds a resource and has no language", mine)
	}
}

func TestReceiptsListsWhatLetMoeAnsweredInItsOwnShape(t *testing.T) {
	_, store := seed(t)
	if _, err := store.Snapshot(nil); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceipt(resEnglishA, 7001, true); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveReceipt(resEnglishB, 7002, false); err != nil {
		t.Fatal(err)
	}
	all, err := store.Receipts()
	if err != nil {
		t.Fatal(err)
	}
	got := map[int]Receipt{}
	for _, r := range all {
		if r.ForumID >= resEnglishA && r.ForumID <= resKorean {
			got[r.ForumID] = r
		}
	}
	if len(got) != 2 || got[resEnglishA].ResourceID != 7001 || !got[resEnglishA].Public ||
		got[resEnglishB].ResourceID != 7002 || got[resEnglishB].Public {
		t.Errorf("receipts = %+v, want the two pushed resources and not the Korean one still waiting", got)
	}
}
