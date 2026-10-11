import { describe, expect, it } from 'vitest'
import type { ShopOffer, ShopOwnedItem } from '#shared/utils/api/schemas'
import { isOfferOwned, offerAction } from './_offer'

const item = (id: string): ShopOffer['rewards'][number]['item'] => ({
  object: 'shop_item',
  id,
  item_type: 'avatar_frame',
  display_name: 'frame',
  description: '',
  artwork: null
})

const offer = (patch: Partial<ShopOffer> = {}): ShopOffer => ({
  object: 'shop_offer',
  id: '9',
  price: 300,
  is_site_exclusive: false,
  rewards: [{ item: item('9'), duration_days: null }],
  purchase_limit: null,
  stock_count: null,
  remaining_count: null,
  sold_count: 0,
  restocked_at: null,
  ends_at: null,
  ...patch
})

const viewer = {
  signedIn: true,
  owned: false,
  used: 0,
  balance: 500,
  subscribed: false
}

describe('offerAction', () => {
  it('lets a signed-in user who can afford it buy', () => {
    expect(offerAction(offer(), viewer)).toEqual({
      label: '购买',
      disabled: false,
      intent: 'buy'
    })
  })

  it('names the shortfall and refuses when the balance is short', () => {
    expect(offerAction(offer(), { ...viewer, balance: 120 })).toEqual({
      label: '还差 180 点',
      disabled: true,
      intent: 'buy'
    })
  })

  it('puts owned, sold out and limit ahead of the balance', () => {
    const poor = { ...viewer, balance: 0 }
    expect(offerAction(offer(), { ...poor, owned: true }).label).toBe('已拥有')
    expect(offerAction(offer({ remaining_count: 0 }), poor).label).toBe(
      '到货提醒我'
    )
    const monthly = offer({
      purchase_limit: { quantity: 2, period: 'month' }
    })
    expect(offerAction(monthly, { ...poor, used: 2 })).toEqual({
      label: '本月已买满',
      disabled: true,
      intent: 'buy'
    })
    const lifetime = offer({
      purchase_limit: { quantity: 1, period: 'lifetime' }
    })
    expect(offerAction(lifetime, { ...poor, used: 1 }).label).toBe('已达限购')
  })

  it('asks an anonymous visitor to sign in instead of judging a balance it does not know', () => {
    expect(
      offerAction(offer(), { ...viewer, signedIn: false, balance: 0 })
    ).toEqual({ label: '登录后购买', disabled: false, intent: 'buy' })
  })

  it('offers a restock alert on a sold-out offer, even past the purchase limit', () => {
    const soldOut = offer({
      remaining_count: 0,
      purchase_limit: { quantity: 1, period: 'month' }
    })
    expect(offerAction(soldOut, { ...viewer, used: 1 })).toEqual({
      label: '到货提醒我',
      disabled: false,
      intent: 'subscribe'
    })
    expect(offerAction(soldOut, { ...viewer, signedIn: false }).intent).toBe(
      'subscribe'
    )
    expect(offerAction(soldOut, { ...viewer, subscribed: true })).toEqual({
      label: '已开启到货提醒',
      disabled: false,
      intent: 'unsubscribe'
    })
  })
})

describe('isOfferOwned', () => {
  const held = (
    id: string,
    patch: Partial<ShopOwnedItem> = {}
  ): ShopOwnedItem => ({
    item: item(id),
    acquired_via: 'purchase',
    acquired_at: '2026-09-20T00:00:00Z',
    expires_at: null,
    is_active: true,
    ...patch
  })

  it('counts only a permanent, active holding', () => {
    expect(isOfferOwned(offer(), [held('9')])).toBe(true)
    expect(
      isOfferOwned(offer(), [held('9', { expires_at: '2026-10-20T00:00:00Z' })])
    ).toBe(false)
    expect(isOfferOwned(offer(), [held('9', { is_active: false })])).toBe(false)
    const bundle = offer({
      rewards: [
        { item: item('9'), duration_days: null },
        { item: item('10'), duration_days: null }
      ]
    })
    expect(isOfferOwned(bundle, [held('9')])).toBe(true)
  })

  it('refuses a bundle once any one of its items is held for good, as the account service does', () => {
    const bundle = offer({
      rewards: [
        { item: item('9'), duration_days: null },
        { item: item('10'), duration_days: null }
      ]
    })
    expect(isOfferOwned(bundle, [held('10')])).toBe(true)
    expect(isOfferOwned(bundle, [])).toBe(false)
  })

  it('never counts a redeem code as owned', () => {
    const code = { ...item('7'), item_type: 'redeem_code' as const }
    const coupon = offer({ rewards: [{ item: code, duration_days: null }] })
    expect(isOfferOwned(coupon, [{ ...held('7'), item: code }])).toBe(false)
  })
})
