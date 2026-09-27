package app

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"kun-galgame-api/pkg/communityclient"
)

func TestAnchorPresentationTriggers(t *testing.T) {
	f := newApnFix(t)
	now := time.Now()
	f.insertGalgame(t, apnWork, true)
	if !f.queueHas(t, 1, strconv.Itoa(apnWork)) || f.queueCount(t) != 1 {
		t.Fatalf("galgame insert queue = %d has=%v", f.queueCount(t), f.queueHas(t, 1, strconv.Itoa(apnWork)))
	}
	f.resetQueue(t)
	f.insertResource(t, apnRes, apnWork, w3UserAlice)
	if !f.queueHas(t, 2, "resource:960200101") || f.queueCount(t) != 1 {
		t.Fatal("resource insert")
	}
	f.resetQueue(t)
	f.insertRating(t, apnRating, apnWork, w3UserAlice)
	if !f.queueHas(t, 2, "rating:960200201") || f.queueCount(t) != 1 {
		t.Fatal("rating insert")
	}
	f.resetQueue(t)
	f.insertQuiz(t, apnQuiz, w3UserAlice, "q", false)
	if !f.queueHas(t, 2, "quiz:960200301") || f.queueCount(t) != 1 {
		t.Fatal("quiz insert")
	}
	f.resetQueue(t)
	f.insertToolset(t, apnTool, w3UserAlice, 0, "t")
	if !f.queueHas(t, 2, "toolset:960200401") || f.queueCount(t) != 1 {
		t.Fatal("toolset insert")
	}
	f.resetQueue(t)
	f.insertWebsite(t, apnSite, "n", "apn.example", "all")
	if !f.queueHas(t, 2, "website:960200501") || f.queueCount(t) != 1 {
		t.Fatal("website insert")
	}

	f.resetQueue(t)
	f.runSQL(t, `DELETE FROM galgame_website WHERE id = ?`, apnSite)
	if !f.queueHas(t, 2, "website:960200501") {
		t.Fatal("website delete")
	}
	f.resetQueue(t)
	f.runSQL(t, `DELETE FROM galgame_toolset WHERE id = ?`, apnTool)
	if !f.queueHas(t, 2, "toolset:960200401") {
		t.Fatal("toolset delete")
	}
	f.resetQueue(t)
	f.runSQL(t, `DELETE FROM galgame_quiz WHERE id = ?`, apnQuiz)
	if !f.queueHas(t, 2, "quiz:960200301") {
		t.Fatal("quiz delete")
	}
	f.resetQueue(t)
	f.runSQL(t, `DELETE FROM galgame_rating WHERE id = ?`, apnRating)
	if !f.queueHas(t, 2, "rating:960200201") {
		t.Fatal("rating delete")
	}
	f.resetQueue(t)
	f.runSQL(t, `DELETE FROM galgame_resource WHERE id = ?`, apnRes)
	if !f.queueHas(t, 2, "resource:960200101") {
		t.Fatal("resource delete")
	}
	f.resetQueue(t)
	f.runSQL(t, `DELETE FROM galgame WHERE id = ?`, apnWork)
	if !f.queueHas(t, 1, strconv.Itoa(apnWork)) {
		t.Fatal("galgame delete")
	}

	f.insertGalgame(t, apnWork, true)
	f.insertResource(t, apnRes, apnWork, w3UserAlice)
	f.insertRating(t, apnRating, apnWork, w3UserAlice)
	f.insertQuiz(t, apnQuiz, w3UserAlice, "q", false)
	f.linkQuiz(t, apnQuiz, apnWork)
	f.resetQueue(t)
	f.runSQL(t, `UPDATE galgame SET view = view + 1 WHERE id = ?`, apnWork)
	if f.queueCount(t) != 0 {
		t.Fatalf("view update enqueued %d", f.queueCount(t))
	}
	f.runSQL(t, `UPDATE galgame SET published = false WHERE id = ?`, apnWork)
	if !f.queueHas(t, 1, strconv.Itoa(apnWork)) || !f.queueHas(t, 2, "resource:960200101") ||
		!f.queueHas(t, 2, "rating:960200201") || !f.queueHas(t, 2, "quiz:960200301") {
		t.Fatal("published flip did not enqueue work and children")
	}
	f.resetQueue(t)
	f.runSQL(t, `UPDATE galgame SET catalog_checked_at = now(), catalog_rendered = catalog_rendered,
		content_limit = content_limit, published = published WHERE id = ?`, apnWork)
	if f.queueCount(t) != 0 {
		t.Fatalf("a mirror poll that changed nothing enqueued %d", f.queueCount(t))
	}
	for _, change := range []string{
		`UPDATE galgame SET catalog_rendered = NOT COALESCE(catalog_rendered, false) WHERE id = ?`,
		`UPDATE galgame SET content_limit = CASE WHEN content_limit = 'nsfw' THEN 'sfw' ELSE 'nsfw' END WHERE id = ?`,
	} {
		f.resetQueue(t)
		f.runSQL(t, change, apnWork)
		if !f.queueHas(t, 1, strconv.Itoa(apnWork)) || !f.queueHas(t, 2, "resource:960200101") ||
			!f.queueHas(t, 2, "rating:960200201") || !f.queueHas(t, 2, "quiz:960200301") {
			t.Fatalf("did not enqueue work and children: %s", change)
		}
	}

	f.resetQueue(t)
	f.runSQL(t, `INSERT INTO galgame_quiz_galgame (quiz_id, work_id) VALUES (?, ?)`, apnQuiz, apnWorkB)
	if !f.queueHas(t, 2, "quiz:960200301") {
		t.Fatal("quiz_galgame insert")
	}

	f.resetQueue(t)
	f.runSQL(t, `INSERT INTO feed_activity (type, source_id, user_id, work_id, content, link, created)
		VALUES ('GALGAME_COMMENT_CREATION', ?, ?, ?, '', '/galgame/960200009', ?)`,
		apnFeedSID, w3UserAlice, apnFeedWork, now)
	if !f.queueHas(t, 1, strconv.Itoa(apnFeedWork)) {
		t.Fatal("comment feed without galgame row")
	}

	f.resetQueue(t)
	f.runSQL(t, `UPDATE galgame_resource SET note = 'x' WHERE id = ?`, apnRes)
	if f.queueCount(t) != 0 {
		t.Fatalf("resource note update enqueued %d", f.queueCount(t))
	}
}

