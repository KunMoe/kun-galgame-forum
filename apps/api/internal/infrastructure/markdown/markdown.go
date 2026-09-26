package markdown

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"kun-galgame-api/pkg/imageclient"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const plainSpoilerSpan = `<span ` + spoilerClass + `>$1</span>`

var (
	plainSpoilerRegex = regexp.MustCompile(`(?s)\|\|(.*?)\|\|`)
	videoLinkRegex    = regexp.MustCompile(`kv:<a href="(https?://[^\s]+?\.(mp4))">[^<]+</a>`)
	codeBlockRegex    = regexp.MustCompile(`(?s)<pre><code class="language-(\w+)"`)
	mentionRegex      = regexp.MustCompile(`<a href="kungal-user:(\d+)"[^>]*>(.*?)</a>`)
	quoteRegex        = regexp.MustCompile(`<a href="kungal-reply:(\d+)"[^>]*>#?(\d+)</a>`)

	md         goldmark.Markdown
	mdHardWrap goldmark.Markdown
	mdContent  goldmark.Markdown
	sanitizer  *bluemonday.Policy
)

var contentImageRefRe = regexp.MustCompile(`^/image/([0-9a-f]{64})(?:_([a-z0-9]+))?$`)

func ParseContentImageRef(dest string) (hash, variant string, ok bool) {
	m := contentImageRefRe.FindStringSubmatch(dest)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

var contentImageScanRe = regexp.MustCompile(`/image/[0-9a-f]{64}`)

func ExtractCoverImages(content string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, limit)
	for _, tk := range contentImageScanRe.FindAllString(content, -1) {
		if _, dup := seen[tk]; dup {
			continue
		}
		seen[tk] = struct{}{}
		if IsSticker(strings.TrimPrefix(tk, "/image/")) {
			continue
		}
		out = append(out, tk)
		if len(out) >= limit {
			break
		}
	}
	return out
}

var contentImageCDNBase string

func SetContentImageCDNBase(base string) {
	contentImageCDNBase = strings.TrimRight(base, "/")
	rememberCDNBaseHost(contentImageCDNBase)
}

var contentSiteBase string

func SetContentSiteBase(base string) {
	contentSiteBase = strings.TrimRight(base, "/")
}

func resolveContentImageRef(dest string) string {
	if contentImageCDNBase == "" {
		return ""
	}
	m := contentImageRefRe.FindStringSubmatch(dest)
	if m == nil {
		return ""
	}
	hash, variant := m[1], m[2]
	if variant != "" {
		return imageclient.VariantURL(contentImageCDNBase, hash, variant, "webp")
	}
	return imageclient.MainURL(contentImageCDNBase, hash, "webp")
}

type TocLink struct {
	ID       string    `json:"id"`
	Text     string    `json:"text"`
	Depth    int       `json:"depth"`
	Children []TocLink `json:"children,omitempty"`
}

func init() {
	md = newGoldmark(false, true)
	mdHardWrap = newGoldmark(true, true)
	mdContent = newGoldmark(false, false)

	sanitizer = newSanitizePolicy()
}

func newGoldmark(hardWraps, imageMeta bool) goldmark.Markdown {
	rendererOpts := []renderer.Option{html.WithUnsafe()}
	if hardWraps {
		rendererOpts = append(rendererOpts, html.WithHardWraps())
	}
	parserOpts := []parser.Option{parser.WithAutoHeadingID()}
	if imageMeta {
		parserOpts = append(parserOpts, parser.WithASTTransformers(
			util.Prioritized(&contentImageMetaTransformer{}, 100),
		))
	}
	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			mathjax.MathJax,
			&h1ToH2Extension{},
			&lazyImageExtension{},
			&spoilerExtension{},
		),
		goldmark.WithParserOptions(parserOpts...),
		goldmark.WithRendererOptions(rendererOpts...),
	)
}

func newSanitizePolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	p.AllowAttrs("class").Globally()
	p.AllowAttrs("id").OnElements("h1", "h2", "h3", "h4", "h5", "h6")
	p.AllowAttrs("loading", "decoding", "data-kun-lazy-image").OnElements("img")
	p.AllowAttrs("width", "height").Matching(regexp.MustCompile(`^[0-9]+$`)).OnElements("img")
	p.AllowAttrs("data-thumbhash").Matching(regexp.MustCompile(`^[A-Za-z0-9+/=]+$`)).OnElements("img")
	p.AllowElements("video", "button")
	p.AllowAttrs("controls", "loop", "playsinline", "width", "src").OnElements("video")
	p.AllowAttrs("title").OnElements("button")
	p.AllowElements("input")
	p.AllowAttrs("type", "checked", "disabled").OnElements("input")
	p.AllowAttrs("data-uid").OnElements("a")
	p.AllowAttrs("data-reply-id", "data-floor").OnElements("span")

	return p
}

func RenderQuestionPlain(source string) string {
	escaped := string(util.EscapeHTML([]byte(source)))
	return plainSpoilerRegex.ReplaceAllString(escaped, plainSpoilerSpan)
}

func Render(source string) string {
	html, _ := RenderWithTOC(source)
	return html
}

