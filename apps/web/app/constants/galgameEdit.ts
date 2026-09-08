import { defineAsyncComponent } from 'vue'
import type {
  EditContextItem,
  EditFieldConfig,
  EditFieldConfigMap,
  EditObjectColumn,
  EditSelectOption
} from '@nextmoe/edit-ui-vue'
import {
  KUN_GALGAME_OFFICIAL_KIND_OPTIONS,
  KUN_GALGAME_OFFICIAL_KIND_DEVELOPER
} from '~/constants/galgameOfficial'
import { galgameImageSourceLabel } from '~/constants/galgameImageSource'

const K = (name: string) => `catalog.work.${name}`

const TitlesField = defineAsyncComponent(
  () => import('~/components/galgame/edit/TitlesField.vue')
)
const IntrosField = defineAsyncComponent(
  () => import('~/components/galgame/edit/IntrosField.vue')
)
const RosterField = defineAsyncComponent(
  () => import('~/components/galgame/edit/RosterField.vue')
)
const CreditsField = defineAsyncComponent(
  () => import('~/components/galgame/edit/CreditsField.vue')
)

export interface GalgameEditNames {
  tag?: Map<number, string>
  official?: Map<number, string>
  engine?: Map<number, string>
  series?: Map<number, string>
  character?: Map<number, string>
  staff?: Map<number, string>
  covers?: GalgameCover[]
  screenshots?: GalgameScreenshot[]
}

// Only the curated lane is editable: an edge imported from VNDB / Bangumi /
// DLsite is reconciled by its own importer and is not in this snapshot at all.
// Without this the form reads as "the tags vanished" next to a detail page
// showing eighty of them.
const UPSTREAM_NOTE =
  '以下内容来自 VNDB / Bangumi / DLsite 等数据源，由各数据源自行维护，无法在这里增删；上方只管理本站补充的部分。'

const CURATED_IMAGE_SOURCE = 'curated'

const missingFrom = (
  map: Map<number, string> | undefined,
  selected: unknown
): EditContextItem[] => {
  if (!map) {
    return []
  }
  const chosen = new Set(
    (Array.isArray(selected) ? selected : []).map((id) => Number(id))
  )
  return [...map]
    .filter(([id]) => !chosen.has(id))
    .map(([, name]) => ({ label: name }))
}

const missingLabelsFrom = (
  map: Map<number, string> | undefined,
  selected: unknown
): EditContextItem[] => {
  const ids = (Array.isArray(selected) ? selected : []).map((row) =>
    Number((row as { label_id?: number }).label_id)
  )
  return missingFrom(map, ids)
}

const upstreamImages = (
  images: (GalgameCover | GalgameScreenshot)[] | undefined
): EditContextItem[] =>
  (images ?? [])
    .filter((image) => image.source !== CURATED_IMAGE_SOURCE)
    .map((image) => ({
      label: galgameImageSourceLabel(image.source),
      image: image.cdn_url
        ? withImageVariant(image.cdn_url, 'mini')
        : galgameImageSrc(image)
    }))

const taxName = (name: unknown): string =>
  typeof name === 'string' ? name : ''

interface TaxonomyHit {
  id: number
  name: unknown
}

const searchTaxonomy =
  (path: string) =>
  async (keyword: string): Promise<EditSelectOption[]> => {
    const data = await kunFetch<TaxonomyHit[]>(path, {
      method: 'GET',
      query: { q: keyword }
    })
    return (data ?? []).map((o) => ({ value: o.id, label: taxName(o.name) }))
  }

const browseTaxonomy =
  (path: string) =>
  async (keyword: string): Promise<EditSelectOption[]> => {
    const data = await kunFetch<TaxonomyHit[]>(path, { method: 'GET' })
    const q = keyword.trim().toLowerCase()
    return (data ?? [])
      .map((o) => ({ value: o.id, label: taxName(o.name) }))
      .filter((o) => !q || o.label.toLowerCase().includes(q))
  }

const searchTags = searchTaxonomy('/galgame-tag/search')
const searchOfficials = searchTaxonomy('/galgame-official/search')
const searchStaff = searchTaxonomy('/galgame-staff/search')
const searchCharacters = searchTaxonomy('/galgame-character/search')
const searchEngines = browseTaxonomy('/galgame-engine')
const searchSeries = browseTaxonomy('/galgame-series')

const resolveFrom =
  (map?: Map<number, string>) =>
  (ids: (string | number)[]): EditSelectOption[] => {
    if (!map) {
      return []
    }
    return ids.flatMap((id) => {
      const name = map.get(Number(id))
      return name ? [{ value: Number(id), label: name }] : []
    })
  }

const idFormatItem =
  (map?: Map<number, string>) =>
  (item: unknown): string =>
    map?.get(Number(item)) ?? `#${String(item)}`