func TestAnchorPresentationFields(t *testing.T) {
	f := newApnFix(t)
	origin := "https://www.kungal.com"
	f.seedWork(t, apnWork, "nsfw")
	f.insertGalgame(t, apnWork, true)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok := f.findItem(1, strconv.Itoa(apnWork))
	if !ok {
		t.Fatal("missing galgame item")
	}
	if it.Title != apnWorkTitle(apnWork) || it.URL != origin+"/galgame/960200001" ||
		it.WorkID != apnWork || it.CoverImageHash != apnCover || it.ContentLimit != "nsfw" || it.Removed {
		t.Fatalf("galgame %+v", it)
	}

	f.resetQueue(t)
	f.insertResource(t, apnRes, apnWork, w3UserAlice)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "resource:960200101")
	if !ok || it.Title != apnWorkTitle(apnWork) || it.URL != origin+"/galgame/resource/960200101" ||
		it.WorkID != apnWork || it.CoverImageHash != apnCover || it.ContentLimit != "nsfw" {
		t.Fatalf("resource %+v ok=%v", it, ok)
	}

	f.resetQueue(t)
	f.insertRating(t, apnRating, apnWork, w3UserAlice)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "rating:960200201")
	if !ok || it.Title != apnWorkTitle(apnWork) || it.URL != origin+"/galgame-rating/960200201" ||
		it.WorkID != apnWork || it.CoverImageHash != apnCover || it.ContentLimit != "nsfw" {
		t.Fatalf("rating %+v ok=%v", it, ok)
	}

	f.seedWork(t, apnWorkB, "sfw")
	f.insertGalgame(t, apnWorkB, true)
	f.resetQueue(t)
	f.insertQuiz(t, apnQuiz, w3UserAlice, "One work Q", false)
	f.linkQuiz(t, apnQuiz, apnWork)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "quiz:960200301")
	if !ok || it.Title != "One work Q" || it.URL != origin+"/galgame-quiz/960200301" ||
		it.WorkID != apnWork || it.CoverImageHash != apnCover || it.ContentLimit != "nsfw" {
		t.Fatalf("quiz one %+v ok=%v", it, ok)
	}

	f.resetQueue(t)
	f.insertQuiz(t, apnQuizB, w3UserAlice, "Two works Q", false)
	f.linkQuiz(t, apnQuizB, apnWork)
	f.linkQuiz(t, apnQuizB, apnWorkB)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "quiz:960200302")
	if !ok || it.WorkID != 0 || it.CoverImageHash != "" || it.ContentLimit != "nsfw" || it.Title != "Two works Q" {
		t.Fatalf("quiz two %+v ok=%v", it, ok)
	}

	f.resetQueue(t)
	f.insertQuiz(t, apnQuizC, w3UserAlice, "Zero works Q", false)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "quiz:960200303")
	if !ok || it.WorkID != 0 || it.CoverImageHash != "" || it.ContentLimit != "sfw" || it.Title != "Zero works Q" {
		t.Fatalf("quiz zero %+v ok=%v", it, ok)
	}

	f.resetQueue(t)
	f.insertQuiz(t, apnQuizD, w3UserAlice, "Hidden Q", true)
	f.linkQuiz(t, apnQuizD, apnWork)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "quiz:960200304")
	if !ok || it.Title != "Hidden Q" || it.WorkID != 0 || it.CoverImageHash != "" || it.ContentLimit != "nsfw" {
		t.Fatalf("quiz hide %+v ok=%v", it, ok)
	}

	f.resetQueue(t)
	f.insertQuiz(t, apnQuizE, w3UserAlice, "![](x)", false)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "quiz:960200305")
	if !ok || it.Title != "Galgame 题目" {
		t.Fatalf("quiz empty %+v ok=%v", it, ok)
	}

	f.resetQueue(t)
	f.insertQuiz(t, apnQuizF, w3UserAlice, "visible ||secret|| more", false)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "quiz:960200306")
	if !ok || !strings.Contains(it.Title, "███") || strings.Contains(it.Title, "secret") {
		t.Fatalf("quiz spoiler %+v ok=%v", it, ok)
	}

	f.resetQueue(t)
	f.insertToolset(t, apnTool, w3UserAlice, 0, "My tools")
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "toolset:960200401")
	if !ok || it.Title != "My tools" || it.URL != origin+"/toolset/960200401" ||
		it.WorkID != 0 || it.CoverImageHash != "" || it.ContentLimit != "sfw" {
		t.Fatalf("toolset %+v ok=%v", it, ok)
	}

	f.resetQueue(t)
	f.insertWebsite(t, apnSite, "Site name", "apn.example", "r18")
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "website:960200501")
	if !ok || it.Title != "Site name" || it.URL != origin+"/website/apn.example" ||
		it.WorkID != 0 || it.CoverImageHash != "" || it.ContentLimit != "nsfw" {
		t.Fatalf("website %+v ok=%v", it, ok)
	}
}

