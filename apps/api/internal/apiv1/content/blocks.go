package content

import (
	"strings"

	"kun-galgame-api/internal/infrastructure/markdown"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

func (c *converter) blocks(parent ast.Node) Blocks {
	if parent == nil {
		return nil
	}
	var out Blocks
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		out = append(out, c.convertBlock(n)...)
	}
	return out
}

func (c *converter) convertBlock(n ast.Node) Blocks {
	switch n.Kind() {
	case ast.KindParagraph, ast.KindTextBlock:
		in := c.inlines(n)
		if len(in) == 0 {
			return nil
		}
		if card, ok := entityCard(in); ok {
			return Blocks{card}
		}
		return Blocks{NewParagraph(in)}
	case ast.KindHeading:
		return c.heading(n.(*ast.Heading))
	case ast.KindThematicBreak:
		return Blocks{NewThematicBreak()}
	case ast.KindBlockquote:
		return Blocks{NewBlockquote(c.blocks(n))}
	case ast.KindList:
		return Blocks{c.list(n.(*ast.List))}
	case ast.KindListItem:
		return c.blocks(n)
	case ast.KindFencedCodeBlock:
		return Blocks{c.fencedCode(n.(*ast.FencedCodeBlock))}
	case ast.KindCodeBlock:
		return Blocks{NewCode(nil, truncateRunes(nodeLines(n, c.src), maxValueRunes))}
	case ast.KindHTMLBlock:
		return c.htmlBlock(n.(*ast.HTMLBlock))
	case markdown.KindSpoilerBlock:
		return Blocks{NewSpoiler(c.blocks(n))}
	case mathjax.KindMathBlock:
		v := strings.TrimSpace(nodeLines(n, c.src))
		return Blocks{NewMath(truncateRunes(v, maxValueRunes))}
	case east.KindTable:
		return Blocks{c.table(n.(*east.Table))}
	default:
		return c.blocks(n)
	}
}

func (c *converter) heading(h *ast.Heading) Blocks {
	depth := h.Level
	if depth < 2 {
		depth = 2
	}
	if depth > 6 {
		depth = 6
	}
	anchor := markdown.HeadingID(h)
	anchor = truncateRunes(anchor, maxAnchor)
	if anchor == "" {
		anchor = "heading"
	}
	return Blocks{NewHeading(depth, anchor, c.inlines(h))}
}

func (c *converter) list(n *ast.List) ListNode {
	var start *int
	if n.IsOrdered() {
		s := n.Start
		if s < 0 {
			s = 0
		}
		start = &s
	}
	var items ListItems
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		li, ok := child.(*ast.ListItem)
		if !ok {
			continue
		}
		items = append(items, c.listItem(li))
	}
	return NewList(n.IsOrdered(), start, !n.IsTight, items)
}

func (c *converter) listItem(n *ast.ListItem) ListItemNode {
	var checked *bool
	if first := n.FirstChild(); first != nil {
		if cb, ok := first.FirstChild().(*east.TaskCheckBox); ok {
			v := cb.IsChecked
			checked = &v
		}
	}
	return NewListItem(checked, c.blocks(n))
}

func (c *converter) fencedCode(n *ast.FencedCodeBlock) CodeNode {
	var info string
	if n.Info != nil {
		info = string(n.Info.Segment.Value(c.src))
	}
	return NewCode(fenceLang(info), truncateRunes(nodeLines(n, c.src), maxValueRunes))
}

func (c *converter) table(n *east.Table) TableNode {
	var rows TableRows
	for row := n.FirstChild(); row != nil; row = row.NextSibling() {
		var cells TableCells
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			tc, ok := cell.(*east.TableCell)
			if !ok {
				continue
			}
			cells = append(cells, NewTableCell(alignPtr(tc.Alignment), c.inlines(tc)))
		}
		rows = append(rows, NewTableRow(cells))
	}
	return NewTable(rows)
}

func alignPtr(a east.Alignment) *string {
	var s string
	switch a {
	case east.AlignLeft:
		s = "left"
	case east.AlignCenter:
		s = "center"
	case east.AlignRight:
		s = "right"
	default:
		return nil
	}
	return &s
}
