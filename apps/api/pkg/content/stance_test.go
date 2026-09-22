package content

import "testing"

func TestFold(t *testing.T) {
	for _, tc := range []struct {
		why            string
		adultConfirmed bool
		display        string
		want           Stance
	}{
		{"a stored blur cannot widen a reader the claim does not cover", false, "blur", StanceHide},
		{"nor can a stored show", false, "show", StanceHide},
		{"hide either way", false, "hide", StanceHide},
		{"adult and blurring", true, "blur", StanceBlur},
		{"adult and showing", true, "show", StanceShow},
		{"adult and hiding", true, "hide", StanceHide},
		{"adult but the claim never arrived", true, "", StanceHide},
		{"a value this build does not know", true, "peek", StanceHide},
		{"no claims at all (a session written before this wave)", false, "", StanceHide},
	} {
		if got := Fold(tc.adultConfirmed, tc.display); got != tc.want {
			t.Errorf("%s: Fold(%v, %q) = %q, want %q", tc.why, tc.adultConfirmed, tc.display, got, tc.want)
		}
	}
}

// ParseStance reads back what the Bearer stance cache wrote, so a value that
// was never a stance has to land on hide rather than on "not hide".
func TestParseStance(t *testing.T) {
	for in, want := range map[string]Stance{
		"hide": StanceHide, "blur": StanceBlur, "show": StanceShow,
		"": StanceHide, "peek": StanceHide, "HIDE": StanceHide,
	} {
		if got := ParseStance(in); got != want {
			t.Errorf("ParseStance(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAllowsNSFW(t *testing.T) {
	if StanceHide.AllowsNSFW() {
		t.Error("hide must not allow nsfw")
	}
	if !StanceBlur.AllowsNSFW() || !StanceShow.AllowsNSFW() {
		t.Error("blur and show both allow nsfw through; blur masks it on the way out")
	}
}
