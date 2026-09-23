export const KUN_FIELD_LABELS: Record<string, string> = {
  overall: '总分',
  short_summary: '简评',
  galgameType: 'Galgame 类型',
  play_status: '游玩状态',
  spoiler_level: '剧透等级',
  recommend: '推荐度',
  art: '美术',
  story: '剧情',
  music: '音乐',
  character: '角色',
  route: '线路',
  system: '系统',
  voice: '配音',
  replay_value: '可重复游玩性',

  name_en_us: '英文名称',
  name_ja_jp: '日文名称',
  name_zh_cn: '简体中文名称',
  name_zh_tw: '繁体中文名称',
  intro_en_us: '英文简介',
  intro_ja_jp: '日文简介',
  intro_zh_cn: '简体中文简介',
  intro_zh_tw: '繁体中文简介',
  aliases: '别名',
  vndb_id: 'VNDB ID',
  release_date: '发售日期',
  original_language: '原始语言',
  age_limit: '年龄分级',
  content_limit: '内容分级',
  contentLimit: '内容分级',
  banner: '预览图',
  introduction: '简介',

  name: '名称',
  title: '标题',
  content: '内容',
  content_markdown: '内容',
  text: '内容',
  sections: '分区',
  description: '描述',
  url: '链接',
  link: '链接',
  email: '邮箱',
  password: '密码',
  tags: '标签',
  targets: '标签',
  category: '分类',
  section: '分区',
  access_scope: '访问范围',
  access_roles: '可见角色',
  access_user_ids: '可见用户',
  size: '大小',
  size_label: '大小',
  code: '验证码',
  link_url: '链接',
  extraction_code: '提取码',
  archive_password: '解压码',
  artifact_id: '上传文件',
  filename: '文件名',
  file_size: '文件大小',
  toolset_type: '工具类型',
  interface_language: '语言',
  release_channel: '版本',
  homepage_urls: '主页',
  rating: '评分',
  note: '备注',
  prompt_text: '题干',
  description_markdown: '描述',
  explanation_markdown: '解析',
  quiz_type: '题型',
  quiz_category: '分类',
  choices: '选项',
  correct_choice_indexes: '正确答案',
  is_statement_true: '判断答案',
  work_ids: '关联作品',
  is_work_hidden: '隐藏关联作品',
  difficulty: '难度'
}

interface KunZodIssue {
  path?: (string | number)[]
  message: string
}

const labelForPath = (path?: (string | number)[]): string => {
  const key = path?.find((p) => typeof p === 'string') as string | undefined
  if (!key) return ''
  return KUN_FIELD_LABELS[key] ?? key
}

export const formatKunZodIssue = (issue: KunZodIssue): string => {
  const label = labelForPath(issue.path)
  return label ? `${label}：${issue.message}` : issue.message
}