func TestAnchorPresentationLiveness(t *testing.T) {
	f := newApnFix(t)
	f.seedWork(t, apnWork, "sfw")
	f.insertGalgame(t, apnWork, false)
	f.insertResource(t, apnRes, apnWork, w3UserAlice)
	f.resetQueue(t)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (2, 'resource:960200101')`)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.writes()) != 0 || f.queueHas(t, 2, "resource:960200101") {
		t.Fatalf("unpublished never-accepted: writes=%d queued=%v", len(f.writes()), f.queueHas(t, 2, "resource:960200101"))
	}
	f.seedSent(t, 2, "resource:960200101", 1, false)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (2, 'resource:960200101')`)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok := f.findItem(2, "resource:960200101")
	keys := apnItemKeys(t, it)
	if !ok || !it.Removed || len(keys) != 4 {
		t.Fatalf("unpublished tombstone %+v keys=%v", it, keys)
	}
	if _, removed, found := f.sent(t, 2, "resource:960200101"); !found || !removed {
		t.Fatalf("accepted tombstone not recorded: removed=%v found=%v", removed, found)
	}
	f.cm.mu.Lock()
	f.cm.puts = nil
	f.cm.mu.Unlock()
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (2, 'resource:960200101')`)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.writes()) != 0 || f.queueHas(t, 2, "resource:960200101") {
		t.Fatalf("an already removed anchor was tombstoned again: %+v", f.writes())
	}

	f.resetQueue(t)
	f.cm.mu.Lock()
	f.cm.puts = nil
	f.cm.mu.Unlock()
	f.insertGalgame(t, apnWorkB, true)
	f.seedWork(t, apnWorkB, "sfw")
	f.insertResource(t, apnRes+1, apnWorkB, w3UserBanned)
	f.resetQueue(t)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (2, 'resource:960200102')`)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.writes()) != 0 {
		t.Fatalf("banned owner sent %+v", f.writes())
	}
	f.seedSent(t, 2, "resource:960200102", 1, false)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (2, 'resource:960200102')`)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "resource:960200102")
	if !ok || !it.Removed || len(apnItemKeys(t, it)) != 4 {
		t.Fatalf("banned tombstone %+v", it)
	}

	f.cm.mu.Lock()
	f.cm.puts = nil
	f.cm.mu.Unlock()
	f.insertResource(t, apnRes+2, apnWorkB, apnGoneUser)
	f.resetQueue(t)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (2, 'resource:960200103')`)
	n0 := len(f.writes())
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.writes()) != n0 {
		t.Fatal("absent resource owner sent")
	}
	f.seedSent(t, 2, "resource:960200103", 1, false)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (2, 'resource:960200103')`)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "resource:960200103")
	if !ok || !it.Removed {
		t.Fatalf("absent resource owner tombstone %+v", it)
	}

	f.cm.mu.Lock()
	f.cm.puts = nil
	f.cm.mu.Unlock()
	f.resetQueue(t)
	f.insertRating(t, apnRating, apnWorkB, apnGoneUser)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "rating:960200201")
	if !ok || it.Removed {
		t.Fatalf("absent rating owner should be live %+v ok=%v", it, ok)
	}

	f.resetQueue(t)
	f.cm.mu.Lock()
	f.cm.puts = nil
	f.cm.mu.Unlock()
	f.insertRating(t, apnRating+1, apnWorkB, w3UserBanned)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := f.findItem(2, "rating:960200202"); ok {
		t.Fatal("banned rating owner sent live")
	}
	f.seedSent(t, 2, "rating:960200202", 1, false)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (2, 'rating:960200202')`)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "rating:960200202")
	if !ok || !it.Removed {
		t.Fatalf("banned rating tombstone %+v", it)
	}

	f.resetQueue(t)
	f.cm.mu.Lock()
	f.cm.puts = nil
	f.cm.mu.Unlock()
	f.insertToolset(t, apnTool, w3UserAlice, 1, "hidden")
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.writes()) != 0 {
		t.Fatal("hidden toolset sent")
	}
	f.seedSent(t, 2, "toolset:960200401", 1, false)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (2, 'toolset:960200401')`)
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(2, "toolset:960200401")
	if !ok || !it.Removed {
		t.Fatalf("hidden toolset tombstone %+v", it)
	}

	f.resetQueue(t)
	f.cm.mu.Lock()
	f.cm.puts = nil
	f.cm.mu.Unlock()
	f.insertGalgame(t, apnWorkC, true)
	f.cat.omitIDs[apnWorkC] = true
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := f.findItem(1, strconv.Itoa(apnWorkC)); ok {
		t.Fatal("omitted galgame sent live")
	}
	f.seedSent(t, 1, strconv.Itoa(apnWorkC), 1, false)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (1, ?)`, strconv.Itoa(apnWorkC))
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok = f.findItem(1, strconv.Itoa(apnWorkC))
	if !ok || !it.Removed {
		t.Fatalf("omitted galgame tombstone %+v", it)
	}
}

