package content

import "testing"

func TestFold(t *testing.T) {
	for _, tc := range []struct {
		why            string
		adultConfirmed bool
		display        string
		want           Stance
	}{
		{"the backfilled majority: blur stored, never attested", false, "blur", StanceHide},
		{"show stored, never attested", false, "show", StanceHide},
		{"hide stored, never attested", false, "hide", StanceHide},
		{"attested and blurring", true, "blur", StanceBlur},
		{"attested and showing", true, "show", StanceShow},
		{"attested and hiding", true, "hide", StanceHide},
		{"attested but the claim never arrived", true, "", StanceHide},
		{"attested with a value this build does not know", true, "peek", StanceHide},
		{"no claims at all (a session written before this wave)", false, "", StanceHide},
	} {
		if got := Fold(tc.adultConfirmed, tc.display); got != tc.want {
			t.Errorf("%s: Fold(%v, %q) = %q, want %q", tc.why, tc.adultConfirmed, tc.display, got, tc.want)
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
