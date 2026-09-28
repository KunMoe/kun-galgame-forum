import type { ShopItemType, ShopSlot } from '#shared/utils/api/schemas'

export interface ShopShelfMeta {
  key: string
  title: string
  note: string
}

export const SHOP_SHELVES: ShopShelfMeta[] = [
  {
    key: 'codes',
    title: '福利兑换',
    note: '兑换码买到后立即发放，在「我的物品」里随时可以找到'
  },
  {
    key: 'perks',
    title: '功能解锁',
    note: '给你的 NextMoe 账号多一项能力，所有站点通用'
  },
  {
    key: 'cosmetics',
    title: '装扮',
    note: '换上之后，所有站点都会显示'
  }
]

export const SHOP_SITE_SHELF: ShopShelfMeta = {
  key: 'site',
  title: '本站专区',
  note: '只在鲲 Galgame 论坛出售，买到后所有站点都能用'
}

export const SHOP_TYPE_SHELF: Record<ShopItemType, string> = {
  redeem_code: 'codes',
  profile_about: 'perks',
  avatar_frame: 'cosmetics',
  profile_background: 'cosmetics'
}

export const SHOP_SLOTS: { slot: ShopSlot; label: string; empty: string }[] = [
  { slot: 'avatar_frame', label: '头像框', empty: '你还没有头像框' },
  { slot: 'profile_background', label: '主页背景', empty: '你还没有主页背景' }
]
