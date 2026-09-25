import { nextmoe } from '~/config/nextmoe'

export interface KunLayoutItem {
  name: string
  label: string
  icon?: string
  router?: string
  hint?: string
  hintColor?: 'primary' | 'danger'
  external?: boolean
  isCollapse?: boolean
  children?: KunLayoutItem[]
}

export const kunLayoutItem: KunLayoutItem[] = [
  {
    name: 'create',
    icon: 'lucide:pencil',
    label: '发布主题',
    isCollapse: false,
    children: [
      {
        name: 'createTopic',
        icon: 'icon-park-outline:topic',
        router: '/edit/topic',
        label: '发布话题',
        hint: '大升级'
      },
      {
        name: 'createGalgame',
        icon: 'lucide:gamepad-2',
        router: '/edit/galgame/publish',
        label: '发布 Galgame'
      },
      {
        name: 'createToolset',
        icon: 'lucide:wrench',
        router: '/edit/toolset/create',
        label: '发布 Gal 工具'
      },
      {
        name: 'createQuiz',
        icon: 'lucide:library-big',
        router: '/edit/galgame/quiz',
        label: '发布 Galgame 习题'
      }
    ]
  },
  {
    name: 'galgame',
    icon: 'lucide:gamepad-2',
    label: 'Galgame',
    isCollapse: false,
    children: [
      {
        name: 'galgame',
        icon: 'lucide:box',
        router: '/galgame',
        label: 'Galgame'
      },
      {
        name: 'galgame-library',
        icon: 'lucide:library-big',
        router: '/gallib',
        label: 'Galgame 数据库',
        hint: '新',
        hintColor: 'danger'
      },
      {
        name: 'galgame-rating',
        icon: 'lucide:lollipop',
        router: '/galgame-rating',
        label: 'Galgame 评分',
        hint: '最强!'
      },
      {
        name: 'galgame-quiz',
        icon: 'lucide:brain',
        router: '/galgame-quiz',
        label: 'Galgame 题库',
        hint: '新',
        hintColor: 'danger'
      },
      {
        name: 'galgame-calendar',
        icon: 'lucide:calendar-days',
        router: '/galgame-calendar',
        label: 'Galgame 发售月历',
        hint: '新',
        hintColor: 'danger'
      },
      {
        name: 'news',
        icon: 'lucide:newspaper',
        router: '/news',
        label: 'Galgame 情报',
        hint: '新',
        hintColor: 'danger'
      },
      {
        name: 'website',
        icon: 'lucide:globe',
        router: '/website',
        label: 'Gal 网站资料库',
        hint: '起飞!'
      },
      {
        name: 'toolset',
        icon: 'lucide:wrench',
        router: '/toolset',
        label: 'Gal 工具资源',
        hint: '最实用!'
      },
      {
        name: 'galgame-resource',
        icon: 'lucide:download',
        router: '/galgame/resource',
        label: 'Gal 资源列表'
      },
      {
        name: 'galgame-official',
        icon: 'cuida:building-outline',
        router: '/galgame/official',
        label: 'Galgame 会社'
      },
      {
        name: 'galgame-tag',
        icon: 'lucide:tag',
        router: '/galgame/tag',
        label: 'Galgame 标签'
      },
      {
        name: 'galgame-character',
        icon: 'lucide:drama',
        router: '/galgame/character',
        label: 'Galgame 角色'
      },
      {
        name: 'galgame-engine',
        icon: 'carbon:ibm-engineering-lifecycle-mgmt',
        router: '/galgame/engine',
        label: 'Galgame 引擎'
      },
      {
        name: 'galgame-series',
        icon: 'lucide:layers',
        router: '/galgame/series',
        label: 'Galgame 系列'
      }
    ]
  },
  {
    name: 'topic',
    icon: 'icon-park-outline:topic',
    label: '话题',
    isCollapse: false,
    children: [
      {
        name: 'topic',
        icon: 'lucide:library-big',
        router: '/topic',
        label: '全部话题'
      },
      {
        name: 'galgame',
        icon: 'lucide:gamepad-2',
        router: '/category/galgame',
        label: 'Galgame 类'
      },
      {
        name: 'technique',
        icon: 'lucide:drafting-compass',
        router: '/category/technique',
        label: '技术交流 类'
      },
      {
        name: 'others',
        icon: 'lucide:circle-ellipsis',
        router: '/category/others',
        label: '其它 类'
      }
    ]
  },
  {
    name: 'activity',
    icon: 'lucide:activity',
    label: '网站动态',
    isCollapse: false,
    children: [
      {
        name: 'timeline',
        router: '/activity',
        label: '动态时间线'
      },
      {
        name: 'activity',
        router: '/activity/category',
        label: '分类动态'
      }
    ]
  },
  {
    name: 'ranking',
    icon: 'lucide:align-end-horizontal',
    label: '排行榜单',
    isCollapse: true,
    children: [
      {
        name: 'ranking',
        router: '/ranking/topic',
        label: '话题排行',
        hint: '优化'
      },
      {
        name: 'galgame',
        router: '/ranking/galgame',
        label: 'Galgame 排行'
      },
      {
        name: 'ranking',
        router: '/ranking/user',
        label: '用户排行',
        hint: '优化'
      }
    ]
  },
  {
    name: 'update',
    icon: 'lucide:arrow-big-up-dash',
    label: '更新日志',
    isCollapse: true,
    children: [
      {
        name: 'history',
        router: '/update/history',
        label: '更新历史'
      },
      {
        name: 'todo',
        router: '/update/todo',
        label: '待办列表'
      }
    ]
  },
  {
    name: 'doc',
    icon: 'lucide:info',
    router: '/doc',
    label: 'Galgame 帮助文档'
  },
  {
    name: 'friends',
    icon: 'lucide:handshake',
    router: '/friend-links',
    label: '友情链接'
  }
]

