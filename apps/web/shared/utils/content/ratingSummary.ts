import type { ContentDocument, InlineNode } from '../api/schemas'
import { ratingPreview } from './ratingPreview'

type RatingSummarySource = {
  short_summary: string
  summary_content?: ContentDocument | null
}

export const ratingSummaryContent = (
  rating: RatingSummarySource
): ContentDocument => {
  if (rating.summary_content) return rating.summary_content
  const source = rating.short_summary
  if (!source) return { object: 'document', children: [] }

  const children: InlineNode[] = /:{3,}spoiler\b/.test(source)
    ? [
        {
          object: 'inline_spoiler',
          children: [{ object: 'text', value: source }]
        }
      ]
    : source.split(/\|\|(.*?)\|\|/gs).map((value, index) =>
        index % 2
          ? {
              object: 'inline_spoiler',
              children: [{ object: 'text', value }]
            }
          : { object: 'text', value }
      )

  return {
    object: 'document',
    children: [{ object: 'paragraph', children }]
  }
}

export const ratingSummaryExcerpt = (
  rating: RatingSummarySource & { spoiler_level: string }
): string => {
  const preview = ratingPreview(ratingSummaryContent(rating))
  if (
    rating.spoiler_level === 'serious' ||
    (rating.spoiler_level === 'portion' && !preview.hasSpoiler)
  )
    return '[剧透内容已屏蔽，点击详情查看]'
  return preview.maskedText
}
