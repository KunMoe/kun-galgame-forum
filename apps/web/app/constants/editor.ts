import type { KunEditorFeatures } from '@kungal/editor-core'
import type { KunEntityView, KunToolbarItem } from '@kungal/editor-vue'
import EntityCardView from '~/components/kun/milkdown/EntityCardView.vue'
import EntityChipView from '~/components/kun/milkdown/EntityChipView.vue'

export const KUN_EDITOR_TOOLBAR_ITEMS: KunToolbarItem[] = [
  'heading',
  '|',
  'picker',
  'entityPicker',
  '|',
  'bold',
  'italic',
  'strike',
  'code',
  'link',
  '|',
  'bulletList',
  'orderedList',
  'quote',
  'codeBlock',
  'hr',
  '|',
  'spoiler'
]

export const KUN_REVIEW_EDITOR_TOOLBAR_ITEMS: KunToolbarItem[] = [
  'bold',
  'italic',
  'strike',
  'code',
  'link',
  '|',
  'bulletList',
  'orderedList',
  'quote',
  '|',
  'spoiler'
]

export const KUN_REVIEW_EDITOR_FEATURES: KunEditorFeatures = {
  quote: false,
  mention: false,
  sticker: false,
  katex: false,
  codeBlock: false,
  entity: false
}

export const KUN_EDITOR_ENTITY_VIEW: KunEntityView = {
  card: EntityCardView,
  chip: EntityChipView
}

export const KUN_EDITOR_TOOLBAR_OVERRIDES = {
  entityPicker: { icon: 'lucide:id-card', label: '插入条目卡片' }
}
