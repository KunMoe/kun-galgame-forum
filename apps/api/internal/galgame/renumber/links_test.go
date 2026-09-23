package renumber

import "testing"

func TestRewriteLinks(t *testing.T) {
	lookup := func(m map[int64]int64) func(int64) (int64, bool) {
		return func(n int64) (int64, bool) {
			v, ok := m[n]
			return v, ok
		}
	}
	ids := lookup(map[int64]int64{12: 11, 923: 922})

	cases := []struct {
		name string
		in   string
		fn   func(int64) (int64, bool)
		want string
		n    int
	}{
		{"relative", "/galgame/12", ids, "/galgame/11", 1},
		{"relative query", "/galgame/923?comment=77&thread=5", ids, "/galgame/922?comment=77&thread=5", 1},
		{"resource path", "/galgame/resource/12", ids, "/galgame/resource/12", 0},
		{"quiz path", "/galgame-quiz/12", ids, "/galgame-quiz/12", 0},
		{"rating path", "/galgame-rating/12", ids, "/galgame-rating/12", 0},
		{"unnamed number", "/galgame/99", ids, "/galgame/99", 0},
		{"same id", "/galgame/12", lookup(map[int64]int64{12: 12}), "/galgame/12", 0},
		{"two links", "/galgame/12 and /galgame/923", ids, "/galgame/11 and /galgame/922", 2},
		{"start of text", "/galgame/12 end", ids, "/galgame/11 end", 1},
		{"end of text", "see /galgame/12", ids, "see /galgame/11", 1},
		{"www kungal https", "https://www.kungal.com/galgame/12", ids, "https://www.kungal.com/galgame/11", 1},
		{"www kungal locale", "https://www.kungal.com/zh-cn/galgame/12", ids, "https://www.kungal.com/zh-cn/galgame/11", 1},
		{"bare kungal locale", "https://kungal.com/zh-cn/galgame/12", ids, "https://kungal.com/zh-cn/galgame/11", 1},
		{"http www kungal", "http://www.kungal.com/galgame/12", ids, "http://www.kungal.com/galgame/11", 1},
		{"www kungal.org", "https://www.kungal.org/galgame/12", ids, "https://www.kungal.org/galgame/11", 1},
		{"host case", "HTTPS://WWW.KUNGAL.COM/galgame/12", ids, "HTTPS://WWW.KUNGAL.COM/galgame/11", 1},
		{"relative locale", "/zh-cn/galgame/12", ids, "/zh-cn/galgame/11", 1},
		{"markdown", "[x](https://www.kungal.com/galgame/12)", ids, "[x](https://www.kungal.com/galgame/11)", 1},
		{"angle autolink", "<https://www.kungal.com/galgame/12>", ids, "<https://www.kungal.com/galgame/11>", 1},
		{"mikugame", "https://mikugame.icu/galgame/12", ids, "https://mikugame.icu/galgame/12", 0},
		{"moyu", "https://www.moyu.moe/galgame/12", ids, "https://www.moyu.moe/galgame/12", 0},
		{"typo host", "https://www.kungalgl.com/galgame/12", ids, "https://www.kungalgl.com/galgame/12", 0},
		{"srsg", "https://srsg.clbug.com/galgame/12", ids, "https://srsg.clbug.com/galgame/12", 0},
		{"no scheme host", "example.com/galgame/12", ids, "example.com/galgame/12", 0},
		{"markdown label without scheme", "[www.kungal.com/galgame/12](http://www.kungal.com/galgame/12)", ids, "[www.kungal.com/galgame/11](http://www.kungal.com/galgame/11)", 2},
		{"bare kungal.com", "see kungal.com/zh-cn/galgame/12", ids, "see kungal.com/zh-cn/galgame/11", 1},
		{"markdown-escaped url", `其实本站就有 https\://www\.kungal.com/galgame/12 和 https\://www\.kungal\.com/galgame/923`, ids, `其实本站就有 https\://www\.kungal.com/galgame/11 和 https\://www\.kungal\.com/galgame/922`, 2},
		{"escaped foreign host", `https\://mikugame\.icu/galgame/12`, ids, `https\://mikugame\.icu/galgame/12`, 0},
		{"lookalike host without scheme", "evil-kungal.com/galgame/12 sub.kungal.com/galgame/12", ids, "evil-kungal.com/galgame/12 sub.kungal.com/galgame/12", 0},
		{"kungal.org without www", "https://kungal.org/galgame/12", ids, "https://kungal.org/galgame/12", 0},
		{"alnum after digits", "/galgame/12foo", ids, "/galgame/12foo", 0},
		{"underscore after", "/galgame/12_x", ids, "/galgame/12_x", 0},
		{"paren relative", "(/galgame/12)", ids, "(/galgame/11)", 1},
		{"bracket relative", "[/galgame/12]", ids, "[/galgame/11]", 1},
		{"quote relative", `"/galgame/12"`, ids, `"/galgame/11"`, 1},
		{"angle relative", "</galgame/12>", ids, "</galgame/11>", 1},
		{"backtick relative", "`/galgame/12`", ids, "`/galgame/11`", 1},
		{"plus relative", "+/galgame/12", ids, "+/galgame/11", 1},
		{"star relative", "*/galgame/12", ids, "*/galgame/11", 1},
		{"mixed kungal and mikugame",
			"see https://www.kungal.com/galgame/12 and https://mikugame.icu/galgame/12",
			ids,
			"see https://www.kungal.com/galgame/11 and https://mikugame.icu/galgame/12",
			1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, n := RewriteLinks(tc.in, tc.fn)
			if got != tc.want || n != tc.n {
				t.Fatalf("got %q n=%d, want %q n=%d", got, n, tc.want, tc.n)
			}
		})
	}
}
