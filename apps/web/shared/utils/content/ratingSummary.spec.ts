import { describe, expect, it } from 'vitest'
import type { ContentDocument } from '../api/schemas'
import { documentPlainText } from './plainText'
import { ratingSummaryContent } from './ratingSummary'

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
