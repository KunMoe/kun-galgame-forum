import { describe, expect, it } from 'vitest'
import { h } from 'vue'
import { renderToString } from 'vue/server-renderer'
import type {
  BlockNode,
  Image,
  InlineNode,
  ListItemNode,
  TableCellNode,
  UserRef
} from '#shared/utils/api/schemas'
import ContentNodes from './Nodes'

type ContentNode = BlockNode | InlineNode

const html = async (nodes: ContentNode[]) =>
  await renderToString(h(ContentNodes, { nodes }))

const ssr = (inner: string) => `<!--[-->${inner}<!--]-->`

const text = (value: string): InlineNode => ({ object: 'text', value })

const p = (...children: InlineNode[]): BlockNode => ({
  object: 'paragraph',
  children
})

const item = (
  children: BlockNode[],
  is_checked: boolean | null = null
): ListItemNode => ({
  object: 'list_item',
  is_checked,
  children
})

const cell = (
  value: string,
  align: TableCellNode['align'] = null
): TableCellNode => ({
  object: 'table_cell',
  align,
  children: [text(value)]
})

const user = (over: Partial<UserRef> = {}): UserRef => ({
  object: 'user',
  id: '3',
  name: 'Alice',
  avatar: null,
  ...over
})

const hostedImage = (over: Partial<Image> = {}): Image => ({
  hash: 'ab',
  url: 'https://cdn.example/full.webp',
  width: 800,
  height: 600,
  thumbhash: 'thumb',
  sexual: 'safe',
  ...over
})

const unknown = (...nodes: unknown[]) => nodes as ContentNode[]

