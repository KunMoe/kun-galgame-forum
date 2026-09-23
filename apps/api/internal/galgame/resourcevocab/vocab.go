package resourcevocab

var TypeKeys = []string{
	"game", "patch", "collection", "crack_fix", "mod", "tool",
	"walkthrough", "ost", "voice", "cg", "wallpaper", "artbook", "video", "other",
}

var LegacyTypeKeys = []string{"image", "ai", "others"}

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
	"renpy-android", "tyranor", "tyranor-next", "other",
}

var VersionLabelKeys = []string{"官方最新", "稳定版", "镜像版", "汉化版", "未知版本"}

var RuntimeRelevantTypes = []string{
	"game", "collection", "patch", "crack_fix", "mod", "tool",
}

var emulatorRuntimes = []string{
	"winlator", "gamehub", "kirikiroid2", "krkrsdl2", "onscripter",
	"joiplay", "easyrpg", "renpy-android", "tyranor", "tyranor-next",
}

func index(keys []string) map[string]int {
	m := make(map[string]int, len(keys))
	for i, k := range keys {
		m[k] = i
	}
	return m
}

var (
	typeIndex     = index(append(append([]string{}, TypeKeys...), LegacyTypeKeys...))
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

func IsType(v string) bool { _, ok := typeIndex[v]; return ok }

func HasRuntimeAxis(resourceType string) bool { return runtimeSet[resourceType] }

func Languages(in []string) (Keys, bool) { return Normalize(in, languageIndex) }
func Platforms(in []string) (Keys, bool) { return Normalize(in, platformIndex) }
func Runtimes(in []string) (Keys, bool)  { return Normalize(in, runtimeIndex) }
