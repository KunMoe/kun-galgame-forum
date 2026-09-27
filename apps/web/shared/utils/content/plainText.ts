import type { ContentDocument } from '../api/schemas'

const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === 'object'

const objectOf = (node: unknown): string | undefined =>
  isRecord(node) && typeof node.object === 'string' ? node.object : undefined

const childrenOf = (node: unknown): unknown[] =>
  isRecord(node) && Array.isArray(node.children) ? node.children : []

const nodesOf = (
  input: ContentDocument | readonly unknown[]
): readonly unknown[] =>
  Array.isArray(input)
    ? input
    : isRecord(input) && Array.isArray(input.children)
      ? input.children
      : []

const stringValue = (node: unknown): string =>
  isRecord(node) && typeof node.value === 'string' ? node.value : ''

const BLOCK_JOIN = new Set([
  'document',
  'blockquote',
  'spoiler',
  'list',
  'list_item',
  'table'
])

const OMIT = new Set(['image', 'video', 'math', 'inline_math'])

const mentionName = (node: unknown, deletedMentionLabel: string): string => {
  if (!isRecord(node) || !isRecord(node.mentioned_user)) {
    return deletedMentionLabel
  }
  const name = node.mentioned_user.name
  return typeof name === 'string' ? name : deletedMentionLabel
}

const replyFloor = (node: unknown): string => {
  if (isRecord(node) && typeof node.floor === 'number') {
    return `#${node.floor}`
  }
  return '#0'
}

const nodeText = (node: unknown, deletedMentionLabel: string): string => {
  const object = objectOf(node)
  if (object === 'text' || object === 'inline_code' || object === 'code') {
    return stringValue(node)
  }
  if (object === 'break') {
    return '\n'
  }
  if (object === 'mention') {
    return `@${mentionName(node, deletedMentionLabel)}`
  }
  if (object === 'reply_reference') {
    return replyFloor(node)
  }
  if (object !== undefined && OMIT.has(object)) {
    return ''
  }
  const children = childrenOf(node)
  if (children.length > 0) {
    if (object === 'table_row') {
      return children
        .map((child) => nodeText(child, deletedMentionLabel))
        .join(' ')
    }
    if (object !== undefined && BLOCK_JOIN.has(object)) {
      return joinBlocks(children, deletedMentionLabel)
    }
    return children
      .map((child) => nodeText(child, deletedMentionLabel))
      .join('')
  }
  return stringValue(node)
}

const joinBlocks = (
  nodes: readonly unknown[],
  deletedMentionLabel: string
): string => nodes.map((node) => nodeText(node, deletedMentionLabel)).join('\n')

export const documentPlainText = (
  input: ContentDocument | readonly unknown[] | null | undefined,
  deletedMentionLabel: string
): string => {
  if (input == null) {
    return ''
  }
  return joinBlocks(nodesOf(input), deletedMentionLabel)
}

const hasOmitted = (nodes: readonly unknown[]): boolean =>
  nodes.some((node) => {
    const object = objectOf(node)
    return (
      (object !== undefined && OMIT.has(object)) || hasOmitted(childrenOf(node))
    )
  })

export const documentHasContent = (
  input: ContentDocument | readonly unknown[] | null | undefined
): boolean =>
  input != null &&
  (documentPlainText(input, '').trim() !== '' || hasOmitted(nodesOf(input)))

export const firstImageUrl = (
  input: ContentDocument | readonly unknown[] | null | undefined
): string | undefined => {
  if (input == null) {
    return undefined
  }
  const walk = (nodes: readonly unknown[]): string | undefined => {
    for (const node of nodes) {
      if (
        isRecord(node) &&
        node.object === 'image' &&
        typeof node.url === 'string' &&
        node.url
      ) {
        return node.url
      }
      const found = walk(childrenOf(node))
      if (found) {
        return found
      }
    }
    return undefined
  }
  return walk(nodesOf(input))
}

export const documentImageHashes = (
  input: ContentDocument | readonly unknown[] | null | undefined
): Set<string> => {
  const hashes = new Set<string>()
  if (input == null) {
    return hashes
  }
  const walk = (nodes: readonly unknown[]) => {
    for (const node of nodes) {
      if (
        isRecord(node) &&
        node.object === 'image' &&
        isRecord(node.image) &&
        typeof node.image.hash === 'string'
      ) {
        hashes.add(node.image.hash)
      }
      walk(childrenOf(node))
    }
  }
  walk(nodesOf(input))
  return hashes
}

export type DocumentHeading = {
  depth: number
  anchor: string
  text: string
}

export const documentHeadings = (
  input: ContentDocument | readonly unknown[] | null | undefined,
  deletedMentionLabel: string
): DocumentHeading[] => {
  const items: DocumentHeading[] = []
  if (input == null) {
    return items
  }
  const walk = (nodes: readonly unknown[]) => {
    for (const node of nodes) {
      if (isRecord(node) && node.object === 'heading') {
        items.push({
          depth: typeof node.depth === 'number' ? node.depth : 2,
          anchor: typeof node.anchor === 'string' ? node.anchor : '',
          text: childrenOf(node)
            .map((child) => nodeText(child, deletedMentionLabel))
            .join('')
        })
      }
      walk(childrenOf(node))
    }
  }
  walk(nodesOf(input))
  return items
}
