import type { DocCategory } from '#shared/utils/api/schemas'

export type DocEditorMode = 'create' | 'rewrite'

export interface DocEditorForm {
  doc_id: string | null
  title: string
  slug: string
  description: string
  banner_image_hash: string
  is_pinned: boolean
  content_markdown: string
  doc_category: DocCategory | ''
}