describe('ContentNodes markup', () => {
  it('renders a paragraph', async () => {
    expect(await html([p(text('hello'))])).toBe(ssr('<p>hello</p>'))
  })

  it('renders an empty paragraph as a kept blank line', async () => {
    expect(await html([p()])).toBe(ssr('<p><br></p>'))
  })

  it('renders a heading of depth 2', async () => {
    expect(
      await html([
        {
          object: 'heading',
          depth: 2,
          anchor: 'intro',
          children: [text('Intro')]
        }
      ])
    ).toBe(ssr('<h2 id="intro">Intro</h2>'))
  })

  it('renders a heading of depth 6', async () => {
    expect(
      await html([
        {
          object: 'heading',
          depth: 6,
          anchor: 'notes',
          children: [text('Notes')]
        }
      ])
    ).toBe(ssr('<h6 id="notes">Notes</h6>'))
  })

  it('renders a thematic break', async () => {
    expect(await html([{ object: 'thematic_break' }])).toBe(ssr('<hr>'))
  })

  it('renders a blockquote', async () => {
    expect(
      await html([{ object: 'blockquote', children: [p(text('q'))] }])
    ).toBe(ssr('<blockquote><p>q</p></blockquote>'))
  })

  it('renders an unordered list', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: false,
          start: null,
          is_spread: false,
          children: [item([p(text('a'))])]
        }
      ])
    ).toBe(ssr('<ul><li>a</li></ul>'))
  })

  it('omits start on an ordered list that starts at 1', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: true,
          start: 1,
          is_spread: false,
          children: [item([p(text('a'))])]
        }
      ])
    ).toBe(ssr('<ol><li>a</li></ol>'))
  })

  it('sets start on an ordered list that starts at 3', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: true,
          start: 3,
          is_spread: false,
          children: [item([p(text('a'))])]
        }
      ])
    ).toBe(ssr('<ol start="3"><li>a</li></ol>'))
  })

  it('sets start on an ordered list that starts at 0', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: true,
          start: 0,
          is_spread: false,
          children: [item([p(text('a'))])]
        }
      ])
    ).toBe(ssr('<ol start="0"><li>a</li></ol>'))
  })

  it('omits start on an ordered list whose start is null', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: true,
          start: null,
          is_spread: false,
          children: [item([p(text('a'))])]
        }
      ])
    ).toBe(ssr('<ol><li>a</li></ol>'))
  })

  it('unwraps paragraphs in a tight list', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: false,
          start: null,
          is_spread: false,
          children: [item([p(text('tight'))])]
        }
      ])
    ).toBe(ssr('<ul><li>tight</li></ul>'))
  })

  it('keeps paragraphs in a spread list', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: false,
          start: null,
          is_spread: true,
          children: [item([p(text('spread'))])]
        }
      ])
    ).toBe(ssr('<ul><li><p>spread</p></li></ul>'))
  })

  it('renders a checked task item', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: false,
          start: null,
          is_spread: false,
          children: [item([p(text('done'))], true)]
        }
      ])
    ).toBe(
      ssr('<ul><li><input type="checkbox" disabled checked> done</li></ul>')
    )
  })

  it('renders an unchecked task item', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: false,
          start: null,
          is_spread: false,
          children: [item([p(text('todo'))], false)]
        }
      ])
    ).toBe(ssr('<ul><li><input type="checkbox" disabled> todo</li></ul>'))
  })

  it("puts a spread task item's checkbox inside its first paragraph", async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: false,
          start: null,
          is_spread: true,
          children: [item([p(text('done')), p(text('more'))], true)]
        }
      ])
    ).toBe(
      ssr(
        '<ul><li><p><input type="checkbox" disabled checked> done</p><p>more</p></li></ul>'
      )
    )
  })

  it('keeps an empty paragraph in a tight list as a blank line', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: false,
          start: null,
          is_spread: false,
          children: [item([p()])]
        }
      ])
    ).toBe(ssr('<ul><li><br></li></ul>'))
  })

  it('puts a task checkbox before a first child that is not a paragraph', async () => {
    expect(
      await html([
        {
          object: 'list',
          is_ordered: false,
          start: null,
          is_spread: false,
          children: [item([{ object: 'thematic_break' }], false)]
        }
      ])
    ).toBe(ssr('<ul><li><input type="checkbox" disabled> <hr></li></ul>'))
  })

  it('clamps a heading depth outside 2 to 6', async () => {
    const heading = (depth: number): BlockNode => ({
      object: 'heading',
      depth,
      anchor: 'a',
      children: [text('A')]
    })
    expect(await html([heading(1), heading(9)])).toBe(
      ssr('<h2 id="a">A</h2><h6 id="a">A</h6>')
    )
  })

  it('renders a code block with a language', async () => {
    expect(await html([{ object: 'code', lang: 'js', value: 'x' }])).toBe(
      ssr(
        '<div class="kun-code-container language-js"><div class="kun-code-header"><span class="lang">js</span><button class="copy" title="Copy code"></button></div><pre><code class="language-js">x</code></pre></div>'
      )
    )
  })

  it('renders a code block without a language', async () => {
    expect(await html([{ object: 'code', lang: null, value: 'x' }])).toBe(
      ssr(
        '<div class="kun-code-container"><div class="kun-code-header"><span class="lang"></span><button class="copy" title="Copy code"></button></div><pre><code>x</code></pre></div>'
      )
    )
  })

  it('renders a header-only table', async () => {
    expect(
      await html([
        {
          object: 'table',
          children: [{ object: 'table_row', children: [cell('H')] }]
        }
      ])
    ).toBe(
      ssr(
        '<div class="kun-table-container"><table><thead><tr><th>H</th></tr></thead></table></div>'
      )
    )
  })

  it('renders a table with body rows and all alignments', async () => {
    expect(
      await html([
        {
          object: 'table',
          children: [
            {
              object: 'table_row',
              children: [
                cell('a', 'left'),
                cell('b', 'center'),
                cell('c', 'right'),
                cell('d', null)
              ]
            },
            {
              object: 'table_row',
              children: [
                cell('e', 'left'),
                cell('f', 'center'),
                cell('g', 'right'),
                cell('h', null)
              ]
            }
          ]
        }
      ])
    ).toBe(
      ssr(
        '<div class="kun-table-container"><table><thead><tr><th style="text-align: left">a</th><th style="text-align: center">b</th><th style="text-align: right">c</th><th>d</th></tr></thead><tbody><tr><td style="text-align: left">e</td><td style="text-align: center">f</td><td style="text-align: right">g</td><td>h</td></tr></tbody></table></div>'
      )
    )
  })

  it('renders a block spoiler', async () => {
    expect(await html([{ object: 'spoiler', children: [p(text('s'))] }])).toBe(
      ssr(
        '<div class="kun-spoiler text-transparent kun-spoiler-hidden"><p>s</p></div>'
      )
    )
  })

  it('renders a text node', async () => {
    expect(await html([text('plain')])).toBe(ssr('plain'))
  })

  it('renders emphasis', async () => {
    expect(await html([{ object: 'emphasis', children: [text('x')] }])).toBe(
      ssr('<em>x</em>')
    )
  })

  it('renders strong', async () => {
    expect(await html([{ object: 'strong', children: [text('x')] }])).toBe(
      ssr('<strong>x</strong>')
    )
  })

  it('renders strikethrough', async () => {
    expect(
      await html([{ object: 'strikethrough', children: [text('x')] }])
    ).toBe(ssr('<del>x</del>'))
  })

  it('renders inline code', async () => {
    expect(await html([{ object: 'inline_code', value: 'x' }])).toBe(
      ssr('<code>x</code>')
    )
  })

  it('renders a break', async () => {
    expect(await html([{ object: 'break' }])).toBe(ssr('<br>'))
  })

  it('renders a link', async () => {
    expect(
      await html([
        {
          object: 'link',
          url: 'https://example.com/',
          children: [text('go')]
        }
      ])
    ).toBe(ssr('<a href="https://example.com/" rel="nofollow">go</a>'))
  })

  it('renders mailto and uppercase-scheme links', async () => {
    expect(
      await html([
        { object: 'link', url: 'mailto:a@example.com', children: [text('m')] },
        { object: 'link', url: 'HTTPS://EXAMPLE.COM/', children: [text('u')] }
      ])
    ).toBe(
      ssr(
        '<a href="mailto:a@example.com" rel="nofollow">m</a><a href="HTTPS://EXAMPLE.COM/" rel="nofollow">u</a>'
      )
    )
  })

  it('renders an image with full metadata', async () => {
    expect(
      await html([
        {
          object: 'image',
          url: 'https://cdn.example/small.webp',
          alt: 'cover',
          image: hostedImage(),
          is_sticker: false
        }
      ])
    ).toBe(
      ssr(
        '<img src="https://cdn.example/small.webp" alt="cover" loading="lazy" decoding="async" data-kun-lazy-image="true" width="800" height="600" data-thumbhash="thumb">'
      )
    )
  })

  it('renders an image with image null', async () => {
    expect(
      await html([
        {
          object: 'image',
          url: 'https://cdn.example/offsite.png',
          alt: 'off',
          image: null,
          is_sticker: false
        }
      ])
    ).toBe(
      ssr(
        '<img src="https://cdn.example/offsite.png" alt="off" loading="lazy" decoding="async" data-kun-lazy-image="true">'
      )
    )
  })

  it('omits width height and thumbhash when those image fields are null', async () => {
    expect(
      await html([
        {
          object: 'image',
          url: 'https://cdn.example/small.webp',
          alt: 'cover',
          image: hostedImage({
            width: null,
            height: null,
            thumbhash: null
          }),
          is_sticker: false
        }
      ])
    ).toBe(
      ssr(
        '<img src="https://cdn.example/small.webp" alt="cover" loading="lazy" decoding="async" data-kun-lazy-image="true">'
      )
    )
  })

  it('renders a video', async () => {
    expect(
      await html([{ object: 'video', url: 'https://cdn.example/a.mp4' }])
    ).toBe(
      ssr(
        '<video controls loop playsinline width="100%" src="https://cdn.example/a.mp4"></video>'
      )
    )
  })

  it('renders an inline spoiler', async () => {
    expect(
      await html([{ object: 'inline_spoiler', children: [text('s')] }])
    ).toBe(
      ssr(
        '<span class="kun-spoiler text-transparent kun-spoiler-hidden">s</span>'
      )
    )
  })

  it('renders a mention with a name', async () => {
    expect(await html([{ object: 'mention', mentioned_user: user() }])).toBe(
      ssr(
        '<a class="kun-mention" data-uid="3" href="/user/3/info" rel="nofollow">@Alice</a>'
      )
    )
  })

  it('renders a mention whose name is null via toKunUser', async () => {
    expect(
      await html([{ object: 'mention', mentioned_user: user({ name: null }) }])
    ).toBe(
      ssr(
        '<a class="kun-mention" data-uid="3" href="/user/3/info" rel="nofollow">@已注销用户</a>'
      )
    )
  })

  it('renders a reply reference', async () => {
    expect(
      await html([{ object: 'reply_reference', reply_id: '48', floor: 2 }])
    ).toBe(
      ssr('<span class="kun-quote" data-reply-id="48" data-floor="2">#2</span>')
    )
  })
})

