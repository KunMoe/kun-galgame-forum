package problem

import (
	"strings"
	"testing"
)

var requiredCodes = []struct {
	code   string
	domain Domain
	status int
}{
	{CodeMalformedBody, DomainPlatform, 400},
	{CodeInvalidParameter, DomainPlatform, 400},
	{CodeUnknownEnumValue, DomainPlatform, 400},
	{CodeLimitTooLarge, DomainPlatform, 400},
	{CodeInvalidCursor, DomainPlatform, 400},
	{CodeUnknownSort, DomainPlatform, 400},
	{CodeMissingCredential, DomainPlatform, 401},
	{CodeInvalidCredential, DomainPlatform, 401},
	{CodeScopeRequired, DomainPlatform, 403},
	{CodeAccountBanned, DomainKungal, 403},
	{CodeNotFound, DomainPlatform, 404},
	{CodeMethodNotAllowed, DomainPlatform, 405},
	{CodeIdempotencyKeyReused, DomainPlatform, 409},
	{CodeIdempotencyRequestInProgress, DomainKungal, 409},
	{CodeUnsupportedMediaType, DomainPlatform, 415},
	{CodeValidationFailed, DomainPlatform, 422},
	{CodeInternalError, DomainPlatform, 500},
	{CodeServiceUnavailable, DomainPlatform, 503},
	{CodePermissionRequired, DomainModeration, 403},
	{CodeContentRejected, DomainKungal, 422},
	{CodeTopicDailyLimitReached, DomainKungal, 429},
	{CodeDraftLimitReached, DomainKungal, 409},
	{CodeLotteryClosed, DomainKungal, 409},
	{CodeLotteryIneligible, DomainKungal, 403},
	{CodeLotteryCreatorIneligible, DomainKungal, 403},
	{CodeLotteryDrawn, DomainKungal, 409},
	{CodeRedemptionCodeForfeited, DomainKungal, 409},
	{CodeMoemoepointInsufficient, DomainKungal, 403},
	{CodeSelfLikeForbidden, DomainKungal, 403},
	{CodeSelfUpvoteForbidden, DomainKungal, 403},
	{CodeRateLimited, DomainPlatform, 429},
	{CodeQuizAnswerRequired, DomainKungal, 403},
	{CodeInvalidStateTransition, DomainMe, 409},
	{CodePollClosed, DomainKungal, 409},
	{CodeVoteAlreadyCast, DomainKungal, 409},
}

func TestRegistryClosedAndExact(t *testing.T) {
	if len(Codes) != len(requiredCodes) {
		t.Fatalf("Codes has %d entries, want %d", len(Codes), len(requiredCodes))
	}
	seen := map[string]string{}
	statusOf := map[string]int{}
	uris := map[string]string{}
	for _, d := range Codes {
		if !NamePattern.MatchString(d.Code) {
			t.Errorf("code %q fails the name pattern", d.Code)
		}
		if len(d.Code) > 63 {
			t.Errorf("code %s longer than 63", d.Code)
		}
		if d.Title == "" || d.Description == "" {
			t.Errorf("code %s missing title/description", d.Code)
		}
		if Kebab(d.Code) == "" || CodeFromKebab(Kebab(d.Code)) != d.Code {
			t.Errorf("code %s is not reversible with kebab %q", d.Code, Kebab(d.Code))
		}
		want := TypeURIPrefix + string(d.Domain) + "/" + Kebab(d.Code)
		if d.TypeURI() != want {
			t.Errorf("type URI %s want %s", d.TypeURI(), want)
		}
		if d.Domain == DomainPlatform {
			plat := "https://developer.nextmoe.dev/problems/platform/" + Kebab(d.Code)
			if d.TypeURI() != plat {
				t.Errorf("platform URI %s want %s", d.TypeURI(), plat)
			}
		}
		if prev, ok := seen[d.Code]; ok {
			t.Errorf("code %s duplicated in %s and %s", d.Code, prev, d.Domain)
		}
		seen[d.Code] = string(d.Domain)
		if prev, ok := statusOf[d.Code]; ok && prev != d.Status {
			t.Errorf("code %s has two statuses %d and %d", d.Code, prev, d.Status)
		}
		statusOf[d.Code] = d.Status
		if other, ok := uris[d.TypeURI()]; ok {
			t.Errorf("type URI %s used by %s and %s", d.TypeURI(), other, d.Code)
		}
		uris[d.TypeURI()] = d.Code
		if d.Status < 400 || d.Status > 599 {
			t.Errorf("code %s has non-error status %d", d.Code, d.Status)
		}
	}
	for _, want := range requiredCodes {
		got, ok := Lookup(want.code)
		if !ok {
			t.Errorf("missing code %s", want.code)
			continue
		}
		if got.Domain != want.domain || got.Status != want.status {
			t.Errorf("%s domain=%s status=%d, want %s %d", want.code, got.Domain, got.Status, want.domain, want.status)
		}
	}
	for _, r := range Reasons {
		if !NamePattern.MatchString(r.Reason) {
			t.Errorf("reason %q fails the name pattern", r.Reason)
		}
		if len(r.Reason) > 63 {
			t.Errorf("reason %s longer than 63", r.Reason)
		}
		if _, clash := seen[r.Reason]; clash {
			t.Errorf("reason %s collides with a top-level code", r.Reason)
		}
		if r.Title == "" || r.Description == "" {
			t.Errorf("reason %s missing title/description", r.Reason)
		}
	}
}

func TestLookupUnknown(t *testing.T) {
	if _, ok := Lookup("NOT_A_CODE"); ok {
		t.Fatal("unknown code looked up")
	}
	if _, ok := LookupReason("NOT_A_REASON"); ok {
		t.Fatal("unknown reason looked up")
	}
	got, ok := Lookup(CodeValidationFailed)
	if !ok || got.Status != 422 {
		t.Fatalf("VALIDATION_FAILED lookup = %+v ok=%v", got, ok)
	}
}

func TestReasonParamsSchema(t *testing.T) {
	want := map[string][]string{
		ReasonTooLong:      {ParamMaxLength},
		ReasonTooShort:     {ParamMinLength},
		ReasonOutOfRange:   {ParamMinimum, ParamMaximum},
		ReasonTooManyItems: {ParamMaxItems},
		ReasonTooFewItems:  {ParamMinItems},
		ReasonUnknownValue: {ParamAllowed},
	}
	for _, r := range Reasons {
		got := r.Params
		exp := want[r.Reason]
		if len(got) != len(exp) {
			t.Errorf("%s params %v want %v", r.Reason, got, exp)
			continue
		}
		for i := range exp {
			if got[i] != exp[i] {
				t.Errorf("%s params %v want %v", r.Reason, got, exp)
			}
		}
	}
}

func TestKebabBijection(t *testing.T) {
	for _, d := range Codes {
		if got := CodeFromKebab(Kebab(d.Code)); got != d.Code {
			t.Errorf("CodeFromKebab(Kebab(%s)) = %s", d.Code, got)
		}
		if strings.Contains(Kebab(d.Code), "_") {
			t.Errorf("kebab %s still has underscore", Kebab(d.Code))
		}
	}
}