const labelFormatItem =
  (map?: Map<number, string>) =>
  (item: unknown): string => {
    const row = item as { label_id?: number; kind?: number }
    const name =
      (row.label_id !== undefined && map?.get(Number(row.label_id))) ||
      `#${row.label_id ?? '?'}`
    const kind = KUN_GALGAME_OFFICIAL_KIND_OPTIONS.find(
      (o) => o.value === row.kind
    )?.label
    return kind ? `${name} · ${kind}` : String(name)
  }

const GROUP_TITLES = '标题'
const GROUP_INTRO = '介绍'
const GROUP_BASIC = '基本信息'
const GROUP_RELATIONS = '关联条目'
const GROUP_CAST = '出演与署名'
const GROUP_EXTRAS = '链接'
const GROUP_IMAGES = '图片'

export const GALGAME_EDIT_GROUP_ORDER = [
  GROUP_TITLES,
  GROUP_INTRO,
  GROUP_BASIC,
  GROUP_RELATIONS,
  GROUP_CAST,
  GROUP_EXTRAS,
  GROUP_IMAGES
]

export const GALGAME_EDIT_TABBED_GROUPS: string[] = [
  GROUP_RELATIONS,
  GROUP_CAST,
  GROUP_IMAGES
]

export const GALGAME_EDIT_IDENTITY_HINT =
  'VNDB / Bangumi ID 属于条目身份, 不能直接修改。如果发现挂错了, 请在「链接」里补上正确的 VNDB / Bangumi 链接, 并在提交说明里写明原因, 由资料库管理员核对后修正。'

const TITLE_KIND_OPTIONS: EditSelectOption[] = [
  { value: 0, label: '官方名' },
  { value: 1, label: '别名' },
  { value: 2, label: '缩写' }
]

// catalog grades an image, not a work: `sexual` is set per image by the nightly
// machine pass and by whoever uploads it, and the wire carries base + the
// token's index in the vocabulary's published order (safe/suggestive/explicit,
// tame/violent/brutal), which is why these are ints and not tokens.
const SEXUAL_OPTIONS: EditSelectOption[] = [
  { value: 0, label: '无性内容' },
  { value: 1, label: '性暗示' },
  { value: 2, label: '露骨' }
]

const VIOLENCE_OPTIONS: EditSelectOption[] = [
  { value: 0, label: '无暴力' },
  { value: 1, label: '暴力' },
  { value: 2, label: '血腥' }
]

const GRADE_COLUMNS: EditObjectColumn[] = [
  {
    key: 'sexual',
    label: '性内容分级',
    control: 'select',
    type: 'integer',
    options: SEXUAL_OPTIONS
  },
  {
    key: 'violence',
    label: '暴力分级',
    control: 'select',
    type: 'integer',
    options: VIOLENCE_OPTIONS
  }
]

// image_hash is deliberately absent: it is the row's identity, and the item
// editor keeps every key the columns do not declare.
const COVER_ITEM_COLUMNS: EditObjectColumn[] = [
  {
    key: 'kind',
    label: '封面类型',
    control: 'input',
    type: 'string',
    placeholder: 'main / dig / pkgfront / pkgback …'
  },
  ...GRADE_COLUMNS
]

const SCREENSHOT_ITEM_COLUMNS: EditObjectColumn[] = [
  {
    key: 'caption',
    label: '说明',
    control: 'input',
    type: 'string',
    placeholder: '这张图的说明, 可留空'
  },
  ...GRADE_COLUMNS
]

const imageRow = (value: unknown): string => {
  if (value && typeof value === 'object') {
    const row = value as { image_hash?: string }
    if (row.image_hash) {
      return galgameImageSrc({ image_hash: row.image_hash })
    }
  }
  return ''
}

const hasImageHash = (items: unknown[], hash: string): boolean =>
  items.some((item) => (item as { image_hash?: string }).image_hash === hash)

const uploadCoverItem = async (
  file: File,
  current: unknown[]
): Promise<unknown | null> => {
  const res = await uploadGalgameImage(file, 'galgame_banner', file.name)
  if (!res) {
    return null
  }
  if (hasImageHash(current, res.hash)) {
    useMessage('已跳过重复图片', 'warn')
    return null
  }
  return { image_hash: res.hash }
}

const uploadScreenshotItem = async (
  file: File,
  current: unknown[]
): Promise<unknown | null> => {
  const res = await uploadGalgameImage(file, 'galgame_screenshot', file.name)
  if (!res) {
    return null
  }
  if (hasImageHash(current, res.hash)) {
    useMessage('已跳过重复图片', 'warn')
    return null
  }
  return { image_hash: res.hash }
}

