package resourcevocab

import "strings"

func LegacyLanguage(scalar string) Keys {
	switch strings.TrimSpace(scalar) {
	case "", "all":
		return Keys{}
	case "others":
		return Keys{"other"}
	default:
		if _, ok := languageIndex[scalar]; ok {
			return Keys{scalar}
		}
		return Keys{}
	}
}

func LegacyPlatform(scalar string) (platforms Keys, runtimes Keys) {
	switch strings.TrimSpace(scalar) {
	case "windows":
		return Keys{"win"}, Keys{"native-win"}
	case "mac":
		return Keys{"mac"}, Keys{}
	case "linux":
		return Keys{"lin"}, Keys{}
	case "app":
		return Keys{"and"}, Keys{"native-and"}
	case "others":
		return Keys{"oth"}, Keys{}
	case "emulator":
		return Keys{}, Keys{}
	default:
		return Keys{}, Keys{}
	}
}

func CompatLanguage(langs Keys) string {
	if len(langs) == 0 {
		return ""
	}
	if langs[0] == "other" {
		return "others"
	}
	return langs[0]
}

func CompatPlatform(platforms, runtimes Keys) string {
	for _, r := range runtimes {
		if emulatorRuntimeSet[r] {
			return "emulator"
		}
	}
	hasWin, hasAnd := false, false
	for _, p := range platforms {
		switch p {
		case "win":
			hasWin = true
		case "and":
			hasAnd = true
		}
	}
	if hasAnd && !hasWin {
		return "app"
	}
	if hasWin {
		return "windows"
	}
	for _, p := range platforms {
		switch p {
		case "mac":
			return "mac"
		case "lin":
			return "linux"
		}
	}
	if len(platforms) == 0 && len(runtimes) == 0 {
		return ""
	}
	return "others"
}

func CompatType(t string) string {
	switch t {
	case "others", "ai":
		return "other"
	default:
		return t
	}
}

func Title(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if strings.ContainsAny(v, "\r\n\t") {
		return "", false
	}
	return v, len([]rune(v)) <= 200
}
