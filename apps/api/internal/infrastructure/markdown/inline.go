package markdown

import (
	"bytes"
	"os"
	"regexp"
	"slices"
	"strings"

	"kun-galgame-api/pkg/imageclient"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

var (
	inlineMd        = newInlineMarkdown()
	inlineSanitizer = newInlineSanitizePolicy()
)

var allowedImageHosts = resolveAllowedImageHosts()

func resolveAllowedImageHosts() []string {
	if v := strings.TrimSpace(os.Getenv("KUNGAL_MESSAGE_IMAGE_HOSTS")); v != "" {
		var hosts []string
		for p := range strings.SplitSeq(v, ",") {
			if h := strings.TrimSpace(p); h != "" {
				hosts = append(hosts, h)
			}
		}
		if len(hosts) > 0 {
			return hosts
		}
	}
	return append(slices.Clone(imageclient.KnownHosts), "sticker.kungal.com")
}

var messageImageSrcPattern = buildImageSrcPattern(allowedImageHosts)

// RenderInline expands a /image/<hash> token to the CDN base before the policy sees
// it, so a base pointing at a host this list does not name makes bluemonday drop the
// img and the picture just disappears from chat. Re-derive both when the base is set.
func allowInlineImageHost(host string) {
	if host == "" {
		return
	}
	for _, h := range allowedImageHosts {
		if strings.EqualFold(h, host) {
			return
		}
	}
	allowedImageHosts = append(allowedImageHosts, host)
	messageImageSrcPattern = buildImageSrcPattern(allowedImageHosts)
	inlineSanitizer = newInlineSanitizePolicy()
}

func buildImageSrcPattern(hosts []string) *regexp.Regexp {
	escaped := make([]string, len(hosts))
	for i, h := range hosts {
		escaped[i] = regexp.QuoteMeta(h)
	}
	return regexp.MustCompile(`^https://(` + strings.Join(escaped, "|") + `)/`)
}

func newInlineMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithParser(parser.NewParser(
			parser.WithBlockParsers(
				util.Prioritized(parser.NewParagraphParser(), 1000),
			),
			parser.WithInlineParsers(parser.DefaultInlineParsers()...),
			parser.WithParagraphTransformers(parser.DefaultParagraphTransformers()...),
		)),
		goldmark.WithExtensions(
			extension.Strikethrough,
			extension.Linkify,
			&lazyImageExtension{},
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)
}

func newInlineSanitizePolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "em", "del", "code", "a", "img")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("src").Matching(messageImageSrcPattern).OnElements("img")
	p.AllowAttrs("alt", "title", "loading", "decoding", "data-kun-lazy-image").OnElements("img")
	p.AllowStandardURLs()
	p.RequireNoFollowOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	return p
}

func RenderInline(source string) string {
	if source == "" {
		return ""
	}
	source = ResolveLegacyStickerRefs(source)
	var buf bytes.Buffer
	if err := inlineMd.Convert([]byte(source), &buf); err != nil {
		return inlineSanitizer.Sanitize(source)
	}
	return inlineSanitizer.Sanitize(buf.String())
}
