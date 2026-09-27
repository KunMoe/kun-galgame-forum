package utils

import "strings"

var providerPatterns = []struct {
	key      string
	patterns []string
}{
	{"baidu", []string{"pan.baidu.com", "tieba.baidu.com", "pan.baidu.", "baidu.com"}},
	{"aliyun", []string{"alipan.com", "aliyun", "aliyundrive", "aliyuncs", "aliyunpan"}},
	{"quark", []string{"pan.quark.cn", "quark.cn", "quark"}},
	{"pan123", []string{
		"123pan", "123684", "123865", "123912", "123912.com",
		"123684.com", "123865.com", "123pan.cn", "vip.123pan",
	}},
	{"tianyiyun", []string{"cloud.189.cn", "189.cn", "ecloud.189.cn"}},
	{"caiyun", []string{"caiyun.139.com", "yun.139.com", "139.com"}},
	{"xunlei", []string{"pan.xunlei.com", "xunlei.com"}},
	{"uc", []string{"drive.uc.cn", "uc.cn"}},
	{"lanzou", []string{
		"lanzou.com", "lanzous.com", "lanzoux.com", "lanzoui.com",
		"lanzouw.com", "lanzouj.com", "lanzouu.com", "lanzouq.com",
	}},
}

func DetectProviderFromURL(url string) string {
	if url == "" {
		return "other"
	}
	s := strings.ToLower(url)
	for _, entry := range providerPatterns {
		for _, p := range entry.patterns {
			if strings.Contains(s, p) {
				return entry.key
			}
		}
	}
	return "other"
}

func DetectProvidersFromURLs(urls []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		p := DetectProviderFromURL(u)
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

var providerNameSubstrs = []struct {
	pattern string
	name    string
}{
	{"ed2k://", "电驴下载"},
	{"magnet", "磁力下载"},
	{"tieba.baidu.com", "百度贴吧"},
	{"baidu.com", "百度网盘"},
	{"quark.cn", "夸克网盘"},
	{"alipan.com", "阿里云盘"},
	{"123912.com", "123 云盘"},
	{"123865.com", "123 云盘"},
	{"123pan.com", "123 云盘"},
	{"123pan.cn", "123 云盘"},
	{"xunlei.com", "迅雷云盘"},
	{"weiyun.com", "腾讯微云"},
	{"139.com", "和彩云 (移动云盘)"},
	{"189.cn", "天翼云盘"},
	{"uc.cn", "UC 网盘"},
	{"lanzou", "蓝奏云"},
	{"ctfile.com", "城通网盘"},
	{"nullcloud.top", "未知云盘"},
	{"mypikpak.com", "PikPak"},
	{"sharepoint.com", "OneDrive"},
	{"sharepoint.cn", "OneDrive"},
	{"1drv.ms", "OneDrive"},
	{"mega.nz", "MEGA"},
	{"google.com", "Google 云盘"},
	{"yandex.com", "Yandex Disk"},
	{"gofile.io", "GoFile"},
	{"ipfs.dweb.link", "IPFS"},
	{"steampowered.com", "Steam"},
	{"epicgames.com", "Epic 游戏商店"},
	{"itch.io", "itch.io"},
	{"github.com", "GitHub"},
	{"bilibili.com", "哔哩哔哩"},
	{"t.me", "Telegram"},
	{"telegram.me", "Telegram"},
	{"archive.org", "Internet Archive"},
	{"nyaa.si", "Nyaa"},
	{"2dfan.com", "2BFun"},
	{"ddfan.org", "2BFun"},
	{"ddfan.top", "2BFun"},
	{"galge.top", "2BFun"},
	{"hacg.uno", "琉璃神社 (HACG)"},
	{"kungal.com", "鲲 Galgame 论坛"},
	{"moyu.moe", "鲲 Galgame 补丁"},
	{"anime-sharing.com", "Anime-Sharing"},
	{"e-hentai.org", "E-Hentai"},
	{"dmm.co.jp", "DMM"},
	{"zi6.cc", "梓澪"},
	{"zi0.cc", "梓澪"},
	{"zi8.cc", "梓澪"},
	{"shinnku.com", "真红小站"},
	{"shinnku.org", "真红小站"},
	{"oo0o.ooo", "真红小站"},
	{"touchgal.io", "TouchGal"},
	{"touchgal.us", "TouchGal"},
	{"dlgal.com", "GGbases"},
	{"lycorisgal.com", "LycorisGal"},
}

func DetectProviderNameFromURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	s := strings.ToLower(rawURL)
	for _, entry := range providerNameSubstrs {
		if strings.Contains(s, entry.pattern) {
			return entry.name
		}
	}
	return hostFromURL(rawURL)
}

func DetectProviderNamesFromURLs(urls []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		name := DetectProviderNameFromURL(u)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func hostFromURL(raw string) string {
	s := raw
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndex(s, "@"); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.LastIndex(s, ":"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimPrefix(strings.ToLower(s), "www.")
	if s == "" {
		return raw
	}
	return s
}
