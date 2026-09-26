import { Fragment, h, type FunctionalComponent, type VNode } from 'vue'
import katex from 'katex'
import { toKunUser } from '~/utils/userRef'
import type {
  BlockNode,
  CodeNode,
  HeadingNode,
  ImageNode,
  InlineNode,
  LinkNode,
  ListItemNode,
  ListNode,
  MentionNode,
  ParagraphNode,
  ReplyReferenceNode,
  TableCellNode,
  TableNode,
  VideoNode
} from '#shared/utils/api/schemas'
import { isDecimalId, isRecord, isSafeHref, isSafeMediaUrl } from './guards'

type ContentNode = BlockNode | InlineNode
type Rendered = VNode | string | (VNode | string)[] | null
type Ctx = { tight: boolean }

const isContentNode = (value: unknown): value is ContentNode =>
  isRecord(value) && typeof value.object === 'string'

const flatten = (rendered: Rendered, out: (VNode | string)[]): void => {
  if (rendered == null) {
    return
  }
  if (Array.isArray(rendered)) {
    for (const item of rendered) {
      flatten(item, out)
    }
    return
  }
  out.push(rendered)
}

const childrenOf = (
  nodes: readonly unknown[],
  ctx: Ctx
): (VNode | string)[] => {
  const out: (VNode | string)[] = []
  for (const node of nodes) {
    if (!isContentNode(node)) {
      continue
    }
    flatten(renderNode(node, ctx), out)
  }
  return out
}

const inlines = (nodes: readonly InlineNode[], ctx: Ctx) =>
  childrenOf(nodes, ctx)

const blocks = (nodes: readonly BlockNode[], ctx: Ctx) => childrenOf(nodes, ctx)

const renderUnknown = (node: unknown, ctx: Ctx): Rendered => {
  if (!isRecord(node)) {
    return null
  }
  if (Array.isArray(node.children)) {
    return childrenOf(node.children, ctx)
  }
  if (typeof node.value === 'string') {
    return node.value
  }
  return null
}

const renderParagraph = (node: ParagraphNode, ctx: Ctx): VNode => {
  if (node.children.length === 0) {
    return h('p', [h('br')])
  }
  return h('p', inlines(node.children, ctx))
}

const renderHeading = (node: HeadingNode, ctx: Ctx): VNode => {
  const depth = Math.min(Math.max(Math.trunc(node.depth), 2), 6)
  return h(`h${depth}`, { id: node.anchor }, inlines(node.children, ctx))
}

const taskBox = (item: ListItemNode): (VNode | string)[] =>
  item.is_checked === null
    ? []
    : [
        h('input', {
          type: 'checkbox',
          disabled: true,
          ...(item.is_checked ? { checked: true } : {})
        }),
        ' '
      ]

const renderListItem = (item: ListItemNode, ctx: Ctx): VNode => {
  const body: (VNode | string)[] = []
  let box = taskBox(item)
  for (const child of item.children) {
    if (child.object !== 'paragraph') {
      body.push(...box)
      box = []
      flatten(renderNode(child, ctx), body)
      continue
    }
    const content = inlines(child.children, ctx)
    const line = content.length > 0 ? [...box, ...content] : [...box, h('br')]
    box = []
    if (ctx.tight) {
      body.push(...line)
    } else {
      body.push(h('p', line))
    }
  }
  body.push(...box)
  return h('li', body)
}

const renderList = (node: ListNode): VNode => {
  const tag = node.is_ordered ? 'ol' : 'ul'
  const props =
    node.is_ordered && node.start !== null && node.start !== 1
      ? { start: node.start }
      : {}
  const itemCtx: Ctx = { tight: !node.is_spread }
  return h(
    tag,
    props,
    node.children.map((item) => renderListItem(item, itemCtx))
  )
}

const renderCode = (node: CodeNode): VNode => {
  const lang = node.lang
  const containerClass = lang
    ? `kun-code-container language-${lang}`
    : 'kun-code-container'
  const codeProps = lang ? { class: `language-${lang}` } : {}
  return h('div', { class: containerClass }, [
    h('div', { class: 'kun-code-header' }, [
      h('span', { class: 'lang' }, lang ?? ''),
      h('button', { class: 'copy', title: 'Copy code' })
    ]),
    h('pre', [h('code', codeProps, node.value)])
  ])
}

const renderMath = (value: string, display: boolean): VNode => {
  const tag = display ? 'div' : 'span'
  const cls = display ? 'math display' : 'math inline'
  try {
    const html = katex.renderToString(value, {
      displayMode: display,
      throwOnError: false
    })
    return h(tag, { class: cls, innerHTML: html })
  } catch {
    return h(tag, { class: cls }, value)
  }
}

