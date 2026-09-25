package entityapiv1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/errors"

	"github.com/gofiber/fiber/v3"
)

type traitFakeCatalog struct {
	Catalog
	traits  []client.CatalogTrait
	rows    []client.CatalogCharacterRow
	queries []client.CatalogCharacterQuery
}

func (f *traitFakeCatalog) CatalogTraitVocabulary(context.Context) ([]client.CatalogTrait, *errors.AppError) {
	return f.traits, nil
}

func (f *traitFakeCatalog) CatalogCharacterList(_ context.Context, in client.CatalogCharacterQuery) (*client.CatalogCharacterPage, *errors.AppError) {
	f.queries = append(f.queries, in)
	return &client.CatalogCharacterPage{Items: f.rows, Total: 1234}, nil
}

func newTraitAPI(t *testing.T) (*fiber.App, *traitFakeCatalog) {
	t.Helper()
	var rows []client.CatalogCharacterRow
	raw := fmt.Sprintf(`[{"id":"23","display_name":"能美クドリャフカ","latin":"Noumi Kudryavka",
		"localized":{"zh-Hans":{"value":"能美库特莉亚芙卡","is_machine":false}},
		"image":{"url":"https://x.example/a.webp","hash":"%064x","width":256,"height":300,"thumbhash":"pUgK","sexual":"explicit"},
		"work_count":4,"matched_trait_ids":["782","41"]}]`, 23)
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		t.Fatal(err)
	}
	cat := &traitFakeCatalog{traits: testTraits(t), rows: rows}
	app := fiber.New(fiber.Config{ErrorHandler: v1.WriteFiberError})
	v1.Setup(app, v1.Deps{}, Register(New(cat, nil, "https://image.test.example")))
	return app, cat
}

func getJSON(t *testing.T, app *fiber.App, path string, wantStatus int) map[string]any {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, path, nil), fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s: status %d, want %d: %v", path, resp.StatusCode, wantStatus, body)
	}
	return body
}

func itemIDs(body map[string]any) []string {
	var out []string
	for _, it := range body["items"].([]any) {
		out = append(out, it.(map[string]any)["id"].(string))
	}
	return out
}

func TestListCharactersForwardsFilters(t *testing.T) {
	app, cat := newTraitAPI(t)
	body := getJSON(t, app, "/api/v1/characters?trait_ids=781,50&trait_match=any&genders=female,other&sort=id_desc&page=3&limit=24&include_nsfw=true", http.StatusOK)
	got := cat.queries[0]
	if !slices.Equal(got.TraitIDs, []int{781, 50}) || !got.MatchAny || !slices.Equal(got.Genders, []string{"female", "other"}) ||
		got.Sort != "newest" || got.Page != 3 || got.Limit != 24 || !got.NSFW {
		t.Fatalf("upstream query %+v", got)
	}
	if body["total"].(float64) != 1234 || body["total_relation"] != "eq" {
		t.Fatalf("total %v %v", body["total"], body["total_relation"])
	}
	item := body["items"].([]any)[0].(map[string]any)
	if item["catalog_work_count"].(float64) != 4 || item["image"].(map[string]any)["sexual"] != "explicit" {
		t.Fatalf("item %v", item)
	}
	var matched []string
	for _, m := range item["matched_traits"].([]any) {
		matched = append(matched, m.(map[string]any)["id"].(string))
	}
	if !slices.Equal(matched, []string{"782", "41"}) {
		t.Fatalf("matched_traits %v", matched)
	}
}

