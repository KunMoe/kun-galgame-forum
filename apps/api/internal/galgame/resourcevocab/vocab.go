package resourcevocab

var TypeKeys = []string{
	"game", "patch", "collection", "crack_fix", "mod", "tool",
	"walkthrough", "ost", "voice", "cg", "wallpaper", "artbook", "video", "other",
}

var ProviderKeys = []string{
	"baidu", "aliyun", "quark", "pan123", "tianyiyun",
	"caiyun", "xunlei", "uc", "lanzou", "other",
}

var LanguageKeys = []string{"zh-cn", "zh-tw", "ja-jp", "en-us", "other"}

var PlatformKeys = []string{
	"win", "and", "ios", "mac", "lin", "web", "mob",
	"swi", "sw2", "n3d", "nds", "wii", "wiu", "gba", "gbc", "nes", "sfc",
	"ps1", "ps2", "ps3", "ps4", "ps5", "psp", "psv",
	"xb1", "xb3", "xbo", "xxs",
	"sat", "smd", "scd", "drc", "pce", "pcf", "tdo",
	"p88", "p98", "x1s", "x68", "fm7", "fm8", "fmt", "msx", "dos",
	"dvd", "bdp", "vnd", "oth",
}

var RuntimeKeys = []string{
	"native-win", "native-and", "native-ios", "winlator", "gamehub",
	"kirikiroid2", "krkrsdl2", "onscripter", "joiplay", "easyrpg",
	"renpy-android", "tyranor", "tyranor-next", "emulator", "other",
}

var RuntimeRelevantTypes = []string{
	"game", "collection", "patch", "crack_fix", "mod", "tool",
}

var emulatorRuntimes = []string{
	"winlator", "gamehub", "kirikiroid2", "krkrsdl2", "onscripter",
	"joiplay", "easyrpg", "renpy-android", "tyranor", "tyranor-next", "emulator",
}

func index(keys []string) map[string]int {
	m := make(map[string]int, len(keys))
	for i, k := range keys {
		m[k] = i
	}
	return m
}

var (
	languageIndex = index(LanguageKeys)
	platformIndex = index(PlatformKeys)
	runtimeIndex  = index(RuntimeKeys)
	runtimeSet    = func() map[string]bool {
		m := map[string]bool{}
		for _, k := range RuntimeRelevantTypes {
			m[k] = true
		}
		return m
	}()
	emulatorRuntimeSet = func() map[string]bool {
		m := map[string]bool{}
		for _, k := range emulatorRuntimes {
			m[k] = true
		}
		return m
	}()
)

func HasRuntimeAxis(resourceType string) bool { return runtimeSet[resourceType] }

func IsEmulatorRuntime(runtime string) bool { return emulatorRuntimeSet[runtime] }

func Platforms(in []string) (Keys, bool) { return Normalize(in, platformIndex) }
func Runtimes(in []string) (Keys, bool)  { return Normalize(in, runtimeIndex) }

// RuntimeFilter widens emulator, which a resource stores when its uploader
// said only that it needs one, to every emulator runtime.
func RuntimeFilter(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if k == "emulator" {
			out = append(out, emulatorRuntimes...)
		} else {
			out = append(out, k)
		}
	}
	return out
}
