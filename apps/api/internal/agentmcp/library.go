package agentmcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type searchQueryInput struct {
	Query string `json:"q"`
}

type searchResourcesInput struct {
	Query  string `json:"q,omitempty"`
	WorkID int64  `json:"work_id,omitempty"`
	Sort   string `json:"sort,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type searchRatingsInput struct {
	WorkID int64  `json:"work_id,omitempty"`
	Sort   string `json:"sort,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type searchToolsetsInput struct {
	Query    string `json:"q,omitempty"`
	Type     string `json:"toolset_type,omitempty"`
	Platform string `json:"platform,omitempty"`
	Sort     string `json:"sort,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

var (
	resourceSorts = map[string]string{"newest": "created_desc", "downloads": "download_desc"}
	ratingSorts   = map[string]string{"newest": "created_desc", "popular": "view_desc", "highest": "overall_desc", "lowest": "overall_asc"}
	toolsetSorts  = map[string]string{"updated": "resource_updated_desc", "views": "view_desc", "newest": "created_desc"}
)

func enumOf(p *jsonschema.Schema, keys ...string) {
	for _, k := range keys {
		p.Enum = append(p.Enum, k)
	}
}

func limitSchema(p *jsonschema.Schema) {
	one, top := 1.0, float64(workLimitMax)
	p.Minimum, p.Maximum = &one, &top
	p.Default = json.RawMessage(strconv.Itoa(workLimit))
	p.Description = "返回几条，默认 8，最多 12。"
}

func searchResourcesTool() *mcp.Tool {
	schema, err := jsonschema.For[searchResourcesInput](nil)
	if err != nil {
		panic(err)
	}
	p := schema.Properties
	p["q"].Description = "在资源说明和作品名里搜索的关键词。不填就按排序列出全站资源。"
	p["work_id"].Description = "只看这部作品的资源（作品 id）。问一部作品的资源时更推荐直接用 get_galgame。"
	enumOf(p["sort"], "newest", "downloads")
	p["sort"].Description = "排序。newest=最新发布（默认），downloads=下载最多。填了 work_id 时按发布时间。"
	limitSchema(p["limit"])
	return &mcp.Tool{
		Name:  "search_resources",
		Title: "搜索本站下载资源",
		Description: "列出鲲 Galgame 论坛（本站）的 Galgame 下载资源：游戏本体、汉化补丁、合集、免CD、攻略、原声带、CG 等，" +
			"每条有资源 id、所属作品、类型、平台、运行方式、语言、大小、下载量和更新时间。" +
			"用户问「最近更新了什么资源」「下载最多的资源」，或按资源说明里的词找资源时调用。下载链接不在这里，请让用户打开资源页面。",
		InputSchema: schema,
		Annotations: readOnly(),
	}
}

func searchRatingsTool() *mcp.Tool {
	schema, err := jsonschema.For[searchRatingsInput](nil)
	if err != nil {
		panic(err)
	}
	p := schema.Properties
	p["work_id"].Description = "只看这部作品的评价（作品 id）。不填就是全站的评价。"
	enumOf(p["sort"], "newest", "popular", "highest", "lowest")
	p["sort"].Description = "排序。newest=最新（默认），popular=浏览最多，highest=总评最高，lowest=总评最低。"
	limitSchema(p["limit"])
	return &mcp.Tool{
		Name:  "search_ratings",
		Title: "查看玩家评价",
		Description: "列出鲲 Galgame 论坛（本站）玩家写的 Galgame 评价（Galgame 评分）：作者、总评（10 分制）、推荐程度、游玩进度和短评。" +
			"用户问某部作品玩家评价怎么样、口碑如何、最近大家在评什么、哪些作品评价最高时调用。短评含剧透时会省略。",
		InputSchema: schema,
		Annotations: readOnly(),
	}
}

func searchUsersTool() *mcp.Tool {
	schema, err := jsonschema.For[searchQueryInput](nil)
	if err != nil {
		panic(err)
	}
	minLen, maxLen := 1, 107
	schema.Properties["q"].MinLength, schema.Properties["q"].MaxLength = &minLen, &maxLen
	schema.Properties["q"].Description = "用户名或其中的一部分。"
	return &mcp.Tool{
		Name:        "search_users",
		Title:       "搜索论坛用户",
		Description: "按用户名搜索鲲 Galgame 论坛的用户，返回用户 id、签名、身份和发帖数。用户问论坛里某个人、想找某位用户的主页时调用。",
		InputSchema: schema,
		Annotations: readOnly(),
	}
}

func searchToolsetsTool() *mcp.Tool {
	schema, err := jsonschema.For[searchToolsetsInput](nil)
	if err != nil {
		panic(err)
	}
	p := schema.Properties
	p["q"].Description = "工具名里的关键词，比如 Kirikiroid2、Locale Emulator、LunaTranslator、GARbro。不填就按其他条件列出。"
	enumOf(p["toolset_type"], "emulator", "translator", "extractor", "converter", "debug", "launcher", "script", "docs", "others")
	p["toolset_type"].Description = "工具类型。emulator=模拟器 / 转区，translator=翻译，extractor=解包，converter=转换，debug=调试，launcher=启动器 / 管理器，script=脚本，docs=文档教程。"
	enumOf(p["platform"], "windows", "mac", "linux", "emulator", "others")
	p["platform"].Description = "工具运行的平台。"
	enumOf(p["sort"], "updated", "views", "newest")
	p["sort"].Description = "排序。updated=最近更新（默认），views=浏览最多，newest=最新收录。"
	limitSchema(p["limit"])
	return &mcp.Tool{
		Name:  "search_toolsets",
		Title: "搜索 Gal 工具",
		Description: "搜索鲲 Galgame 论坛（本站）的 Gal 工具资源：转区、模拟器、机翻 / 翻译器、解包、存档、启动器等玩 Galgame 用的工具和教程。" +
			"用户问游戏乱码、怎么转区、怎么翻译、怎么解包、用什么工具时调用；游戏运行问题的讨论帖用 search_topics。",
		InputSchema: schema,
		Annotations: readOnly(),
	}
}

func (s *server) searchResources(ctx context.Context, req *mcp.CallToolRequest, in searchResourcesInput) (*mcp.CallToolResult, Cards, error) {
	v, err := s.viewer(ctx, req)
	if err != nil {
		return nil, Cards{}, err
	}
	sort, ok := resourceSorts[orDefault(in.Sort, "newest")]
	if !ok {
		return nil, Cards{}, fmt.Errorf("不认识的排序 %q", in.Sort)
	}
	limit := workFilter{Limit: in.Limit}.limit()
	var res page[resource]
	if in.WorkID > 0 {
		path := "/works/" + strconv.FormatInt(in.WorkID, 10) + "/resources"
		err = s.API.get(ctx, path, url.Values{"limit": {strconv.Itoa(limit)}}, &res)
	} else {
		q := url.Values{"limit": {strconv.Itoa(limit)}, "sort": {sort}}
		if t := strings.TrimSpace(in.Query); t != "" {
			q.Set("q", t)
		}
		err = s.API.get(ctx, "/galgame-resources", nsfwParam(q, v.includeNSFW), &res)
	}
	if errors.Is(err, errNotFound) {
		return nil, Cards{}, fmt.Errorf("本站没有 id 为 %d 的作品。", in.WorkID)
	}
	if err != nil {
		return nil, Cards{}, apiFail("search resources", err)
	}
	if len(res.Items) == 0 {
		return textResult("没有找到符合条件的资源。"), Cards{Cards: []Card{}}, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "符合条件的资源共 %d 个，列出 %d 个：\n", res.Total, len(res.Items))
	cards := make([]Card, 0, len(res.Items))
	for _, r := range res.Items {
		work := ""
		if r.Work != nil {
			work = fmt.Sprintf("《%s》（作品 #%s）", r.Work.zhName(), r.Work.ID)
		}
		fmt.Fprintf(&b, "- %s %s\n", work, resourceLine(r))
		cards = append(cards, resourceCard(r, v.includeNSFW))
	}
	return textResult(b.String()), Cards{Cards: cards}, nil
}

func (s *server) searchRatings(ctx context.Context, req *mcp.CallToolRequest, in searchRatingsInput) (*mcp.CallToolResult, Cards, error) {
	v, err := s.viewer(ctx, req)
	if err != nil {
		return nil, Cards{}, err
	}
	sort, ok := ratingSorts[orDefault(in.Sort, "newest")]
	if !ok {
		return nil, Cards{}, fmt.Errorf("不认识的排序 %q", in.Sort)
	}
	q := url.Values{"limit": {strconv.Itoa(workFilter{Limit: in.Limit}.limit())}, "sort": {sort}}
	if in.WorkID > 0 {
		q.Set("work_id", strconv.FormatInt(in.WorkID, 10))
	}
	var res page[rating]
	if err := s.API.get(ctx, "/ratings", nsfwParam(q, v.includeNSFW), &res); err != nil {
		return nil, Cards{}, apiFail("search ratings", err)
	}
	if len(res.Items) == 0 {
		return textResult("还没有符合条件的玩家评价。"), Cards{Cards: []Card{}}, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "玩家评价共 %d 篇，列出 %d 篇：\n", res.Total, len(res.Items))
	cards := make([]Card, 0, len(res.Items))
	for _, r := range res.Items {
		work := ""
		if r.Work != nil {
			work = fmt.Sprintf("《%s》（作品 #%s）", r.Work.zhName(), r.Work.ID)
		}
		fmt.Fprintf(&b, "- %s %s\n", work, ratingLine(r))
		cards = append(cards, ratingCard(r, v.includeNSFW))
	}
	return textResult(b.String()), Cards{Cards: cards}, nil
}

func (s *server) searchUsers(ctx context.Context, req *mcp.CallToolRequest, in searchQueryInput) (*mcp.CallToolResult, Cards, error) {
	if _, err := s.viewer(ctx, req); err != nil {
		return nil, Cards{}, err
	}
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return nil, Cards{}, errors.New("q 不能为空。")
	}
	var res page[userHit]
	if err := s.API.get(ctx, "/search/users", url.Values{"q": {q}, "limit": {"8"}}, &res); err != nil {
		return nil, Cards{}, apiFail("search users", err)
	}
	if len(res.Items) == 0 {
		return textResult("没有找到名字含「" + q + "」的用户。"), Cards{Cards: []Card{}}, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "名字含「%s」的用户，列出 %d 位：\n", q, len(res.Items))
	cards := make([]Card, 0, len(res.Items))
	for _, u := range res.Items {
		fmt.Fprintf(&b, "- 用户 #%s %s%s · 话题 %d · 回复 %d", u.ID, u.Name, roleText(u.Roles), u.TopicCount, u.ReplyCount)
		if bio := strings.TrimSpace(u.Bio); bio != "" {
			fmt.Fprintf(&b, " · 签名：%s", bio)
		}
		b.WriteString("\n")
		cards = append(cards, userCard(u))
	}
	return textResult(b.String()), Cards{Cards: cards}, nil
}

func (s *server) searchToolsets(ctx context.Context, req *mcp.CallToolRequest, in searchToolsetsInput) (*mcp.CallToolResult, Cards, error) {
	if _, err := s.viewer(ctx, req); err != nil {
		return nil, Cards{}, err
	}
	sort, ok := toolsetSorts[orDefault(in.Sort, "updated")]
	if !ok {
		return nil, Cards{}, fmt.Errorf("不认识的排序 %q", in.Sort)
	}
	q := url.Values{"limit": {strconv.Itoa(workFilter{Limit: in.Limit}.limit())}, "sort": {sort}}
	if t := strings.TrimSpace(in.Query); t != "" {
		q.Set("q", t)
	}
	if in.Type != "" {
		q.Set("toolset_type", in.Type)
	}
	if in.Platform != "" {
		q.Set("platform", in.Platform)
	}
	var res page[toolset]
	if err := s.API.get(ctx, "/toolsets", q, &res); err != nil {
		return nil, Cards{}, apiFail("search toolsets", err)
	}
	if len(res.Items) == 0 {
		return textResult("本站没有符合条件的 Gal 工具。"), Cards{Cards: []Card{}}, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "符合条件的 Gal 工具共 %d 个，列出 %d 个：\n", res.Total, len(res.Items))
	cards := make([]Card, 0, len(res.Items))
	for _, t := range res.Items {
		fmt.Fprintf(&b, "- 工具 #%s %s%s · %s · 下载 %d · 浏览 %d\n", t.ID, t.Title, aliasText(t.Aliases, 3), toolsetMeta(t), t.DownloadCount, t.ViewCount)
		cards = append(cards, toolsetCard(t))
	}
	return textResult(b.String()), Cards{Cards: cards}, nil
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func roleText(roles []string) string {
	for _, r := range roles {
		switch r {
		case "admin":
			return "（管理员）"
		case "moderator":
			return "（版主）"
		case "creator":
			return "（创作者）"
		}
	}
	return ""
}

func toolsetMeta(t toolset) string {
	return joinNonEmpty(" · ", label(t.Type, toolsetTypeLabels), label(t.Platform, toolsetPlatformLabels), label(t.Language, languageLabels))
}

func resourceCard(r resource, includeNSFW bool) Card {
	card := Card{
		Kind: kindResource, ID: r.ID, Title: label(r.Type, resourceTypeLabels),
		Subtitle: joinNonEmpty(" · ", labels(r.Platforms, platformLabels), labels(r.Languages, languageLabels), r.Size),
		Summary:  r.Title, URL: "/galgame/resource/" + r.ID, ContentLimit: "sfw",
	}
	if r.Work != nil {
		card.Title = "《" + r.Work.zhName() + "》" + label(r.Type, resourceTypeLabels)
		var adult bool
		card.Cover, adult = coverOf(*r.Work, includeNSFW)
		card.ContentLimit = limitOf(adult)
	}
	facts := []Fact{{"类型", label(r.Type, resourceTypeLabels)}}
	if len(r.Platforms) > 0 {
		facts = append(facts, Fact{"平台", labels(r.Platforms, platformLabels)})
	}
	if len(r.Runtimes) > 0 {
		facts = append(facts, Fact{"运行", labels(r.Runtimes, runtimeLabels)})
	}
	if len(r.Languages) > 0 {
		facts = append(facts, Fact{"语言", labels(r.Languages, languageLabels)})
	}
	if r.Size != "" {
		facts = append(facts, Fact{"大小", r.Size})
	}
	facts = append(facts, Fact{"下载", strconv.Itoa(r.DownloadCount)})
	if len(r.UpdatedAt) >= 10 {
		facts = append(facts, Fact{"更新", r.UpdatedAt[:10]})
	}
	card.Facts = facts
	return card
}

func ratingCard(r rating, includeNSFW bool) Card {
	author := "已注销用户"
	if r.Author != nil && r.Author.Name != "" {
		author = r.Author.Name
	}
	card := Card{
		Kind: kindRating, ID: r.ID, Title: author + " 的评价",
		Subtitle: fmt.Sprintf("总评 %d/10 · %s", r.Overall, label(r.Recommend, ratingRecommendLabels)),
		Summary:  ratingSummary(r), URL: "/galgame-rating/" + r.ID, ContentLimit: "sfw",
		Facts: []Fact{{"总评", fmt.Sprintf("%d/10", r.Overall)}, {"推荐", label(r.Recommend, ratingRecommendLabels)}},
	}
	if r.PlayStatus != nil {
		card.Facts = append(card.Facts, Fact{"进度", label(*r.PlayStatus, playStatusLabels)})
	}
	if r.Work != nil {
		card.Title = "《" + r.Work.zhName() + "》· " + author + " 的评价"
		var adult bool
		card.Cover, adult = coverOf(*r.Work, includeNSFW)
		card.ContentLimit = limitOf(adult)
	}
	return card
}

func userCard(u userHit) Card {
	bio, _ := clip(strings.TrimSpace(u.Bio), 60)
	return Card{
		Kind: kindUser, ID: u.ID, Title: u.Name, Subtitle: bio, Cover: https(u.Avatar),
		Facts: []Fact{{"话题", strconv.Itoa(u.TopicCount)}, {"回复", strconv.Itoa(u.ReplyCount)}},
		URL:   "/user/" + u.ID, ContentLimit: "sfw",
	}
}

func toolsetCard(t toolset) Card {
	facts := []Fact{{"类型", label(t.Type, toolsetTypeLabels)}, {"平台", label(t.Platform, toolsetPlatformLabels)},
		{"语言", label(t.Language, languageLabels)}, {"下载", strconv.Itoa(t.DownloadCount)}}
	if t.Practicality != nil {
		facts = append(facts, Fact{"实用度", fmt.Sprintf("%.1f", *t.Practicality)})
	}
	return Card{
		Kind: kindToolset, ID: t.ID, Title: t.Title, Subtitle: toolsetMeta(t),
		Facts: facts, URL: "/toolset/" + t.ID, ContentLimit: "sfw",
	}
}
