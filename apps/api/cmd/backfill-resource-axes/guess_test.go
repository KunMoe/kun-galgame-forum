package main

import "testing"

func TestGuessFromEmulatorNote(t *testing.T) {
	g := guessFromText("【PC+安卓直装+KR&TY模拟器双端】", "11.06GB", "emulator")
	if !contains(g.Runtimes, "tyranor") || !contains(g.Runtimes, "kirikiroid2") ||
		!contains(g.Runtimes, "native-and") {
		t.Fatalf("runtimes %v", g.Runtimes)
	}
	if !contains(g.Platforms, "win") || !contains(g.Platforms, "and") {
		t.Fatalf("platforms %v", g.Platforms)
	}
}

func TestGuessNamedEmulatorAddsNoPlaceholder(t *testing.T) {
	g := guessFromText("KRKR模拟器", "", "emulator")
	if contains(g.Runtimes, "emulator") {
		t.Fatalf("runtimes %v", g.Runtimes)
	}
}

func TestGuessDoesNotTripOnEnglishTy(t *testing.T) {
	g := guessFromText("quality entity notes", "2 GB", "windows")
	if contains(g.Runtimes, "tyranor") {
		t.Fatalf("false tyranor %v", g.Runtimes)
	}
}

func TestGuessKRAddsAndroidPlatform(t *testing.T) {
	g := guessFromText("KRKR模拟器，用专门工具解压lz4", "", "emulator")
	if !contains(g.Runtimes, "kirikiroid2") {
		t.Fatalf("runtimes %v", g.Runtimes)
	}
	if !contains(g.Platforms, "and") {
		t.Fatalf("platforms %v", g.Platforms)
	}
}

func contains(keys []string, v string) bool {
	for _, k := range keys {
		if k == v {
			return true
		}
	}
	return false
}
