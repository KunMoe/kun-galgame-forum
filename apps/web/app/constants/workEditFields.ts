const WORK_EDIT_FIELD_LABELS: Record<string, string> = {
  display_name: '条目名',
  titles: '标题与别名',
  intros: '介绍',
  olang: '原始语言',
  content_rating: '年龄分级',
  display_nsfw: '展示素材为 NSFW',
  tag_ids: '标签',
  tag_spoilers: '标签剧透等级',
  labels: '会社',
  platforms: '平台',
  engine_ids: '引擎',
  series_ids: '所属系列',
  series_members: '系列内位置',
  relations: '关联作品',
  links: '相关链接',
  covers: '封面',
  covers_upstream: '资料源封面',
  cover_order: '封面顺序',
  screenshots: '画廊',
  screenshots_upstream: '资料源截图',
  screenshot_order: '截图顺序',
  credits: '制作人员',
  roster: '出演名单'
}

const SUPPRESSED_SUFFIX = '.suppressed'

export const workEditFieldLabel = (key: string): string => {
  const bare = key.replace(/^catalog\.work\./, '')
  if (bare.endsWith(SUPPRESSED_SUFFIX)) {
    const parent = bare.slice(0, -SUPPRESSED_SUFFIX.length)
    return `隐藏的${WORK_EDIT_FIELD_LABELS[parent] ?? parent}`
  }
  return WORK_EDIT_FIELD_LABELS[bare] ?? bare
}
