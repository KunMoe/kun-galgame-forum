package renumber

import (
	"strings"
	"testing"
)

func TestLoadMapOK(t *testing.T) {
	m, err := LoadMap(strings.NewReader("1\t1\tcurated\n10\t9\tclaim\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Len() != 2 || m.TSVRows() != 2 {
		t.Fatalf("len=%d tsv=%d", m.Len(), m.TSVRows())
	}
	if got, ok := m.Lookup(10); !ok || got != 9 {
		t.Fatalf("Lookup(10)=%d %v", got, ok)
	}
	if _, ok := m.Lookup(3); ok {
		t.Fatal("Lookup of an unnamed id must miss")
	}
	m.AddUnchanged([]int64{1, 3, 0, -1})
	if got, ok := m.Lookup(3); !ok || got != 3 {
		t.Fatalf("unchanged 3: %d %v", got, ok)
	}
	if m.Len() != 3 {
		t.Fatalf("len after unchanged=%d", m.Len())
	}
	how := m.HowCounts()
	if how[HowCurated] != 1 || how[HowClaim] != 1 || how[HowUnchanged] != 1 {
		t.Fatalf("how counts %+v", how)
	}
	if got, ok := m.Lookup(1); !ok || got != 1 {
		t.Fatalf("id 1 stayed curated, not overwritten: %d %v", got, ok)
	}
}

func TestLoadMapRejects(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"malformed fields", "1\t2\n", "map line 1:"},
		{"four fields", "1\t2\tcurated\textra\n", "map line 1:"},
		{"old_id not a number", "x\t2\tcurated\n", "map line 1: old_id"},
		{"new_id not a number", "1\ty\tclaim\n", "map line 1: new_id"},
		{"non-positive old", "0\t2\tcurated\n", "map line 1: ids must be positive"},
		{"non-positive new", "1\t-3\tclaim\n", "map line 1: ids must be positive"},
		{"unknown how", "1\t2\tunchanged\n", "map line 1: unknown how"},
		{"empty how", "1\t2\t\n", "map line 1: unknown how"},
		{"duplicate old_id", "1\t2\tcurated\n1\t3\tclaim\n", "map line 2: duplicated old_id 1"},
		{"duplicate after blank", "1\t2\tcurated\n\n1\t9\tclaim\n", "map line 3: duplicated old_id 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadMap(strings.NewReader(tc.in))
			if err == nil {
				t.Fatal("want error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q, want substring %q", err, tc.want)
			}
		})
	}
}

func TestLoadMapSkipsBlankLines(t *testing.T) {
	m, err := LoadMap(strings.NewReader("\n1\t2\tcurated\n\r\n3\t4\tclaim\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Len() != 2 {
		t.Fatalf("len=%d", m.Len())
	}
}
