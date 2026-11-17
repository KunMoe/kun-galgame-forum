package agentmcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"kun-galgame-api/internal/apiv1/content"
	searchRepo "kun-galgame-api/internal/search/repository"
	topicapiv1 "kun-galgame-api/internal/topic/apiv1"
	"kun-galgame-api/internal/topic/model"
	topicRepo "kun-galgame-api/internal/topic/repository"
	"kun-galgame-api/pkg/userclient"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	searchLimit    = 10
	excerptRunes   = 150
	bodyRunes      = 600
	replyRunes     = 200
	highlightLimit = 3
	cardKind       = "kungal:topic"
)

var cst = time.FixedZone("CST", 8*60*60)

type Card struct {
	Kind         string `json:"kind"`
	ID           string `json:"id"`
	Title        string `json:"title"`
	Subtitle     string `json:"subtitle,omitempty"`
	Cover        string `json:"cover,omitempty"`
	Summary      string `json:"summary,omitempty"`
	Facts        []Fact `json:"facts"`
	URL          string `json:"url"`
	ContentLimit string `json:"content_limit"`
	SameAs       string `json:"same_as,omitempty"`
}

type Fact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Cards struct {
	Cards []Card `json:"cards"`
}

type searchTopicsInput struct {
	Query   string `json:"query"`
	Section string `json:"section,omitempty"`
	Sort    string `json:"sort,omitempty"`
}

var topicSorts = map[string]searchRepo.TopicOrder{
	"relevance": searchRepo.OrderRelevance,
	"newest":    searchRepo.OrderNewest,
	"active":    searchRepo.OrderActive,
	"views":     searchRepo.OrderViews,
	"likes":     searchRepo.OrderLikes,
}

type getTopicInput struct {
	TopicID int `json:"topic_id"`
}

func readOnly() *mcp.ToolAnnotations {
	closed := false
	return &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &closed}
}

func searchTopicsTool() *mcp.Tool {
	schema, err := jsonschema.For[searchTopicsInput](nil)
	if err != nil {
		panic(err)
	}
	query, section, sort := schema.Properties["query"], schema.Properties["section"], schema.Properties["sort"]
	minLen, maxLen := 1, 107
	query.MinLength, query.MaxLength = &minLen, &maxLen
	query.Description = "关键词，用空格分隔，每个词都必须出现在标题或正文里。" +
		"用话题里会出现的原文写法：作品的中文名、日文名或常用简称，一次一到三个词。"
	for _, key := range topicapiv1.SectionKeys() {
		section.Enum = append(section.Enum, key)
	}
	section.Description = "只搜这个分区，不填搜全部分区。galgame-discussion=Galgame 讨论，galgame-walkthrough=Galgame 攻略，" +
		"review=长文评价，news=新作资讯，resource-seeking=寻求资源，resource-release=资源发布（已归档），tech-help=技术求助，" +
		"game-running=游戏运行（模拟器、手机、Linux 等），development=开发技术，acgn=动画漫画音乐小说，daily=日常杂谈，site=站务相关。"
	for _, key := range []string{"relevance", "newest", "active", "views", "likes"} {
		sort.Enum = append(sort.Enum, key)
	}
	sort.Default = json.RawMessage(`"relevance"`)
	sort.Description = "排序。relevance=相关度（默认），newest=最新发布，active=最近有回复，views=浏览最多，likes=点赞最多。"
	return &mcp.Tool{
		Name:  "search_topics",
		Title: "搜索论坛话题",
		Description: "搜索鲲 Galgame 论坛的话题（帖子）。论坛里有玩家对 galgame 的讨论、攻略、长文评价、新作资讯，" +
			"以及汉化、补丁、游戏运行（模拟器、手机、Linux）等问题的求助和经验。" +
			"用户想知道玩家怎么评价、怎么攻略、遇到问题怎么解决、论坛里有没有相关帖子时调用；" +
			"作品本身的资料（简介、角色、会社、标签）请用资料库工具。" +
			"按关键词匹配标题和正文，每个词都必须出现，所以关键词要少而准。" +
			"返回最多 10 条，每条有话题 id、分区、作者、日期和摘要；要看正文和高赞回复时再调用 get_topic。",
		InputSchema: schema,
		Annotations: readOnly(),
	}
}

func getTopicTool() *mcp.Tool {
	schema, err := jsonschema.For[getTopicInput](nil)
	if err != nil {
		panic(err)
	}
	one := 1.0
	schema.Properties["topic_id"].Minimum = &one
	schema.Properties["topic_id"].Description = "话题 id：search_topics 结果里 # 后的数字，或链接 /topic/<id> 里的数字。"
	return &mcp.Tool{
		Name:  "get_topic",
		Title: "查看论坛话题",
		Description: "按 id 读取鲲 Galgame 论坛的一个话题：标题、分区、作者、时间、正文节选（前几百字）、" +
			"最佳答案和点赞最多的几条回复。需要引用或总结某个话题的具体内容时调用，id 来自 search_topics 或用户给的 /topic/<id> 链接。",
		InputSchema: schema,
		Annotations: readOnly(),
	}
}

