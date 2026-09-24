const CODE_PATTERNS = [
  /提取码\s*[：:]\s*([a-zA-Z0-9]+)/g,
  /访问码\s*[：:]\s*([a-zA-Z0-9]+)/g,
  /pwd\s*[：:=]\s*([a-zA-Z0-9]+)/gi
]

const FALLBACK_CODE_PATTERN = /密码\s*[：:]\s*([a-zA-Z0-9]+)/g

const PASSWORD_PATTERNS = [
  /解压码\s*[：:]\s*([^\s,，]+)/g,
  /解压密码\s*[：:]\s*([^\s,，]+)/g,
  /unzip(?:\s*code|\s*password)?\s*[：:=]\s*([^\s,，]+)/gi
]

const URL_BODY = String.raw`[^\s<>"'，。！？、；：【】（）()\u4e00-\u9fff]+`

const LINK_RE = new RegExp(
  String.raw`(?:(?:https?|ftp|ftps|thunder|ed2k):\/\/${URL_BODY}|magnet:\??${URL_BODY})`,
  'gi'
)

const TRAILING_PUNCTUATION_RE = /[,.!?，。！？、；：」』"'`)\]】）>]+$/

export const RESOURCE_PROVIDER_KEYS = [
  'baidu',
  'aliyun',
  'quark',
  'pan123',
  'tianyiyun',
  'caiyun',
  'xunlei',
  'uc',
  'lanzou',
  'other'
] as const

export type ResourceProviderKey = (typeof RESOURCE_PROVIDER_KEYS)[number]

const PROVIDER_KEY_PATTERNS: readonly {
  key: Exclude<ResourceProviderKey, 'other'>
  patterns: readonly string[]
}[] = [
  {
    key: 'baidu',
    patterns: ['pan.baidu.com', 'tieba.baidu.com', 'pan.baidu.', 'baidu.com']
  },
  {
    key: 'aliyun',
    patterns: ['alipan.com', 'aliyun', 'aliyundrive', 'aliyuncs', 'aliyunpan']
  },
  { key: 'quark', patterns: ['pan.quark.cn', 'quark.cn', 'quark'] },
  {
    key: 'pan123',
    patterns: [
      '123pan',
      '123684',
      '123865',
      '123912',
      '123912.com',
      '123684.com',
      '123865.com',
      '123pan.cn',
      'vip.123pan'
    ]
  },
  { key: 'tianyiyun', patterns: ['cloud.189.cn', '189.cn', 'ecloud.189.cn'] },
  { key: 'caiyun', patterns: ['caiyun.139.com', 'yun.139.com', '139.com'] },
  { key: 'xunlei', patterns: ['pan.xunlei.com', 'xunlei.com'] },
  { key: 'uc', patterns: ['drive.uc.cn', 'uc.cn'] },
  {
    key: 'lanzou',
    patterns: [
      'lanzou.com',
      'lanzous.com',
      'lanzoux.com',
      'lanzoui.com',
      'lanzouw.com',
      'lanzouj.com',
      'lanzouu.com',
      'lanzouq.com'
    ]
  }
]

const PROVIDER_KEY_LABELS: Record<ResourceProviderKey, string> = {
  baidu: '百度网盘',
  aliyun: '阿里云盘',
  quark: '夸克网盘',
  pan123: '123盘',
  tianyiyun: '天翼云盘',
  caiyun: '和彩云',
  xunlei: '迅雷网盘',
  uc: 'UC网盘',
  lanzou: '蓝奏云',
  other: '其他 (自建网盘等不限速)'
}

export const detectProviderKeyFromURL = (url: string): ResourceProviderKey => {
  if (!url) return 'other'
  const s = url.toLowerCase()
  for (const entry of PROVIDER_KEY_PATTERNS) {
    for (const pattern of entry.patterns) {
      if (s.includes(pattern)) return entry.key
    }
  }
  return 'other'
}

export interface ParsedResourceLinks {
  links: string[]
  code: string
  password: string
  providers: string[]
}

export interface ApplyResourceLinkPasteInput {
  pasted: string
  existingLinks: string[]
  existingCode: string
  existingPassword: string
  replaceAll: boolean
}

export interface ApplyResourceLinkPasteResult {
  applied: boolean
  links: string[]
  code: string
  password: string
  providers: string[]
  notify: string
}

const unique = (values: string[]): string[] => {
  const seen = new Set<string>()
  const out: string[] = []
  for (const value of values) {
    if (!value || seen.has(value)) continue
    seen.add(value)
    out.push(value)
  }
  return out
}

const collectMatches = (input: string, patterns: RegExp[]): string[] => {
  const out: string[] = []
  for (const pattern of patterns) {
    const re = new RegExp(
      pattern.source,
      pattern.flags.includes('g') ? pattern.flags : `${pattern.flags}g`
    )
    for (const match of input.matchAll(re)) {
      if (match[1]) out.push(match[1])
    }
  }
  return unique(out)
}

const cleanMatchedUrl = (raw: string): string =>
  raw.replace(TRAILING_PUNCTUATION_RE, '')

const parseCodeFromSearchParams = (url: URL): string =>
  url.searchParams.get('pwd') ||
  url.searchParams.get('password') ||
  url.searchParams.get('code') ||
  ''

export const splitResourceLinkText = (raw: string): string[] =>
  raw
    .split(/[,，]/)
    .map((item) => item.trim())
    .filter(Boolean)

const extractUrls = (input: string): string[] => {
  const re = new RegExp(LINK_RE.source, 'gi')
  const links: string[] = []
  for (const match of input.matchAll(re)) {
    const url = cleanMatchedUrl(match[0] ?? '')
    if (url) links.push(url)
  }
  return unique(links)
}

export const parseResourceLinks = (input: string): ParsedResourceLinks => {
  const trimmed = input.trim()
  if (!trimmed) {
    return { links: [], code: '', password: '', providers: [] }
  }

  const links = extractUrls(trimmed)
  const queryCodes: string[] = []
  for (const link of links) {
    try {
      const code = parseCodeFromSearchParams(new URL(link))
      if (code) queryCodes.push(code)
    } catch {
      /* query codes need a parseable URL; the matched link is still kept */
    }
  }

  const textCodes = collectMatches(trimmed, CODE_PATTERNS)
  const codes = unique([...queryCodes, ...textCodes])
  if (!codes.length) {
    codes.push(...collectMatches(trimmed, [FALLBACK_CODE_PATTERN]))
  }

  const passwords = collectMatches(trimmed, PASSWORD_PATTERNS)

  const keys = links.map(detectProviderKeyFromURL)

  return {
    links,
    code: codes.join(', '),
    password: passwords.join(', '),
    providers: unique(keys.map((key) => PROVIDER_KEY_LABELS[key]))
  }
}

export const isCleanLinkDump = (
  raw: string,
  parsed: ParsedResourceLinks = parseResourceLinks(raw)
): boolean => {
  const parts = splitResourceLinkText(raw)
  if (!parts.length || parts.length !== parsed.links.length) return false
  return parts.every((part, index) => part === parsed.links[index])
}

const buildNotify = (opts: {
  filledCode: boolean
  filledPassword: boolean
  cleaned: boolean
  merged: boolean
  linkCount: number
  providers: string[]
}): string => {
  const bits: string[] = []
  if (opts.cleaned || opts.merged) {
    bits.push(opts.linkCount === 1 ? '链接' : `${opts.linkCount} 条链接`)
  }
  if (opts.filledCode) bits.push('提取码')
  if (opts.filledPassword) bits.push('解压码')
  if (!bits.length) return ''
  const pan = opts.providers.length ? `（${opts.providers.join('、')}）` : ''
  return `已自动识别：${bits.join('、')}${pan}`
}

export const applyResourceLinkPaste = (
  opts: ApplyResourceLinkPasteInput
): ApplyResourceLinkPasteResult => {
  const parsed = parseResourceLinks(opts.pasted)
  const filledCode = Boolean(parsed.code && !opts.existingCode)
  const filledPassword = Boolean(parsed.password && !opts.existingPassword)

  if (!parsed.links.length) {
    if (!filledCode && !filledPassword) {
      return {
        applied: false,
        links: opts.existingLinks,
        code: opts.existingCode,
        password: opts.existingPassword,
        providers: [],
        notify: ''
      }
    }
    return {
      applied: true,
      links: opts.existingLinks,
      code: filledCode ? parsed.code : opts.existingCode,
      password: filledPassword ? parsed.password : opts.existingPassword,
      providers: [],
      notify: buildNotify({
        filledCode,
        filledPassword,
        cleaned: false,
        merged: false,
        linkCount: 0,
        providers: []
      })
    }
  }

  const incoming = unique(parsed.links)
  const links = opts.replaceAll
    ? incoming
    : unique([...opts.existingLinks, ...incoming])
  const cleaned = !isCleanLinkDump(opts.pasted, parsed)
  const merged =
    !opts.replaceAll &&
    incoming.some((link) => !opts.existingLinks.includes(link))

  return {
    applied: true,
    links,
    code: filledCode ? parsed.code : opts.existingCode,
    password: filledPassword ? parsed.password : opts.existingPassword,
    providers: parsed.providers,
    notify: buildNotify({
      filledCode,
      filledPassword,
      cleaned,
      merged,
      linkCount: incoming.length,
      providers: parsed.providers
    })
  }
}

export const applyResourceLinkBlur = (
  raw: string,
  existingCode: string,
  existingPassword: string
): ApplyResourceLinkPasteResult => {
  const parsed = parseResourceLinks(raw)
  if (!parsed.links.length) {
    return {
      applied: false,
      links: splitResourceLinkText(raw),
      code: existingCode,
      password: existingPassword,
      providers: [],
      notify: ''
    }
  }

  const filledCode = Boolean(parsed.code && !existingCode)
  const filledPassword = Boolean(parsed.password && !existingPassword)
  const cleaned = !isCleanLinkDump(raw, parsed)
  const applied = cleaned || filledCode || filledPassword

  return {
    applied,
    links: parsed.links,
    code: filledCode ? parsed.code : existingCode,
    password: filledPassword ? parsed.password : existingPassword,
    providers: parsed.providers,
    notify: applied
      ? buildNotify({
          filledCode,
          filledPassword,
          cleaned,
          merged: false,
          linkCount: parsed.links.length,
          providers: parsed.providers
        })
      : ''
  }
}
