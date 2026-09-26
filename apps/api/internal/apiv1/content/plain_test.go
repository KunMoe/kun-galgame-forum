package content_test

import (
	"strings"
	"testing"

	"kun-galgame-api/internal/apiv1/content"
)

func TestPlainText(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"autolink", "泪目了，还有方舟遗佬\n\n<https://www.kungal.com/doc/who-is-kun>\n", "泪目了，还有方舟遗佬\nhttps://www.kungal.com/doc/who-is-kun"},
		{"marker characters in text", "C# 和 a > b，snake_case~ x|y", "C# 和 a > b，snake_case~ x|y"},
		{"underscore in a bare url", "https://example.com/a_b_c", "https://example.com/a_b_c"},
		{"emphasis and heading", "# 标题\n\n**粗** _斜_ ~~删~~", "标题\n粗 斜 删"},
		{"blockquote and list", "> 引用\n\n- 一\n- 二", "引用\n一\n二"},
		{"link text", "看[这里](https://example.com)好", "看这里好"},
		{"image and sticker dropped", "a![图](https://example.com/x.png)b![鲲 Galgame 表情包 \\[1\\] - 56](/image/" + strings.Repeat("a", 64) + "_320 \"t\")c", "abc"},
		{"mention and reply reference", "[@kun](kungal-user:1) [#3](kungal-reply:9) 你好", "@kun #3 你好"},
		{"spoilers masked", "前||剧透||后\n\n:::spoiler\n整段剧透\n:::", "前███后\n███"},
		{"code", "用 `a*b` 吧\n\n```go\nx := 1\n```", "用 a*b 吧\nx := 1"},
		{"escapes and entities", `a \* b &amp; c`, "a * b & c"},
		{"html", "a<br>b\n\n<div>块</div>", "a\nb\n块"},
		{"table", "| a | b |\n| - | - |\n| 1 | 2 |", "a b\n1 2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := content.PlainText(tc.in, 1000); got != tc.want {
				t.Fatalf("PlainText(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
	if got := content.PlainText("**一二三四五**", 3); got != "一二三" {
		t.Fatalf("cut to 3 runes: %q", got)
	}
}
