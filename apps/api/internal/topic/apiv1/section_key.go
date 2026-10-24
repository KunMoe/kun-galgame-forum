package apiv1

import (
	"slices"

	"github.com/danielgtaylor/huma/v2"
)

var sectionKeys = []string{
	"galgame-discussion", "galgame-walkthrough", "review", "news",
	"resource-seeking", "resource-release",
	"tech-help", "game-running", "development",
	"acgn", "daily", "site",
}

// Migration 216 retired these keys, but App builds pinned to an older spec
// still send them on writes and filters.
var legacySectionKeys = map[string]string{
	"g-chatting": "galgame-discussion", "g-other": "galgame-discussion",
	"g-walkthrough": "galgame-walkthrough", "g-article": "review",
	"g-news": "news", "g-releases": "news", "g-seeking": "resource-seeking",
	"t-help":    "tech-help",
	"t-android": "game-running", "t-linux": "game-running", "t-practical": "game-running",
	"t-crack": "development", "t-web": "development", "t-languages": "development",
	"t-ai": "development", "t-algorithm": "development", "t-adobe": "development",
	"t-other": "development",
	"o-anime": "acgn", "o-comics": "acgn", "o-music": "acgn", "o-novel": "acgn",
	"o-daily": "daily", "o-essay": "daily", "o-other": "daily",
	"o-forum": "site", "o-patch": "site",
}

func IsSectionKey(s string) bool {
	return slices.Contains(sectionKeys, s)
}

// NormalizeSectionKey maps a current or retired key to the current one.
func NormalizeSectionKey(s string) (string, bool) {
	if IsSectionKey(s) {
		return s, true
	}
	key, ok := legacySectionKeys[s]
	return key, ok
}

func enumSchema(values []string, description string) *huma.Schema {
	enum := make([]any, len(values))
	maxLen := 0
	for i, v := range values {
		enum[i] = v
		maxLen = max(maxLen, len(v))
	}
	return &huma.Schema{Type: huma.TypeString, Enum: enum, MaxLength: &maxLen, Description: description}
}

type TopicSectionKey string

func (TopicSectionKey) Schema(huma.Registry) *huma.Schema {
	return enumSchema(sectionKeys, "Topic section key. Hyphenated URL segment of /section/{key}. listSections gives each one's label, group and rules.")
}

type SectionSlug string

func (SectionSlug) Schema(huma.Registry) *huma.Schema {
	values := slices.Clone(sectionKeys)
	for old := range legacySectionKeys {
		values = append(values, old)
	}
	slices.Sort(values[len(sectionKeys):])
	return enumSchema(values, "Deprecated: read section instead. A section key, or one of the 27 keys retired on 2026-10-24 (g-, t- and o- prefixed), "+
		"which requests may still send and which map to their successor. Responses carry only current keys.")
}
