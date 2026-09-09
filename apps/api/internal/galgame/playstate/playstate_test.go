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

func TestFromCatalogDisplayOnlyDone(t *testing.T) {
	if got := FromCatalog("done", nil); got != "done" {
		t.Errorf("FromCatalog(done, nil) = %q, want done", got)
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

func TestValidRejectsDisplayOnlyDone(t *testing.T) {
	if Valid("done") {
		t.Fatal("Valid(done) = true, want false")
	}
	for _, s := range All() {
		if !Valid(s) {
			t.Errorf("Valid(%q) = false, want true", s)
		}
	}
}

func TestToCatalogRejectsDisplayOnlyDone(t *testing.T) {
	_, _, ok := ToCatalog("done")
	if ok {
		t.Fatal("ToCatalog(done) ok = true, want false")
	}
}