func (s *server) searchTopics(ctx context.Context, req *mcp.CallToolRequest, in searchTopicsInput) (*mcp.CallToolResult, Cards, error) {
	v, err := s.viewer(ctx, req)
	if err != nil {
		return nil, Cards{}, err
	}
	keywords := strings.Fields(in.Query)
	if len(keywords) == 0 {
		return nil, Cards{}, errors.New("query 不能为空。")
	}
	rows, total, err := s.Search.SearchTopicRowsV1(searchRepo.V1Query{
		Keywords: keywords, Limit: searchLimit, Authenticated: true, IncludeNSFW: v.includeNSFW,
		Section: in.Section, Order: topicSorts[in.Sort],
	})
	if err != nil {
		return nil, Cards{}, fail("search topics", err)
	}
	items, prob := s.Topics.Summaries(ctx, rows)
	if prob != nil {
		return nil, Cards{}, fail("search topics", prob)
	}
	if len(items) == 0 {
		return textResult(fmt.Sprintf("没有找到包含「%s」的话题。可以换个写法或减少关键词再试。", in.Query)), Cards{Cards: []Card{}}, nil
	}
	labels, err := s.Taxonomy.SectionLabels()
	if err != nil {
		return nil, Cards{}, fail("search topics", err)
	}
	bodies, err := s.bodies(items)
	if err != nil {
		return nil, Cards{}, fail("search topics", err)
	}
	created := make(map[string]time.Time, len(rows))
	for _, row := range rows {
		created[strconv.Itoa(row.ID)] = row.Created
	}

	var b strings.Builder
	fmt.Fprintf(&b, "「%s」共找到 %d 个话题，以下是前 %d 个：\n", in.Query, total, len(items))
	out := Cards{Cards: make([]Card, 0, len(items))}
	for _, item := range items {
		summary := excerpt(item, bodies[string(item.ID)])
		card := topicCard(item, labels, created[string(item.ID)], summary)
		out.Cards = append(out.Cards, card)
		fmt.Fprintf(&b, "\n#%s %s\n  %s · 回复 %d · 点赞 %d%s\n", item.ID, item.Title, card.Subtitle,
			item.ReplyCount, item.LikeCount, nsfwMark(item.IsNSFW))
		if summary != "" {
			fmt.Fprintf(&b, "  %s\n", summary)
		}
	}
	return textResult(b.String()), out, nil
}

func (s *server) getTopic(ctx context.Context, req *mcp.CallToolRequest, in getTopicInput) (*mcp.CallToolResult, Cards, error) {
	v, err := s.viewer(ctx, req)
	if err != nil {
		return nil, Cards{}, err
	}
	notVisible := fmt.Errorf("话题 #%d 不存在，或提问用户无权查看。", in.TopicID)
	topic, prob := s.Topics.ReadableTopic(ctx, in.TopicID, v.user)
	if prob != nil {
		if prob.Status == http.StatusNotFound {
			return nil, Cards{}, notVisible
		}
		return nil, Cards{}, fail("get topic", prob)
	}
	if topic.IsNSFW && !v.includeNSFW {
		return nil, Cards{}, fmt.Errorf("话题 #%d 是成人内容，提问用户的内容设置不显示成人内容。", in.TopicID)
	}
	items, prob := s.Topics.Summaries(ctx, []topicRepo.TopicKeysetRow{keysetRow(topic)})
	if prob != nil {
		return nil, Cards{}, fail("get topic", prob)
	}
	if len(items) == 0 {
		return nil, Cards{}, notVisible
	}
	item := items[0]
	labels, err := s.Taxonomy.SectionLabels()
	if err != nil {
		return nil, Cards{}, fail("get topic", err)
	}
	replies, authors, err := s.highlights(ctx, topic)
	if err != nil {
		return nil, Cards{}, fail("get topic", err)
	}

	body, cut := clip(plain(topic.Content, bodyRunes+1), bodyRunes)
	card := topicCard(item, labels, topic.CreatedAt, excerpt(item, topic.Content))

	var b strings.Builder
	fmt.Fprintf(&b, "#%d %s\n%s · 浏览 %d · 回复 %d · 点赞 %d%s\n", topic.ID, topic.Title, card.Subtitle,
		topic.View, topic.ReplyCount, topic.LikeCount, nsfwMark(topic.IsNSFW))
	if topic.Status == 1 {
		b.WriteString("（这个话题已被隐藏，只有作者和管理员能看到）\n")
	}
	if topic.AISummary != nil && *topic.AISummary != "" {
		fmt.Fprintf(&b, "\nAI 摘要：%s\n", *topic.AISummary)
	}
	fmt.Fprintf(&b, "\n正文：\n%s", body)
	if cut {
		b.WriteString("……（正文较长，只截取了开头）")
	}
	b.WriteString("\n")
	if len(replies) > 0 {
		b.WriteString("\n精选回复：\n")
		for _, r := range replies {
			text, cut := clip(plain(r.Content, replyRunes+1), replyRunes)
			if cut {
				text += "……"
			}
			tag := fmt.Sprintf("%d 赞", r.LikeCount)
			if topic.BestAnswerID != nil && *topic.BestAnswerID == r.ID {
				tag = "最佳答案，" + tag
			}
			fmt.Fprintf(&b, "- %d 楼 %s（%s）：%s\n", r.Floor, authors[r.UserID], tag, text)
		}
	}
	return textResult(b.String()), Cards{Cards: []Card{card}}, nil
}

