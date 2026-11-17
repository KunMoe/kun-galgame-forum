package agentmcp

import (
	"strings"

	"kun-galgame-api/internal/galgame/resourcevocab"
)

var resourceTypeLabels = map[string]string{
	"game": "游戏本体", "collection": "合集", "patch": "补丁", "crack_fix": "免CD / 修正", "mod": "MOD",
	"tool": "工具", "walkthrough": "攻略", "ost": "原声带", "voice": "语音", "cg": "鉴赏 CG",
	"wallpaper": "壁纸", "artbook": "设定资料", "video": "视频", "other": "其它",
}

var languageLabels = map[string]string{
	"zh-cn": "简体中文", "zh-tw": "繁体中文", "ja-jp": "日语", "en-us": "英语", "other": "其他语言", "others": "其他语言",
}

var platformLabels = map[string]string{
	"win": "Windows", "and": "安卓", "ios": "iOS", "mac": "macOS", "lin": "Linux", "web": "网页",
	"mob": "功能机", "swi": "Switch", "sw2": "Switch 2", "psp": "PSP", "psv": "PS Vita",
	"ps2": "PS2", "ps3": "PS3", "ps4": "PS4", "ps5": "PS5", "nds": "NDS", "n3d": "3DS",
	"dvd": "DVD 播放机", "oth": "其它平台",
}

var runtimeLabels = map[string]string{
	"native-win": "Windows 原生", "native-and": "安卓直装", "native-ios": "iOS 原生",
	"tyranor-next": "Tyranor Next", "yukihub": "YukiHub", "winlator": "Winlator", "gamehub": "GameHub",
	"kirikiroid2": "Kirikiroid2", "krkrsdl2": "krkrsdl2", "onscripter": "ONScripter", "joiplay": "JoiPlay",
	"easyrpg": "EasyRPG", "renpy-android": "Ren'Py 安卓版", "tyranor": "Tyranor", "emulator": "模拟器",
	"other": "其它运行环境",
}

var toolsetTypeLabels = map[string]string{
	"emulator": "模拟器", "translator": "翻译工具", "extractor": "解包工具", "converter": "转换工具",
	"debug": "调试工具", "launcher": "启动器 / 管理器", "script": "脚本", "docs": "文档教程", "others": "其他",
}

var toolsetPlatformLabels = map[string]string{
	"windows": "Windows", "mac": "macOS", "linux": "Linux", "emulator": "模拟器", "others": "其他平台",
}

var ratingRecommendLabels = map[string]string{
	"strong_yes": "强烈推荐", "yes": "推荐", "neutral": "一般", "no": "不推荐", "strong_no": "强烈不推荐",
}

var playStatusLabels = map[string]string{
	"wish": "想玩", "doing": "在玩", "done_one_route": "通关一条线", "done_main": "通关主线",
	"done_all": "全线通关", "done": "已通关", "on_hold": "搁置", "dropped": "弃坑",
}

// mobilePlatforms and mobileRuntimes are the web's 手机可玩: a resource for an Android or iOS
// platform, or one that runs through any emulator runtime.
var mobilePlatforms, mobileRuntimes = "and,ios", "emulator"

func labels(keys []string, table map[string]string) string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, label(k, table))
	}
	return strings.Join(out, "/")
}

func label(key string, table map[string]string) string {
	if v, ok := table[key]; ok {
		return v
	}
	return key
}

func validKey(key string, keys []string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

var (
	platformKeys = resourcevocab.PlatformKeys
	languageKeys = resourcevocab.LanguageKeys
	typeKeys     = resourcevocab.TypeKeys
)
