export interface KunFooterLink {
  label: string
  link: string
}

export const KUN_FOOTER_COLUMNS: KunFooterLink[][] = [
  [
    { label: '鲲 Galgame 补丁', link: 'https://www.moyu.moe/' },
    { label: '鲲 Galgame 表情包', link: 'https://sticker.kungal.com/' },
    { label: '鲲 Galgame 导航', link: 'https://nav.kungal.org/' }
  ],
  [
    { label: 'NextMoe 账号', link: 'https://account.nextmoe.com/' },
    { label: '开发者平台', link: 'https://developer.nextmoe.dev/' },
    { label: '编辑者中心', link: 'https://editor.nextmoe.com/' }
  ],
  [
    { label: '鲲 UI', link: 'https://ui.kungal.com/' },
    { label: 'NextMoe·未萌', link: 'https://www.nextmoe.com/' },
    { label: 'LetMoe·一起萌', link: 'https://www.letmoe.com/' }
  ]
]

export const KUN_FOOTER_ORG = {
  label: '鲲 Galgame',
  link: 'https://github.com/kungal'
}
