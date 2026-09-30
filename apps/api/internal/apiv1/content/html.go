package content

import (
	"bytes"
	"regexp"
	"strings"

	"kun-galgame-api/internal/infrastructure/markdown"

	"github.com/yuin/goldmark/ast"
	"golang.org/x/net/html"
)

var (
	onlyBRRe = regexp.MustCompile(`(?is)^(?:\s*<br\s*/?\s*>)+\s*$`)
	brTagRe  = regexp.MustCompile(`(?i)<br\s*/?\s*>`)
)

func htmlBlockSource(n *ast.HTMLBlock, source []byte) string {
	var b strings.Builder
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		b.Write(line.Value(source))
	}
	if n.HasClosure() {
		b.Write(n.ClosureLine.Value(source))
	}
	return b.String()
}

func (c *converter) htmlBlock(n *ast.HTMLBlock) Blocks {
	raw := htmlBlockSource(n, c.src)
	if c.untrusted {
		return literalHTMLBlock(raw)
	}
	if onlyBRRe.MatchString(raw) {
		count := len(brTagRe.FindAllString(raw, -1))
		out := make(Blocks, 0, count)
		for i := 0; i < count; i++ {
			out = append(out, NewParagraph(nil))
		}
		return out
	}
	in := trimBreaks(c.tokenizeHTML(raw, true))
	if len(in) == 0 {
		return nil
	}
	return Blocks{NewParagraph(mergeInlines(in))}
}

func (c *converter) rawHTML(n *ast.RawHTML) Inlines {
	raw := string(n.Segments.Value(c.src))
	if c.untrusted {
		return literalRawHTML(raw)
	}
	return c.tokenizeHTML(raw, false)
}

func trimBreaks(in Inlines) Inlines {
	for len(in) > 0 {
		if _, ok := in[0].(BreakNode); ok {
			in = in[1:]
			continue
		}
		break
	}
	for len(in) > 0 {
		if _, ok := in[len(in)-1].(BreakNode); ok {
			in = in[:len(in)-1]
			continue
		}
		break
	}
	return in
}

// atom.Lookup also knows attribute names, so a user who typed <src> or <id> as
// prose lost the word; this is the WHATWG element index plus the obsolete
// presentational elements old posts use.
var htmlElements = func() map[string]bool {
	m := map[string]bool{}
	for _, n := range strings.Fields(`a abbr address area article aside audio b base bdi bdo blockquote body br button
		canvas caption cite code col colgroup data datalist dd del details dfn dialog div dl dt em embed fieldset
		figcaption figure footer form h1 h2 h3 h4 h5 h6 head header hgroup hr html i iframe img input ins kbd label
		legend li link main map mark menu meta meter nav noscript object ol optgroup option output p picture pre
		progress q rp rt ruby s samp script search section select slot small source span strong style sub summary
		sup table tbody td template textarea tfoot th thead time title tr track u ul var video wbr
		acronym basefont big center dir font frame frameset marquee nobr noframes strike tt xmp`) {
		m[n] = true
	}
	return m
}()

func isHTMLElementName(name []byte) bool {
	return htmlElements[strings.ToLower(string(name))]
}

func isName(name, want []byte) bool {
	return bytes.Equal(name, want)
}

var (
	tagBR          = []byte("br")
	tagHR          = []byte("hr")
	tagImg         = []byte("img")
	tagScript      = []byte("script")
	tagStyle       = []byte("style")
	blockEndBreaks = map[string]bool{
		"p": true, "div": true, "li": true, "ul": true, "ol": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"blockquote": true, "pre": true, "table": true, "tr": true,
	}
)

func (c *converter) tokenizeHTML(raw string, blockMode bool) Inlines {
	if raw == "" {
		return nil
	}
	z := html.NewTokenizer(strings.NewReader(raw))
	var out Inlines
	inRaw := false
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		switch tt {
		case html.TextToken:
			if inRaw {
				continue
			}
			if t := string(z.Text()); t != "" {
				out = append(out, NewText(t))
			}
		case html.CommentToken, html.DoctypeToken:
		case html.StartTagToken, html.SelfClosingTagToken, html.EndTagToken:
			rawTok := string(z.Raw())
			name, hasAttr := z.TagName()
			nameCopy := append([]byte(nil), name...)
			if inRaw {
				if tt == html.EndTagToken && (isName(nameCopy, tagScript) || isName(nameCopy, tagStyle)) {
					inRaw = false
				}
				continue
			}
			if tt == html.StartTagToken && (isName(nameCopy, tagScript) || isName(nameCopy, tagStyle)) {
				inRaw = true
				continue
			}
			if isName(nameCopy, tagBR) {
				out = append(out, NewBreak())
				continue
			}
			if isName(nameCopy, tagImg) && tt != html.EndTagToken {
				src, alt, hasSrc := imgAttrs(z, hasAttr)
				if hasSrc {
					out = append(out, c.convertImage(src, alt)...)
				}
				continue
			}
			if blockMode && isName(nameCopy, tagHR) {
				out = append(out, NewBreak())
				continue
			}
			if blockMode && tt == html.EndTagToken && blockEndBreaks[string(nameCopy)] {
				out = append(out, NewBreak())
				continue
			}
			if isHTMLElementName(nameCopy) {
				continue
			}
			if rawTok != "" {
				out = append(out, NewText(rawTok))
			}
		}
	}
	return out
}

func imgAttrs(z *html.Tokenizer, hasAttr bool) (src, alt string, ok bool) {
	if !hasAttr {
		return "", "", false
	}
	for {
		key, val, more := z.TagAttr()
		switch string(key) {
		case "src":
			src = string(val)
			ok = true
		case "alt":
			alt = string(val)
		}
		if !more {
			break
		}
	}
	return src, alt, ok
}

func collectHTMLImageHashes(raw []byte, addHash func(string)) {
	if len(raw) == 0 {
		return
	}
	z := html.NewTokenizer(bytes.NewReader(raw))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		name, hasAttr := z.TagName()
		if !bytes.Equal(name, tagImg) || !hasAttr {
			continue
		}
		src, _, ok := imgAttrs(z, hasAttr)
		if !ok {
			continue
		}
		if h, _, ok := markdown.ParseContentImageRef(src); ok {
			addHash(h)
		}
	}
}
