import { describe, expect, it } from 'vitest'
import type { BlockNode, InlineNode } from '../api/schemas'
import { documentPlainText } from './plainText'
import { ratingPreview } from './ratingPreview'

const formatted: InlineNode[] = [
  { object: 'text', value: '主角' },
  { object: 'strong', children: [{ object: 'text', value: '最终' }] },
  { object: 'emphasis', children: [{ object: 'text', value: '选择' }] },
  {
    object: 'link',
    url: 'https://example.com',
    children: [{ object: 'text', value: '牺牲' }]
  }
]

const cases: { name: string; node: BlockNode; expected: string }[] = [
  {
    name: 'keeps mixed inline formatting on the same line',
    node: {
      object: 'paragraph',
      children: [{ object: 'inline_spoiler', children: formatted }]
    },
    expected: '主角最终选择牺牲'
  },
  {
    name: 'preserves an explicit break inside an inline spoiler',
    node: {
      object: 'paragraph',
      children: [
        {
          object: 'inline_spoiler',
          children: [
            ...formatted,
            { object: 'break' },
            { object: 'text', value: '第二行' }
          ]
        }
      ]
    },
    expected: '主角最终选择牺牲\n第二行'
  },
  {
    name: 'preserves paragraph boundaries inside a block spoiler',
    node: {
      object: 'spoiler',
      children: [
        { object: 'paragraph', children: formatted },
        { object: 'paragraph', children: [{ object: 'text', value: '第二段' }] }
      ]
    },
    expected: '主角最终选择牺牲\n第二段'
  }
]

describe('rating spoiler preview text', () => {
  it.each(cases)('$name', ({ node, expected }) => {
    const preview = ratingPreview({ object: 'document', children: [node] })

    expect(preview.hasSpoiler).toBe(true)
    expect(documentPlainText(preview.document, '')).toBe(expected)
    const paragraph = preview.document.children[0]
    expect(paragraph?.object).toBe('paragraph')
    if (paragraph?.object === 'paragraph') {
      expect(paragraph.children[0]?.object).toBe('inline_spoiler')
    }
  })
})
