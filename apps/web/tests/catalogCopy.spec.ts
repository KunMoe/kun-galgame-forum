// @vitest-environment node
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  writeFileSync
} from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

type AllowedPhrase = { phrase: string; reason: string }

const FORBIDDEN = /NextMoe\s*目录|资料库/
const SOURCE = /\.(?:ts|vue|json|m?js)$/
const SKIPPED = /\.(?:d|spec)\.ts$/

export const ALLOWED_PHRASES: AllowedPhrase[] = [
  {
    phrase: 'Galgame 网站资料库',
    reason: 'the forum’s own website directory (/website)'
  },
  { phrase: 'Gal 网站资料库', reason: 'nav label of the website directory' },
  { phrase: '网站资料库管理', reason: 'admin page of the website directory' },
  {
    phrase: '网站资料库页面',
    reason: 'admin hint about the website directory page'
  },
  {
    phrase: '本资料库仅会收录免费网站',
    reason: 'website directory description'
  },
  { phrase: '本资料库拥有', reason: 'website category page count' },
  {
    phrase: '资讯网站, Galgame 资料库, Telegram 社群',
    reason: 'kinds of sites the website directory collects'
  },
  { phrase: 'Galgame 资源资料库', reason: 'section name of /galgame' },
  { phrase: 'Galgame 标签资料库', reason: 'section name of /galgame/tag' },
  {
    phrase: 'Galgame 会社 / 厂商资料库',
    reason: 'section name of /galgame/official'
  },
  {
    phrase: 'Galgame 会社 / 制作商资料库',
    reason: 'SEO title of /galgame/official'
  },
  {
    phrase: 'Galgame 制作商资料库',
    reason: 'SEO description of /galgame/official'
  },
  {
    phrase: 'Galgame 会社资料库',
    reason: 'search page link to /galgame/official'
  },
  { phrase: 'Galgame 引擎资料库', reason: 'section name of /galgame/engine' },
  { phrase: '· 资源资料库 ·', reason: 'OG card slogan' },
  { phrase: "'Galgame 资料库',", reason: 'SEO keyword in the site config' }
]

type SourceFile = { path: string; shown: string }

const sourceFiles = (roots: string[]): SourceFile[] => {
  const files: SourceFile[] = []
  const walk = (dir: string, base: string) => {
    let entries
    try {
      entries = readdirSync(dir, { withFileTypes: true })
    } catch {
      return
    }
    for (const entry of entries) {
      const path = join(dir, entry.name)
      if (entry.isDirectory()) {
        walk(path, base)
      } else if (
        entry.isFile() &&
        SOURCE.test(entry.name) &&
        !SKIPPED.test(entry.name)
      ) {
        files.push({ path, shown: relative(base, path) })
      }
    }
  }
  for (const root of roots) {
    walk(root, dirname(root))
  }
  return files
}

export const catalogCopyViolations = (
  roots: string[],
  allowed: AllowedPhrase[]
): string[] => {
  const files = sourceFiles(roots)
  if (files.length === 0) {
    return [
      'scanned zero source files; the tree must contain application sources'
    ]
  }
  const phrases = allowed
    .map((entry) => entry.phrase)
    .sort((a, b) => b.length - a.length)
  const used = new Set<string>()
  const violations: string[] = []
  for (const { path, shown } of files) {
    const lines = readFileSync(path, 'utf8').split('\n')
    lines.forEach((line, index) => {
      let rest = line
      for (const phrase of phrases) {
        if (rest.includes(phrase)) {
          used.add(phrase)
          rest = rest.replaceAll(phrase, '\0')
        }
      }
      if (FORBIDDEN.test(rest)) {
        violations.push(`${shown}:${index + 1}: ${line.trim()}`)
      }
    })
  }
  for (const phrase of phrases) {
    if (!used.has(phrase)) {
      violations.push(`allowed phrase 「${phrase}」 matches nothing; remove it`)
    }
  }
  return violations
}

const webRoot = fileURLToPath(new URL('..', import.meta.url))

describe('catalog copy says 鲲 Galgame 论坛数据', () => {
  it('finds no 资料库 / NextMoe 目录 outside the allowed phrases', () => {
    expect(
      catalogCopyViolations(
        ['app', 'server', 'shared', 'i18n'].map((dir) => join(webRoot, dir)),
        ALLOWED_PHRASES
      )
    ).toEqual([])
  })

  it('catches a catalog 资料库 even on a line that also holds an allowed phrase', () => {
    const root = mkdtempSync(join(tmpdir(), 'catalog-copy-'))
    const app = join(root, 'app')
    const i18n = join(root, 'i18n')
    mkdirSync(app)
    mkdirSync(i18n)
    writeFileSync(
      join(app, 'a.vue'),
      [
        '<KunInfo description="资料库暂时不可用" />',
        '<KunHeader name="Galgame 网站资料库" />',
        '需要 NextMoe 资料库版主资格',
        '数据来自 NextMoe 目录',
        'Galgame 网站资料库 与 Galgame 资料库服务'
      ].join('\n')
    )
    writeFileSync(join(app, 'b.spec.ts'), "it('资料库')\n")
    writeFileSync(join(app, 'c.d.ts'), '// 资料库\n')
    writeFileSync(join(i18n, 'd.json'), '{ "x": "资料库中已有同名作品" }\n')

    expect(
      catalogCopyViolations(
        [app, i18n],
        [
          { phrase: 'Galgame 网站资料库', reason: 'control' },
          { phrase: 'Galgame 引擎资料库', reason: 'stale' }
        ]
      )
    ).toEqual([
      'app/a.vue:1: <KunInfo description="资料库暂时不可用" />',
      'app/a.vue:3: 需要 NextMoe 资料库版主资格',
      'app/a.vue:4: 数据来自 NextMoe 目录',
      'app/a.vue:5: Galgame 网站资料库 与 Galgame 资料库服务',
      'i18n/d.json:1: { "x": "资料库中已有同名作品" }',
      'allowed phrase 「Galgame 引擎资料库」 matches nothing; remove it'
    ])
  })

  it('fails when the tree has no files', () => {
    const root = mkdtempSync(join(tmpdir(), 'catalog-copy-empty-'))
    expect(catalogCopyViolations([root], [])).toEqual([
      'scanned zero source files; the tree must contain application sources'
    ])
  })
})
