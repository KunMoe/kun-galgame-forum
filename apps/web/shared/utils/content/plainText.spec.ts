import { describe, expect, it } from 'vitest'
import type {
  BlockNode,
  ContentDocument,
  InlineNode,
  ListItemNode,
  TableCellNode,
  UserRef
} from '../api/schemas'
import { documentPlainText } from './plainText'

const deleted = '已注销用户'

const text = (value: string): InlineNode => ({ object: 'text', value })

const p = (...children: InlineNode[]): BlockNode => ({
  object: 'paragraph',
  children
})

const doc = (...children: BlockNode[]): ContentDocument => ({
  object: 'document',
  children
})

const user = (over: Partial<UserRef> = {}): UserRef => ({
  object: 'user',
  id: '3',
  name: 'Alice',
  avatar: null,
  ...over
})

const item = (children: BlockNode[]): ListItemNode => ({
  object: 'list_item',
  is_checked: null,
  children
})

const cell = (value: string): TableCellNode => ({
  object: 'table_cell',
  align: null,
  children: [text(value)]
})

const plain = (input: ContentDocument | readonly unknown[]) =>
  documentPlainText(input, deleted)

describe('documentPlainText', () => {
  it('returns an empty string for an empty document', () => {
    expect(plain(doc())).toBe('')
  })

  it('keeps text, inline code, and code values as is', () => {
    expect(
      plain(doc(p(text('hello '), { object: 'inline_code', value: 'x' })))
    ).toBe('hello x')
    expect(
      plain(
        doc({
          object: 'code',
          lang: 'ts',
          value: 'const a = 1'
        })
      )
    ).toBe('const a = 1')
  })

  it('separates blocks with newlines and treats break as a newline', () => {
    expect(plain(doc(p(text('a')), p(text('b'))))).toBe('a\nb')
    expect(plain(doc(p(text('a'), { object: 'break' }, text('b'))))).toBe(
      'a\nb'
    )
  })

  it('renders a living mention as @name and a deleted mention with the given label', () => {
    expect(plain(doc(p({ object: 'mention', mentioned_user: user() })))).toBe(
      '@Alice'
    )
    expect(
      plain(doc(p({ object: 'mention', mentioned_user: user({ name: null }) })))
    ).toBe('@已注销用户')
  })

  it('renders a reply_reference as #floor', () => {
    expect(
      plain(
        doc(
          p({
            object: 'reply_reference',
            reply_id: '9',
            floor: 7
          })
        )
      )
    ).toBe('#7')
  })

  it('omits images, videos, and math', () => {
    expect(
      plain(
        doc(
          p({
            object: 'image',
            url: 'https://cdn.example/a.webp',
            alt: 'pic',
            image: null,
            is_sticker: false
          }),
          {
            object: 'math',
            value: 'e=mc^2'
          },
          p({
            object: 'video',
            url: 'https://cdn.example/a.mp4'
          }),
          p({ object: 'inline_math', value: 'x^2' }, text('kept'))
        )
      )
    ).toBe('\n\n\nkept')
  })

  it('walks heading, emphasis, strong, strikethrough, link, list, blockquote, spoiler, and table children', () => {
    expect(
      plain(
        doc(
          {
            object: 'heading',
            depth: 2,
            anchor: 'hi',
            children: [text('Title')]
          },
          p({ object: 'emphasis', children: [text('em')] }),
          p({ object: 'strong', children: [text('b')] }),
          p({ object: 'strikethrough', children: [text('s')] }),
          p({
            object: 'link',
            url: 'https://example.com',
            children: [text('go')]
          }),
          {
            object: 'blockquote',
            children: [p(text('q'))]
          },
          {
            object: 'spoiler',
            children: [p(text('hid'))]
          },
          {
            object: 'list',
            is_ordered: false,
            start: null,
            is_spread: false,
            children: [item([p(text('one'))]), item([p(text('two'))])]
          },
          {
            object: 'table',
            children: [
              {
                object: 'table_row',
                children: [cell('A'), cell('B')]
              }
            ]
          },
          p({
            object: 'inline_spoiler',
            children: [text('psst')]
          }),
          { object: 'thematic_break' }
        )
      )
    ).toBe('Title\nem\nb\ns\ngo\nq\nhid\none\ntwo\nA B\npsst\n')
  })

  it('uses children of an unknown node, else its string value', () => {
    expect(plain([{ object: 'weird', children: [text('xy')] }])).toBe('xy')
    expect(plain([{ object: 'weird', value: 'zval' }])).toBe('zval')
  })
})
