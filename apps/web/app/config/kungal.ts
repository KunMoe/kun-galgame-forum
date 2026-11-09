import type { KunSiteConfig } from './config'

const KUN_SITE_NAME = '鲲 Galgame'
const KUN_SITE_SHORT = '鲲 Galgame 论坛'
const KUN_SITE_MENTION = '@kungalgame'
const KUN_SITE_TITLE = '鲲 Galgame 论坛 🐳 开源 Galgame 网站'
const KUN_SITE_DESCRIPTION =
  '世界上最萌的 Galgame 论坛! 世界上最先进的 Galgame 资源资料库网站! 永远免费! Next Generation Galgame Encyclopedia Website!'
const KUN_SITE_URL = 'https://www.kungal.com'
const KUN_SITE_URL_BACKUP = 'https://www.kungal.org'
const KUN_SITE_NAV = 'https://nav.kungal.org'
const KUN_SITE_PATCH = 'https://www.moyu.moe'
const KUN_SITE_STICKER = 'https://sticker.kungal.com'
const KUN_SITE_OSS_DOMAIN = 'https://kun-galgame-forum.iloveren.link'
const KUN_SITE_DEVELOPMENT_DOCUMENTATION =
  'https://soft.moe/kun-visualnovel-docs/kun-forum.html'
const KUN_SITE_TELEGRAM_GROUP = 'https://telegram.me/kungalgame'
const KUN_SITE_GITHUB = 'https://github.com/KunMoe/kun-galgame-forum'
const KUN_SITE_AUTHOR_GITHUB = 'https://github.com/KunMoe'
const KUN_SITE_LIST = [
  { name: '鲲 Galgame 导航', url: KUN_SITE_NAV },
  { name: '鲲 Galgame 补丁', url: KUN_SITE_PATCH },
  { name: '鲲 Galgame 表情包', url: KUN_SITE_STICKER },
  { name: '鲲 Galgame 论坛 (备用)', url: KUN_SITE_URL_BACKUP },
  { name: '鲲 Galgame 开发文档', url: KUN_SITE_DEVELOPMENT_DOCUMENTATION }
]
const KUN_SITE_THEME_COLOR = '#006FEE'
const KUN_SITE_VALID_DOMAIN_LIST = [
  'www.kungal.com',
  'www.kungal.org',
  'localhost',
  '127.0.0.1'
]

const KUN_SITE_KEYWORDS = [
  'Galgame',
  'Galgame 论坛',
  'Galgame 下载',
  'Galgame 资源',
  'Galgame 资料库',
  'Galgame 网站',
  'Galgame 系列',
  'Galgame 会社',
  'Galgame 标签',
  'Galgame 引擎',
  'Galgame 评测',
  'Galgame 数据分析',
  'Galgame 新作动态',
  'Galgame 汉化 / 国际化',
  'Galgame 制作',
  'Galgame 讨论',
  'Galgame 制作',
  '技术交流',
  '其他交流'
]

export const kungal: KunSiteConfig = {
  name: KUN_SITE_NAME,
  title: KUN_SITE_TITLE,
  titleShort: KUN_SITE_SHORT,
  titleTemplate: `%s - ${KUN_SITE_TITLE}`,
  description: KUN_SITE_DESCRIPTION,
  keywords: KUN_SITE_KEYWORDS,
  canonical: KUN_SITE_URL,
  themeColor: KUN_SITE_THEME_COLOR,
  github: KUN_SITE_GITHUB,
  authorGitHub: KUN_SITE_AUTHOR_GITHUB,
  validDomain: KUN_SITE_VALID_DOMAIN_LIST,
  author: [
    { name: KUN_SITE_TITLE, url: KUN_SITE_URL },
    { name: 'GitHub', url: KUN_SITE_GITHUB },
    ...KUN_SITE_LIST
  ],
  creator: {
    name: KUN_SITE_SHORT,
    mention: KUN_SITE_MENTION,
    url: KUN_SITE_URL
  },
  publisher: {
    name: KUN_SITE_SHORT,
    mention: KUN_SITE_MENTION,
    url: KUN_SITE_URL
  },
  domain: {
    main: KUN_SITE_URL,
    imageBed: 'https://img.kungal.com',
    storage: 'https://oss.kungal.com',
    telegram_group: KUN_SITE_TELEGRAM_GROUP,
    patch: KUN_SITE_PATCH,
    backup: KUN_SITE_URL_BACKUP,
    sticker: KUN_SITE_STICKER,
    nav: KUN_SITE_NAV,
    doc: KUN_SITE_DEVELOPMENT_DOCUMENTATION,
    oss: KUN_SITE_OSS_DOMAIN
  },
  og: {
    title: KUN_SITE_TITLE,
    description: KUN_SITE_DESCRIPTION,
    image: '/kungalgame.webp',
    url: KUN_SITE_URL
  },
  ad: [
    {
      name: 'aify',
      banner: '/a/kungal1.webp',
      icon: '/a/icon1.webp'
    }
  ],
  images: [
    {
      url: '/kungalgame.webp',
      fullUrl: `${KUN_SITE_URL}/kungalgame.webp`,
      width: 1672,
      height: 941,
      alt: KUN_SITE_TITLE
    }
  ]
}
