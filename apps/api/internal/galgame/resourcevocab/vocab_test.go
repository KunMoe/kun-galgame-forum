package resourcevocab

import (
	"strings"
	"testing"
)

func TestLegacyAndCompatRoundTrip(t *testing.T) {
	cases := []struct {
		old  string
		plat Keys
		run  Keys
		back string
	}{
		{old: "windows", plat: Keys{"win"}, run: Keys{"native-win"}, back: "windows"},
		{old: "app", plat: Keys{"and"}, run: Keys{"native-and"}, back: "app"},
		{old: "mac", plat: Keys{"mac"}, back: "mac"},
		{old: "linux", plat: Keys{"lin"}, back: "linux"},
		{old: "others", plat: Keys{"oth"}, back: "others"},
		{old: "emulator", run: Keys{"emulator"}, back: "emulator"},
	}
	for _, c := range cases {
		p, r := LegacyPlatform(c.old)
		if got := stringsJoin(p); got != stringsJoin(c.plat) {
			t.Errorf("%s platforms %v want %v", c.old, p, c.plat)
		}
		if got := stringsJoin(r); got != stringsJoin(c.run) {
			t.Errorf("%s runtimes %v want %v", c.old, r, c.run)
		}
		if back := CompatPlatform(p, r); back != c.back {
			t.Errorf("%s compat %q want %q", c.old, back, c.back)
		}
	}
}

func TestCompatEmulatorRuntimeWins(t *testing.T) {
	got := CompatPlatform(Keys{"win", "and"}, Keys{"native-win", "tyranor"})
	if got != "emulator" {
		t.Fatalf("got %q", got)
	}
}

func TestRuntimeFilterWidensEmulator(t *testing.T) {
	got := RuntimeFilter([]string{"native-and", "emulator"})
	want := append([]string{"native-and"}, emulatorRuntimes...)
	if stringsJoin(got) != stringsJoin(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if got := RuntimeFilter([]string{"kirikiroid2"}); stringsJoin(got) != "kirikiroid2" {
		t.Fatalf("a named emulator stays itself, got %v", got)
	}
}

// The web lists the emulator runtimes as every runtime that is neither native-*
// nor other, so a key added here must follow that rule.
func TestEmulatorRuntimesAreTheNonNativeKeys(t *testing.T) {
	var want Keys
	for _, k := range RuntimeKeys {
		if !strings.HasPrefix(k, "native-") && k != "other" {
			want = append(want, k)
		}
	}
	if stringsJoin(want) != stringsJoin(emulatorRuntimes) {
		t.Fatalf("emulatorRuntimes %v, non-native runtimes %v", emulatorRuntimes, want)
	}
}

func TestRuntimeKeysAreLetMoesInItsOrder(t *testing.T) {
	want := "native-win,native-and,tyranor-next,yukihub,native-ios,winlator,gamehub,kirikiroid2," +
		"krkrsdl2,onscripter,joiplay,easyrpg,renpy-android,tyranor,emulator,other"
	if got := stringsJoin(RuntimeKeys); got != want {
		t.Fatalf("runtime keys\n got %s\nwant %s", got, want)
	}
}

func TestRuntimesNormalizeToVocabularyOrder(t *testing.T) {
	got, ok := Runtimes([]string{"tyranor", "yukihub", "native-ios", "tyranor-next", "native-and"})
	if !ok || stringsJoin(got) != "native-and,tyranor-next,yukihub,native-ios,tyranor" {
		t.Fatalf("got %v ok=%v", got, ok)
	}
}

func TestYukiHubIsAnEmulatorRuntime(t *testing.T) {
	if !IsEmulatorRuntime("yukihub") {
		t.Fatal("yukihub launches other engines; it is an emulator runtime")
	}
	widened := "," + stringsJoin(RuntimeFilter([]string{"emulator"})) + ","
	if !strings.Contains(widened, ",yukihub,") {
		t.Fatalf("the emulator filter must reach yukihub: %s", widened)
	}
	if got := CompatPlatform(Keys{"and"}, Keys{"yukihub"}); got != "emulator" {
		t.Fatalf("compat platform %q", got)
	}
}

func TestHasRuntimeAxis(t *testing.T) {
	if !HasRuntimeAxis("collection") || HasRuntimeAxis("ost") {
		t.Fatal("collection needs a runtime, ost does not")
	}
}

func stringsJoin(k Keys) string {
	if len(k) == 0 {
		return ""
	}
	out := k[0]
	for i := 1; i < len(k); i++ {
		out += "," + k[i]
	}
	return out
}