func TestListCharactersGates(t *testing.T) {
	app, cat := newTraitAPI(t)
	body := getJSON(t, app, "/api/v1/characters?trait_ids=41", http.StatusBadRequest)
	if body["code"] != "INVALID_PARAMETER" {
		t.Fatalf("adult trait without include_nsfw: %v", body)
	}
	getJSON(t, app, "/api/v1/characters?sort=relevance_desc", http.StatusBadRequest)
	if len(cat.queries) != 0 {
		t.Fatalf("a refused request reached catalog: %+v", cat.queries)
	}

	body = getJSON(t, app, "/api/v1/characters?q=kud", http.StatusOK)
	if q := cat.queries[0]; q.Q != "kud" || q.Sort != "relevance" || q.TraitIDs != nil {
		t.Fatalf("upstream query %+v", q)
	}
	if m := body["items"].([]any)[0].(map[string]any)["matched_traits"].([]any); len(m) != 0 {
		t.Fatalf("matched_traits without trait_ids: %v", m)
	}

	body = getJSON(t, app, "/api/v1/characters?trait_ids=781", http.StatusOK)
	if q := cat.queries[1]; q.Sort != "popularity" || q.Page != 1 || q.Limit != 24 {
		t.Fatalf("defaults %+v", q)
	}
	var matched []string
	for _, m := range body["items"].([]any)[0].(map[string]any)["matched_traits"].([]any) {
		matched = append(matched, m.(map[string]any)["id"].(string))
	}
	if !slices.Equal(matched, []string{"782"}) {
		t.Fatalf("an SFW reader sees adult matched traits: %v", matched)
	}
}

func TestTraitFaces(t *testing.T) {
	app, _ := newTraitAPI(t)

	if got := itemIDs(getJSON(t, app, "/api/v1/traits", http.StatusOK)); !slices.Equal(got, []string{"1", "3", "4", "99"}) {
		t.Fatalf("SFW roots %v", got)
	}
	if got := itemIDs(getJSON(t, app, "/api/v1/traits?include_nsfw=true", http.StatusOK)); !slices.Equal(got, []string{"1", "3", "4", "625", "99"}) {
		t.Fatalf("NSFW roots %v", got)
	}
	if got := itemIDs(getJSON(t, app, "/api/v1/traits?parent_id=40", http.StatusOK)); !slices.Equal(got, []string{"781"}) {
		t.Fatalf("SFW children of Footwear %v", got)
	}
	if got := itemIDs(getJSON(t, app, "/api/v1/traits?ids=783,41,20", http.StatusOK)); !slices.Equal(got, []string{"783", "20"}) {
		t.Fatalf("SFW ids %v", got)
	}
	getJSON(t, app, "/api/v1/traits?q=x&ids=1", http.StatusBadRequest)

	hit := getJSON(t, app, "/api/v1/traits?q=%E5%85%BD%E8%80%B3", http.StatusOK)["items"].([]any)[0].(map[string]any)
	if hit["id"] != "30" || hit["trait_group_id"] != "3" || hit["parents"].([]any)[0].(map[string]any)["id"] != "31" {
		t.Fatalf("兽耳 hit %v", hit)
	}

	getJSON(t, app, "/api/v1/traits/41", http.StatusNotFound)
	getJSON(t, app, "/api/v1/traits/41?include_nsfw=true", http.StatusOK)

	footwear := getJSON(t, app, "/api/v1/traits/40", http.StatusOK)
	if footwear["character_count"].(float64) != 393 || footwear["child_count"].(float64) != 1 || footwear["is_searchable"] != false {
		t.Fatalf("SFW footwear %v", footwear)
	}
	footwear = getJSON(t, app, "/api/v1/traits/40?include_nsfw=true", http.StatusOK)
	if footwear["character_count"].(float64) != 400 || footwear["child_count"].(float64) != 2 {
		t.Fatalf("NSFW footwear %v", footwear)
	}

	boots := getJSON(t, app, "/api/v1/traits/781", http.StatusOK)
	if boots["description"] != "This character wears boots." || boots["trait_group"].(map[string]any)["display_name"] != "Clothes" {
		t.Fatalf("boots %v", boots)
	}
	var sub []string
	for _, s := range boots["subtraits"].([]any) {
		sub = append(sub, s.(map[string]any)["id"].(string))
	}
	if !slices.Equal(sub, []string{"782", "783"}) {
		t.Fatalf("subtraits of boots %v", sub)
	}
}