func TestAnchorPresentationTransient(t *testing.T) {
	f := newApnFix(t)
	f.seedWork(t, apnWork, "sfw")
	f.insertGalgame(t, apnWork, true)
	f.cat.fail.Store(true)
	n := f.queueCount(t)
	if _, err := f.presenter.RunOnce(context.Background()); err == nil {
		t.Fatal("catalog fail returned nil")
	}
	if f.queueCount(t) != n || len(f.writes()) != 0 {
		t.Fatalf("catalog fail drained queue or sent: q=%d w=%d", f.queueCount(t), len(f.writes()))
	}
	f.cat.fail.Store(false)

	f.insertResource(t, apnRes, apnWork, w3UserAlice)
	f.failOA.Store(true)
	n = f.queueCount(t)
	if _, err := f.presenter.RunOnce(context.Background()); err == nil {
		t.Fatal("users fail returned nil")
	}
	if f.queueCount(t) != n || len(f.writes()) != 0 {
		t.Fatalf("users fail drained: q=%d", f.queueCount(t))
	}
	f.failOA.Store(false)

	f.cm.mu.Lock()
	f.cm.fail503 = true
	f.cm.mu.Unlock()
	n = f.queueCount(t)
	if _, err := f.presenter.RunOnce(context.Background()); err == nil {
		t.Fatal("503 returned nil")
	}
	if f.queueCount(t) != n {
		t.Fatalf("503 drained queue to %d", f.queueCount(t))
	}
}

