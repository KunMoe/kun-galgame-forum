<script setup lang="ts">
import type { ShopOffer } from '#shared/utils/api/schemas'
import type { OfferAction } from './_offer'

const props = defineProps<{
  offer: ShopOffer
  viewer: KunUser
  action: OfferAction
  used: number
  subscribed: boolean
  busy: boolean
}>()

const emit = defineEmits<{ buy: []; restock: [subscribe: boolean] }>()

const restockDay = new Intl.DateTimeFormat('zh-CN', {
  timeZone: 'Asia/Shanghai',
  month: 'long',
  day: 'numeric'
})

const lead = computed(() => props.offer.rewards[0])
const title = computed(() =>
  props.offer.rewards.map((r) => r.item.display_name).join(' + ')
)
const isCode = computed(() => lead.value?.item.item_type === 'redeem_code')

const facts = computed(() => {
  const o = props.offer
  const out: string[] = []
  if (!isCode.value) {
    out.push(
      lead.value?.duration_days
        ? `有效期 ${lead.value.duration_days} 天`
        : '永久'
    )
  }
  if (o.stock_count !== null) out.push(`限量 ${o.stock_count} 份`)
  const unit = isCode.value ? '张' : '份'
  if (o.remaining_count !== null && o.remaining_count > 0)
    out.push(`还剩 ${o.remaining_count} ${unit}`)
  if (o.sold_count > 0) out.push(`已售 ${o.sold_count} ${unit}`)
  if (o.restocked_at)
    out.push(`${restockDay.format(new Date(o.restocked_at))}补货`)
  const limit = o.purchase_limit
  if (limit) {
    const period = limit.period === 'month' ? '每月' : '每人'
    out.push(
      props.used
        ? `${period}限购 ${limit.quantity}，已买 ${props.used}`
        : `${period}限购 ${limit.quantity}`
    )
  }
  if (o.ends_at) out.push(`${formatDate(o.ends_at, { isShowYear: true })} 下架`)
  return out
})
</script>

<template>
  <KunCard
    padding="none"
    :is-hoverable="true"
    class-name="gap-0 overflow-hidden"
    content-class="gap-0"
  >
    <ShopStage :item="lead?.item" :viewer="viewer" class-name="h-48" />

    <div class="flex flex-1 flex-col p-5">
      <h3 class="text-foreground text-base leading-snug font-semibold">
        {{ title }}
      </h3>
      <p
        v-if="lead?.item.description"
        class="text-default-500 mt-1.5 line-clamp-2 text-sm"
      >
        {{ lead.item.description }}
      </p>
      <p v-if="facts.length" class="text-default-400 mt-3 text-xs">
        {{ facts.join(' · ') }}
      </p>

      <div class="mt-auto flex items-center justify-between gap-3 pt-5">
        <ShopPrice :amount="offer.price" class-name="text-2xl" />
        <span v-if="action.disabled" class="text-default-500 text-sm">
          {{ action.label }}
        </span>
        <KunButton v-else-if="action.intent === 'buy'" @click="emit('buy')">
          {{ action.label }}
        </KunButton>
        <KunButton
          v-else
          :variant="action.intent === 'unsubscribe' ? 'flat' : 'solid'"
          :loading="busy"
          @click="emit('restock', action.intent === 'subscribe')"
        >
          <KunIcon
            :name="
              action.intent === 'unsubscribe'
                ? 'lucide:bell-ring'
                : 'lucide:bell'
            "
          />
          {{ action.label }}
        </KunButton>
      </div>
      <KunButton
        v-if="subscribed && action.intent !== 'unsubscribe'"
        size="sm"
        variant="light"
        color="default"
        class-name="mt-2 self-end"
        :loading="busy"
        @click="emit('restock', false)"
      >
        取消到货提醒
      </KunButton>
    </div>
  </KunCard>
</template>