const renderCell = (tag: 'th' | 'td', cell: TableCellNode, ctx: Ctx): VNode => {
  const props =
    cell.align !== null ? { style: `text-align: ${cell.align}` } : {}
  return h(tag, props, inlines(cell.children, ctx))
}

const renderTable = (node: TableNode, ctx: Ctx): VNode => {
  const [header, ...body] = node.children
  const tableChildren: VNode[] = []
  if (header) {
    tableChildren.push(
      h('thead', [
        h(
          'tr',
          header.children.map((cell) => renderCell('th', cell, ctx))
        )
      ])
    )
  }
  if (body.length > 0) {
    tableChildren.push(
      h(
        'tbody',
        body.map((row) =>
          h(
            'tr',
            row.children.map((cell) => renderCell('td', cell, ctx))
          )
        )
      )
    )
  }
  return h('div', { class: 'kun-table-container' }, [h('table', tableChildren)])
}

const renderLink = (node: LinkNode, ctx: Ctx): Rendered => {
  const children = inlines(node.children, ctx)
  if (!isSafeHref(node.url)) {
    return children
  }
  return h('a', { href: node.url, rel: 'nofollow' }, children)
}

const renderImage = (node: ImageNode): Rendered => {
  if (!isSafeMediaUrl(node.url)) {
    return node.alt
  }
  const props: Record<string, string | number> = {
    src: node.url,
    alt: node.alt,
    loading: 'lazy',
    decoding: 'async',
    'data-kun-lazy-image': 'true'
  }
  if (node.is_sticker) {
    props.class = 'kun-sticker'
  }
  if (node.image) {
    if (node.image.width !== null) {
      props.width = node.image.width
    }
    if (node.image.height !== null) {
      props.height = node.image.height
    }
    if (node.image.thumbhash !== null) {
      props['data-thumbhash'] = node.image.thumbhash
    }
  }
  return h('img', props)
}

const renderVideo = (node: VideoNode): Rendered => {
  if (!isSafeMediaUrl(node.url)) {
    return null
  }
  return h('video', {
    controls: true,
    loop: true,
    playsinline: '',
    width: '100%',
    src: node.url
  })
}

const renderMention = (node: MentionNode): Rendered => {
  const label = `@${toKunUser(node.mentioned_user).name}`
  const id = node.mentioned_user.id
  if (!isDecimalId(id)) {
    return label
  }
  return h(
    'a',
    {
      class: 'kun-mention',
      'data-uid': id,
      href: `/user/${id}/info`,
      rel: 'nofollow'
    },
    label
  )
}

const renderReplyReference = (node: ReplyReferenceNode): Rendered => {
  const label = `#${node.floor}`
  if (!isDecimalId(node.reply_id)) {
    return label
  }
  return h(
    'span',
    {
      class: 'kun-quote',
      'data-reply-id': node.reply_id,
      'data-floor': node.floor
    },
    label
  )
}

const renderNode = (node: ContentNode, ctx: Ctx): Rendered => {
  switch (node.object) {
    case 'paragraph':
      return renderParagraph(node, ctx)
    case 'heading':
      return renderHeading(node, ctx)
    case 'thematic_break':
      return h('hr')
    case 'blockquote':
      return h('blockquote', blocks(node.children, ctx))
    case 'list':
      return renderList(node)
    case 'code':
      return renderCode(node)
    case 'math':
      return renderMath(node.value, true)
    case 'table':
      return renderTable(node, ctx)
    case 'spoiler':
      return h(
        'div',
        { class: 'kun-spoiler text-transparent kun-spoiler-hidden' },
        blocks(node.children, ctx)
      )
    case 'text':
      return node.value
    case 'emphasis':
      return h('em', inlines(node.children, ctx))
    case 'strong':
      return h('strong', inlines(node.children, ctx))
    case 'strikethrough':
      return h('del', inlines(node.children, ctx))
    case 'inline_code':
      return h('code', node.value)
    case 'inline_math':
      return renderMath(node.value, false)
    case 'break':
      return h('br')
    case 'link':
      return renderLink(node, ctx)
    case 'image':
      return renderImage(node)
    case 'video':
      return renderVideo(node)
    case 'inline_spoiler':
      return h(
        'span',
        { class: 'kun-spoiler text-transparent kun-spoiler-hidden' },
        inlines(node.children, ctx)
      )
    case 'mention':
      return renderMention(node)
    case 'reply_reference':
      return renderReplyReference(node)
    default:
      return renderUnknown(node, ctx)
  }
}

const ContentNodes: FunctionalComponent<{ nodes: ContentNode[] }> = (props) =>
  h(Fragment, childrenOf(props.nodes, { tight: false }))

ContentNodes.props = {
  nodes: {
    type: Array,
    required: true
  }
}

export default ContentNodes