export interface KunSubSite {
  short: string
  name: string
  description: string
  link: string
  github?: string
  icon: string
  hint?: string
}

export const kunSubSites: KunSubSite[] = [
  {
    short: '补丁站',
    name: '鲲 Galgame 补丁站',
    description: '鲲 Galgame 补丁站，提供 Galgame 补丁、整合与相关资源下载。',
    link: 'https://www.moyu.moe/',
    icon: 'lucide:puzzle',
    hint: '震憾上线'
  },
  {
    short: '账号中心',
    name: nextmoe.account,
    description: `${nextmoe.account}，统一 NextMoe 站点集群的所有用户账户，一个账号即可登录旗下的所有网站！`,
    link: 'https://account.nextmoe.com/',
    icon: 'lucide:key-round'
  },
  {
    short: '表情包',
    name: '鲲 Galgame 表情包',
    description:
      'Galgame 表情包网站，鲲 Galgame 表情包下载站，海量 Visual Novel 表情包。',
    link: 'https://sticker.kungal.com/',
    github: 'https://github.com/KUN1007/kun-galgame-stickers-sveltekit',
    icon: 'lucide:image',
    hint: '全网最全'
  },
  {
    short: '导航站',
    name: '鲲 Galgame 导航',
    description: '完全开源的导航站，可以前往鲲 Galgame 旗下的所有子网站！',
    link: 'https://nav.kungal.org/',
    github: 'https://github.com/KUN1007/kun-galgame-nav-solid',
    icon: 'lucide:navigation',
    hint: '防失联'
  },
  {
    short: '开发文档',
    name: '鲲 Galgame 论坛开发文档',
    description: '鲲 Galgame 论坛是完全开源免费的！ 开发文档完全公开！',
    link: 'https://docs-kungal.nextmoe.dev/',
    github: 'https://github.com/KUN1007/soft.moe',
    icon: 'lucide:code-xml'
  }
]

export interface KunRailLink {
  label: string
  router: string
  icon?: string
  hint?: string
  external?: boolean
}

export interface KunRailSection {
  label?: string
  items: KunRailLink[]
}

export interface KunRailGroup {
  name: string
  label: string
  icon: string
  router?: string
  sections: KunRailSection[]
}

