package agentmcp

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	kindGalgame  = "kungal:galgame"
	kindResource = "kungal:resource"
	kindRating   = "kungal:rating"
	kindUser     = "kungal:user"
	kindEngine   = "kungal:engine"
	kindToolset  = "kungal:toolset"
)

func limitOf(nsfw bool) string {
	if nsfw {
		return "nsfw"
	}
	return "sfw"
}

func https(img *image) string {
	if img == nil || !strings.HasPrefix(img.URL, "https://") {
		return ""
	}
	return img.URL
}

// coverOf hides an adult cover from a reader whose setting hides adult
// content, falling back to the work's safe cover.
func coverOf(w workRef, includeNSFW bool) (string, bool) {
	c := w.Cover
	if c != nil && c.Sexual != nil && *c.Sexual != "safe" && !includeNSFW {
		c = w.CoverSFW
	}
	adult := w.IsNSFW || (c != nil && c.Sexual != nil && *c.Sexual != "safe")
	return https(c), adult
}

func year(date *string) string {
	if date == nil || len(*date) < 4 {
		return ""
	}
	return (*date)[:4]
}

func (w workSummary) maker() string {
	if w.Maker == nil {
		return ""
	}
	return w.Maker.zhName()
}

func (w workSummary) score() string {
	if w.RatingScore == nil || w.RatingCount == 0 {
		return "暂无"
	}
	return fmt.Sprintf("%.1f（%d 人）", *w.RatingScore, w.RatingCount)
}

func (w workSummary) resourceLine() string {
	if len(w.ResourcePlatforms) == 0 && len(w.ResourceLanguages) == 0 {
		return "本站暂无资源"
	}
	return "资源平台 " + labels(w.ResourcePlatforms, platformLabels) + " · 语言 " + labels(w.ResourceLanguages, languageLabels)
}