func TestAnchorPresentationSending(t *testing.T) {
	f := newApnFix(t)
	f.seedWork(t, apnWork, "sfw")
	f.seedWork(t, apnWorkB, "sfw")
	f.insertGalgame(t, apnWork, true)
	f.insertGalgame(t, apnWorkB, true)
	f.cm.mu.Lock()
	f.cm.bisect = true
	f.cm.badKey = apnKey(1, strconv.Itoa(apnWorkB))
	f.cm.mu.Unlock()
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := f.sent(t, 1, strconv.Itoa(apnWork)); !ok {
		t.Fatal("good item not sent")
	}
	if _, _, ok := f.sent(t, 1, strconv.Itoa(apnWorkB)); ok {
		t.Fatal("bad item recorded")
	}
	if f.queueHas(t, 1, strconv.Itoa(apnWorkB)) {
		t.Fatal("bad item stayed queued")
	}

	f.cm.mu.Lock()
	f.cm.bisect = false
	f.cm.short = true
	f.cm.puts = nil
	f.cm.mu.Unlock()
	f.insertGalgame(t, apnWorkC, true)
	f.seedWork(t, apnWorkC, "sfw")
	f.resetQueue(t)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (1, ?), (1, ?)`,
		strconv.Itoa(apnWork), strconv.Itoa(apnWorkC))
	if _, err := f.presenter.RunOnce(context.Background()); err == nil {
		t.Fatal("short results accepted")
	}
	if f.queueCount(t) != 2 {
		t.Fatalf("short results queue = %d", f.queueCount(t))
	}

	f.cm.mu.Lock()
	f.cm.short = false
	f.cm.outcome[apnKey(1, strconv.Itoa(apnWorkC))] = "invalid"
	f.cm.mu.Unlock()
	f.resetQueue(t)
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (1, ?)`, strconv.Itoa(apnWorkC))
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := f.sent(t, 1, strconv.Itoa(apnWorkC)); ok {
		t.Fatal("invalid recorded")
	}
	if f.queueHas(t, 1, strconv.Itoa(apnWorkC)) {
		t.Fatal("invalid stayed queued")
	}

	rev1, _, ok := f.sent(t, 1, strconv.Itoa(apnWork))
	if !ok {
		t.Fatal("missing sent for stale")
	}
	f.cm.mu.Lock()
	f.cm.outcome[apnKey(1, strconv.Itoa(apnWork))] = "stale"
	f.cm.mu.Unlock()
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (1, ?)
		ON CONFLICT (anchor_kind, anchor_id) DO UPDATE SET enqueued = clock_timestamp()`, strconv.Itoa(apnWork))
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	rev2, _, ok := f.sent(t, 1, strconv.Itoa(apnWork))
	if !ok || rev2 != rev1 {
		t.Fatalf("stale moved sent %d → %d", rev1, rev2)
	}

	f.cm.mu.Lock()
	delete(f.cm.outcome, apnKey(1, strconv.Itoa(apnWorkC)))
	f.cm.mu.Unlock()
	f.runSQL(t, `INSERT INTO anchor_presentation_queue (anchor_kind, anchor_id) VALUES (1, ?)`, strconv.Itoa(apnWorkC))
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it, ok := f.findItem(1, strconv.Itoa(apnWorkC))
	rev, removed, found := f.sent(t, 1, strconv.Itoa(apnWorkC))
	if !ok || !found || removed || rev != it.Revision {
		t.Fatalf("accepted sent rev=%d item=%d found=%v", rev, it.Revision, found)
	}
}

func TestAnchorPresentationRevision(t *testing.T) {
	f := newApnFix(t)
	f.seedWork(t, apnWork, "sfw")
	f.insertGalgame(t, apnWork, true)
	before := time.Now()
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	it := f.lastItems(t)[0]
	f.cm.mu.Lock()
	putAt := f.cm.putAt
	f.cm.mu.Unlock()
	if it.Revision < before.UnixMicro() || it.Revision > putAt.UnixMicro() {
		t.Fatalf("revision %d outside [%d, %d]", it.Revision, before.UnixMicro(), putAt.UnixMicro())
	}
}

func TestAnchorPresentationReconcile(t *testing.T) {
	f := newApnFix(t)
	f.seedWork(t, apnWork, "sfw")
	f.insertGalgame(t, apnWork, true)
	f.insertGalgame(t, apnWorkC, true)
	f.cat.omitIDs[apnWorkC] = true
	if _, err := f.presenter.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.resetQueue(t)
	f.cm.mu.Lock()
	view := f.cm.stored[apnKey(1, strconv.Itoa(apnWork))]
	f.cm.list = []communityclient.AnchorPresentationView{view}
	f.cm.mu.Unlock()
	f.presenter.Reconcile(context.Background())
	if f.queueHas(t, 1, strconv.Itoa(apnWork)) || len(f.writes()) != 0 {
		t.Fatalf("identical enqueued, or a never-accepted anchor tombstoned: %+v", f.writes())
	}

	for name, drift := range map[string]func(*communityclient.AnchorPresentationView){
		"cover":   func(v *communityclient.AnchorPresentationView) { v.CoverImageHash = nil },
		"work_id": func(v *communityclient.AnchorPresentationView) { v.WorkID = nil },
		"url":     func(v *communityclient.AnchorPresentationView) { v.URL += "?x" },
	} {
		f.resetQueue(t)
		f.cm.mu.Lock()
		view = f.cm.stored[apnKey(1, strconv.Itoa(apnWork))]
		drift(&view)
		f.cm.list = []communityclient.AnchorPresentationView{view}
		f.cm.mu.Unlock()
		f.presenter.Reconcile(context.Background())
		if !f.queueHas(t, 1, strconv.Itoa(apnWork)) {
			t.Fatalf("%s drift not enqueued", name)
		}
	}

	f.cm.mu.Lock()
	view = f.cm.stored[apnKey(1, strconv.Itoa(apnWork))]
	view.Title = "drifted"
	f.cm.list = []communityclient.AnchorPresentationView{view}
	f.cm.mu.Unlock()
	f.presenter.Reconcile(context.Background())
	if !f.queueHas(t, 1, strconv.Itoa(apnWork)) {
		t.Fatal("title drift not enqueued")
	}

	f.resetQueue(t)
	f.cm.mu.Lock()
	view = f.cm.stored[apnKey(1, strconv.Itoa(apnWork))]
	view.Title = " " + view.Title
	f.cm.list = []communityclient.AnchorPresentationView{view}
	f.cm.mu.Unlock()
	f.presenter.Reconcile(context.Background())
	if !f.queueHas(t, 1, strconv.Itoa(apnWork)) {
		t.Fatal("a stored title that differs only in whitespace was not enqueued")
	}

	f.resetQueue(t)
	f.cm.mu.Lock()
	view = f.cm.stored[apnKey(1, strconv.Itoa(apnWork))]
	view.ContentLimit = "nsfw"
	f.cm.list = []communityclient.AnchorPresentationView{view}
	f.cm.mu.Unlock()
	f.presenter.Reconcile(context.Background())
	if !f.queueHas(t, 1, strconv.Itoa(apnWork)) {
		t.Fatal("content_limit drift not enqueued")
	}

	f.resetQueue(t)
	f.cm.mu.Lock()
	view = f.cm.stored[apnKey(1, strconv.Itoa(apnWork))]
	view.Removed = true
	f.cm.list = []communityclient.AnchorPresentationView{view}
	f.cm.mu.Unlock()
	f.presenter.Reconcile(context.Background())
	if !f.queueHas(t, 1, strconv.Itoa(apnWork)) {
		t.Fatal("stored tombstone of live not enqueued")
	}

	f.seedWork(t, apnWorkB, "sfw")
	f.insertGalgame(t, apnWorkB, true)
	f.cat.omitIDs[apnWorkB] = true
	f.resetQueue(t)
	const hidden, orphan = "960200002", "960299999"
	f.cm.mu.Lock()
	drifted := f.cm.stored[apnKey(1, strconv.Itoa(apnWork))]
	drifted.ContentLimit = "nsfw"
	f.cm.list = []communityclient.AnchorPresentationView{
		drifted,
		{AnchorKind: 1, AnchorID: hidden, Removed: false, Revision: 1},
		{AnchorKind: 1, AnchorID: orphan, Removed: false, Revision: 1},
	}
	f.cm.puts = nil
	f.cm.mu.Unlock()
	f.presenter.Reconcile(context.Background())
	var enqueued time.Time
	if err := f.db.Raw(`SELECT enqueued FROM anchor_presentation_queue WHERE anchor_kind = 1 AND anchor_id = ?`,
		strconv.Itoa(apnWork)).Row().Scan(&enqueued); err != nil {
		t.Fatalf("drifted row was not enqueued: %v", err)
	}
	tombs := map[string]communityclient.AnchorPresentationItem{}
	for _, it := range f.allItems() {
		if it.Removed {
			tombs[it.AnchorID] = it
		}
	}
	laterRead := enqueued.UnixMicro()
	if tb, ok := tombs[hidden]; !ok || tb.Revision >= laterRead {
		t.Fatalf("hidden-row tombstone revision %d, want before %d", tb.Revision, laterRead)
	}
	f.cm.mu.Lock()
	listed := f.cm.listedAt
	f.cm.mu.Unlock()
	if tb, ok := tombs[orphan]; !ok || tb.Revision >= listed.UnixMicro() {
		t.Fatalf("orphan tombstone revision %d, want before %d", tb.Revision, listed.UnixMicro())
	}
	_, removed, ok := f.sent(t, 1, hidden)
	if !ok || !removed {
		t.Fatalf("accepted reconcile tombstone sent removed=%v found=%v", removed, ok)
	}

	f.cat.fail.Store(true)
	f.resetQueue(t)
	f.cm.mu.Lock()
	f.cm.puts = nil
	f.cm.list = []communityclient.AnchorPresentationView{
		{AnchorKind: 1, AnchorID: orphan, Removed: false, Revision: 1},
	}
	f.cm.mu.Unlock()
	f.presenter.Reconcile(context.Background())
	if len(f.writes()) != 0 {
		t.Fatalf("catalog fail during scan sent tombstones: %+v", f.writes())
	}
}