func RenderHardWrap(source string) string {
	return renderWith(mdHardWrap, source)
}

func RenderWithTOC(source string) (string, []TocLink) {
	source = ResolveLegacyStickerRefs(source)
	src := []byte(source)
	reader := text.NewReader(src)
	ctx := parser.NewContext(parser.WithIDs(newUnicodeIDs()))
	root := md.Parser().Parse(reader, parser.WithContext(ctx))

	toc := buildTOCTree(collectHeadings(root, src), 3)

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, src, root); err != nil {
		return source, nil
	}

	return applyTransforms(buf.String()), toc
}

func renderWith(m goldmark.Markdown, source string) string {
	source = ResolveLegacyStickerRefs(source)
	src := []byte(source)
	reader := text.NewReader(src)
	ctx := parser.NewContext(parser.WithIDs(newUnicodeIDs()))
	root := m.Parser().Parse(reader, parser.WithContext(ctx))

	var buf bytes.Buffer
	if err := m.Renderer().Render(&buf, src, root); err != nil {
		return source
	}

	return applyTransforms(buf.String())
}

func applyTransforms(result string) string {
	result = codeBlockRegex.ReplaceAllStringFunc(result, func(match string) string {
		lang := codeBlockRegex.FindStringSubmatch(match)
		if len(lang) < 2 {
			return match
		}
		return `<div class="kun-code-container language-` + lang[1] + `">` +
			`<div class="kun-code-header">` +
			`<span class="lang">` + lang[1] + `</span>` +
			`<button class="copy" title="Copy code"></button>` +
			`</div>` +
			`<pre><code class="language-` + lang[1] + `"`
	})
	result = strings.ReplaceAll(result, "<pre><code>",
		`<div class="kun-code-container">`+
			`<div class="kun-code-header">`+
			`<span class="lang"></span>`+
			`<button class="copy" title="Copy code"></button>`+
			`</div>`+
			`<pre><code>`)
	result = strings.ReplaceAll(result, "</code></pre>", "</code></pre></div>")

	result = strings.ReplaceAll(result, "<table>", `<div class="kun-table-container"><table>`)
	result = strings.ReplaceAll(result, "</table>", `</table></div>`)

	result = videoLinkRegex.ReplaceAllString(result,
		`<video controls loop playsinline width="100%" src="$1"></video>`)

	result = mentionRegex.ReplaceAllString(result,
		`<a class="kun-mention" data-uid="$1" href="`+contentSiteBase+`/user/$1/info">$2</a>`)
	result = quoteRegex.ReplaceAllString(result,
		`<span class="kun-quote" data-reply-id="$1" data-floor="$2">#$2</span>`)

	return sanitizer.Sanitize(result)
}

type unicodeIDs struct {
	used map[string]int
	anon int
}

func newUnicodeIDs() *unicodeIDs {
	return &unicodeIDs{used: map[string]int{}}
}

func (u *unicodeIDs) Generate(value []byte, _ ast.NodeKind) []byte {
	base := slugify(string(value))
	if base == "" {
		base = fmt.Sprintf("heading-%d", u.anon)
		u.anon++
	}
	id := base
	if n := u.used[base]; n > 0 {
		u.used[base] = n + 1
		id = fmt.Sprintf("%s-%d", base, n)
	} else {
		u.used[base] = 1
	}
	u.used[id] = 1
	return []byte(id)
}

func (u *unicodeIDs) Put(value []byte) {
	u.used[string(value)] = 1
}

func collectHeadings(root ast.Node, source []byte) []TocLink {
	if root == nil {
		return nil
	}
	var out []TocLink
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		depth := h.Level
		if depth == 1 {
			depth = 2
		}
		out = append(out, TocLink{
			ID:    headingID(h),
			Text:  headingText(h, source),
			Depth: depth,
		})
		return ast.WalkContinue, nil
	})
	return out
}

func headingID(h *ast.Heading) string {
	attr, found := h.Attribute([]byte("id"))
	if !found {
		return ""
	}
	if b, ok := attr.([]byte); ok {
		return string(b)
	}
	return ""
}

func headingText(h *ast.Heading, source []byte) string {
	var b strings.Builder
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		if t, ok := n.(*ast.Text); ok {
			b.Write(t.Segment.Value(source))
			return
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(h)
	return strings.TrimSpace(b.String())
}

func slugify(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case unicode.IsLetter(r):
			if r < 128 {
				b.WriteRune(unicode.ToLower(r))
			} else {
				b.WriteRune(r)
			}
			prevDash = false
		case unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		case unicode.IsSpace(r), r == '-', r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.TrimRight(b.String(), "-")
	runes := []rune(out)
	if len(runes) > 100 {
		out = string(runes[:100])
	}
	return out
}

func buildTOCTree(flat []TocLink, maxDepth int) []TocLink {
	var roots []TocLink
	type frame struct {
		depth int
		list  *[]TocLink
	}
	stack := []frame{{depth: 1, list: &roots}}

	for _, h := range flat {
		if h.Depth < 2 || h.Depth > maxDepth {
			continue
		}
		for len(stack) > 1 && h.Depth <= stack[len(stack)-1].depth {
			stack = stack[:len(stack)-1]
		}
		top := stack[len(stack)-1]
		*top.list = append(*top.list, h)
		newEntry := &(*top.list)[len(*top.list)-1]
		stack = append(stack, frame{depth: h.Depth, list: &newEntry.Children})
	}
	return roots
}

type h1ToH2Extension struct{}

func (e *h1ToH2Extension) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&h1ToH2Renderer{}, 100),
	))
}

