package main

import (
	"slices"
	"strings"

	"kun-galgame-api/internal/galgame/resourcevocab"
)

type guess struct {
	Platforms resourcevocab.Keys
	Runtimes  resourcevocab.Keys
}

func guessFromText(note, size, legacyPlatform string) guess {
	blob := strings.ToLower(note + "\n" + size)
	plats := map[string]bool{}
	runs := map[string]bool{}

	addPlat := func(k string) { plats[k] = true }
	addRun := func(k string) { runs[k] = true }

	if strings.Contains(blob, "winlator") {
		addRun("winlator")
		addPlat("win")
	}
	if strings.Contains(blob, "盖世") || strings.Contains(blob, "gamehub") {
		addRun("gamehub")
		addPlat("win")
	}
	if strings.Contains(blob, "tyranor next") || strings.Contains(blob, "tyranor-next") {
		addRun("tyranor-next")
		addPlat("and")
	} else if strings.Contains(blob, "tyrano") || containsTok(blob, "ty") {
		addRun("tyranor")
		addPlat("and")
	}
	if strings.Contains(blob, "kirikiroid") || strings.Contains(blob, "xp3") ||
		containsTok(blob, "kr") {
		addRun("kirikiroid2")
		addPlat("and")
	}
	if strings.Contains(blob, "onscripter") || containsTok(blob, "ons") {
		addRun("onscripter")
		addPlat("and")
	}
	if strings.Contains(blob, "joiplay") {
		addRun("joiplay")
		addPlat("and")
	}
	if strings.Contains(blob, "easyrpg") {
		addRun("easyrpg")
		addPlat("and")
	}
	if strings.Contains(blob, "ren'py") || strings.Contains(blob, "renpy") {
		addRun("renpy-android")
		addPlat("and")
	}
	if strings.Contains(blob, "krkrsdl2") {
		addRun("krkrsdl2")
		addPlat("and")
	}
	if strings.Contains(blob, "直装") || strings.Contains(blob, "apk") {
		addRun("native-and")
		addPlat("and")
	}
	if strings.Contains(blob, "ios") {
		addRun("native-ios")
		addPlat("ios")
	}
	if strings.Contains(blob, "pc") || strings.Contains(blob, "windows") {
		addPlat("win")
		if legacyPlatform == "windows" || strings.Contains(blob, "pc") {
			addRun("native-win")
		}
	}

	p, _ := resourcevocab.LegacyPlatform(legacyPlatform)
	for _, k := range p {
		addPlat(k)
	}
	if legacyPlatform == "windows" {
		addRun("native-win")
	}
	if legacyPlatform == "app" {
		addRun("native-and")
		addPlat("and")
	}
	if legacyPlatform == "emulator" && !slices.ContainsFunc(setKeys(runs), resourcevocab.IsEmulatorRuntime) {
		addRun("emulator")
	}

	outP, _ := resourcevocab.Platforms(setKeys(plats))
	outR, _ := resourcevocab.Runtimes(setKeys(runs))
	return guess{Platforms: outP, Runtimes: outR}
}

func containsTok(blob, tok string) bool {
	for _, p := range []string{
		tok + "模拟器", "+" + tok, tok + "&", tok + "/", tok + "版",
		" " + tok + " ", tok + "+",
	} {
		if strings.Contains(blob, p) {
			return true
		}
	}
	return false
}

func setKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
