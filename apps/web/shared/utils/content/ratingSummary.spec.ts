import { describe, expect, it } from 'vitest'
import type { ContentDocument } from '../api/schemas'
import { documentPlainText } from './plainText'
import { ratingSummaryContent, ratingSummaryExcerpt } from './ratingSummary'

describe('rating summaries from an older API', () => {
  it('uses literal review text when summary_content is absent', () => {
    const document = ratingSummaryContent({ short_summary: '**音乐很好**' })

    expect(documentPlainText(document, '')).toBe('**音乐很好**')
  })

  it('keeps surrounding text and masks inline spoilers when the document is null', () => {
    const document = ratingSummaryContent({
      short_summary: '音乐很好，||第一行\n第二行||，推荐',
      summary_content: null
    })

    expect(document.children).toEqual([
      {
        object: 'paragraph',
        children: [
          { object: 'text', value: '音乐很好，' },
          {
            object: 'inline_spoiler',
            children: [{ object: 'text', value: '第一行\n第二行' }]
          },
          { object: 'text', value: '，推荐' }
        ]
      }
    ])
  })

  it('masks the whole legacy source when it contains a block spoiler', () => {
    const source = '音乐很好\n\n:::spoiler\n秘密结局\n:::'
    const document = ratingSummaryContent({ short_summary: source })

    expect(document.children).toEqual([
      {
        object: 'paragraph',
        children: [
          {
            object: 'inline_spoiler',
            children: [{ object: 'text', value: source }]
          }
        ]
      }
    ])
  })

  it('returns an empty document for an empty legacy summary', () => {
    expect(ratingSummaryContent({ short_summary: '' })).toEqual({
      object: 'document',
      children: []
    })
  })

  it('keeps a document supplied by the new API, including an empty one', () => {
    const document: ContentDocument = { object: 'document', children: [] }

    expect(
      ratingSummaryContent({
        short_summary: '**音乐很好**',
        summary_content: document
      })
    ).toBe(document)
  })
})

describe('public rating excerpts', () => {
  const document: ContentDocument = {
    object: 'document',
    children: [
      {
        object: 'paragraph',
        children: [
          {
            object: 'strong',
            children: [{ object: 'text', value: '音乐很好，' }]
          },
          {
            object: 'inline_spoiler',
            children: [{ object: 'text', value: '秘密结局' }]
          }
        ]
      },
      {
        object: 'spoiler',
        children: [
          {
            object: 'paragraph',
            children: [{ object: 'text', value: '整段秘密' }]
          }
        ]
      }
    ]
  }

  it.each(['none', 'portion'])(
    'masks inline and block spoilers at level %s',
    (level) => {
      expect(
        ratingSummaryExcerpt({
          short_summary:
            '**音乐很好，**||秘密结局||\n\n:::spoiler\n整段秘密\n:::',
          summary_content: document,
          spoiler_level: level
        })
      ).toBe('音乐很好，███\n███')
    }
  )

  it('replaces the entire excerpt with a notice for a serious spoiler', () => {
    expect(
      ratingSummaryExcerpt({
        short_summary: '秘密',
        summary_content: document,
        spoiler_level: 'serious'
      })
    ).toBe('[剧透内容已屏蔽，点击详情查看]')
  })

  it('replaces a partial review without an explicit spoiler mask with a notice', () => {
    expect(
      ratingSummaryExcerpt({
        short_summary: '秘密结局',
        spoiler_level: 'portion'
      })
    ).toBe('[剧透内容已屏蔽，点击详情查看]')
  })

  it('keeps an empty review without spoilers empty', () => {
    expect(
      ratingSummaryExcerpt({
        short_summary: '',
        spoiler_level: 'none'
      })
    ).toBe('')
  })

  it('does not leak block spoilers from an older API response', () => {
    expect(
      ratingSummaryExcerpt({
        short_summary: ':::spoiler\n秘密结局\n:::',
        spoiler_level: 'none'
      })
    ).toBe('███')
  })
})