// edit-ui 0.3.0's guardEditControl downgrades a scalar-list control to read-only
// when the value holds objects, which is right — but resolveControl does not know
// a field has a `component`, so an object list rendered by one of ours still
// resolves to `string-list` first and gets guarded. On 0.4.0 that put a
// 「结构不支持」 badge on 标题与别名 / 介绍 / 出演名单 / 本站署名 and passed
// disabled=true into the component: four whole tabs went read-only on upgrade.
// Naming the control the field actually has is the fix; the component still wins
// at render time, so nothing else changes.
const OBJECT_LIST: Pick<EditFieldConfig, 'control'> = { control: 'object-list' }

export const createGalgameEditConfig = (
  names: GalgameEditNames = {}
): EditFieldConfigMap => ({
  [K('titles')]: {
    label: '标题与别名',
    group: GROUP_TITLES,
    component: TitlesField,
    ...OBJECT_LIST,
    formatItem: (item) => {
      const row = item as { lang?: string; title?: string; kind?: number }
      const kind =
        TITLE_KIND_OPTIONS.find((o) => o.value === row.kind)?.label ?? ''
      return `${row.title ?? ''}${row.lang ? ` (${row.lang})` : ''}${kind ? ` · ${kind}` : ''}`
    },
    description: '至少要有一条官方名。别名可以留空语言。'
  },

  [K('display_name')]: {
    label: '条目名',
    group: GROUP_TITLES,
    description: '条目在跨站场合显示的单一名称, 通常与原文官方名一致'
  },

  [K('intros')]: {
    label: '介绍',
    group: GROUP_INTRO,
    component: IntrosField,
    ...OBJECT_LIST,
    formatItem: (item) => {
      const row = item as { lang?: string; intro?: string }
      return `${row.lang ?? ''}: ${(row.intro ?? '').slice(0, 40)}`
    },
    description: '每种语言一条，支持 Markdown。图片会被后端剥离。'
  },

  [K('olang')]: {
    label: '原始语言',
    group: GROUP_BASIC,
    options: [
      { value: 'ja', label: '日语' },
      { value: 'zh-Hans', label: '简体中文' },
      { value: 'zh-Hant', label: '繁体中文' },
      { value: 'en', label: '英语' }
    ]
  },
  [K('content_rating')]: {
    label: '年龄分级',
    group: GROUP_BASIC,
    options: [
      { value: 0, label: '全年龄' },
      { value: 1, label: '敏感' },
      { value: 2, label: 'R18' }
    ]
  },
  [K('display_nsfw')]: {
    label: '展示素材为 NSFW',
    group: GROUP_BASIC,
    control: 'switch',
    description: '封面 / 画廊 / 简介是否含成人内容, 与年龄分级是两条独立的轴'
  },

  [K('series_ids')]: {
    label: '所属系列',
    tabLabel: '系列',
    group: GROUP_RELATIONS,
    control: 'entity-picker',
    multiple: true,
    searchEntities: searchSeries,
    resolveEntities: resolveFrom(names.series),
    formatItem: idFormatItem(names.series),
    description: '搜索并选择所属系列。只有本站创建的系列可以在这里增删。',
    contextNote: UPSTREAM_NOTE,
    contextItems: (value) => missingFrom(names.series, value)
  },
  [K('tag_ids')]: {
    label: '标签',
    tabLabel: '标签',
    group: GROUP_RELATIONS,
    control: 'entity-picker',
    multiple: true,
    searchEntities: searchTags,
    resolveEntities: resolveFrom(names.tag),
    formatItem: idFormatItem(names.tag),
    description: '搜索标签名称添加',
    contextNote: UPSTREAM_NOTE,
    contextItems: (value) => missingFrom(names.tag, value)
  },
  [K('labels')]: {
    label: '会社',
    tabLabel: '会社',
    group: GROUP_RELATIONS,
    control: 'entity-kind-picker',
    entityIdKey: 'label_id',
    entityKinds: KUN_GALGAME_OFFICIAL_KIND_OPTIONS,
    entityDefaultKind: KUN_GALGAME_OFFICIAL_KIND_DEVELOPER,
    searchEntities: searchOfficials,
    resolveEntities: resolveFrom(names.official),
    formatItem: labelFormatItem(names.official),
    description: '搜索会社名称添加, 并选择它在本作中的身份 (开发商 / 发行商 …)',
    contextNote: UPSTREAM_NOTE,
    contextItems: (value) => missingLabelsFrom(names.official, value)
  },
  [K('engine_ids')]: {
    label: '引擎',
    tabLabel: '引擎',
    group: GROUP_RELATIONS,
    control: 'entity-picker',
    multiple: true,
    searchEntities: searchEngines,
    resolveEntities: resolveFrom(names.engine),
    formatItem: idFormatItem(names.engine),
    description: '搜索引擎名称添加',
    contextNote: UPSTREAM_NOTE,
    contextItems: (value) => missingFrom(names.engine, value)
  },

  [K('roster')]: {
    label: '出演名单',
    tabLabel: '角色',
    group: GROUP_CAST,
    component: RosterField,
    ...OBJECT_LIST,
    pairsSuppressed: true,
    fieldProps: { names: names.character },
    formatItem: (item) => {
      const row = item as { character_id?: number; kind?: number }
      return (
        names.character?.get(Number(row.character_id)) ??
        `#${row.character_id ?? '?'}`
      )
    },
    description:
      '只能改已有出演边的主配和剧透档，不能在这里新建或删除。隐藏会从读面拿掉这条边，行仍留在表里给上游导入器。'
  },
  [K('roster.suppressed')]: {
    label: '隐藏的角色'
  },
  [K('credits')]: {
    label: '本站署名',
    tabLabel: '署名',
    group: GROUP_CAST,
    component: CreditsField,
    ...OBJECT_LIST,
    fieldProps: {
      searchNames: searchStaff,
      searchCharacters,
      resolveNames: resolveFrom(names.staff),
      resolveCharacters: resolveFrom(names.character)
    },
    formatItem: (item) => {
      const row = item as { credit_name_id?: number; role_id?: number }
      const name =
        names.staff?.get(Number(row.credit_name_id)) ??
        `#${row.credit_name_id ?? '?'}`
      return name
    },
    description:
      '只管理本站补充的署名。上游导入的署名在详情页制作人员里，不能在这里改，只能整行隐藏。',
    contextNote: UPSTREAM_NOTE,
    contextItems: () =>
      [...(names.staff ?? [])].map(([, name]) => ({ label: name }))
  },
  [K('credits.suppressed')]: {
    label: '隐藏的上游署名',
    tabLabel: '隐藏署名',
    group: GROUP_CAST,
    control: 'string-list',
    placeholder: '粘贴 credit:职务:名义:角色 后回车',
    description:
      '上游署名的行身份键，回显即可、不要自己拼。隐藏后读面不再出现这条署名。'
  },
  [K('titles.suppressed')]: {
    label: '隐藏的标题',
    group: GROUP_TITLES,
    control: 'string-list',
    placeholder: '粘贴 title:类型:语言:标题 后回车',
    description:
      '被隐藏的上游标题仍留在表里，读面不再出现。键由 catalog 按标题内容派生，不要改格式。'
  },

  [K('links')]: {
    label: '相关链接',
    group: GROUP_EXTRAS,
    control: 'string-list',
    placeholder: '粘贴 http(s) 链接后回车添加',
    description: GALGAME_EDIT_IDENTITY_HINT
  },

  [K('covers')]: {
    label: '封面',
    tabLabel: '封面',
    group: GROUP_IMAGES,
    resolveImage: imageRow,
    formatItem: (item) => imageRow(item) || JSON.stringify(item),
    uploadImage: uploadCoverItem,
    itemColumns: COVER_ITEM_COLUMNS,
    pinItemFlag: { key: 'portrait_pinned', label: '竖版封面' },
    description:
      '“竖版封面”是卡片和列表渲染的那一张，由这里的置顶决定，与顺序无关；详情页头图由系统在横版图中自动挑选（近正方形的碟面、盒背等永不入选）。拖拽只调整画廊里的展示次序。',
    contextNote: UPSTREAM_NOTE,
    contextItems: () => upstreamImages(names.covers)
  },
  [K('screenshots')]: {
    label: '画廊',
    tabLabel: '画廊',
    group: GROUP_IMAGES,
    resolveImage: imageRow,
    formatItem: (item) => imageRow(item) || JSON.stringify(item),
    uploadImage: uploadScreenshotItem,
    itemColumns: SCREENSHOT_ITEM_COLUMNS,
    description: '拖拽可调整展示顺序, 点每张图的铅笔可以改它的说明与分级',
    contextNote: UPSTREAM_NOTE,
    contextItems: () => upstreamImages(names.screenshots)
  }
})

export const GALGAME_EDIT_FIELD_CONFIG: EditFieldConfigMap =
  createGalgameEditConfig()

export const galgameEditLabel = (key: string): string =>
  GALGAME_EDIT_FIELD_CONFIG[key]?.label ?? key.replace('catalog.work.', '')

export const galgameEditFieldConfig = (
  key: string
): EditFieldConfig | undefined => GALGAME_EDIT_FIELD_CONFIG[key]

export const GALGAME_EDIT_LEGACY_ACTION_LABELS: Record<string, string> = {
  created: '创建',
  updated: '更新',
  claimed: '认领',
  approved: '过审',
  declined: '拒绝',
  banned: '封禁',
  unbanned: '解封',
  reverted: '回滚',
  merged: '合并',
  edited_pending: '待审编辑',
  status_changed: '状态变更'
}
