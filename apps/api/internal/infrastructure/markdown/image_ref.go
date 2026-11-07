package markdown

import (
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"kun-galgame-api/pkg/imageclient"
)

//go:embed legacy_sticker_refs.tsv
var legacyStickerRefsTSV string

type stickerKey struct {
	pack, position int
}

var (
	legacyStickerHashByKey = mustParseLegacyStickerRefs(legacyStickerRefsTSV)

	legacyStickerURLRe = regexp.MustCompile(`https?://sticker\.kungal\.com/stickers/KUNgal(\d{1,2})/(\d{1,3})\.webp`)
	absoluteImageURLRe = regexp.MustCompile(`https?://([A-Za-z0-9.\-]+)/([0-9a-f]{2})/([0-9a-f]{2})/([0-9a-f]{64})(_[a-z0-9]+)?\.webp`)
	legacyHashRe       = regexp.MustCompile(`^[0-9a-f]{64}$`)

	imageServiceHosts = knownImageServiceHosts()
)

func mustParseLegacyStickerRefs(tsv string) map[stickerKey]string {
	out := make(map[stickerKey]string)
	for line := range strings.SplitSeq(tsv, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		// The TSV is compiled into the binary; a bad line is a broken build, not a runtime miss.
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			panic(fmt.Sprintf("legacy_sticker_refs.tsv: want 3 tab-separated fields, got %q", line))
		}
		pack, err := strconv.Atoi(fields[0])
		if err != nil {
			panic(fmt.Sprintf("legacy_sticker_refs.tsv: bad pack %q", line))
		}
		position, err := strconv.Atoi(fields[1])
		if err != nil {
			panic(fmt.Sprintf("legacy_sticker_refs.tsv: bad position %q", line))
		}
		hash := fields[2]
		if !legacyHashRe.MatchString(hash) {
			panic(fmt.Sprintf("legacy_sticker_refs.tsv: bad hash %q", line))
		}
		key := stickerKey{pack: pack, position: position}
		if _, dup := out[key]; dup {
			panic(fmt.Sprintf("legacy_sticker_refs.tsv: duplicate pack/position %q", line))
		}
		out[key] = hash
	}
	return out
}

func LegacyStickerHash(pack, position int) (string, bool) {
	h, ok := legacyStickerHashByKey[stickerKey{pack: pack, position: position}]
	return h, ok
}

func knownImageServiceHosts() map[string]struct{} {
	out := make(map[string]struct{}, len(imageclient.KnownHosts))
	for _, h := range imageclient.KnownHosts {
		out[h] = struct{}{}
	}
	return out
}

func addImageServiceHost(host string) {
	host = strings.ToLower(host)
	if host == "" {
		return
	}
	imageServiceHosts[host] = struct{}{}
}

func isImageServiceHost(host string) bool {
	_, ok := imageServiceHosts[strings.ToLower(host)]
	return ok
}

func rememberCDNBaseHost(base string) {
	s := strings.TrimRight(base, "/")
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if j := strings.IndexAny(s, "/?"); j >= 0 {
		s = s[:j]
	}
	addImageServiceHost(s)
	allowInlineImageHost(strings.ToLower(s))
}

func ResolveLegacyStickerRefs(source string) string {
	if !strings.Contains(source, "sticker.kungal.com/stickers/KUNgal") {
		return source
	}
	return legacyStickerURLRe.ReplaceAllStringFunc(source, func(m string) string {
		sub := legacyStickerURLRe.FindStringSubmatch(m)
		pack, _ := strconv.Atoi(sub[1])
		position, _ := strconv.Atoi(sub[2])
		hash, ok := LegacyStickerHash(pack, position)
		if !ok {
			return m
		}
		// _320 is what the live sticker picker inserts; the main and _320 derivatives both 200.
		return "/image/" + hash + "_320"
	})
}

func NormalizeImageRefs(source string) string {
	return absoluteImageURLRe.ReplaceAllStringFunc(source, func(m string) string {
		sub := absoluteImageURLRe.FindStringSubmatch(m)
		host, aa, bb, hash, variant := sub[1], sub[2], sub[3], sub[4], sub[5]
		if !isImageServiceHost(host) {
			return m
		}
		// Shards must be the hash's own, or a coincidentally-shaped URL on another site would fold.
		if aa != hash[:2] || bb != hash[2:4] {
			return m
		}
		return "/image/" + hash + variant
	})
}

func NormalizeStoredContent(source string) string {
	return NormalizeImageRefs(ResolveLegacyStickerRefs(source))
}