func (s *server) bodies(items []topicapiv1.TopicSummary) (map[string]string, error) {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		if item.AISummary == nil {
			id, _ := strconv.Atoi(string(item.ID))
			ids = append(ids, id)
		}
	}
	out := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	topics, err := s.Rows.FindByIDs(ids)
	if err != nil {
		return nil, err
	}
	for _, t := range topics {
		out[strconv.Itoa(t.ID)] = t.Content
	}
	return out, nil
}

func (s *server) highlights(ctx context.Context, topic *model.Topic) ([]model.TopicReply, map[int]string, error) {
	rows, err := s.Replies.BestAndMostLikedReplies(topic.ID, topic.BestAnswerID, highlightLimit)
	if err != nil || len(rows) == 0 {
		return nil, nil, err
	}
	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.UserID
	}
	users, err := s.Users.Users(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	kept := rows[:0]
	names := make(map[int]string, len(rows))
	for _, r := range rows {
		u, ok := users[r.UserID]
		if ok && !userclient.IsRenderable(u) {
			continue
		}
		names[r.UserID] = displayName(u.Name)
		kept = append(kept, r)
	}
	return kept, names, nil
}

func keysetRow(t *model.Topic) topicRepo.TopicKeysetRow {
	return topicRepo.TopicKeysetRow{
		ID: t.ID, Title: t.Title, View: t.View, Status: t.Status, IsNSFW: t.IsNSFW,
		LikeCount: t.LikeCount, ReplyCount: t.ReplyCount, CommentCount: t.CommentCount,
		BestAnswerID: t.BestAnswerID, StatusUpdateTime: t.StatusUpdateTime, Created: t.CreatedAt,
		UpvoteTime: t.UpvoteTime, Edited: t.Edited, CoverImages: t.CoverImages, UserID: t.UserID,
		Category: t.Category, FavoriteCount: t.FavoriteCount, UpvoteCount: t.UpvoteCount, AISummary: t.AISummary,
	}
}

func topicCard(item topicapiv1.TopicSummary, labels map[string]string, created time.Time, summary string) Card {
	section := labels[string(item.Section)]
	if section == "" {
		section = string(item.Section)
	}
	author := ""
	if item.Author.Name != nil {
		author = *item.Author.Name
	}
	author = displayName(author)
	date := created.In(cst).Format("2006-01-02")
	card := Card{
		Kind:     cardKind,
		ID:       string(item.ID),
		Title:    item.Title,
		Subtitle: section + " · " + author + " · " + date,
		Summary:  summary,
		Facts: []Fact{
			{"版块", section},
			{"作者", author},
			{"发布", date},
			{"回复", strconv.Itoa(item.ReplyCount)},
			{"浏览", strconv.Itoa(item.ViewCount)},
			{"点赞", strconv.Itoa(item.LikeCount)},
		},
		URL:          "/topic/" + string(item.ID),
		ContentLimit: "sfw",
	}
	if item.IsNSFW {
		card.ContentLimit = "nsfw"
	}
	if len(item.CoverImages) > 0 {
		card.Cover = item.CoverImages[0].URL
	}
	return card
}

func excerpt(item topicapiv1.TopicSummary, body string) string {
	if item.AISummary != nil && *item.AISummary != "" {
		return *item.AISummary
	}
	text, cut := clip(plain(body, excerptRunes+1), excerptRunes)
	if cut {
		text += "……"
	}
	return text
}

func plain(source string, maxRunes int) string {
	return strings.Join(strings.Fields(content.PlainText(source, maxRunes)), " ")
}

func clip(s string, n int) (string, bool) {
	if utf8.RuneCountInString(s) <= n {
		return s, false
	}
	return string([]rune(s)[:n]), true
}

func displayName(name string) string {
	if name == "" {
		return "已注销用户"
	}
	return name
}

func nsfwMark(nsfw bool) string {
	if nsfw {
		return " · 成人内容"
	}
	return ""
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func fail(op string, err error) error {
	slog.Error("agent MCP: "+op, "error", err)
	return errUnavailable
}
