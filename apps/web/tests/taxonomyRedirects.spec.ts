import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, it, expect } from 'vitest'
import {
  parseWikiId,
  resolveLegacyTag,
  resolveLegacyEngine,
  resolveRenamedTaxonomyPath
} from '../server/utils/kunTaxonomyRedirects'
import {
  TAXONOMY_FAMILIES,
  taxonomyIndexPath,
  taxonomyDetailPath
} from '../shared/utils/kunTaxonomyPaths'
import type { TaxonomyFamily } from '../shared/utils/kunTaxonomyPaths'
import tagRedirects from '../server/data/wiki-tag-redirects.json'
import engineRedirects from '../server/data/wiki-engine-redirects.json'
import tagGone from '../server/data/wiki-tag-gone.json'

const FINAL_DETAIL = /^\/galgame\/(tag|official|engine)\/\d+$/
const isFinalForm = (path: string) =>
  FINAL_DETAIL.test(path) &&
  !path.includes('/galgame-') &&
  !path.includes('/c/')

describe('taxonomy path builders', () => {
  it('builds the current index path per family', () => {
    expect(taxonomyIndexPath('tag')).toBe('/galgame/tag')
    expect(taxonomyIndexPath('official')).toBe('/galgame/official')
    expect(taxonomyIndexPath('engine')).toBe('/galgame/engine')
  })

  it('builds a bare catalog-id detail path with no discriminator segment', () => {
    expect(taxonomyDetailPath('tag', 55)).toBe('/galgame/tag/55')
    expect(taxonomyDetailPath('official', 9)).toBe('/galgame/official/9')
    expect(taxonomyDetailPath('engine', 138)).toBe('/galgame/engine/138')
  })
})

describe('gen-2 `/galgame-{family}` → current, one hop', () => {
  it.each([...TAXONOMY_FAMILIES])('redirects the %s index', (family) => {
    expect(resolveRenamedTaxonomyPath(`/galgame-${family}`)).toBe(
      `/galgame/${family}`
    )
  })

  it.each([...TAXONOMY_FAMILIES])(
    'drops the /c/ segment on %s detail, keeping the id',
    (family) => {
      expect(resolveRenamedTaxonomyPath(`/galgame-${family}/c/1234`)).toBe(
        `/galgame/${family}/1234`
      )
    }
  )

  it('leaves live and unrelated paths alone', () => {
    expect(resolveRenamedTaxonomyPath('/galgame/tag/55')).toBeNull()
    expect(resolveRenamedTaxonomyPath('/galgame/tag')).toBeNull()
    expect(resolveRenamedTaxonomyPath('/galgame-resource')).toBeNull()
    expect(resolveRenamedTaxonomyPath('/galgame-rating/12')).toBeNull()
    expect(resolveRenamedTaxonomyPath('/galgame-tag/5')).toBeNull()
    expect(resolveRenamedTaxonomyPath('/galgame-tag/c/abc')).toBeNull()
    expect(resolveRenamedTaxonomyPath('/galgame-tag/c/5/extra')).toBeNull()
    expect(resolveRenamedTaxonomyPath('/galgame-series/c/5')).toBeNull()
  })
})

describe('gen-1 wiki-id → current, one hop (no 301 chain)', () => {
  it('resolves a mapped tag straight to the final form', () => {
    expect(resolveLegacyTag(1)).toEqual({
      kind: 'redirect',
      to: '/galgame/tag/55'
    })
  })

  it('resolves a non-identity engine straight to the final form', () => {
    expect(resolveLegacyEngine(140)).toEqual({
      kind: 'redirect',
      to: '/galgame/engine/138'
    })
  })

  it('never emits an intermediate form for ANY mapped tag', () => {
    const bad = Object.keys(tagRedirects as Record<string, number>)
      .map((wikiId) => resolveLegacyTag(Number(wikiId)))
      .filter((r) => r.kind !== 'redirect' || !isFinalForm(r.to))
    expect(bad).toEqual([])
  })

  it('never emits an intermediate form for ANY mapped engine', () => {
    const bad = Object.keys(engineRedirects as Record<string, number>)
      .map((wikiId) => resolveLegacyEngine(Number(wikiId)))
      .filter((r) => r.kind !== 'redirect' || !isFinalForm(r.to))
    expect(bad).toEqual([])
  })

  it('410s a retired tag and 404s an unknown one', () => {
    const gone = (tagGone as number[])[0]!
    expect(resolveLegacyTag(gone)).toEqual({ kind: 'gone' })
    expect(resolveLegacyTag(99_999_999)).toEqual({ kind: 'missing' })
    expect(resolveLegacyEngine(99_999_999)).toEqual({ kind: 'missing' })
  })
})