describe('ContentNodes XSS guards', () => {
  it('escapes HTML in a text node', async () => {
    expect(await html([p(text('<img src=x onerror=alert(1)>'))])).toBe(
      ssr('<p>&lt;img src=x onerror=alert(1)&gt;</p>')
    )
  })

  it('drops a javascript: link and keeps its children', async () => {
    expect(
      await html([
        {
          object: 'link',
          url: 'javascript:alert(1)',
          children: [text('click')]
        }
      ])
    ).toBe(ssr('click'))
  })

  it('renders a data: image as its alt text', async () => {
    expect(
      await html([
        {
          object: 'image',
          url: 'data:image/png;base64,xx',
          alt: 'safe alt',
          image: null,
          is_sticker: false
        }
      ])
    ).toBe(ssr('safe alt'))
  })

  it('renders a mention with a non-decimal id as plain text', async () => {
    const out = await html([
      {
        object: 'mention',
        mentioned_user: user({ id: '1" onmouseover="x' })
      }
    ])
    expect(out).toBe(ssr('@Alice'))
    expect(out).not.toContain('onmouseover')
    expect(out).not.toContain('data-uid')
  })
})

describe('ContentNodes media and reference guards', () => {
  it('renders nothing for a video whose url is not http(s)', async () => {
    expect(
      await html([p({ object: 'video', url: 'javascript:alert(1)' })])
    ).toBe(ssr('<p></p>'))
  })

  it('renders a reply reference with a non-decimal id as plain text', async () => {
    const out = await html([
      { object: 'reply_reference', reply_id: '48" onclick="x', floor: 2 }
    ])
    expect(out).toBe(ssr('#2'))
  })
})

