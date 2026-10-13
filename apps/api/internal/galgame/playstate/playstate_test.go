package playstate

import "testing"

func TestRoundTripAllPickerValues(t *testing.T) {
	for _, want := range All() {
		state, completion, ok := ToCatalog(want)
		if !ok {
			t.Fatalf("ToCatalog(%q) ok = false", want)
		}
		got := FromCatalog(state, completion)
		if got != want {
			t.Errorf("FromCatalog(ToCatalog(%q)) = %q, want %q", want, got, want)
		}
	}
}

func TestDoneWithoutCompletionRoundTrips(t *testing.T) {
	if got := FromCatalog("done", nil); got != Done {
		t.Errorf("FromCatalog(done, nil) = %q, want done", got)
	}
	state, completion, ok := ToCatalog(Done)
	if !ok || state != "done" || completion != nil {
		t.Errorf("ToCatalog(done) = %q, %v, %v; want done with no completion", state, completion, ok)
	}
}

func TestFromCatalogEmptyAndUnknown(t *testing.T) {
	if got := FromCatalog("", nil); got != "" {
		t.Errorf("FromCatalog(\"\", nil) = %q, want empty", got)
	}
	if got := FromCatalog("garbage", nil); got != "" {
		t.Errorf("FromCatalog(garbage, nil) = %q, want empty", got)
	}
}
