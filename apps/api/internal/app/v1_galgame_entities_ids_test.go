package app

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestV1TagsIDsResolve(t *testing.T) {
	f := newGEFix(t)
	resp, body := f.get(t, "/api/v1/tags?ids=5101,5102,5999,5101", "/tags")
	geStatus(t, resp, body, http.StatusOK, "")
	ids := geItemIDs(body)
	geWant(t, "request order, skip adult and missing, dedupe", ids, "5101")
	if body["total"] != float64(1) {
		t.Fatalf("total %v includes the adult or missing id", body["total"])
	}

	resp, body = f.get(t, "/api/v1/tags?ids=5101,5102&include_nsfw=true", "/tags")
	geStatus(t, resp, body, http.StatusOK, "")
	geWant(t, "adult with include_nsfw", geItemIDs(body), "5101", "5102")
}

func TestV1TagsIDsRejectsBadAndTooMany(t *testing.T) {
	f := newGEFix(t)
	resp, body := f.get(t, "/api/v1/tags?ids=abc", "/tags")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	resp, body = f.get(t, "/api/v1/tags?ids=0", "/tags")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	resp, body = f.get(t, "/api/v1/tags?q=x&ids=5101", "/tags")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	if errs := body["errors"].([]any); errs[0].(map[string]any)["reason"] != "INCONSISTENT_WITH" {
		t.Fatalf("q+ids errors %+v", errs)
	}

	ids := make([]string, 101)
	for i := range ids {
		ids[i] = strconv.Itoa(5200 + i)
	}
	resp, body = f.get(t, "/api/v1/tags?ids="+strings.Join(ids, ","), "/tags")
	geStatus(t, resp, body, http.StatusBadRequest, "INVALID_PARAMETER")
	if errs := body["errors"].([]any); errs[0].(map[string]any)["reason"] != "TOO_MANY_ITEMS" {
		t.Fatalf("101 ids errors %+v", errs)
	}
}

func TestV1CompaniesIDsResolve(t *testing.T) {
	f := newGEFix(t)
	resp, body := f.get(t, "/api/v1/companies?ids=6101,6999,6101", "/companies")
	geStatus(t, resp, body, http.StatusOK, "")
	geWant(t, "companies ids", geItemIDs(body), "6101")
	resp, body = f.get(t, "/api/v1/companies?ids=6101&company_kind=doujin_circle", "/companies")
	geStatus(t, resp, body, http.StatusOK, "")
	if len(geItemIDs(body)) != 0 {
		t.Fatalf("kind filter should drop the brand: %v", geItemIDs(body))
	}
}
