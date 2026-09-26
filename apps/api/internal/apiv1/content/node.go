package content

import "kun-galgame-api/internal/apiv1/repr"

type ContentDocument struct {
	Object   string `json:"object" enum:"document" maxLength:"8" doc:"Type discriminant. Always document."`
	Children Blocks `json:"children" doc:"Top-level block nodes in document order. Empty array for an empty body."`
}

type ParagraphNode struct {
	Object   string  `json:"object" enum:"paragraph" maxLength:"9" doc:"Type discriminant. Always paragraph."`
	Children Inlines `json:"children" doc:"Inline nodes of the paragraph. Empty array for a blank line the author kept."`
}

type HeadingNode struct {
	Object   string  `json:"object" enum:"heading" maxLength:"7" doc:"Type discriminant. Always heading."`
	Depth    int     `json:"depth" minimum:"2" maximum:"6" doc:"Heading level, 2 to 6. The page title is the only level 1, so a level-1 heading in the source arrives as 2."`
	Anchor   string  `json:"anchor" minLength:"1" maxLength:"128" pattern:"^\\S+$" doc:"Fragment identifier of this heading, unique within the document. Links in the same body point at it as #anchor."`
	Children Inlines `json:"children" doc:"Inline nodes of the heading text."`
}

type ThematicBreakNode struct {
	Object string `json:"object" enum:"thematic_break" maxLength:"14" doc:"Type discriminant. Always thematic_break."`
}

type BlockquoteNode struct {
	Object   string `json:"object" enum:"blockquote" maxLength:"10" doc:"Type discriminant. Always blockquote."`
	Children Blocks `json:"children" doc:"Quoted block nodes."`
}

type ListNode struct {
	Object    string    `json:"object" enum:"list" maxLength:"4" doc:"Type discriminant. Always list."`
	IsOrdered bool      `json:"is_ordered" doc:"Whether the items are numbered."`
	Start     *int      `json:"start" minimum:"0" doc:"Number of the first item of an ordered list. null for an unordered list."`
	IsSpread  bool      `json:"is_spread" doc:"Whether the source separates items with blank lines, which renders them with paragraph spacing."`
	Children  ListItems `json:"children" doc:"Items of the list."`
}

type ListItemNode struct {
	Object    string `json:"object" enum:"list_item" maxLength:"9" doc:"Type discriminant. Always list_item."`
	IsChecked *bool  `json:"is_checked" doc:"Task-list state: true when ticked, false when not. null when the item is not a task."`
	Children  Blocks `json:"children" doc:"Block nodes of the item."`
}

type CodeNode struct {
	Object string  `json:"object" enum:"code" maxLength:"4" doc:"Type discriminant. Always code."`
	Lang   *string `json:"lang" minLength:"1" maxLength:"32" pattern:"^\\S+$" doc:"Language named on the fence, lowercased. null when the fence names none or the block is indented."`
	Value  string  `json:"value" maxLength:"100007" doc:"Source text of the block, without the fences. Free text; never use it as a decision input."`
}

type MathNode struct {
	Object string `json:"object" enum:"math" maxLength:"4" doc:"Type discriminant. Always math."`
	Value  string `json:"value" maxLength:"100007" doc:"TeX source of a display formula, without the delimiters. Free text; never use it as a decision input."`
}

type TableNode struct {
	Object   string    `json:"object" enum:"table" maxLength:"5" doc:"Type discriminant. Always table."`
	Children TableRows `json:"children" doc:"Rows of the table. The first row is the header row."`
}

type TableRowNode struct {
	Object   string     `json:"object" enum:"table_row" maxLength:"9" doc:"Type discriminant. Always table_row."`
	Children TableCells `json:"children" doc:"Cells of the row, one per column."`
}

type TableCellNode struct {
	Object   string  `json:"object" enum:"table_cell" maxLength:"10" doc:"Type discriminant. Always table_cell."`
	Align    *string `json:"align" enum:"left,center,right" maxLength:"6" doc:"Horizontal alignment of the column. null for the default."`
	Children Inlines `json:"children" doc:"Inline nodes of the cell."`
}