describe('ContentNodes unknown nodes', () => {
  it('renders the children of an unknown node', async () => {
    expect(
      await html(
        unknown({
          object: 'future_block',
          children: [text('kept')]
        })
      )
    ).toBe(ssr('kept'))
  })

  it('renders the value of an unknown node as text', async () => {
    expect(
      await html(unknown({ object: 'future_inline', value: 'plain' }))
    ).toBe(ssr('plain'))
  })

  it('renders nothing for an unknown node with neither children nor value', async () => {
    expect(await html(unknown({ object: 'future_void' }))).toBe(ssr(''))
  })
})

describe('ContentNodes math', () => {
  it('renders display math with KaTeX markup', async () => {
    const out = await html([{ object: 'math', value: 'x^2' }])
    expect(out.startsWith('<!--[--><div class="math display">')).toBe(true)
    expect(out).toContain('class="katex"')
    expect(out.endsWith('</div><!--]-->')).toBe(true)
  })

  it('renders inline math with KaTeX markup', async () => {
    const out = await html([{ object: 'inline_math', value: 'x^2' }])
    expect(out.startsWith('<!--[--><span class="math inline">')).toBe(true)
    expect(out).toContain('class="katex"')
    expect(out.endsWith('</span><!--]-->')).toBe(true)
  })

  it('shows invalid display math as a KaTeX error with its source', async () => {
    const out = await html([{ object: 'math', value: '\\invalid{{{' }])
    expect(out.startsWith('<!--[--><div class="math display">')).toBe(true)
    expect(out).toContain('katex-error')
    expect(out).toContain('\\invalid{{{')
  })

  it('shows invalid inline math as a KaTeX error with its source', async () => {
    const out = await html([{ object: 'inline_math', value: '\\invalid{{{' }])
    expect(out.startsWith('<!--[--><span class="math inline">')).toBe(true)
    expect(out).toContain('katex-error')
    expect(out).toContain('\\invalid{{{')
  })
})
