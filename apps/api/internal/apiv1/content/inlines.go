package content

import (
	"strings"

	"kun-galgame-api/internal/apiv1/repr"
	"kun-galgame-api/internal/infrastructure/markdown"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

func (c *converter) inlines(parent ast.Node) Inlines {
	if parent == nil {
		return nil
	}
	var out Inlines
	for n := parent.FirstChild(); n != nil; {
		next := n.NextSibling()
		if c.emitVideo(n, next) {
			if prefix, _ := c.kvPrefix(n); prefix != "" {
				out = append(out, NewText(prefix))
			}
			if url, ok := c.videoURL(next); ok {
				out = append(out, NewVideo(url))
			}
			n = next.NextSibling()
			continue
		}
		out = append(out, c.convertInline(n)...)
		n = next
	}
	return mergeInlines(out)
}

func (c *converter) convertInline(n ast.Node) Inlines {
	switch n.Kind() {
	case ast.KindText:
		return c.convertText(n.(*ast.Text))
	case ast.KindString:
		v := c.stringValue(n.(*ast.String))
		if v == "" {
			return nil
		}
		return Inlines{NewText(v)}
	case ast.KindEmphasis:
		e := n.(*ast.Emphasis)
		kids := c.inlines(e)
		if e.Level >= 2 {
			return Inlines{NewStrong(kids)}
		}
		return Inlines{NewEmphasis(kids)}
	case ast.KindCodeSpan:
		return Inlines{NewInlineCode(codeSpanValue(n.(*ast.CodeSpan), c.src))}
	case ast.KindLink:
		return c.convertLink(n.(*ast.Link))
	case ast.KindAutoLink:
		return c.convertAutoLink(n.(*ast.AutoLink))
	case ast.KindImage:
		img := n.(*ast.Image)
		return c.convertImage(string(img.Destination), truncateRunes(c.plainText(img), maxAltRunes))
	case ast.KindRawHTML:
		return c.rawHTML(n.(*ast.RawHTML))
	case east.KindStrikethrough:
		return Inlines{NewStrikethrough(c.inlines(n))}
	case east.KindTaskCheckBox:
		return nil
	case markdown.KindSpoilerInline:
		return Inlines{NewInlineSpoiler(c.inlines(n))}
	case mathjax.KindInlineMath:
		v := strings.TrimSpace(inlineMathValue(n, c.src))
		return Inlines{NewInlineMath(truncateRunes(v, maxValueRunes))}
	default:
		return c.inlines(n)
	}
}

func (c *converter) convertText(n *ast.Text) Inlines {
	val := c.textValue(n)
	var out Inlines
	if val != "" {
		out = append(out, NewText(val))
	}
	if n.HardLineBreak() {
		out = append(out, NewBreak())
		return out
	}
	if !n.SoftLineBreak() {
		return out
	}
	l, lk := c.runeBefore(n)
	if r, ok := lastRune(val); ok {
		l, lk = r, neighborRune
	}
	r, rk := c.runeAfter(n)
	if lk == neighborNone || rk == neighborNone || lk == neighborLineBreak || rk == neighborLineBreak {
		return out
	}
	if lk == neighborRune && rk == neighborRune && removeSoftBreak(l, r) {
		return out
	}
	out = append(out, NewText(" "))
	return out
}

func inlineMathValue(n ast.Node, source []byte) string {
	var b strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		t, ok := child.(*ast.Text)
		if !ok {
			continue
		}
		value := t.Segment.Value(source)
		if len(value) > 0 && value[len(value)-1] == '\n' {
			b.Write(value[:len(value)-1])
			if child != n.LastChild() {
				b.WriteByte(' ')
			}
		} else {
			b.Write(value)
		}
	}
	return b.String()
}

func (c *converter) convertLink(n *ast.Link) Inlines {
	dest := string(n.Destination)
	if id, ok := parseUserMention(dest); ok {
		return Inlines{NewMention(c.userRef(id))}
	}
	text := c.plainText(n)
	if rid, floor, ok := parseReplyRef(dest, text); ok {
		return Inlines{NewReplyReference(rid, floor)}
	}
	kids := c.inlines(n)
	if url, _, ok := c.imageServiceURL(dest); ok {
		return Inlines{NewLink(url, kids)}
	}
	if url, ok := c.normalizeLinkURL(dest); ok {
		return Inlines{NewLink(url, kids)}
	}
	return kids
}

func (c *converter) convertAutoLink(n *ast.AutoLink) Inlines {
	label := string(n.Label(c.src))
	var dest string
	if n.AutoLinkType == ast.AutoLinkEmail {
		dest = "mailto:" + label
	} else {
		dest = string(n.URL(c.src))
	}
	url, ok := parseAllowedURL(dest)
	if !ok {
		if label == "" {
			return nil
		}
		return Inlines{NewText(label)}
	}
	if label == "" {
		return Inlines{NewLink(url, nil)}
	}
	return Inlines{NewLink(url, Inlines{NewText(label)})}
}

func (c *converter) convertImage(dest, alt string) Inlines {
	alt = truncateRunes(alt, maxAltRunes)
	if url, hash, ok := c.imageServiceURL(dest); ok {
		node := NewImage(url, alt, repr.NewImage(c.cdn, hash, c.imageMeta(hash)))
		node.IsSticker = markdown.IsSticker(hash)
		return Inlines{node}
	}
	if url, ok := c.normalizeImageURL(dest); ok {
		return Inlines{NewImage(url, alt, nil)}
	}
	if alt != "" {
		return Inlines{NewText(alt)}
	}
	return nil
}

func (c *converter) emitVideo(n, next ast.Node) bool {
	if next == nil {
		return false
	}
	if _, ok := c.kvPrefix(n); !ok {
		return false
	}
	_, ok := c.videoURL(next)
	return ok
}

func (c *converter) kvPrefix(n ast.Node) (string, bool) {
	s := c.nodeDecoded(n)
	if !strings.HasSuffix(s, "kv:") {
		return "", false
	}
	return strings.TrimSuffix(s, "kv:"), true
}

func (c *converter) nodeDecoded(n ast.Node) string {
	switch t := n.(type) {
	case *ast.Text:
		return c.textValue(t)
	case *ast.String:
		return c.stringValue(t)
	default:
		return ""
	}
}

func (c *converter) videoURL(n ast.Node) (string, bool) {
	switch t := n.(type) {
	case *ast.Link:
		return mp4VideoURL(string(t.Destination))
	case *ast.AutoLink:
		return mp4VideoURL(string(t.URL(c.src)))
	default:
		return "", false
	}
}
