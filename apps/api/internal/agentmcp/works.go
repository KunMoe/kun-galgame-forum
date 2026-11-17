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

const (
	workLimit    = 8
	workLimitMax = 12
	introRunes   = 300
	tagLimit     = 10
	resourceList = 20
)

var workSorts = map[string]string{
	"updated": "resource_updated_desc", "popular": "view_7d_desc", "views": "view_desc",
	"rating": "rating_desc", "newest": "release_date_desc", "added": "created_desc",
}

const detailHint = "这里只有概要。回答某一部作品有哪些资源、用什么引擎、玩家评价如何之前，先对它调用 get_galgame。"

var searchSorts = map[string]string{
	"popular": "popularity_desc", "views": "popularity_desc", "newest": "released_desc", "updated": "updated_desc",
}

var sortLabels = map[string]string{
	"updated": "资源更新时间", "popular": "近 7 天热度", "views": "总浏览量", "rating": "本站评分",
	"newest": "发售日期", "added": "收录时间", "relevance": "相关度",
}

type workFilter struct {
	Platform string `json:"platform,omitempty"`
	Mobile   bool   `json:"mobile,omitempty"`
	Language string `json:"language,omitempty"`
	Type     string `json:"resource_type,omitempty"`
	Sort     string `json:"sort,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type searchGalgamesInput struct {
	Query string `json:"q,omitempty"`
	workFilter
	TagIDs    []int64 `json:"tag_ids,omitempty"`
	CompanyID int64   `json:"company_id,omitempty"`
}

type idInput struct {
	ID int64 `json:"id"`
}

type engineWorksInput struct {
	ID int64 `json:"id"`
	workFilter
}

func describeFilter(p map[string]*jsonschema.Schema) {
	for _, k := range platformKeys {
		p["platform"].Enum = append(p["platform"].Enum, k)
	}
	p["platform"].Description = "只要有这个平台资源的作品。电脑 / PC=win，安卓=and，iPhone / iPad=ios，Mac=mac，Linux=lin，Switch=swi，PSP=psp，其他见枚举。" +
		"问的是某一种手机系统时用这个，不要用 mobile。"
	p["mobile"].Description = "只要手机上能玩的作品：有安卓 / iOS 资源，或能用手机模拟器（Kirikiroid2、Tyranor、Winlator、ONScripter、JoiPlay 等，基本只有安卓版）运行，" +
		"和网站的「手机可玩」一致。只在用户没说是哪种手机时用；问 iPhone 一定用 platform=ios。"
	for _, k := range languageKeys {
		p["language"].Enum = append(p["language"].Enum, k)
	}
	p["language"].Description = "只要有这种语言资源的作品。zh-cn=简体中文（问「有没有汉化」「中文版」时用），zh-tw=繁体中文，ja-jp=日语，en-us=英语。"
	for _, k := range typeKeys {
		p["resource_type"].Enum = append(p["resource_type"].Enum, k)
	}
	p["resource_type"].Description = "只要有这类资源的作品。game=游戏本体，patch=补丁（汉化补丁等），collection=合集，crack_fix=免CD / 修正，" +
		"walkthrough=攻略，ost=原声带，cg=鉴赏 CG，tool=工具，其他见枚举。"
	for _, k := range []string{"updated", "popular", "views", "rating", "newest", "added"} {
		p["sort"].Enum = append(p["sort"].Enum, k)
	}
	p["sort"].Description = "排序。updated=最近更新资源（默认），popular=近 7 天热门，views=总浏览最多，rating=本站评分最高，newest=发售日最新，added=最近收录。" +
		"填了 q 时默认按相关度，rating 和 added 不可用。"
	one, top := 1.0, float64(workLimitMax)
	p["limit"].Minimum, p["limit"].Maximum = &one, &top
	p["limit"].Default = json.RawMessage(strconv.Itoa(workLimit))
	p["limit"].Description = "返回几部，默认 8，最多 12。"
}

func searchGalgamesTool() *mcp.Tool {
	schema, err := jsonschema.For[searchGalgamesInput](nil)
	if err != nil {
		panic(err)
	}
	p := schema.Properties
	maxQuery := 107
	p["q"].MaxLength = &maxQuery
	p["q"].Description = "作品名或名字里的关键词，中文、日文、英文名和常用简称都能匹配。不填就按其他条件列出本站有资源的作品。" +
		"q 只匹配作品名：找某个会社 / 社团的作品时，先用 search_entities 查到会社 id，再填 company_id，不要把会社名填进 q。"
	describeFilter(p)
	p["tag_ids"].Description = "作品标签 id（来自 search_tags），要同时带有全部这些标签，最多 10 个。"
	tags := tagLimit
	p["tag_ids"].MaxItems = &tags
	p["company_id"].Description = "只要这个会社的作品，会社 id 来自 search_entities。"
	return &mcp.Tool{
		Name:  "search_galgames",
		Title: "搜索本站的 Galgame",
		Description: "搜索鲲 Galgame 论坛（本站）的 Galgame，以及本站有没有它们的下载资源（游戏本体、汉化补丁、合集等）。" +
			"可以按名字、资源平台、手机能不能玩、资源语言、资源类型、标签、会社筛选，按资源更新、热度、浏览、本站评分、发售日排序；所有条件都可以不填。" +
			"用户问本站有没有某部作品、能不能下载、有没有汉化 / 中文、能不能在手机或某个平台玩、最近更新了什么资源、本站热门或高分作品时调用。" +
			"返回总数和每部作品的 id、名字、会社、发售日、资源平台和语言、本站评分；要看某一部的资源明细、引擎、评价时再调用 get_galgame。" +
			"作品 id 就是资料库的作品 id。平台照实说：安卓资源不能说成能在 iPhone 上玩。",
		InputSchema: schema,
		Annotations: readOnly(),
	}
}

func getGalgameTool() *mcp.Tool {
	schema, err := jsonschema.For[idInput](nil)
	if err != nil {
		panic(err)
	}
	one := 1.0
	schema.Properties["id"].Minimum = &one
	schema.Properties["id"].Description = "作品 id：search_galgames 结果里 # 后的数字、资料库的作品 id，或链接 /galgame/<id> 里的数字。"
	return &mcp.Tool{
		Name:  "get_galgame",
		Title: "查看本站的 Galgame",
		Description: "按 id 读取鲲 Galgame 论坛（本站）的一部 Galgame：名字和别名、会社、发售日、引擎、系列、标签、简介摘录、本站浏览 / 收藏 / 评分，" +
			"本站每个下载资源的类型、平台、运行方式、语言、版本、大小、下载量、是否失效，以及浏览最多的几篇玩家评价。" +
			"用户问到某一部作品的资源、平台、能不能在手机上玩、有没有中文、多大、用什么引擎、玩家评价怎么样时都要调用，即使前面的列表里已经出现过它；" +
			"只知道名字时先用 search_galgames 找到 id。回答只依据这里列出的资源，没有的平台就说没有。下载链接不在这里，请让用户打开作品页面下载。",
		InputSchema: schema,
		Annotations: readOnly(),
	}
}

func searchEnginesTool() *mcp.Tool {
	schema, err := jsonschema.For[searchQueryInput](nil)
	if err != nil {
		panic(err)
	}
	minLen, maxLen := 1, 64
	schema.Properties["q"].MinLength, schema.Properties["q"].MaxLength = &minLen, &maxLen
	schema.Properties["q"].Description = "引擎名或别名的一部分，比如 KiriKiri、Ren'Py、TyranoScript、Unity、NScripter。引擎名大多是英文，中文俗称要换成英文名：吉里吉里 / krkr=KiriKiri。"
	return &mcp.Tool{
		Name:  "search_engines",
		Title: "搜索游戏引擎",
		Description: "按名字搜索 Galgame 使用的游戏引擎（KiriKiri / 吉里吉里、Ren'Py、TyranoScript、Unity、NScripter、RPG Maker 等），返回引擎 id、别名和收录作品数。" +
			"用户问某个引擎、某引擎有哪些作品、某引擎的游戏能不能在手机上玩时先用它找到 id，再用 get_engine。作品用什么引擎请用 get_galgame。",
		InputSchema: schema,
		Annotations: readOnly(),
	}
}

func getEngineTool() *mcp.Tool {
	schema, err := jsonschema.For[engineWorksInput](nil)
	if err != nil {
		panic(err)
	}
	p := schema.Properties
	one := 1.0
	p["id"].Minimum = &one
	p["id"].Description = "引擎 id：search_engines 或 get_galgame 结果里 # 后的数字，或链接 /galgame/engine/<id> 里的数字。"
	describeFilter(p)
	return &mcp.Tool{
		Name:  "get_engine",
		Title: "查看游戏引擎",
		Description: "按 id 读取一个游戏引擎，并列出本站用这个引擎、且有下载资源的作品，可以按资源平台、手机可玩、语言、类型筛选和排序。" +
			"用户问「吉里吉里的游戏有哪些能在手机上玩」「Ren'Py 有什么汉化作品」这类问题时用。",
		InputSchema: schema,
		Annotations: readOnly(),
	}
}

func (s *server) searchGalgames(ctx context.Context, req *mcp.CallToolRequest, in searchGalgamesInput) (*mcp.CallToolResult, Cards, error) {
	v, err := s.viewer(ctx, req)
	if err != nil {
		return nil, Cards{}, err
	}
	if err := in.check(); err != nil {
		return nil, Cards{}, err
	}
	if len(in.TagIDs) > tagLimit {
		return nil, Cards{}, fmt.Errorf("tag_ids 最多 %d 个", tagLimit)
	}
	q := strings.TrimSpace(in.Query)
	if q == "" {
		params := in.params(v.includeNSFW)
		if len(in.TagIDs) > 0 {
			params.Set("tag_ids", joinIDs(in.TagIDs))
		}
		if in.CompanyID > 0 {
			params.Set("company_id", strconv.FormatInt(in.CompanyID, 10))
		}
		var res page[workSummary]
		if err := s.API.get(ctx, "/works", params, &res); err != nil {
			return nil, Cards{}, apiFail("browse works", err)
		}
		head := filterHead(in.workFilter, "")
		if len(res.Items) == 0 {
			return textResult(head + "本站没有符合这些条件、有下载资源的作品。"), Cards{Cards: []Card{}}, nil
		}
		text, cards := s.workList(res.Items, v.includeNSFW)
		return textResult(fmt.Sprintf("%s本站符合条件的作品共 %d 部，按%s排列的前 %d 部：\n%s%s",
			head, res.Total, sortLabels[in.sortKey("updated")], len(res.Items), text, detailHint)), Cards{Cards: cards}, nil
	}

	params := url.Values{"q": {q}, "limit": {"30"}, "sort": {"relevance_desc"}}
	if key, ok := searchSorts[in.Sort]; ok {
		params.Set("sort", key)
	}
	if len(in.TagIDs) > 0 {
		params.Set("tag_ids", joinIDs(in.TagIDs))
	}
	if in.CompanyID > 0 {
		params.Set("company_id", strconv.FormatInt(in.CompanyID, 10))
	}
	var res page[workSummary]
	if err := s.API.get(ctx, "/search/works", nsfwParam(params, v.includeNSFW), &res); err != nil {
		return nil, Cards{}, apiFail("search works", err)
	}
	if len(res.Items) == 0 {
		return textResult("本站和资料库里都没有找到名字含「" + q + "」的作品，可以换个写法（日文名、简称）再试。"), Cards{Cards: []Card{}}, nil
	}
	matched := in.matching(res.Items)
	limit := in.limit()
	head := filterHead(in.workFilter, q)
	if len(matched) == 0 {
		shown := res.Items[:min(limit, len(res.Items))]
		text, cards := s.workList(shown, v.includeNSFW)
		return textResult(fmt.Sprintf("%s没有符合的作品。不加资源条件时，名字含「%s」的作品有 %d 部：\n%s也就是说本站有这些作品，但没有符合条件的资源。",
			head, q, res.Total, text)), Cards{Cards: cards}, nil
	}
	shown := matched[:min(limit, len(matched))]
	text, cards := s.workList(shown, v.includeNSFW)
	return textResult(fmt.Sprintf("%s名字含「%s」的作品，列出 %d 部：\n%s%s", head, q, len(shown), text, detailHint)), Cards{Cards: cards}, nil
}

func (s *server) getGalgame(ctx context.Context, req *mcp.CallToolRequest, in idInput) (*mcp.CallToolResult, Cards, error) {
	v, err := s.viewer(ctx, req)
	if err != nil {
		return nil, Cards{}, err
	}
	if in.ID <= 0 {
		return nil, Cards{}, errors.New("id 必须是正整数")
	}
	id := strconv.FormatInt(in.ID, 10)
	var w workDetail
	if err := s.API.get(ctx, "/works/"+id, nsfwParam(url.Values{}, v.includeNSFW), &w); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, Cards{}, fmt.Errorf("本站没有 id 为 %d 的作品，或者提问用户的内容设置看不到它。", in.ID)
		}
		return nil, Cards{}, apiFail("get work", err)
	}
	var resources page[resource]
	if err := s.API.get(ctx, "/works/"+id+"/resources", url.Values{"limit": {strconv.Itoa(resourceList)}}, &resources); err != nil {
		return nil, Cards{}, apiFail("work resources", err)
	}
	var ratings page[rating]
	ratingQuery := url.Values{"work_id": {id}, "sort": {"view_desc"}, "limit": {"3"}}
	if err := s.API.get(ctx, "/ratings", nsfwParam(ratingQuery, v.includeNSFW), &ratings); err != nil {
		return nil, Cards{}, apiFail("work ratings", err)
	}
	card := s.workCard(w.workSummary, v.includeNSFW)
	card.Summary = introOf(w.Intros, 120)
	return textResult(workDetailText(w, resources, ratings, v.includeNSFW)), Cards{Cards: []Card{card}}, nil
}

func (s *server) searchEngines(ctx context.Context, req *mcp.CallToolRequest, in searchQueryInput) (*mcp.CallToolResult, Cards, error) {
	if _, err := s.viewer(ctx, req); err != nil {
		return nil, Cards{}, err
	}
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return nil, Cards{}, errors.New("q 不能为空。")
	}
	var res page[engine]
	if err := s.API.get(ctx, "/engines", url.Values{"q": {q}, "limit": {"10"}}, &res); err != nil {
		return nil, Cards{}, apiFail("search engines", err)
	}
	if len(res.Items) == 0 {
		return textResult("没有找到名字含「" + q + "」的引擎。引擎名大多是英文，换成英文名再试，比如吉里吉里=KiriKiri。"), Cards{Cards: []Card{}}, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "名字含「%s」的引擎共 %d 个：\n", q, res.Total)
	cards := make([]Card, 0, len(res.Items))
	for _, e := range res.Items {
		fmt.Fprintf(&b, "#%s %s%s · 资料库收录 %d 部作品\n", e.ID, e.DisplayName, aliasText(e.Aliases, 4), e.WorkCount)
		cards = append(cards, engineCard(e))
	}
	return textResult(b.String()), Cards{Cards: cards}, nil
}

func (s *server) getEngine(ctx context.Context, req *mcp.CallToolRequest, in engineWorksInput) (*mcp.CallToolResult, Cards, error) {
	v, err := s.viewer(ctx, req)
	if err != nil {
		return nil, Cards{}, err
	}
	if in.ID <= 0 {
		return nil, Cards{}, errors.New("id 必须是正整数")
	}
	if err := in.check(); err != nil {
		return nil, Cards{}, err
	}
	id := strconv.FormatInt(in.ID, 10)
	var e engine
	if err := s.API.get(ctx, "/engines/"+id, nil, &e); err != nil {
		if errors.Is(err, errNotFound) {
			return nil, Cards{}, fmt.Errorf("没有 id 为 %d 的引擎。", in.ID)
		}
		return nil, Cards{}, apiFail("get engine", err)
	}
	params := in.params(v.includeNSFW)
	// The engine lane takes one value per resource filter, not a list.
	params.Del("resource_platforms")
	params.Del("resource_runtimes")
	params.Del("resource_languages")
	switch {
	case in.Platform != "":
		params.Set("resource_platform", in.Platform)
	case in.Mobile:
		params.Set("resource_platform", "and")
		params.Set("resource_runtime", mobileRuntimes)
	}
	if in.Language != "" {
		params.Set("resource_language", in.Language)
	}
	var res page[workSummary]
	if err := s.API.get(ctx, "/engines/"+id+"/works", params, &res); err != nil {
		return nil, Cards{}, apiFail("engine works", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "#%s %s%s · 资料库收录 %d 部作品\n", e.ID, e.DisplayName, aliasText(e.Aliases, 6), e.WorkCount)
	b.WriteString(filterHead(in.workFilter, ""))
	cards := []Card{engineCard(e)}
	if len(res.Items) == 0 {
		b.WriteString("本站没有符合条件、有下载资源的这个引擎的作品。")
		return textResult(b.String()), Cards{Cards: cards}, nil
	}
	text, works := s.workList(res.Items, v.includeNSFW)
	fmt.Fprintf(&b, "本站有资源的作品共 %d 部，按%s排列的前 %d 部：\n%s%s", res.Total, sortLabels[in.sortKey("updated")], len(res.Items), text, detailHint)
	return textResult(b.String()), Cards{Cards: append(cards, works...)}, nil
}

func (f workFilter) check() error {
	if f.Platform != "" && !validKey(f.Platform, platformKeys) {
		return fmt.Errorf("不认识的平台 %q", f.Platform)
	}
	if f.Language != "" && !validKey(f.Language, languageKeys) {
		return fmt.Errorf("不认识的语言 %q", f.Language)
	}
	if f.Type != "" && !validKey(f.Type, typeKeys) {
		return fmt.Errorf("不认识的资源类型 %q", f.Type)
	}
	if _, ok := workSorts[f.Sort]; f.Sort != "" && !ok {
		return fmt.Errorf("不认识的排序 %q", f.Sort)
	}
	return nil
}

func (f workFilter) limit() int {
	if f.Limit <= 0 {
		return workLimit
	}
	return min(f.Limit, workLimitMax)
}

func (f workFilter) sortKey(fallback string) string {
	if f.Sort == "" {
		return fallback
	}
	return f.Sort
}

func (f workFilter) params(includeNSFW bool) url.Values {
	q := url.Values{"limit": {strconv.Itoa(f.limit())}, "sort": {workSorts[f.sortKey("updated")]}}
	switch {
	case f.Platform != "":
		q.Set("resource_platforms", f.Platform)
	case f.Mobile:
		q.Set("resource_platforms", mobilePlatforms)
		q.Set("resource_runtimes", mobileRuntimes)
	}
	if f.Language != "" {
		q.Set("resource_languages", f.Language)
	}
	if f.Type != "" {
		q.Set("resource_type", f.Type)
	}
	return nsfwParam(q, includeNSFW)
}

// Catalog search knows nothing of forum resources, so a named search is
// narrowed afterwards by the platforms and languages each summary carries.
func (f workFilter) matching(items []workSummary) []workSummary {
	if f.Platform == "" && !f.Mobile && f.Language == "" {
		return items
	}
	out := make([]workSummary, 0, len(items))
	for _, w := range items {
		switch {
		case f.Platform != "" && !validKey(f.Platform, w.ResourcePlatforms):
		case f.Mobile && !validKey("and", w.ResourcePlatforms) && !validKey("ios", w.ResourcePlatforms):
		case f.Language != "" && !validKey(f.Language, w.ResourceLanguages):
		default:
			out = append(out, w)
		}
	}
	return out
}

func filterHead(f workFilter, q string) string {
	var parts []string
	if q != "" {
		parts = append(parts, "名字含「"+q+"」")
	}
	if f.Platform != "" {
		parts = append(parts, "平台 "+label(f.Platform, platformLabels))
	}
	if f.Mobile {
		parts = append(parts, "手机可玩")
	}
	if f.Language != "" {
		parts = append(parts, "语言 "+label(f.Language, languageLabels))
	}
	if f.Type != "" {
		parts = append(parts, "资源类型 "+label(f.Type, resourceTypeLabels))
	}
	if len(parts) == 0 {
		return ""
	}
	return "筛选：" + strings.Join(parts, "；") + "\n"
}

func joinIDs(ids []int64) string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(out, ",")
}

func aliasText(aliases []string, n int) string {
	if len(aliases) == 0 {
		return ""
	}
	return "（别名：" + strings.Join(aliases[:min(n, len(aliases))], "、") + "）"
}
