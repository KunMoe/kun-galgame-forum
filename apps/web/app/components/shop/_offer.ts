import type {
  ShopInventory,
  ShopOffer,
  ShopOwnedItem
} from '#shared/utils/api/schemas'

export interface OfferAction {
  label: string
  disabled: boolean
  intent: 'buy' | 'subscribe' | 'unsubscribe'
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

export const isRestockSubscribed = (
  offer: ShopOffer,
  inventory: ShopInventory | null
) => inventory?.restock_alert_offer_ids.includes(offer.id) ?? false

export const offerAction = (
  offer: ShopOffer,
  viewer: {
    signedIn: boolean
    owned: boolean
    used: number
    balance: number
    subscribed: boolean
  }
): OfferAction => {
  if (viewer.owned) {
    return { label: '已拥有', disabled: true, intent: 'buy' }
  }
  if (offer.remaining_count === 0) {
    return viewer.subscribed
      ? { label: '已开启到货提醒', disabled: false, intent: 'unsubscribe' }
      : { label: '到货提醒我', disabled: false, intent: 'subscribe' }
  }
  const limit = offer.purchase_limit
  if (limit && viewer.used >= limit.quantity) {
    return {
      label: limit.period === 'month' ? '本月已买满' : '已达限购',
      disabled: true,
      intent: 'buy'
    }
  }
  if (!viewer.signedIn) {
    return { label: '登录后购买', disabled: false, intent: 'buy' }
  }
  const short = offer.price - viewer.balance
  if (short > 0) {
    return { label: `还差 ${short} 点`, disabled: true, intent: 'buy' }
  }
  return { label: '购买', disabled: false, intent: 'buy' }
}
