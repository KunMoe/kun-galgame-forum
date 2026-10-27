import type { Article, BreadcrumbList, WithContext } from 'schema-dts'
import type { Walkthrough, Work } from '#shared/utils/api/schemas'
import { KUN_GALGAME_WALKTHROUGH_SOURCE } from '~/constants/galgameWalkthrough'

export const useGalgameWalkthroughSeo = (
  walkthrough: Walkthrough,
  work: Work
) => {
  const route = useRoute()
  const nameOf = useCatalogName()
  const workName = useWorkName()

  const name = workName(work)
  if (!work.is_published || work.is_nsfw || walkthrough.is_nsfw) {
    useKunDisableSeo(`${name} 攻略`)
    return
  }

  const { original } = nameOf(work)
  const alias = original && original !== name ? original : ''
  const characterNames = walkthrough.chart_characters.map((c) =>
    c.catalog_character
      ? nameOf(c.catalog_character.catalog_name).name
      : c.source_name
  )
  const source =
    KUN_GALGAME_WALKTHROUGH_SOURCE[walkthrough.source_site] ??
    walkthrough.source_site
  const developer = work.maker ? nameOf(work.maker).name : ''
  const pageUrl = `${kungal.domain.main}${route.path}`
  const workUrl = `${kungal.domain.main}/galgame/${work.id}`
  const image = work.banner?.url || work.cover?.url

  const shownNames = characterNames.slice(0, 4).join('、')
  const description = truncateRunes(
    `《${name}》${alias ? `(${alias})` : ''}攻略: ` +
      `${shownNames ? `按${shownNames}${characterNames.length > 4 ? ' 等' : ''} ${characterNames.length} 名角色整理路线, ` : ''}` +
      `共 ${walkthrough.ending_count} 个结局、${walkthrough.save_count} 个存档点, ` +
      `逐条标出每条路线要选的选项、存档时机与到达的结局, 并可用流程图查看分支。` +
      `${walkthrough.source_page_url ? `选项流程整理自${source}` : `由 ${source} 编辑者撰写`}${developer ? `, 游戏由 ${developer} 制作` : ''}。`,
    160
  )
  const title = `${name} 攻略 | 全 ${walkthrough.ending_count} 结局路线选项与存档点`

  const keywords = [
    `${name} 攻略`,
    `${name} 全结局`,
    `${name} 选项`,
    `${name} 存档`,
    ...(alias ? [`${alias} 攻略`] : []),
    ...characterNames.slice(0, 6).map((c) => `${c} 攻略`),
    'Galgame 攻略'
  ].join(',')

  const articleLd: WithContext<Article> = {
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: `${name} 攻略`,
    description,
    url: pageUrl,
    mainEntityOfPage: pageUrl,
    inLanguage: 'zh-Hans',
    ...(image ? { image } : {}),
    dateModified: walkthrough.updated_at,
    ...(walkthrough.source_page_url
      ? { isBasedOn: walkthrough.source_page_url }
      : {}),
    about: {
      '@type': 'VideoGame',
      name,
      ...(alias ? { alternateName: alias } : {}),
      url: workUrl,
      ...(work.release_date ? { datePublished: work.release_date } : {}),
      ...(developer
        ? { publisher: { '@type': 'Organization', name: developer } }
        : {})
    },
    mentions: characterNames.map((c) => ({
      '@type': 'Thing',
      name: c
    })),
    publisher: {
      '@type': 'Organization',
      name: kungal.titleShort,
      url: kungal.domain.main
    }
  }

  const breadcrumbLd: WithContext<BreadcrumbList> = {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: '首页',
        item: kungal.domain.main
      },
      {
        '@type': 'ListItem',
        position: 2,
        name: 'Galgame',
        item: `${kungal.domain.main}/galgame`
      },
      { '@type': 'ListItem', position: 3, name, item: workUrl },
      { '@type': 'ListItem', position: 4, name: '攻略', item: pageUrl }
    ]
  }

  useHead({
    script: [
      {
        id: 'schema-org-walkthrough',
        type: 'application/ld+json',
        innerHTML: articleLd
      },
      {
        id: 'schema-org-breadcrumb',
        type: 'application/ld+json',
        innerHTML: breadcrumbLd
      }
    ]
  })

  useKunSeoMeta({
    title,
    description,
    keywords,
    ogType: 'article',
    ...(image ? { ogImage: image } : {}),
    articleModifiedTime: walkthrough.updated_at
  })
}
