import type {
  BlockNode,
  ContentDocument,
  InlineNode,
  ListItemNode,
  TableCellNode,
  TableRowNode
} from '../api/schemas'
import { documentPlainText } from './plainText'

type Node = BlockNode | InlineNode | ListItemNode | TableCellNode | TableRowNode

const BLOCK_END = new Set([
  'paragraph',
  'heading',
  'code',
  'blockquote',
  'list_item',
  'table_row',
  'spoiler'
])

export const ratingPreview = (
  document: ContentDocument
): { document: ContentDocument; hasSpoiler: boolean } => {
  const children: InlineNode[] = []
  let hasSpoiler = false

  const appendText = (value: string) => {
    if (!value) return
    const last = children.at(-1)
    if (last?.object === 'text') {
      last.value += value
    } else {
      children.push({ object: 'text', value })
    }
  }

  const walk = (nodes: readonly Node[]) => {
    for (const node of nodes) {
      switch (node.object) {
        case 'inline_spoiler':
        case 'spoiler':
          hasSpoiler = true
          children.push({
            object: 'inline_spoiler',
            children: [
              {
                object: 'text',
                value: documentPlainText(node.children, '')
              }
            ]
          })
          break
        case 'text':
        case 'inline_code':
        case 'code':
          appendText(node.value)
          break
        case 'break':
          appendText('\n')
          break
        case 'mention':
        case 'reply_reference':
          appendText(documentPlainText([node], '已删除用户'))
          break
        case 'image':
        case 'video':
        case 'math':
        case 'inline_math':
          break
        default:
          if ('children' in node) {
            walk(node.children as readonly Node[])
          }
      }
      if (BLOCK_END.has(node.object)) appendText('\n')
      if (node.object === 'table_cell') appendText(' ')
    }
  }

  walk(document.children)
  const last = children.at(-1)
  if (last?.object === 'text') last.value = last.value.trimEnd()

  return {
    document: {
      object: 'document',
      children: children.length ? [{ object: 'paragraph', children }] : []
    },
    hasSpoiler
  }
}