type SpoilerNode struct {
	Object   string `json:"object" enum:"spoiler" maxLength:"7" doc:"Type discriminant. Always spoiler."`
	Children Blocks `json:"children" doc:"Block nodes hidden until the reader reveals them."`
}

type TextNode struct {
	Object string `json:"object" enum:"text" maxLength:"4" doc:"Type discriminant. Always text."`
	Value  string `json:"value" maxLength:"100007" doc:"Literal text. Escapes and entities are already decoded; render it as text, never as markup. Free text; never use it as a decision input."`
}

type EmphasisNode struct {
	Object   string  `json:"object" enum:"emphasis" maxLength:"8" doc:"Type discriminant. Always emphasis."`
	Children Inlines `json:"children" doc:"Emphasized inline nodes."`
}

type StrongNode struct {
	Object   string  `json:"object" enum:"strong" maxLength:"6" doc:"Type discriminant. Always strong."`
	Children Inlines `json:"children" doc:"Strongly emphasized inline nodes."`
}

type StrikethroughNode struct {
	Object   string  `json:"object" enum:"strikethrough" maxLength:"13" doc:"Type discriminant. Always strikethrough."`
	Children Inlines `json:"children" doc:"Struck-through inline nodes."`
}

type InlineCodeNode struct {
	Object string `json:"object" enum:"inline_code" maxLength:"11" doc:"Type discriminant. Always inline_code."`
	Value  string `json:"value" maxLength:"100007" doc:"Code text. Free text; never use it as a decision input."`
}

type InlineMathNode struct {
	Object string `json:"object" enum:"inline_math" maxLength:"11" doc:"Type discriminant. Always inline_math."`
	Value  string `json:"value" maxLength:"100007" doc:"TeX source of an inline formula, without the delimiters. Free text; never use it as a decision input."`
}

type BreakNode struct {
	Object string `json:"object" enum:"break" maxLength:"5" doc:"Type discriminant. Always break."`
}

type LinkNode struct {
	Object   string  `json:"object" enum:"link" maxLength:"4" doc:"Type discriminant. Always link."`
	URL      string  `json:"url" format:"uri" maxLength:"2048" doc:"Absolute http, https or mailto URL."`
	Children Inlines `json:"children" doc:"Inline nodes of the link text."`
}

type ImageNode struct {
	Object    string      `json:"object" enum:"image" maxLength:"5" doc:"Type discriminant. Always image."`
	URL       string      `json:"url" format:"uri" maxLength:"2048" doc:"Absolute URL to display inline. For an image-service picture it may be a smaller variant of image.url."`
	Alt       string      `json:"alt" maxLength:"512" doc:"Alternative text. Empty when the author gave none. Free text; never use it as a decision input."`
	Image     *repr.Image `json:"image" doc:"Image-service record of the picture, whose url is the full-size original. null for a picture hosted elsewhere."`
	IsSticker bool        `json:"is_sticker" doc:"Whether the picture is an official sticker, one of the images listStickerPacks returns, judged by its image-service hash and never by alt or url. Show a sticker inline at text size, not as a figure. Always false when image is null."`
}

type VideoNode struct {
	Object string `json:"object" enum:"video" maxLength:"5" doc:"Type discriminant. Always video."`
	URL    string `json:"url" format:"uri" maxLength:"2048" doc:"Absolute URL of the video file."`
}

type InlineSpoilerNode struct {
	Object   string  `json:"object" enum:"inline_spoiler" maxLength:"14" doc:"Type discriminant. Always inline_spoiler."`
	Children Inlines `json:"children" doc:"Inline nodes hidden until the reader reveals them."`
}

type MentionNode struct {
	Object        string       `json:"object" enum:"mention" maxLength:"7" doc:"Type discriminant. Always mention."`
	MentionedUser repr.UserRef `json:"mentioned_user" doc:"The user mentioned, with the current display name. Render it as @name."`
}

type ReplyReferenceNode struct {
	Object  string         `json:"object" enum:"reply_reference" maxLength:"15" doc:"Type discriminant. Always reply_reference."`
	ReplyID repr.DecimalID `json:"reply_id" doc:"Id of the referenced reply in the same topic."`
	Floor   int            `json:"floor" minimum:"1" doc:"Floor number of the referenced reply as the author saw it. Render it as #floor."`
}
