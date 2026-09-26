package content

import (
	"strings"

	"kun-galgame-api/internal/infrastructure/markdown"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

const spoilerMask = "███"

// PlainText is stored Markdown as the text a reader sees, at most maxLen
// runes: images and tags dropped, links as their text, spoilers masked.
func PlainText(source string, maxLen int) string {
	root, src := markdown.ParseStored(source)
	c := &converter{src: src}
	var b strings.Builder
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			switch {
			case n.Kind() == east.KindTableCell:
				b.WriteByte(' ')
			case n.Type() == ast.TypeBlock:
				b.WriteByte('\n')
			}
			return ast.WalkContinue, nil
		}
		switch n.Kind() {
		case ast.KindText:
			t := n.(*ast.Text)
			b.WriteString(c.textValue(t))
			if t.SoftLineBreak() || t.HardLineBreak() {
				b.WriteByte('\n')
			}
		case ast.KindString:
			b.WriteString(c.stringValue(n.(*ast.String)))
		case ast.KindAutoLink:
			b.Write(n.(*ast.AutoLink).Label(src))
		case ast.KindCodeSpan:
			b.WriteString(codeSpanValue(n.(*ast.CodeSpan), src))
		case mathjax.KindInlineMath:
			b.WriteString(inlineMathValue(n, src))
		case ast.KindCodeBlock, ast.KindFencedCodeBlock, mathjax.KindMathBlock:
			b.WriteString(nodeLines(n, src))
		case ast.KindRawHTML:
			writeInlineText(&b, c.rawHTML(n.(*ast.RawHTML)))
		case ast.KindHTMLBlock:
			for _, block := range c.htmlBlock(n.(*ast.HTMLBlock)) {
				if p, ok := block.(ParagraphNode); ok {
					writeInlineText(&b, p.Children)
				}
				b.WriteByte('\n')
			}
		case markdown.KindSpoilerInline, markdown.KindSpoilerBlock:
			b.WriteString(spoilerMask)
		case ast.KindImage:
		default:
			return ast.WalkContinue, nil
		}
		return ast.WalkSkipChildren, nil
	})
	lines := strings.Split(b.String(), "\n")
	kept := lines[:0]
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			kept = append(kept, line)
		}
	}
	return truncateRunes(strings.Join(kept, "\n"), maxLen)
}

func writeInlineText(b *strings.Builder, in Inlines) {
	for _, n := range in {
		switch t := n.(type) {
		case TextNode:
			b.WriteString(t.Value)
		case BreakNode:
			b.WriteByte('\n')
		}
	}
}
