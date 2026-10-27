<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import { settle } from '#shared/utils/api/problem'
import type {
  ShopInventory,
  ShopLoadoutSlot,
  ShopOffer
} from '#shared/utils/api/schemas'
import {
  SHOP_SHELVES,
  SHOP_SITE_SHELF,
  SHOP_TYPE_SHELF
} from '~/constants/shop'
import {
  isOfferOwned,
  isRestockSubscribed,
  offerAction,
  purchasedCount
} from './_offer'

const api = useApiClient()
const { id, name, avatar, moemoepoint } = storeToRefs(usePersistUserStore())
const signedIn = computed(() => id.value > 0)

const viewer = computed<KunUser>(() => ({
  id: 0,
  name: name.value || '你',
  avatar: avatar.value
}))

const {
  data: offers,
  problem: offersProblem,
  pending: offersPending,
  refresh: refreshOffers
} = useKunApi('shop-offers', (client, { signal }) =>
  client.GET('/shop/offers', { signal })
)

const inventory = ref<ShopInventory | null>(null)
const loadInventory = async () => {
  if (!signedIn.value) {
    inventory.value = null
    return
  }
  const result = await settle(api.GET('/me/shop'))
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  inventory.value = result.data
  moemoepoint.value = result.data.moemoepoint
}
onMounted(loadInventory)
watch(id, loadInventory)

const balance = computed(() => inventory.value?.moemoepoint ?? null)

const tab = ref('store')
const tabs = computed<KunTabItem[]>(() => [
  { value: 'store', textValue: '商店' },
  ...(signedIn.value ? [{ value: 'wardrobe', textValue: '我的物品' }] : [])
])

const shelves = computed(() => {
  const list = offers.value?.items ?? []
  const zone = {
    ...SHOP_SITE_SHELF,
    offers: list.filter((o) => o.is_site_exclusive)
  }
  const shared = list.filter((o) => !o.is_site_exclusive)
  const byType = SHOP_SHELVES.map((shelf) => ({
    ...shelf,
    offers: shared.filter(
      (o) => SHOP_TYPE_SHELF[o.rewards[0]!.item.item_type] === shelf.key
    )
  }))
  return [zone, ...byType].filter((s) => s.offers.length)
})

const restockBusy = ref<string | null>(null)

const stateOf = (offer: ShopOffer) => {
  const used = purchasedCount(offer, inventory.value)
  const subscribed = isRestockSubscribed(offer, inventory.value)
  const action = offerAction(offer, {
    signedIn: signedIn.value,
    owned: isOfferOwned(offer, inventory.value?.items ?? []),
    used,
    balance: balance.value ?? moemoepoint.value,
    subscribed
  })
  return { used, action, subscribed, busy: restockBusy.value === offer.id }
}

const setRestockAlert = async (offer: ShopOffer, subscribe: boolean) => {
  if (!signedIn.value) {
    startOAuthLogin({ returnTo: '/shop' })
    return
  }
  const params = { path: { offer_id: offer.id } }
  restockBusy.value = offer.id
  const result = await settle(
    subscribe
      ? api.PUT('/shop/offers/{offer_id}/restock-alert', { params })
      : api.DELETE('/shop/offers/{offer_id}/restock-alert', { params })
  )
  restockBusy.value = null
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  const inv = inventory.value
  if (inv) {
    const rest = inv.restock_alert_offer_ids.filter((id) => id !== offer.id)
    inv.restock_alert_offer_ids = result.data.is_subscribed
      ? [...rest, offer.id]
      : rest
  }
  useMessage(
    result.data.is_subscribed ? '补货时会在站内信通知你' : '已取消到货提醒',
    'success'
  )
}

const buying = ref<ShopOffer | null>(null)
const purchaseOpen = ref(false)
const openPurchase = (offer: ShopOffer) => {
  if (!signedIn.value) {
    startOAuthLogin({ returnTo: '/shop' })
    return
  }
  buying.value = offer
  purchaseOpen.value = true
}

const reload = () => Promise.all([loadInventory(), refreshOffers()])

const onPurchased = async () => {
  await reload()
  tab.value = 'wardrobe'
}

const onWorn = (slot: ShopLoadoutSlot) => {
  const inv = inventory.value
  if (!inv) return
  inv.loadout = inv.loadout.map((l) =>
    l.scope === slot.scope && l.slot === slot.slot ? slot : l
  )
}
</script>

<template>
  <div class="space-y-8 pb-12">
    <div class="flex flex-wrap items-end justify-between gap-6">
      <KunHeader
        name="萌萌点商店"
        description="用萌萌点换装扮、功能和福利。买到的东西属于你的 NextMoe 账号，在所有站点通用。商店只收萌萌点，不涉及任何真实货币。"
        class="max-w-2xl"
      />
      <ShopWallet v-if="signedIn" :balance="balance" />
    </div>

    <KunTab v-model="tab" :items="tabs" variant="underlined" />

    <template v-if="tab === 'store'">
      <div
        v-if="offersPending"
        class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3"
        aria-busy="true"
      >
        <KunCard
          v-for="n in 3"
          :key="n"
          padding="none"
          class-name="gap-0 overflow-hidden"
        >
          <KunSkeleton height="12rem" rounded="none" />
          <div class="space-y-3 p-5">
            <KunSkeleton variant="text" width="50%" />
            <KunSkeleton variant="text" />
            <KunSkeleton variant="text" width="30%" />
          </div>
        </KunCard>
      </div>

      <KunNull
        v-else-if="offersProblem"
        description="商店暂时打不开，过一会儿再来看看吧"
      />

      <KunNull
        v-else-if="shelves.length === 0"
        description="商店还在准备中，过几天再来看看吧"
      />

      <div v-else class="space-y-12">
        <ShopShelf
          v-for="shelf in shelves"
          :key="shelf.key"
          :title="shelf.title"
          :note="shelf.note"
        >
          <ShopOfferCard
            v-for="offer in shelf.offers"
            :key="offer.id"
            :offer="offer"
            :viewer="viewer"
            v-bind="stateOf(offer)"
            @buy="openPurchase(offer)"
            @restock="setRestockAlert(offer, $event)"
          />
        </ShopShelf>
      </div>
    </template>

    <template v-else>
      <KunLoading v-if="!inventory" />
      <ShopWardrobe
        v-else
        :inventory="inventory"
        :viewer="viewer"
        @worn="onWorn"
        @go-store="tab = 'store'"
      />
    </template>

    <ShopPurchaseModal
      v-model="purchaseOpen"
      :offer="buying"
      :viewer="viewer"
      :balance="balance ?? moemoepoint"
      @purchased="onPurchased"
      @stale="reload"
    />
  </div>
</template>