type h1ToH2Renderer struct{}

func (r *h1ToH2Renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, r.renderHeading)
}

func (r *h1ToH2Renderer) renderHeading(
	w util.BufWriter, source []byte, node ast.Node, entering bool,
) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	level := n.Level
	if level == 1 {
		level = 2
	}
	tag := byte('0' + level)

	if entering {
		w.WriteString("<h")
		w.WriteByte(tag)
		for _, attr := range n.Attributes() {
			w.WriteByte(' ')
			w.Write(attr.Name)
			w.WriteString(`="`)
			w.Write(util.EscapeHTML(attr.Value.([]byte)))
			w.WriteByte('"')
		}
		w.WriteByte('>')
	} else {
		w.WriteString("</h")
		w.WriteByte(tag)
		w.WriteString(">\n")
	}
	return ast.WalkContinue, nil
}

type lazyImageExtension struct{}

func (e *lazyImageExtension) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&lazyImageRenderer{}, 100),
	))
}

type lazyImageRenderer struct{}

func (r *lazyImageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *lazyImageRenderer) renderImage(
	w util.BufWriter, source []byte, node ast.Node, entering bool,
) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)

	var altBuf bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			altBuf.Write(t.Value(source))
		}
	}

	dest := n.Destination
	if resolved := resolveContentImageRef(string(dest)); resolved != "" {
		dest = []byte(resolved)
	}

	w.WriteString(`<img src="`)
	w.Write(util.EscapeHTML(dest))
	w.WriteString(`" alt="`)
	w.Write(util.EscapeHTML(altBuf.Bytes()))
	w.WriteString(`"`)
	if n.Title != nil {
		w.WriteString(` title="`)
		w.Write(util.EscapeHTML(n.Title))
		w.WriteString(`"`)
	}
	writeImageAttr(w, n, "width")
	writeImageAttr(w, n, "height")
	writeImageAttr(w, n, "data-thumbhash")
	w.WriteString(` loading="lazy" decoding="async" data-kun-lazy-image="true" />`)
	return ast.WalkSkipChildren, nil
}

func writeImageAttr(w util.BufWriter, n *ast.Image, name string) {
	v, ok := n.AttributeString(name)
	if !ok {
		return
	}
	b, ok := v.([]byte)
	if !ok || len(b) == 0 {
		return
	}
	w.WriteByte(' ')
	w.WriteString(name)
	w.WriteString(`="`)
	w.Write(util.EscapeHTML(b))
	w.WriteByte('"')
}

var contentImageMetaResolve func(hashes []string) map[string]imageclient.ImageMeta

func SetContentImageMetaResolver(fn func(hashes []string) map[string]imageclient.ImageMeta) {
	contentImageMetaResolve = fn
}

func ResolveContentImageMeta(tokens []string) map[string]imageclient.ImageMeta {
	resolve := contentImageMetaResolve
	if resolve == nil || len(tokens) == 0 {
		return nil
	}
	hashByToken := make(map[string]string, len(tokens))
	hashes := make([]string, 0, len(tokens))
	for _, tk := range tokens {
		if h := contentImageHash(tk); h != "" {
			hashByToken[tk] = h
			hashes = append(hashes, h)
		}
	}
	if len(hashes) == 0 {
		return nil
	}
	metas := resolve(hashes)
	out := make(map[string]imageclient.ImageMeta, len(hashByToken))
	for tk, h := range hashByToken {
		if m, ok := metas[h]; ok {
			out[tk] = m
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func contentImageHash(dest string) string {
	hash, _, ok := ParseContentImageRef(dest)
	if !ok {
		return ""
	}
	return hash
}

type contentImageMetaTransformer struct{}

func (t *contentImageMetaTransformer) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	resolve := contentImageMetaResolve
	if resolve == nil {
		return
	}

	var imgs []*ast.Image
	var hashes []string
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if img, ok := n.(*ast.Image); ok {
			if h := contentImageHash(string(img.Destination)); h != "" {
				imgs = append(imgs, img)
				hashes = append(hashes, h)
			}
		}
		return ast.WalkContinue, nil
	})
	if len(imgs) == 0 {
		return
	}

	metas := resolve(hashes)
	for i, img := range imgs {
		m, ok := metas[hashes[i]]
		if !ok {
			continue
		}
		if m.Width > 0 {
			img.SetAttributeString("width", []byte(strconv.Itoa(m.Width)))
		}
		if m.Height > 0 {
			img.SetAttributeString("height", []byte(strconv.Itoa(m.Height)))
		}
		if m.Thumbhash != "" {
			img.SetAttributeString("data-thumbhash", []byte(m.Thumbhash))
		}
	}
}
