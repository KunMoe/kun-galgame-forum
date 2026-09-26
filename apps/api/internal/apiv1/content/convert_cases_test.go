package content_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"kun-galgame-api/internal/apiv1/content"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/userclient"
)

func docJSON(children string) string {
	if children == "" {
		return `{"object":"document","children":[]}`
	}
	return `{"object":"document","children":[` + children + `]}`
}

func paraJSON(inlines string) string {
	return `{"object":"paragraph","children":[` + inlines + `]}`
}

func textJSON(v string) string {
	return `{"object":"text","value":` + quoteJSON(v) + `}`
}

func quoteJSON(v string) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func allConvertCases() []convertCase {
	var cs []convertCase
	cs = append(cs, blockCases()...)
	cs = append(cs, htmlCases()...)
	cs = append(cs, textCases()...)
	cs = append(cs, linkCases()...)
	cs = append(cs, imageCases()...)
	cs = append(cs, docCases()...)
	return cs
}

func blockCases() []convertCase {
	h := func(depth int, anchor, text string) string {
		return `{"object":"heading","depth":` + strconv.Itoa(depth) + `,"anchor":` + quoteJSON(anchor) +
			`,"children":[` + textJSON(text) + `]}`
	}
	return []convertCase{
		{name: "R1 paragraph", src: "hello", want: docJSON(paraJSON(textJSON("hello")))},
		{name: "R2 h1", src: "# Title", want: docJSON(h(2, "title", "Title"))},
		{name: "R2 h2", src: "## Title", want: docJSON(h(2, "title", "Title"))},
		{name: "R2 h3", src: "### Title", want: docJSON(h(3, "title", "Title"))},
		{name: "R2 h4", src: "#### Title", want: docJSON(h(4, "title", "Title"))},
		{name: "R2 h5", src: "##### Title", want: docJSON(h(5, "title", "Title"))},
		{name: "R2 h6", src: "###### Title", want: docJSON(h(6, "title", "Title"))},
		{name: "R2 duplicate headings", src: "## x\n\n## x", want: docJSON(h(2, "x", "x") + "," + h(2, "x-1", "x"))},
		{name: "R2 CJK heading", src: "## 引入", want: docJSON(h(2, "引入", "引入"))},
		{name: "R2 heading in blockquote", src: "> ## Quoted", want: docJSON(
			`{"object":"blockquote","children":[` + h(2, "quoted", "Quoted") + `]}`)},
		{name: "R3 thematic break", src: "---", want: docJSON(`{"object":"thematic_break"}`)},
		{name: "R3 blockquote", src: "> hi", want: docJSON(
			`{"object":"blockquote","children":[` + paraJSON(textJSON("hi")) + `]}`)},
		{name: "R3 nested blockquote", src: "> > inner", want: docJSON(
			`{"object":"blockquote","children":[{"object":"blockquote","children":[` + paraJSON(textJSON("inner")) + `]}]}`)},
		{name: "R4 tight list", src: "- a\n- b", want: docJSON(listJSON(false, "null", false,
			itemJSON("null", paraJSON(textJSON("a")))+","+itemJSON("null", paraJSON(textJSON("b")))))},
		{name: "R4 loose list", src: "- a\n\n- b", want: docJSON(listJSON(false, "null", true,
			itemJSON("null", paraJSON(textJSON("a")))+","+itemJSON("null", paraJSON(textJSON("b")))))},
		{name: "R4 ordered start 1", src: "1. a", want: docJSON(listJSON(true, "1", false,
			itemJSON("null", paraJSON(textJSON("a")))))},
		{name: "R4 ordered start 2", src: "2. a", want: docJSON(listJSON(true, "2", false,
			itemJSON("null", paraJSON(textJSON("a")))))},
		{name: "R4 ordered start 0", src: "0. a", want: docJSON(listJSON(true, "0", false,
			itemJSON("null", paraJSON(textJSON("a")))))},
		{name: "R4 unordered", src: "* a", want: docJSON(listJSON(false, "null", false,
			itemJSON("null", paraJSON(textJSON("a")))))},
		{name: "R4 task mixed", src: "- [x] done\n- [ ] todo\n- plain", want: docJSON(listJSON(false, "null", false,
			itemJSON("true", paraJSON(textJSON("done")))+","+
				itemJSON("false", paraJSON(textJSON("todo")))+","+
				itemJSON("null", paraJSON(textJSON("plain")))))},
		{name: "R4 nested list", src: "- a\n  - b", want: docJSON(listJSON(false, "null", false,
			itemJSON("null", paraJSON(textJSON("a"))+","+listJSON(false, "null", false,
				itemJSON("null", paraJSON(textJSON("b")))))))},
		{name: "R5 fenced lang", src: "```go\nx\n```", want: docJSON(codeJSON(`"go"`, "x"))},
		{name: "R5 fenced extras", src: "```typescript {2} showLineNumbers\nx\n```", want: docJSON(codeJSON(`"typescript"`, "x"))},
		{name: "R5 fenced uppercase", src: "```GO\nx\n```", want: docJSON(codeJSON(`"go"`, "x"))},
		{name: "R5 fenced no lang", src: "```\nx\n```", want: docJSON(codeJSON("null", "x"))},
		{name: "R5 fenced 33-char lang", src: "```" + strings.Repeat("a", 33) + "\nx\n```", want: docJSON(codeJSON("null", "x"))},
		{name: "R5 indented", src: "    indented", want: docJSON(codeJSON("null", "indented"))},
		{name: "R6 inline math", src: "$m_t$", want: docJSON(paraJSON(`{"object":"inline_math","value":"m_t"}`))},
		{name: "R6 block math", src: "$$\ne=mc^2\n$$", want: docJSON(`{"object":"math","value":"e=mc^2"}`)},
		{name: "R8 inline spoiler", src: "||secret||", want: docJSON(paraJSON(
			`{"object":"inline_spoiler","children":[` + textJSON("secret") + `]}`))},
		{name: "R8 block spoiler", src: ":::spoiler\nhidden\n:::", want: docJSON(
			`{"object":"spoiler","children":[` + paraJSON(textJSON("hidden")) + `]}`)},
		{name: "R7 table alignments and br", src: "" +
			"| l | c | r | n |\n" +
			"|:--|:-:|--:|---|\n" +
			"| 1 | 2 | 3 | x<br>y |\n", want: docJSON(tableJSON())},
	}
}

