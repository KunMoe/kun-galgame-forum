<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import { settle } from '#shared/utils/api/problem'
import type {
  ShopInventory,
  ShopLoadoutSlot,
  ShopScope,
  ShopSlot
} from '#shared/utils/api/schemas'
import { SHOP_SLOTS } from '~/constants/shop'

const props = defineProps<{ inventory: ShopInventory; viewer: KunUser }>()
const emit = defineEmits<{ worn: [slot: ShopLoadoutSlot]; goStore: [] }>()

const api = useApiClient()
const accountProfile = `${useRuntimeConfig().public.oauthFrontendUrl}/profile`

const scope = ref<ShopScope>('everywhere')
const scopes: KunTabItem[] = [
  { value: 'everywhere', textValue: '所有站点' },
  { value: 'this_site', textValue: '只在本站' }
]
const pending = ref('')

const ownedIn = (slot: ShopSlot) =>
  props.inventory.items.filter((o) => o.item.item_type === slot && o.is_active)
const wornIn = (slot: ShopSlot) =>
  props.inventory.loadout.find(
    (l) => l.scope === scope.value && l.slot === slot
  )?.worn_item_id ?? null

const perks = computed(() =>
  props.inventory.items.filter(
    (o) => o.is_active && o.item.item_type === 'profile_about'
  )
)
const lapsed = computed(() => props.inventory.items.filter((o) => !o.is_active))
const codes = computed(() =>
  props.inventory.orders
    .filter((o) => o.state === 'completed')
    .flatMap((o) =>
      o.redeem_codes.map((c) => ({
        ...c,
        name: o.rewards[0]?.item.display_name ?? '兑换码',
        boughtAt: o.created_at
      }))
    )
)
const hasCosmetics = computed(() =>
  SHOP_SLOTS.some((s) => ownedIn(s.slot).length > 0)
)
const isEmpty = computed(
  () => props.inventory.items.length === 0 && codes.value.length === 0
)

const wear = async (slot: ShopSlot, itemId: string | null) => {
  pending.value = `${slot}:${itemId ?? 'off'}`
  const path = { scope: scope.value, slot }
  const result = await settle(
    itemId
      ? api.PUT('/me/shop/loadout/{scope}/{slot}', {
          params: { path },
          body: { item_id: itemId }
        })
      : api.DELETE('/me/shop/loadout/{scope}/{slot}', { params: { path } })
  )
  pending.value = ''
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  emit('worn', result.data)
  const where =
    scope.value === 'everywhere' ? '所有站点都会显示' : '只在本站显示'
  useMessage(itemId ? `已换上，${where}` : '已取下', 'success')
}

const expiryOf = (expiresAt: string | null) =>
  expiresAt ? `${formatDate(expiresAt, { isShowYear: true })} 到期` : '永久'
</script>

