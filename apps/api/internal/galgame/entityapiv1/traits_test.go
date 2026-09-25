package entityapiv1

import (
	"encoding/json"
	"slices"
	"testing"

	"kun-galgame-api/internal/galgame/client"
)

const testTraitsJSON = `[
	{"id":"1","display_name":"Hair","name_zh":"毛发","root_order":1,"character_count":900},
	{"id":"3","display_name":"Body","name_zh":"身体","root_order":3,"character_count":800},
	{"id":"4","display_name":"Clothes","name_zh":"服装","root_order":4,"character_count":700},
	{"id":"625","display_name":"Subject of (Sexual)","name_zh":"被动(性)","group_id":null,"root_order":11,"is_sexual":true,"character_count":50},
	{"id":"21","display_name":"Hairstyle","name_zh":"发型","group_id":"1","parents":[{"id":"1"}],"character_count":500},
	{"id":"20","display_name":"Kemonomimi","name_zh":"兽耳","group_id":"1","parents":[{"id":"21"}],"is_searchable":true,"character_count":40,"aliases":["Animal Ears"]},
	{"id":"31","display_name":"Features","name_zh":"特征","group_id":"3","parents":[{"id":"3"}],"character_count":300},
	{"id":"30","display_name":"Kemonomimi","name_zh":"兽耳","group_id":"3","parents":[{"id":"31"}],"is_searchable":true,"character_count":90},
	{"id":"32","display_name":"Kemonomimi Headband","name_zh":"兽耳发箍","group_id":"3","parents":[{"id":"31"}],"is_searchable":true,"character_count":95},
	{"id":"40","display_name":"Footwear","name_zh":"鞋类","group_id":"4","parents":[{"id":"4"}],"character_count":400},
	{"id":"50","display_name":"Latex","name_zh":"乳胶","group_id":"4","parents":[{"id":"4"}],"is_searchable":true,"character_count":30},
	{"id":"781","display_name":"Boots","name_zh":"靴子","group_id":"4","parents":[{"id":"40"}],"is_searchable":true,"character_count":200,"description":"This character wears boots."},
	{"id":"782","display_name":"Knee-high Boots","name_zh":"及膝靴","group_id":"4","parents":[{"id":"781"}],"is_searchable":true,"character_count":60},
	{"id":"783","display_name":"Latex Knee-high Boots","name_zh":"乳胶过膝靴","group_id":"4","parents":[{"id":"782"},{"id":"50"}],"is_searchable":true,"character_count":5},
	{"id":"41","display_name":"Barefoot (Sexual)","name_zh":"赤足(性)","group_id":"4","parents":[{"id":"40"}],"is_sexual":true,"is_searchable":true,"character_count":7},
	{"id":"626","display_name":"Groped","name_zh":"被摸","group_id":"625","parents":[{"id":"31"},{"id":"625"}],"is_sexual":true,"is_searchable":true,"character_count":3},
	{"id":"99","display_name":"Orphan","parents":[{"id":"12345"}]}
]`

var testSFWCounts = map[int64]int{4: 690, 40: 393}

func testTraits(t *testing.T) []client.CatalogTrait {
	t.Helper()
	var wire []client.CatalogTrait
	if err := json.Unmarshal([]byte(testTraitsJSON), &wire); err != nil {
		t.Fatal(err)
	}
	for i := range wire {
		wire[i].SFWCharacterCount = wire[i].CharacterCount
		if n, ok := testSFWCounts[wire[i].ID]; ok {
			wire[i].SFWCharacterCount = n
		}
		if wire[i].Sexual {
			wire[i].SFWCharacterCount = 0
		}
	}
	return wire
}

func testVocab(t *testing.T) traitVocab {
	t.Helper()
	return newTraitVocab(testTraits(t))
}

func summaryIDs(rows []TraitSummary) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, string(r.ID))
	}
	return out
}

func TestTraitVocabRootsAndGroups(t *testing.T) {
	v := testVocab(t)
	if want := []int{1, 3, 4, 625, 99}; !slices.Equal(v.roots, want) {
		t.Fatalf("roots %v, want %v", v.roots, want)
	}
	s := v.summary(v.byID[783], true)
	if s.TraitGroup.Localized["zh-Hans"].Value != "服装" {
		t.Fatalf("group of 783: %+v", s.TraitGroup)
	}
	if got := summaryIDs([]TraitSummary{{ID: s.Parents[0].ID}, {ID: s.Parents[1].ID}}); !slices.Equal(got, []string{"782", "50"}) {
		t.Fatalf("parents %v", got)
	}
	if root := v.summary(v.byID[1], false); root.TraitGroup.DisplayName != "Hair" || len(root.Parents) != 0 {
		t.Fatalf("a root is its own group: %+v", root)
	}
	groped := v.summary(v.byID[626], true)
	if groped.TraitGroupID != "625" || groped.Parents[0].TraitGroupID != "3" || groped.Parents[1].TraitGroupID != "625" {
		t.Fatalf("626 sits in catalog's group 625 though its first root is Body: %+v", groped)
	}
}

func TestTraitVocabSubtraitsTwoLevels(t *testing.T) {
	v := testVocab(t)
	got := summaryIDs(v.subtraits(4, true))
	if want := []string{"40", "50", "781", "41", "783"}; !slices.Equal(got, want) {
		t.Fatalf("subtraits of Clothes %v, want %v", got, want)
	}
	got = summaryIDs(v.subtraits(4, false))
	if want := []string{"40", "50", "781", "783"}; !slices.Equal(got, want) {
		t.Fatalf("SFW subtraits of Clothes %v, want %v", got, want)
	}
	if n := v.summary(v.byID[40], false).ChildCount; n != 1 {
		t.Fatalf("SFW child_count of Footwear %d, want 1", n)
	}
}

func TestTraitVocabSearch(t *testing.T) {
	v := testVocab(t)
	ids := func(q string, nsfw bool) []int {
		var out []int
		for _, n := range v.search(q, nsfw) {
			out = append(out, n.id)
		}
		return out
	}
	if got, want := ids("兽耳", false), []int{30, 20, 32}; !slices.Equal(got, want) {
		t.Fatalf("兽耳 %v, want %v", got, want)
	}
	if got, want := ids("animal ears", false), []int{20}; !slices.Equal(got, want) {
		t.Fatalf("alias %v, want %v", got, want)
	}
	if got := ids("(性)", false); len(got) != 0 {
		t.Fatalf("adult traits leak into an SFW search: %v", got)
	}
	if got, want := ids("(性)", true), []int{625, 41}; !slices.Equal(got, want) {
		t.Fatalf("NSFW search %v, want %v", got, want)
	}
}