func joinNonEmpty(sep string, parts ...string) string {
	kept := parts[:0:0]
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

func (s *server) workList(items []workSummary, includeNSFW bool) (string, []Card) {
	var b strings.Builder
	cards := make([]Card, 0, len(items))
	for i, w := range items {
		name := w.zhName()
		original := ""
		if w.DisplayName != name {
			original = w.DisplayName
		}
		fmt.Fprintf(&b, "%d. #%s 《%s》（%s）%s · 本站评分 %s%s\n", i+1, w.ID, name,
			joinNonEmpty(" · ", original, w.maker(), year(w.ReleaseDate)), w.resourceLine(), w.score(), nsfwMark(w.IsNSFW))
		cards = append(cards, s.workCard(w, includeNSFW))
	}
	return b.String(), cards
}

func (s *server) workCard(w workSummary, includeNSFW bool) Card {
	cover, adult := coverOf(w.workRef, includeNSFW)
	facts := []Fact{}
	add := func(label, value string) {
		if value != "" {
			facts = append(facts, Fact{label, value})
		}
	}
	if w.ReleaseDate != nil {
		add("发售", *w.ReleaseDate)
	}
	add("会社", w.maker())
	add("平台", labels(w.ResourcePlatforms, platformLabels))
	add("语言", labels(w.ResourceLanguages, languageLabels))
	add("评分", w.score())
	add("浏览", strconv.Itoa(w.ViewCount))
	return Card{
		Kind: kindGalgame, ID: w.ID, Title: w.zhName(),
		Subtitle: joinNonEmpty(" · ", w.maker(), year(w.ReleaseDate)),
		Cover:    cover, Facts: facts, URL: "/galgame/" + w.ID,
		ContentLimit: limitOf(adult), SameAs: "work:" + w.ID,
	}
}

func engineCard(e engine) Card {
	return Card{
		Kind: kindEngine, ID: e.ID, Title: e.DisplayName,
		Subtitle: fmt.Sprintf("游戏引擎 · 收录 %d 部作品", e.WorkCount),
		Facts:    []Fact{{"作品数", strconv.Itoa(e.WorkCount)}},
		URL:      "/galgame/engine/" + e.ID, ContentLimit: "sfw",
	}
}

func introOf(intros []intro, runes int) string {
	pick := ""
	for _, pref := range []string{"zh", "ja", "en"} {
		for _, in := range intros {
			if strings.HasPrefix(in.Locale, pref) && strings.TrimSpace(in.Value) != "" {
				pick = in.Value
				break
			}
		}
		if pick != "" {
			break
		}
	}
	text, cut := clip(strings.Join(strings.Fields(pick), " "), runes)
	if cut {
		text += "……"
	}
	return text
}

func workDetailText(w workDetail, resources page[resource], ratings page[rating], includeNSFW bool) string {
	var b strings.Builder
	name := w.zhName()
	fmt.Fprintf(&b, "#%s 《%s》%s\n", w.ID, name, nsfwMark(w.IsNSFW))
	if w.DisplayName != name {
		fmt.Fprintf(&b, "原名：%s\n", w.DisplayName)
	}
	if len(w.Aliases) > 0 {
		fmt.Fprintf(&b, "别名：%s\n", strings.Join(w.Aliases[:min(6, len(w.Aliases))], "、"))
	}
	companies := make([]string, 0, len(w.Companies))
	for _, c := range w.Companies {
		companies = append(companies, c.zhName())
	}
	if len(companies) == 0 && w.Maker != nil {
		companies = append(companies, w.maker())
	}
	if len(companies) > 0 {
		fmt.Fprintf(&b, "会社：%s\n", strings.Join(companies, "、"))
	}
	if w.ReleaseDate != nil {
		fmt.Fprintf(&b, "发售：%s\n", *w.ReleaseDate)
	}
	if len(w.Engines) > 0 {
		engines := make([]string, len(w.Engines))
		for i, e := range w.Engines {
			engines[i] = fmt.Sprintf("%s（引擎 #%s）", e.DisplayName, e.ID)
		}
		fmt.Fprintf(&b, "引擎：%s\n", strings.Join(engines, "、"))
	}
	if len(w.Series) > 0 {
		series := make([]string, len(w.Series))
		for i, s := range w.Series {
			series[i] = s.zhName()
		}
		fmt.Fprintf(&b, "系列：%s\n", strings.Join(series, "、"))
	}
	tags := make([]string, 0, tagLimit)
	for _, t := range w.Tags {
		if len(tags) == tagLimit {
			break
		}
		if t.IsSexual && !includeNSFW {
			continue
		}
		tags = append(tags, t.zhName())
	}
	if len(tags) > 0 {
		fmt.Fprintf(&b, "标签：%s\n", strings.Join(tags, "、"))
	}
	if intro := introOf(w.Intros, introRunes); intro != "" {
		fmt.Fprintf(&b, "简介：%s\n", intro)
	}
	fmt.Fprintf(&b, "本站：浏览 %d · 点赞 %d · 收藏 %d · 评分 %s\n", w.ViewCount, w.LikeCount, w.FavoriteCount, w.score())

	switch {
	case w.HostedOn:
		b.WriteString("\n这部作品的原语言不是日语或中文，本站不收录它的下载资源，资源在 LetMoe（一起萌，letmoe.com）上，作品 id 相同。\n")
	case len(resources.Items) == 0:
		b.WriteString("\n本站还没有这部作品的下载资源。\n")
	default:
		fmt.Fprintf(&b, "\n本站资源共 %d 个", resources.Total)
		if resources.Total > len(resources.Items) {
			fmt.Fprintf(&b, "，列出最近的 %d 个", len(resources.Items))
		}
		b.WriteString("：\n")
		for _, r := range resources.Items {
			fmt.Fprintf(&b, "- %s\n", resourceLine(r))
		}
	}
	if len(ratings.Items) > 0 {
		b.WriteString("\n玩家评价（浏览最多的几篇）：\n")
		for _, r := range ratings.Items {
			fmt.Fprintf(&b, "- %s\n", ratingLine(r))
		}
	}
	fmt.Fprintf(&b, "\n页面：/galgame/%s", w.ID)
	return b.String()
}

func resourceLine(r resource) string {
	parts := []string{label(r.Type, resourceTypeLabels)}
	if r.Title != "" {
		parts[0] += "「" + r.Title + "」"
	}
	if r.VersionLabel != nil && *r.VersionLabel != "" {
		parts = append(parts, "版本 "+*r.VersionLabel)
	}
	if len(r.Platforms) > 0 {
		parts = append(parts, "平台 "+labels(r.Platforms, platformLabels))
	}
	if len(r.Runtimes) > 0 {
		parts = append(parts, "运行 "+labels(r.Runtimes, runtimeLabels))
	}
	if len(r.Languages) > 0 {
		parts = append(parts, labels(r.Languages, languageLabels))
	}
	if r.Size != "" {
		parts = append(parts, r.Size)
	}
	parts = append(parts, fmt.Sprintf("下载 %d", r.DownloadCount))
	if len(r.UpdatedAt) >= 10 {
		parts = append(parts, "更新 "+r.UpdatedAt[:10])
	}
	if r.State == "expired" {
		parts = append(parts, "链接已失效")
	}
	return "资源 #" + r.ID + " " + strings.Join(parts, " · ")
}

func ratingLine(r rating) string {
	author := "已注销用户"
	if r.Author != nil && r.Author.Name != "" {
		author = r.Author.Name
	}
	line := fmt.Sprintf("评价 #%s %s 总评 %d/10，%s", r.ID, author, r.Overall, label(r.Recommend, ratingRecommendLabels))
	if r.PlayStatus != nil {
		line += "，" + label(*r.PlayStatus, playStatusLabels)
	}
	if summary := ratingSummary(r); summary != "" {
		line += "：" + summary
	}
	return line
}

func ratingSummary(r rating) string {
	if r.SummarySpoils {
		return "（短评含剧透，已省略）"
	}
	text, cut := clip(strings.Join(strings.Fields(r.Summary), " "), replyRunes)
	if cut {
		text += "……"
	}
	return text
}