<template>
  <div class="space-y-8">
    <div v-if="isEmpty" class="flex flex-col items-center gap-3">
      <KunNull description="你还没有任何物品" />
      <KunButton size="sm" variant="flat" @click="emit('goStore')">
        去商店看看
      </KunButton>
    </div>

    <template v-else>
      <div
        v-if="hasCosmetics"
        class="flex flex-wrap items-center justify-between gap-3"
      >
        <KunTab v-model="scope" :items="scopes" variant="bordered" size="sm" />
        <p class="text-default-500 text-xs">
          {{
            scope === 'everywhere'
              ? '在所有 NextMoe 站点显示'
              : '只在鲲 Galgame 论坛显示，会盖过「所有站点」的选择'
          }}
        </p>
      </div>

      <section v-for="s in SHOP_SLOTS" :key="s.slot" class="space-y-3">
        <div class="flex items-center justify-between gap-3">
          <h2 class="text-foreground font-semibold">{{ s.label }}</h2>
          <KunButton
            v-if="wornIn(s.slot) !== null"
            size="sm"
            variant="light"
            color="default"
            :loading="pending === `${s.slot}:off`"
            @click="wear(s.slot, null)"
          >
            取下{{ s.label }}
          </KunButton>
        </div>
        <p v-if="ownedIn(s.slot).length === 0" class="text-default-400 text-sm">
          {{ s.empty }}
        </p>
        <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <KunCard
            v-for="owned in ownedIn(s.slot)"
            :key="owned.item.id"
            padding="none"
            class-name="gap-0 overflow-hidden"
            content-class="gap-0"
          >
            <ShopStage :item="owned.item" :viewer="viewer" class-name="h-44">
              <KunChip
                v-if="wornIn(s.slot) === owned.item.id"
                color="success"
                variant="solid"
                size="sm"
                class-name="absolute top-3 right-3"
              >
                使用中
              </KunChip>
            </ShopStage>
            <div class="flex flex-1 items-end justify-between gap-3 p-5">
              <div class="min-w-0">
                <h3 class="text-foreground font-semibold">
                  {{ owned.item.display_name }}
                </h3>
                <p class="text-default-400 mt-1 text-xs">
                  {{ expiryOf(owned.expires_at) }} ·
                  {{ owned.acquired_via === 'grant' ? '赠予' : '购买' }}于
                  {{ formatDate(owned.acquired_at, { isShowYear: true }) }}
                </p>
              </div>
              <KunButton
                v-if="wornIn(s.slot) !== owned.item.id"
                size="sm"
                :loading="pending === `${s.slot}:${owned.item.id}`"
                @click="wear(s.slot, owned.item.id)"
              >
                换上
              </KunButton>
            </div>
          </KunCard>
        </div>
      </section>

      <section v-if="perks.length" class="space-y-3">
        <h2 class="text-foreground font-semibold">功能</h2>
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <KunCard
            v-for="owned in perks"
            :key="owned.item.id"
            padding="md"
            content-class="flex-row items-center justify-start gap-4"
          >
            <div
              class="bg-primary-50 text-primary-600 flex size-12 shrink-0 items-center justify-center rounded-xl"
            >
              <KunIcon name="lucide:notebook-pen" class="size-6" />
            </div>
            <div class="min-w-0 flex-1">
              <h3 class="text-foreground font-semibold">
                {{ owned.item.display_name }}
              </h3>
              <p class="text-default-400 text-xs">
                {{ expiryOf(owned.expires_at) }}
              </p>
            </div>
            <KunButton
              :href="accountProfile"
              target="_blank"
              rel="noopener"
              size="sm"
              variant="flat"
            >
              去写介绍
            </KunButton>
          </KunCard>
        </div>
      </section>

      <section v-if="codes.length" class="space-y-3">
        <h2 class="text-foreground font-semibold">我的兑换码</h2>
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <KunCard
            v-for="c in codes"
            :key="c.code"
            padding="md"
            class-name="border-warning-300 border border-dashed"
          >
            <div class="flex items-center gap-2">
              <div
                class="bg-warning-100 text-warning-600 flex size-8 shrink-0 items-center justify-center rounded-lg"
              >
                <KunIcon name="lucide:ticket-percent" class="size-4" />
              </div>
              <h3 class="text-foreground truncate font-semibold">
                {{ c.name }}
              </h3>
            </div>
            <KunCopy :text="c.code" variant="flat" class-name="font-mono" />
            <p class="text-default-400 text-xs">
              {{ formatDate(c.boughtAt, { isShowYear: true }) }} 购买
              <template v-if="c.expiry_date">
                · {{ c.expiry_date }} 前有效（日本时间）
              </template>
            </p>
          </KunCard>
        </div>
      </section>

      <div v-if="lapsed.length" class="space-y-2">
        <h3 class="text-default-500 text-sm font-medium">已过期</h3>
        <div class="flex flex-wrap gap-2">
          <KunChip v-for="o in lapsed" :key="o.item.id" size="sm">
            {{ o.item.display_name }}
          </KunChip>
        </div>
      </div>
    </template>
  </div>
</template>
