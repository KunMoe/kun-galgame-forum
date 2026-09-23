package client

import (
	"context"
	"encoding/json"
	"testing"
)

// Wave 212's second half reshapes the works intro blocks: the list brief's
// object of four product-locale slots becomes the [{lang, intro, …}] array the
// rest of the catalog already sends, and the detail face's "intro" key is
// renamed "intros". Both faces must read the same on either side of that
// deploy — the forum and catalog ship independently, so whichever goes first
// must not blank every game's introduction.
func TestCatalogIntros_ListBriefReadsBothShapes(t *testing.T) {
	const object = `{"id":9,"display_name":"Kun","intros":{` +
		`"ja-jp":{"intro":"日本語","source":"vndb"},` +
		`"zh-cn":{"intro":"简体","machine":true},` +
		`"zh-tw":{"intro":""}}}`
	const array = `{"id":9,"display_name":"Kun","intros":[` +
		`{"lang":"ja","intro":"日本語","source":"vndb"},` +
		`{"lang":"zh-Hans","intro":"简体","machine":true}]}`

	for _, tc := range []struct{ shape, body string }{
		{"the retiring object of product-locale slots", object},
		{"the array every other catalog face sends", array},
	} {
		var it CatalogWorkListItem
		if err := json.Unmarshal([]byte(tc.body), &it); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.shape, err)
		}
		b := CatalogItemToDetailBrief(context.Background(), &it)
		// Both shapes have to land on the same canonical tags, and the site
		// reads Chinese first, so zh-Hans leads whichever way the wire came.
		if len(b.Intros) != 2 {
			t.Fatalf("%s: intros = %+v, want exactly the two languages present", tc.shape, b.Intros)
		}
		if b.Intros[0].Lang != "zh-Hans" || b.Intros[0].Intro != "简体" {
			t.Errorf("%s: first = %+v, want zh-Hans/简体", tc.shape, b.Intros[0])
		}
		if b.Intros[1].Lang != "ja" || b.Intros[1].Intro != "日本語" {
			t.Errorf("%s: second = %+v, want ja/日本語", tc.shape, b.Intros[1])
		}
	}
}

func TestCatalogIntros_ListBriefKeepsTheMachineFlagAcrossShapes(t *testing.T) {
	for _, tc := range []struct{ shape, body string }{
		{"object", `{"intros":{"zh-cn":{"intro":"简体","machine":true}}}`},
		{"array", `{"intros":[{"lang":"zh-Hans","intro":"简体","machine":true}]}`},
	} {
		var it CatalogWorkListItem
		if err := json.Unmarshal([]byte(tc.body), &it); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.shape, err)
		}
		if len(it.Intros) != 1 || !it.Intros[0].Machine {
			t.Errorf("%s: Intros = %+v, want the machine flag carried through the "+
				"shape change — it is the only thing separating a translation from the "+
				"publisher's own text", tc.shape, it.Intros)
		}
	}
}

func TestCatalogIntros_ListBriefSurvivesAnAbsentBlock(t *testing.T) {
	for _, body := range []string{`{"id":9}`, `{"id":9,"intros":null}`} {
		var it CatalogWorkListItem
		if err := json.Unmarshal([]byte(body), &it); err != nil {
			t.Fatalf("%s: unmarshal: %v", body, err)
		}
		if got := OrderIntros(it.Intros); len(got) != 0 {
			t.Errorf("%s: intros = %+v, want none", body, got)
		}
	}
}

func TestCatalogIntros_DetailReadsBothKeys(t *testing.T) {
	const row = `{"lang":"ja","intro":"日本語"},{"lang":"zh-Hans","intro":"简体","machine":true}`

	for _, tc := range []struct{ key, body string }{
		{"intro", `{"id":9,"intro":[` + row + `]}`},
		{"intros", `{"id":9,"intros":[` + row + `]}`},
	} {
		var d CatalogWorkDetail
		if err := json.Unmarshal([]byte(tc.body), &d); err != nil {
			t.Fatalf("%s: unmarshal: %v", tc.key, err)
		}
		rows := OrderIntros(d.IntroRows())
		if len(rows) != 2 || rows[0].Lang != "zh-Hans" || rows[1].Lang != "ja" {
			t.Fatalf("%s: rows = %+v, want zh-Hans then ja", tc.key, rows)
		}
		if rows[0].Intro != "简体" || rows[1].Intro != "日本語" {
			t.Errorf("%s: text = %q / %q, want 简体 / 日本語", tc.key, rows[0].Intro, rows[1].Intro)
		}
		if !rows[0].Machine || rows[1].Machine {
			t.Errorf("%s: machine = %v / %v, want only the Chinese row flagged",
				tc.key, rows[0].Machine, rows[1].Machine)
		}
	}
}

// The rename ships as a dual-emit window, so a payload carrying both keys must
// resolve to one answer rather than concatenating or picking by struct order.
func TestCatalogIntros_DetailPrefersTheNewKeyWhenBothArrive(t *testing.T) {
	var d CatalogWorkDetail
	body := `{"intro":[{"lang":"zh-Hans","intro":"旧"}],"intros":[{"lang":"zh-Hans","intro":"新"}]}`
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	rows := OrderIntros(d.IntroRows())
	if len(rows) != 1 || rows[0].Intro != "新" {
		t.Errorf("rows = %+v, want 新 — during dual emit the renamed key is the live one", rows)
	}
}