export const kunSidebarRail: KunRailGroup[] = [
  {
    name: 'create',
    label: '发布',
    icon: 'lucide:pencil',
    router: '/edit/topic',
    sections: [
      {
        items: [
          {
            label: '发布话题',
            router: '/edit/topic',
            icon: 'icon-park-outline:topic'
          },
          {
            label: '发布 Gal 工具',
            router: '/edit/toolset/create',
            icon: 'lucide:wrench'
          },
          {
            label: '发布 Galgame',
            router: '/edit/galgame/publish',
            icon: 'lucide:gamepad-2'
          },
          {
            label: '发布 Galgame 习题',
            router: '/edit/galgame/quiz',
            icon: 'lucide:library-big'
          }
        ]
      }
    ]
  },
  {
    name: 'topic',
    label: '话题',
    icon: 'icon-park-outline:topic',
    router: '/topic',
    sections: [
      {
        items: [
          { label: '全部话题', router: '/topic', icon: 'lucide:library-big' },
          {
            label: 'Galgame 类',
            router: '/category/galgame',
            icon: 'lucide:gamepad-2'
          },
          {
            label: '技术交流 类',
            router: '/category/technique',
            icon: 'lucide:drafting-compass'
          },
          {
            label: '其它 类',
            router: '/category/others',
            icon: 'lucide:circle-ellipsis'
          }
        ]
      }
    ]
  },
  {
    name: 'galgame',
    label: 'Galgame',
    icon: 'lucide:gamepad-2',
    router: '/galgame',
    sections: [
      {
        items: [
          { label: 'Galgame', router: '/galgame', icon: 'lucide:box' },
          {
            label: 'Galgame 数据库',
            router: '/gallib',
            icon: 'lucide:library-big'
          },
          {
            label: 'Galgame 评分',
            router: '/galgame-rating',
            icon: 'lucide:lollipop'
          },
          {
            label: 'Galgame 题库',
            router: '/galgame-quiz',
            icon: 'lucide:brain'
          },
          {
            label: 'Galgame 发售月历',
            router: '/galgame-calendar',
            icon: 'lucide:calendar-days'
          },
          {
            label: 'Galgame 情报',
            router: '/news',
            icon: 'lucide:newspaper'
          },
          { label: 'Gal 网站资料库', router: '/website', icon: 'lucide:globe' },
          { label: 'Gal 工具资源', router: '/toolset', icon: 'lucide:wrench' },
          {
            label: 'Gal 资源列表',
            router: '/galgame/resource',
            icon: 'lucide:download'
          },
          {
            label: 'Galgame 会社',
            router: '/galgame/official',
            icon: 'cuida:building-outline'
          },
          { label: 'Galgame 标签', router: '/galgame/tag', icon: 'lucide:tag' },
          {
            label: 'Galgame 角色',
            router: '/galgame/character',
            icon: 'lucide:drama'
          },
          {
            label: 'Galgame 引擎',
            router: '/galgame/engine',
            icon: 'carbon:ibm-engineering-lifecycle-mgmt'
          },
          {
            label: 'Galgame 系列',
            router: '/galgame/series',
            icon: 'lucide:layers'
          }
        ]
      }
    ]
  },
  {
    name: 'others',
    label: '其他',
    icon: 'lucide:layout-grid',
    router: '/doc',
    sections: [
      {
        label: '排行榜',
        items: [
          {
            label: '话题排行',
            router: '/ranking/topic',
            icon: 'lucide:list-ordered'
          },
          {
            label: 'Galgame 排行',
            router: '/ranking/galgame',
            icon: 'lucide:gamepad-2'
          },
          { label: '用户排行', router: '/ranking/user', icon: 'lucide:users' }
        ]
      },
      {
        label: '更新日志',
        items: [
          {
            label: '更新历史',
            router: '/update/history',
            icon: 'lucide:history'
          },
          {
            label: '待办列表',
            router: '/update/todo',
            icon: 'lucide:list-checks'
          }
        ]
      },
      {
        label: '网站动态',
        items: [
          { label: '动态时间线', router: '/activity', icon: 'lucide:activity' },
          {
            label: '分类动态',
            router: '/activity/category',
            icon: 'lucide:layers'
          }
        ]
      },
      {
        items: [
          { label: 'Galgame 帮助文档', router: '/doc', icon: 'lucide:info' },
          {
            label: '友情链接',
            router: '/friend-links',
            icon: 'lucide:handshake'
          },
          { label: '子网站', router: '/sites', icon: 'lucide:app-window' }
        ]
      }
    ]
  }
]
