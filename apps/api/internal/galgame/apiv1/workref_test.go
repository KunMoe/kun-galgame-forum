package apiv1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	v1 "kun-galgame-api/internal/apiv1"
	"kun-galgame-api/internal/apiv1/gates"
	"kun-galgame-api/internal/apiv1/repr"
	galgameapiv1 "kun-galgame-api/internal/galgame/apiv1"
	"kun-galgame-api/internal/galgame/client"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

const (
	portraitHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	bannerHash   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func item(t *testing.T, raw string) *client.CatalogWorkListItem {
	t.Helper()
	var it client.CatalogWorkListItem
	if err := json.Unmarshal([]byte(raw), &it); err != nil {
		t.Fatal(err)
	}
	return &it
}

func TestWorkRefOf(t *testing.T) {
	it := item(t, `{
		"id": 19658, "display_name": "紅殻のパンドラ", "latin": "Koukaku no Pandora", "content_rating": "all", "content_limit": "nsfw",
		"localized": {"zh-Hans": {"value": "红壳的潘多拉", "machine": false}, "en": {"value": "Pandora", "is_machine": true}, "ko": {"value": ""}},
		"claim": {"site": "kungal", "state": "live", "content_limit": "nsfw"},
		"cover_slots": {
			"portrait": {"url": "https://img.example/aa/aa/`+portraitHash+`.webp", "width": 600, "height": 850, "thumbhash": "AbC+"},
			"banner": {"url": "https://img.example/bb/bb/`+bannerHash+`_mini.webp", "width": 1280, "height": 720}
		}
	}`)
	ref := galgameapiv1.WorkRefOf(context.Background(), it, "https://image.test.example")
	if ref.Object != "work" || ref.ID != "19658" || ref.DisplayName != "紅殻のパンドラ" || ref.Latin == nil || *ref.Latin != "Koukaku no Pandora" {
		t.Fatalf("identity %+v", ref)
	}
	if got := ref.Localized["zh-Hans"]; got.Value != "红壳的潘多拉" || got.IsMachine {
		t.Errorf("zh-Hans %+v", got)
	}
	if got := ref.Localized["en"]; !got.IsMachine {
		t.Errorf("en must keep its machine flag: %+v", got)
	}
	if _, has := ref.Localized["ko"]; has {
		t.Error("an empty locale slot is not a name")
	}
	if ref.Cover == nil || ref.Cover.Hash != portraitHash || strings.Contains(ref.Cover.URL, "_mini") ||
		ref.Cover.Width == nil || *ref.Cover.Width != 600 || ref.Cover.Thumbhash == nil {
		t.Errorf("cover must be the portrait original: %+v", ref.Cover)
	}
	if !ref.IsNSFW {
		t.Error("is_nsfw follows catalog's content_limit")
	}

	sfwClaimOnAdultRating := item(t, `{"id": 7, "display_name": "x", "content_rating": "r18", "content_limit": "sfw",
		"claim": {"site": "kungal", "state": "live", "content_limit": "sfw"}}`)
	if ref := galgameapiv1.WorkRefOf(context.Background(), sfwClaimOnAdultRating, "https://image.test.example"); ref.IsNSFW || ref.Cover != nil {
		t.Errorf("the editorial axis wins over the age rating, and no portrait means no cover: %+v", ref)
	}
}

func TestWorkRefPassesTheContractGates(t *testing.T) {
	probe := func(api huma.API) {
		huma.Register(api, v1.Public(huma.Operation{
			OperationID: "probeWorkRef",
			Method:      http.MethodGet,
			Path:        "/probe-works",
			Summary:     "Probe",
			Description: "Returns a work reference.",
		}), func(context.Context, *struct{}) (*struct{ Body repr.WorkRef }, error) { return nil, nil })
	}
	doc := v1.Setup(fiber.New(), v1.Deps{}, probe).OpenAPI()
	if errs := gates.CheckAll(doc); len(errs) > 0 {
		t.Fatalf("gates: %v", errs)
	}
}