func listJSON(ordered bool, start string, spread bool, items string) string {
	return `{"object":"list","is_ordered":` + boolJSON(ordered) + `,"start":` + start +
		`,"is_spread":` + boolJSON(spread) + `,"children":[` + items + `]}`
}

func itemJSON(checked, children string) string {
	return `{"object":"list_item","is_checked":` + checked + `,"children":[` + children + `]}`
}

func codeJSON(lang, value string) string {
	return `{"object":"code","lang":` + lang + `,"value":` + quoteJSON(value) + `}`
}

func boolJSON(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func tableJSON() string {
	cell := func(align, inner string) string {
		return `{"object":"table_cell","align":` + align + `,"children":[` + inner + `]}`
	}
	row := func(cells string) string {
		return `{"object":"table_row","children":[` + cells + `]}`
	}
	header := row(cell(`"left"`, textJSON("l")) + "," + cell(`"center"`, textJSON("c")) + "," +
		cell(`"right"`, textJSON("r")) + "," + cell("null", textJSON("n")))
	body := row(cell(`"left"`, textJSON("1")) + "," + cell(`"center"`, textJSON("2")) + "," +
		cell(`"right"`, textJSON("3")) + "," + cell("null", textJSON("x")+`,{"object":"break"},`+textJSON("y")))
	return `{"object":"table","children":[` + header + "," + body + `]}`
}

func htmlCases() []convertCase {
	emptyP := `{"object":"paragraph","children":[]}`
	return []convertCase{
		{name: "R9 br block one", src: "<br />", want: docJSON(emptyP)},
		{name: "R9 br block three", src: "<br />\n<br />\n<br />", want: docJSON(emptyP + "," + emptyP + "," + emptyP)},
		{name: "R10 inline br", src: "a<br>b", want: docJSON(paraJSON(textJSON("a") + `,{"object":"break"},` + textJSON("b")))},
		{name: "R10 u tags dropped", src: "<u>x</u>", want: docJSON(paraJSON(textJSON("x")))},
		{name: "R10 non-element literal", src: "<bucket>", want: docJSON(paraJSON(textJSON("<bucket>")))},
		{name: "R10 attribute-named non-element literal", src: "a <src> b <id>", want: docJSON(paraJSON(textJSON("a <src> b <id>")))},
		{name: "R10 html comment", src: "a<!-- x -->b", want: docJSON(paraJSON(textJSON("ab")))},
		{name: "R11 img html block", src: `<img src="https://e.com/i.png" alt="pic">`, want: docJSON(paraJSON(
			`{"object":"image","url":"https://e.com/i.png","alt":"pic","image":null,"is_sticker":false}`))},
		{name: "R10 inline img data alt", src: `<img src="data:image/png;base64,xx" alt="hello">`, want: docJSON(paraJSON(textJSON("hello")))},
		{name: "R10 inline img long alt truncated", src: `<img src="data:x" alt="` + strings.Repeat("长", 600) + `">`,
			want: docJSON(paraJSON(textJSON(strings.Repeat("长", 512))))},
		{name: "R11 p br p block", src: "<p>a<br>b</p><p>c</p>", want: docJSON(paraJSON(
			textJSON("a") + `,{"object":"break"},` + textJSON("b") + `,{"object":"break"},` + textJSON("c")))},
		{name: "R11 script dropped", src: "<script>bad()</script>", want: docJSON("")},
	}
}

func textCases() []convertCase {
	br := `{"object":"break"}`
	return []convertCase{
		{name: "R12 backslash escape", src: `foo\*bar`, want: docJSON(paraJSON(textJSON("foo*bar")))},
		{name: "R12 named entity", src: `foo&amp;bar`, want: docJSON(paraJSON(textJSON("foo&bar")))},
		{name: "R12 numeric entity", src: `foo&#65;bar`, want: docJSON(paraJSON(textJSON("fooAbar")))},
		{name: "R13 hard break spaces", src: "a  \nb", want: docJSON(paraJSON(textJSON("a") + "," + br + "," + textJSON("b")))},
		{name: "R13 hard break backslash", src: "a\\\nb", want: docJSON(paraJSON(textJSON("a") + "," + br + "," + textJSON("b")))},
		{name: "R14 soft CJK-CJK", src: "中\n文", want: docJSON(paraJSON(textJSON("中文")))},
		{name: "R14 soft CJK-Latin", src: "中\nA", want: docJSON(paraJSON(textJSON("中 A")))},
		{name: "R14 soft Latin-Latin", src: "a\nb", want: docJSON(paraJSON(textJSON("a b")))},
		{name: "R14 soft Hangul", src: "한\n글", want: docJSON(paraJSON(textJSON("한 글")))},
		{name: "R14 soft CJK punct", src: "中\n。", want: docJSON(paraJSON(textJSON("中。")))},
		{name: "R14 soft after emphasis", src: "*word*\nnext", want: docJSON(paraJSON(
			`{"object":"emphasis","children":[` + textJSON("word") + `]},` + textJSON(" next")))},
		{name: "R14 soft after CJK strong", src: "**粗体**\n中文", want: docJSON(paraJSON(
			`{"object":"strong","children":[` + textJSON("粗体") + `]},` + textJSON("中文")))},
		{name: "R14 soft after link", src: "[l](https://a.example)\nnext", want: docJSON(paraJSON(
			`{"object":"link","url":"https://a.example","children":[` + textJSON("l") + `]},` + textJSON(" next")))},
		{name: "R14 soft after image", src: "![](https://a.example/x.png)\n中", want: docJSON(paraJSON(
			`{"object":"image","url":"https://a.example/x.png","alt":"","image":null,"is_sticker":false},` + textJSON(" 中")))},
		{name: "R14 soft after br", src: "a<br>\nb", want: docJSON(paraJSON(
			textJSON("a") + `,{"object":"break"},` + textJSON("b")))},
		{name: "R15 emphasis", src: "*i*", want: docJSON(paraJSON(`{"object":"emphasis","children":[` + textJSON("i") + `]}`))},
		{name: "R15 strong", src: "**b**", want: docJSON(paraJSON(`{"object":"strong","children":[` + textJSON("b") + `]}`))},
		{name: "R15 strikethrough", src: "~~s~~", want: docJSON(paraJSON(`{"object":"strikethrough","children":[` + textJSON("s") + `]}`))},
		{name: "R15 inline code", src: "`x&y\\*`", want: docJSON(paraJSON(`{"object":"inline_code","value":` + quoteJSON("x&y\\*") + `}`))},
	}
}

func linkCases() []convertCase {
	link := func(url, text string) string {
		return `{"object":"link","url":` + quoteJSON(url) + `,"children":[` + textJSON(text) + `]}`
	}
	long := "https://example.com/" + strings.Repeat("a", 2049)
	return []convertCase{
		{name: "R16 https", src: "[x](https://example.com)", want: docJSON(paraJSON(link("https://example.com", "x")))},
		{name: "R16 http", src: "[x](http://example.com)", want: docJSON(paraJSON(link("http://example.com", "x")))},
		{name: "R16 mailto", src: "[x](mailto:a@b.com)", want: docJSON(paraJSON(link("mailto:a@b.com", "x")))},
		{name: "R16 uppercase scheme", src: "[x](HTTPS://example.com)", want: docJSON(paraJSON(link("https://example.com", "x")))},
		{name: "R16 non-ASCII path", src: "[x](https://example.com/你好)", want: docJSON(paraJSON(link("https://example.com/%E4%BD%A0%E5%A5%BD", "x")))},
		{name: "R16 protocol-relative", src: "[x](//example.com/a)", want: docJSON(paraJSON(link("https://example.com/a", "x")))},
		{name: "R16 rooted topic", src: "[x](/topic/1)", want: docJSON(paraJSON(link("https://www.kungal.com/topic/1", "x")))},
		{name: "R16 rooted image token", src: "[x](/image/" + testHash + ")", want: docJSON(paraJSON(link(
			"https://cdn.example/aa/aa/"+testHash+".webp", "x")))},
		{name: "R16 unwrap kungal.com", src: "[x](kungal.com)", want: docJSON(paraJSON(textJSON("x")))},
		{name: "R16 unwrap fragment", src: "[x](#anchor)", want: docJSON(paraJSON(textJSON("x")))},
		{name: "R16 unwrap vscode-file", src: "[x](vscode-file://a)", want: docJSON(paraJSON(textJSON("x")))},
		{name: "R16 unwrap javascript", src: "[x](javascript:alert(1))", want: docJSON(paraJSON(textJSON("x")))},
		{name: "R16 unwrap uppercase javascript", src: "[x](JavaScript:alert(1))", want: docJSON(paraJSON(textJSON("x")))},
		{name: "R16 unwrap data", src: "[x](data:text/html,hi)", want: docJSON(paraJSON(textJSON("x")))},
		{name: "R16 unwrap http without host", src: "[x](http:foo)", want: docJSON(paraJSON(textJSON("x")))},
		{name: "R19 javascript autolink is text", src: "<javascript:alert(1)>", want: docJSON(paraJSON(textJSON("javascript:alert(1)")))},
		{name: "R16 unwrap 2049 url", src: "[x](" + long + ")", want: docJSON(paraJSON(textJSON("x")))},
		{name: "R16 title dropped", src: `[x](https://example.com "t")`, want: docJSON(paraJSON(link("https://example.com", "x")))},
		{name: "R16 mention valid", src: "[@n](kungal-user:3)", want: docJSON(paraJSON(
			`{"object":"mention","mentioned_user":{"object":"user","id":"3","name":"u3","avatar":null}}`))},
		{name: "R16 mention id 0", src: "[@n](kungal-user:0)", want: docJSON(paraJSON(textJSON("@n")))},
		{name: "R16 mention non-numeric", src: "[@n](kungal-user:x)", want: docJSON(paraJSON(textJSON("@n")))},
		{name: "R16 mention missing user", src: "[@n](kungal-user:3)", want: docJSON(paraJSON(
			`{"object":"mention","mentioned_user":{"object":"user","id":"3","name":null,"avatar":null}}`)), conv: missingUserConverter()},
		{name: "R16 reply #2", src: "[#2](kungal-reply:48)", want: docJSON(paraJSON(
			`{"object":"reply_reference","reply_id":"48","floor":2}`))},
		{name: "R16 reply 2", src: "[2](kungal-reply:48)", want: docJSON(paraJSON(
			`{"object":"reply_reference","reply_id":"48","floor":2}`))},
		{name: "R16 reply #x unwrap", src: "[#x](kungal-reply:48)", want: docJSON(paraJSON(textJSON("#x")))},
		{name: "R16 reply id 0 unwrap", src: "[#2](kungal-reply:0)", want: docJSON(paraJSON(textJSON("#2")))},
		{name: "R19 autolink url", src: "<https://example.com>", want: docJSON(paraJSON(link("https://example.com", "https://example.com")))},
		{name: "R19 autolink email", src: "<user@example.com>", want: docJSON(paraJSON(link("mailto:user@example.com", "user@example.com")))},
		{name: "R18 kv autolink", src: "kv:<https://e.com/x.mp4>", want: docJSON(paraJSON(`{"object":"video","url":"https://e.com/x.mp4"}`))},
		{name: "R18 kv inline link", src: "kv:[v](https://e.com/x.mp4)", want: docJSON(paraJSON(`{"object":"video","url":"https://e.com/x.mp4"}`))},
		{name: "R18 kv png ordinary", src: "kv:[v](https://e.com/x.png)", want: docJSON(paraJSON(
			textJSON("kv:") + "," + link("https://e.com/x.png", "v")))},
		{name: "R18 mp4 without kv", src: "[v](https://e.com/x.mp4)", want: docJSON(paraJSON(link("https://e.com/x.mp4", "v")))},
		{name: "R18 text before kv", src: "see kv:<https://e.com/x.mp4>", want: docJSON(paraJSON(
			textJSON("see ") + `,{"object":"video","url":"https://e.com/x.mp4"}`))},
	}
}

func missingUserConverter() *content.Converter {
	c := defaultConverter()
	c.Users = func(ctx context.Context, ids []int) (map[int]userclient.User, error) {
		return map[int]userclient.User{}, nil
	}
	return c
}

func imageCases() []convertCase {
	main := "https://cdn.example/aa/aa/" + testHash + ".webp"
	variant := "https://cdn.example/aa/aa/" + testHash + "_320.webp"
	img := func(url, alt, rec string) string {
		return `{"object":"image","url":` + quoteJSON(url) + `,"alt":` + quoteJSON(alt) + `,"image":` + rec + `,"is_sticker":false}`
	}
	nullRec := `{"url":` + quoteJSON(main) + `,"hash":` + quoteJSON(testHash) + `,"width":null,"height":null,"thumbhash":null,"sexual":null}`
	metaRec := `{"url":` + quoteJSON(main) + `,"hash":` + quoteJSON(testHash) + `,"width":8,"height":6,"thumbhash":"th","sexual":null}`
	metaConv := defaultConverter()
	metaConv.Images = func(hashes []string) map[string]imageclient.ImageMeta {
		return map[string]imageclient.ImageMeta{testHash: {Width: 8, Height: 6, Thumbhash: "th"}}
	}
	alt600 := strings.Repeat("ä", 600)
	alt512 := strings.Repeat("ä", 512)
	return []convertCase{
		{name: "R17 token no variant", src: "![](/image/" + testHash + ")", want: docJSON(paraJSON(img(main, "", nullRec)))},
		{name: "R17 token variant", src: "![](/image/" + testHash + "_320)", want: docJSON(paraJSON(img(variant, "", metaRec))), conv: metaConv},
		{name: "R17 token missing meta", src: "![](/image/" + testHash + ")", want: docJSON(paraJSON(img(main, "", nullRec)))},
		{name: "R17 https", src: "![a](https://e.com/i.png)", want: docJSON(paraJSON(img("https://e.com/i.png", "a", "null")))},
		{name: "R17 http", src: "![a](http://e.com/i.png)", want: docJSON(paraJSON(img("http://e.com/i.png", "a", "null")))},
		{name: "R17 protocol-relative", src: "![a](//e.com/i.png)", want: docJSON(paraJSON(img("https://e.com/i.png", "a", "null")))},
		{name: "R17 relative with alt", src: "![hello](./x.png)", want: docJSON(paraJSON(textJSON("hello")))},
		{name: "R17 data without alt", src: "![](data:image/png;base64,xx)", want: docJSON("")},
		{name: "R17 empty destination", src: "![]()", want: docJSON("")},
		{name: "R17 unusable only no alt drops paragraph", src: "![](./rel.png)", want: docJSON("")},
		{name: "R17 alt truncated", src: "![" + alt600 + "](https://e.com/i.png)", want: docJSON(paraJSON(img("https://e.com/i.png", alt512, "null")))},
		{name: "R17 sticker token variant", src: "![鲲 Galgame 表情包 \\[1\\] - 2](/image/" + legacySticker(2) + "_320)",
			want: docJSON(paraJSON(stickerImage(legacySticker(2), "_320", "鲲 Galgame 表情包 [1] - 2")))},
		{name: "R17 sticker token no variant", src: "![Sticker](/image/" + legacySticker(3) + ")",
			want: docJSON(paraJSON(stickerImage(legacySticker(3), "", "Sticker")))},
		{name: "R17 offsite url carrying a sticker hash", src: "![](https://e.com/" + legacySticker(4) + "_320.webp)",
			want: docJSON(paraJSON(img("https://e.com/"+legacySticker(4)+"_320.webp", "", "null")))},
		{name: "R17 image inside link", src: "[![a](https://e.com/i.png)](https://e.com/)", want: docJSON(paraJSON(
			`{"object":"link","url":"https://e.com/","children":[` + img("https://e.com/i.png", "a", "null") + `]}`))},
	}
}

func docCases() []convertCase {
	main := "https://cdn.example/aa/aa/" + testHash + ".webp"
	nullRec := `{"url":` + quoteJSON(main) + `,"hash":` + quoteJSON(testHash) + `,"width":null,"height":null,"thumbhash":null,"sexual":null}`
	sticker := "https://sticker.kungal.com/stickers/KUNgal1/1.webp"
	abs := "https://image.kungal.com/aa/aa/" + testHash + ".webp"
	return []convertCase{
		{name: "empty string", src: "", want: docJSON("")},
		{name: "whitespace only", src: "  \n\t", want: docJSON("")},
		{name: "legacy sticker url", src: "![](" + sticker + ")", want: stickerWant()},
		{name: "absolute image-service url", src: "![](" + abs + ")", want: docJSON(paraJSON(
			`{"object":"image","url":` + quoteJSON(main) + `,"alt":"","image":` + nullRec + `,"is_sticker":false}`))},
	}
}

func stickerWant() string {
	return docJSON(paraJSON(stickerImage(legacySticker(1), "_320", "")))
}

func legacySticker(position int) string {
	hash, ok := markdown.LegacyStickerHash(1, position)
	if !ok {
		panic("sticker 1/" + strconv.Itoa(position))
	}
	return hash
}

func stickerImage(hash, variant, alt string) string {
	display := "https://cdn.example/" + hash[:2] + "/" + hash[2:4] + "/" + hash + variant + ".webp"
	orig := "https://cdn.example/" + hash[:2] + "/" + hash[2:4] + "/" + hash + ".webp"
	rec := `{"url":` + quoteJSON(orig) + `,"hash":` + quoteJSON(hash) + `,"width":null,"height":null,"thumbhash":null,"sexual":null}`
	return `{"object":"image","url":` + quoteJSON(display) + `,"alt":` + quoteJSON(alt) + `,"image":` + rec + `,"is_sticker":true}`
}
