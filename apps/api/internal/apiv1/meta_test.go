package apiv1

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"kun-galgame-api/pkg/problem"
)

func TestMetaEndpointsMatchRegistry(t *testing.T) {
	app, _ := newTestAPI(t, Deps{})

	codes := do(t, app, http.MethodGet, "/api/v1/problems", "", nil)
	if codes.StatusCode != http.StatusOK {
		t.Fatalf("problems status %d", codes.StatusCode)
	}
	assertCacheControl(t, codes)
	var codeList struct {
		Object string        `json:"object"`
		Items  []problemType `json:"items"`
	}
	if err := json.Unmarshal(readBody(t, codes), &codeList); err != nil {
		t.Fatal(err)
	}
	if codeList.Object != "list" {
		t.Errorf("object %s", codeList.Object)
	}
	wantCodes := ProblemTypes()
	if len(codeList.Items) != len(problem.Codes) || len(codeList.Items) != len(wantCodes) {
		t.Fatalf("items %d, registry %d", len(codeList.Items), len(problem.Codes))
	}
	for i := range wantCodes {
		if codeList.Items[i] != wantCodes[i] {
			t.Errorf("item %d %+v != %+v", i, codeList.Items[i], wantCodes[i])
		}
	}
	for i := 1; i < len(codeList.Items); i++ {
		a, b := codeList.Items[i-1], codeList.Items[i]
		if domainRank(a.Domain) > domainRank(b.Domain) || (a.Domain == b.Domain && a.Code > b.Code) {
			t.Errorf("unsorted %s/%s then %s/%s", a.Domain, a.Code, b.Domain, b.Code)
		}
		if a.Object != "problem_type" {
			t.Errorf("object %s", a.Object)
		}
	}

	reasons := do(t, app, http.MethodGet, "/api/v1/problems/reasons", "", nil)
	if reasons.StatusCode != http.StatusOK {
		t.Fatalf("reasons status %d", reasons.StatusCode)
	}
	var reasonList struct {
		Object string          `json:"object"`
		Items  []problemReason `json:"items"`
	}
	if err := json.Unmarshal(readBody(t, reasons), &reasonList); err != nil {
		t.Fatal(err)
	}
	wantReasons := ProblemReasons()
	if len(reasonList.Items) != len(problem.Reasons) || len(reasonList.Items) != len(wantReasons) {
		t.Fatalf("reason items %d, registry %d", len(reasonList.Items), len(problem.Reasons))
	}
	for i := range wantReasons {
		got, want := reasonList.Items[i], wantReasons[i]
		if got.Reason != want.Reason || got.Title != want.Title || got.Description != want.Description || got.Object != "problem_reason" {
			t.Errorf("reason %d %+v != %+v", i, got, want)
		}
		if !slices.Equal(got.ParamNames, want.ParamNames) {
			t.Errorf("reason %s param_names %v != %v", got.Reason, got.ParamNames, want.ParamNames)
		}
	}
	for i := 1; i < len(reasonList.Items); i++ {
		if reasonList.Items[i-1].Reason > reasonList.Items[i].Reason {
			t.Errorf("reasons unsorted %s then %s", reasonList.Items[i-1].Reason, reasonList.Items[i].Reason)
		}
	}
}

// The enum was a hand-written tag while domains live in problem.DomainOrder;
// adding the `me` domain made GET /problems answer a value its own schema forbids.
func TestProblemTypeDomainEnumCoversRegistry(t *testing.T) {
	f, ok := reflect.TypeOf(problemType{}).FieldByName("Domain")
	if !ok {
		t.Fatal("problemType has no Domain field")
	}
	enum := strings.Split(f.Tag.Get("enum"), ",")
	for _, d := range problem.DomainOrder {
		if !slices.Contains(enum, string(d)) {
			t.Errorf("domain %q is registered but missing from problemType.domain enum %v", d, enum)
		}
	}
}