describe('parseWikiId', () => {
  it('accepts positive integers only', () => {
    expect(parseWikiId('5')).toBe(5)
    expect(parseWikiId('0')).toBeNull()
    expect(parseWikiId('-3')).toBeNull()
    expect(parseWikiId('1.5')).toBeNull()
    expect(parseWikiId('abc')).toBeNull()
    expect(parseWikiId(undefined)).toBeNull()
  })
})

describe('merged catalog id hop', () => {
  it.each([...TAXONOMY_FAMILIES])(
    'builds a final-form %s target from a survivor id',
    (family) => {
      const to = taxonomyDetailPath(family, 6935)
      expect(isFinalForm(to)).toBe(true)
      expect(to).toBe(`/galgame/${family}/6935`)
    }
  )

  it('lands on a URL that no other rule would redirect again', () => {
    const to = taxonomyDetailPath('official', 6935)
    expect(resolveRenamedTaxonomyPath(to)).toBeNull()
  })

  it('keeps a 会社 reader on the games sub-page across the hop', () => {
    const to = `${taxonomyDetailPath('official', 6935)}/game`
    expect(to).toBe('/galgame/official/6935/game')
    expect(resolveRenamedTaxonomyPath(to)).toBeNull()
  })
})

describe('merged-id pages stop rendering after the hop', () => {
  const read = (path: string) =>
    readFileSync(resolve(process.cwd(), path), 'utf-8')

  const DETAIL_PAGES: Record<TaxonomyFamily, string[]> = {
    tag: ['app/pages/galgame/tag/[id].vue'],
    engine: ['app/pages/galgame/engine/[id].vue'],
    official: [
      'app/pages/galgame/official/[id]/index.vue',
      'app/pages/galgame/official/[id]/game.vue'
    ]
  }

  const HOP_COMPOSABLE = 'app/composables/useGalgameOfficialDetail.ts'
  const hops = (source: string) =>
    source.includes('navigateTo(') ||
    source.includes('useGalgameOfficialDetail(')

  it.each(
    (Object.keys(DETAIL_PAGES) as TaxonomyFamily[]).flatMap((family) =>
      DETAIL_PAGES[family].map((path) => [family, path] as const)
    )
  )('%s (%s): a hop in setup comes with a template gate', (_family, path) => {
    const source = read(path)
    if (!hops(source)) {
      expect(source).not.toContain('moved_to')
      return
    }
    // A merged id leaves data null after the hop, so the root gate on data
    // is what keeps the page from rendering a half-empty body under the 301.
    const root = source.match(/<template>\s*\n\s*<div ([^>]*)>/)
    expect(root?.[1]).toContain('v-if="data"')
  })

  it('the shared 会社 hop is the one that parks the 301', () => {
    const source = read(HOP_COMPOSABLE)
    expect(source).toContain('navigateTo(')
    expect(source).toContain('taxonomyDetailPath(')
    expect(source).toContain('redirectCode: 301')
  })

  it.each(DETAIL_PAGES.official)(
    '%s: the SEO block is inside the not-moved gate',
    (path) => {
      const source = read(path)
      const gate = source.indexOf('if (official)')
      expect(gate).toBeGreaterThan(-1)
      expect(source.indexOf('useKunSeoMeta(')).toBeGreaterThan(gate)
    }
  )
})
