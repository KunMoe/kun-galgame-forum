package content

import (
	"context"
	"strings"

	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/imageclient"

	"github.com/yuin/goldmark/ast"
)

func (c *Converter) ConvertUntrusted(ctx context.Context, source string) (ContentDocument, error) {
	if strings.TrimSpace(source) == "" {
		return NewDocument(nil), nil
	}
	root, src := markdown.ParseStored(source)
	var hashes []string
	seen := map[string]struct{}{}
	collectMarkdownImageHashes(root, func(h string) {
		if h == "" {
			return
		}
		if _, ok := seen[h]; ok {
			return
		}
		seen[h] = struct{}{}
		hashes = append(hashes, h)
	})
	var images map[string]imageclient.ImageMeta
	if len(hashes) > 0 && c.Images != nil {
		images = c.Images(hashes)
	}
	cv := &converter{
		cdn:       c.CDNBase,
		site:      strings.TrimRight(c.SiteBase, "/"),
		src:       src,
		images:    images,
		untrusted: true,
	}
	return NewDocument(cv.blocks(root)), nil
}

func collectMarkdownImageHashes(root ast.Node, addHash func(string)) {
	if root == nil {
		return
	}
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		img, ok := n.(*ast.Image)
		if ok {
			if h, _, parsed := markdown.ParseContentImageRef(string(img.Destination)); parsed {
				addHash(h)
			}
		}
		return ast.WalkContinue, nil
	})
}

func literalRawHTML(raw string) Inlines {
	if raw == "" {
		return nil
	}
	return Inlines{NewText(raw)}
}

func literalHTMLBlock(raw string) Blocks {
	raw = strings.TrimSuffix(raw, "\n")
	if raw == "" {
		return nil
	}
	lines := strings.Split(raw, "\n")
	in := make(Inlines, 0, len(lines)*2-1)
	for i, line := range lines {
		if i > 0 {
			in = append(in, NewBreak())
		}
		if line != "" {
			in = append(in, NewText(line))
		}
	}
	in = mergeInlines(in)
	if len(in) == 0 {
		return nil
	}
	return Blocks{NewParagraph(in)}
}
