import type {
  ShopInventory,
  ShopOffer,
  ShopOwnedItem
} from '#shared/utils/api/schemas'

export interface OfferAction {
  label: string
  disabled: boolean
}

export const isOfferOwned = (offer: ShopOffer, items: ShopOwnedItem[]) =>
  offer.rewards.some(
    (r) =>
      r.item.item_type !== 'redeem_code' &&
      items.some((o) => o.item.id === r.item.id && o.is_active && !o.expires_at)
  )

export const purchasedCount = (
  offer: ShopOffer,
  inventory: ShopInventory | null
) =>
  inventory?.limit_usage.find((u) => u.offer_id === offer.id)
    ?.purchased_count ?? 0

export const offerAction = (
  offer: ShopOffer,
  viewer: { signedIn: boolean; owned: boolean; used: number; balance: number }
): OfferAction => {
  if (viewer.owned) {
    return { label: '已拥有', disabled: true }
  }
  if (offer.remaining_count === 0) {
    return { label: '已售罄', disabled: true }
  }
  const limit = offer.purchase_limit
  if (limit && viewer.used >= limit.quantity) {
    return {
      label: limit.period === 'month' ? '本月已买满' : '已达限购',
      disabled: true
    }
  }
  if (!viewer.signedIn) {
    return { label: '登录后购买', disabled: false }
  }
  const short = offer.price - viewer.balance
  if (short > 0) {
    return { label: `还差 ${short} 点`, disabled: true }
  }
  return { label: '购买', disabled: false }
}
