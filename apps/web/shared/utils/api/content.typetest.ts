import type { components } from '../../types/api/v1'

type Equal<A, B> =
  (<T>() => T extends A ? 1 : 2) extends <T>() => T extends B ? 1 : 2
    ? true
    : false

type Schemas = components['schemas']

declare const block: Schemas['BlockNode']
declare const inline: Schemas['InlineNode']

const _blockTokens: Equal<
  Schemas['BlockNode']['object'],
  | 'paragraph'
  | 'heading'
  | 'thematic_break'
  | 'blockquote'
  | 'list'
  | 'code'
  | 'math'
  | 'table'
  | 'spoiler'
  | 'entity_card'
> = true
void _blockTokens

const _inlineTokens: Equal<
  Schemas['InlineNode']['object'],
  | 'text'
  | 'emphasis'
  | 'strong'
  | 'strikethrough'
  | 'inline_code'
  | 'inline_math'
  | 'break'
  | 'link'
  | 'image'
  | 'video'
  | 'inline_spoiler'
  | 'mention'
  | 'reply_reference'
  | 'entity_link'
> = true
void _inlineTokens

if (inline.object === 'mention') {
  const _name: string | null = inline.mentioned_user.name
  void _name
}

if (inline.object === 'text') {
  // @ts-expect-error a text node has no children
  void inline.children
}

if (block.object === 'list') {
  const _items: Schemas['ListItemNode'][] = block.children
  void _items
}

if (block.object === 'paragraph') {
  // @ts-expect-error a paragraph holds inline nodes, not blocks
  const _blocks: Schemas['BlockNode'][] = block.children
  void _blocks
}
