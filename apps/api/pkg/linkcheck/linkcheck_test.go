package linkcheck

import (
	"slices"
	"testing"
)

func TestStatuses(t *testing.T) {
	d := Result{Status: StatusDead}
	a := Result{Status: StatusAlive}
	u := Result{Status: StatusUnknown}
	odd := Result{Status: "weird"}

	cases := []struct {
		name string
		in   []Result
		n    int
		want []Status
	}{
		{"each link keeps its own verdict", []Result{d, a, u}, 3, []Status{StatusDead, StatusAlive, StatusUnknown}},
		{"an unknown token reads unknown", []Result{odd, d}, 2, []Status{StatusUnknown, StatusDead}},
		{"a short answer is not trusted", []Result{d}, 2, []Status{StatusUnknown, StatusUnknown}},
		{"no answer is all unknown", nil, 1, []Status{StatusUnknown}},
	}
	for _, tc := range cases {
		if got := statuses(tc.in, tc.n); !slices.Equal(got, tc.want) {
			t.Errorf("%s: statuses=%v want %v", tc.name, got, tc.want)
		}
	}
}
