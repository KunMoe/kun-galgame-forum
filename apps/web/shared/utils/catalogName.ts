import type { components } from '../types/api/v1'

export type CatalogName = Pick<
  components['schemas']['WorkSummary'],
  'display_name' | 'latin' | 'localized'
>

const zhLocales = ['zh-Hans', 'zh', 'zh-Hant']

const localizedZh = (n: CatalogName): string => {
  for (const locale of zhLocales) {
    const value = n.localized[locale]?.value
    if (value) {
      return value
    }
  }
  return ''
}

// The rule the server used before v1 sent the whole primitive: Chinese first,
// then the entity's own name, then its romanization. A reader who asked for
// original names flips the head of that chain.
export const catalogEntityName = (
  n: CatalogName,
  preferOriginal = false
): { name: string; original: string } => {
  const zh = localizedZh(n)
  const latin = n.latin ?? ''
  const name = preferOriginal
    ? n.display_name || zh || latin
    : zh || n.display_name || latin
  const other = preferOriginal ? zh : n.display_name
  return { name, original: other && other !== name ? other : '' }
}

// Tags and traits were imported under English vocabulary tokens, so the
// original-name preference would turn 金发 into Blonde; they never follow it.
export const catalogVocabularyName = (n: CatalogName): string =>
  localizedZh(n) || n.display_name

type Intro = { locale: string; value: string; is_machine: boolean }

const introLocales = ['zh-Hans', 'zh', 'zh-Hant', 'ja', 'en']

// One intro for a page that shows one: the first language this site reads.
export const pickCatalogIntro = <T extends Intro>(intros: T[]): T | undefined =>
  introLocales
    .map((locale) => intros.find((i) => i.locale === locale))
    .find(Boolean) ?? intros[0]
