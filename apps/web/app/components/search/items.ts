export interface SearchCategory {
  value: SearchType
  textValue: string
  icon: string
  /** Measure word + noun, as it reads after 共 N. */
  countUnit: string
}

export const SEARCH_CATEGORIES: SearchCategory[] = [
  {
    value: 'all',
    textValue: '全部',
    icon: 'lucide:layout-grid',
    countUnit: '个结果'
  },
  {
    value: 'galgame',
    textValue: 'Galgame',
    icon: 'lucide:gamepad-2',
    countUnit: '个 Galgame'
  },
  {
    value: 'topic',
    textValue: '话题',
    icon: 'lucide:message-square-text',
    countUnit: '个话题'
  },
  {
    value: 'entity',
    textValue: '论坛数据',
    icon: 'lucide:library',
    countUnit: '个条目'
  },
  {
    value: 'resource',
    textValue: 'Galgame 资源',
    icon: 'lucide:package',
    countUnit: '个资源'
  },
  {
    value: 'user',
    textValue: '用户',
    icon: 'lucide:user-round',
    countUnit: '位用户'
  },
  {
    value: 'reply',
    textValue: '回复',
    icon: 'carbon:reply',
    countUnit: '条回复'
  },
  {
    value: 'comment',
    textValue: '评论',
    icon: 'uil:comment-dots',
    countUnit: '条评论'
  },
  {
    value: 'galcomment',
    textValue: '游戏评论',
    icon: 'lucide:message-circle',
    countUnit: '条评论'
  },
  {
    value: 'toolset',
    textValue: 'Gal 工具',
    icon: 'lucide:wrench',
    countUnit: '个工具'
  }
]

export const SEARCH_CATEGORY_MAP = Object.fromEntries(
  SEARCH_CATEGORIES.map((category) => [category.value, category])
) as Record<SearchType, SearchCategory>

export interface SearchEntityFamilyMeta {
  value: SearchEntityFamily
  textValue: string
  icon: string
  path: string
  /** Reads after N; a company and a tag count works, a character does not. */
  countUnit?: string
}

export const SEARCH_ENTITY_FAMILIES: SearchEntityFamilyMeta[] = [
  {
    value: 'character',
    textValue: '角色',
    icon: 'lucide:drama',
    path: '/galgame/character'
  },
  {
    value: 'company',
    textValue: '会社',
    icon: 'lucide:building-2',
    path: '/galgame/official',
    countUnit: '部作品'
  },
  {
    value: 'staff',
    textValue: 'Staff',
    icon: 'lucide:signature',
    path: '/galgame/staff'
  },
  {
    value: 'tag',
    textValue: '标签',
    icon: 'lucide:tag',
    path: '/galgame/tag',
    countUnit: '部作品'
  },
  {
    value: 'series',
    textValue: '系列',
    icon: 'lucide:layers',
    path: '/galgame/series',
    countUnit: '部作品'
  },
  {
    value: 'engine',
    textValue: '引擎',
    icon: 'carbon:ibm-engineering-lifecycle-mgmt',
    path: '/galgame/engine',
    countUnit: '部作品'
  }
]

export const SEARCH_ENTITY_FAMILY_MAP = Object.fromEntries(
  SEARCH_ENTITY_FAMILIES.map((family) => [family.value, family])
) as Record<SearchEntityFamily, SearchEntityFamilyMeta>

export const searchEntityPath = (item: SearchEntityItem) =>
  `${SEARCH_ENTITY_FAMILY_MAP[item.family].path}/${item.id}`

// Catalog answers 400 past ten tag ids.
export const TAG_FILTER_MAX = 10

// The works index's own sort vocabulary. 评分 is not in it — the index carries
// no rating attribute, which is why the filter bar says so instead of offering
// one that would be ignored.
export const SEARCH_GALGAME_SORTS: FilterOption[] = [
  { value: 'relevance', label: '相关度' },
  { value: 'popularity', label: '热度' },
  { value: 'released_desc', label: '最新发售' },
  { value: 'released_asc', label: '最早发售' },
  { value: 'updated', label: '资料更新' }
]
