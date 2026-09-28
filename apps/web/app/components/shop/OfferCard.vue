<script setup lang="ts">
import type { ShopOffer } from '#shared/utils/api/schemas'
import type { OfferAction } from './_offer'

const props = defineProps<{
  offer: ShopOffer
  viewer: KunUser
  action: OfferAction
  owned: boolean
  used: number
}>()

const emit = defineEmits<{ buy: [] }>()

const lead = computed(() => props.offer.rewards[0])
const title = computed(() =>
  props.offer.rewards.map((r) => r.item.display_name).join(' + ')
)
const unit = computed(() =>
  lead.value?.item.item_type === 'redeem_code' ? '张' : '份'
)

const facts = computed(() => {
  const o = props.offer
  const out: string[] = []
  if (o.stock_count !== null) out.push(`限量 ${o.stock_count} 份`)
  if (o.remaining_count !== null && o.remaining_count > 0)
    out.push(`还剩 ${o.remaining_count} ${unit.value}`)
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
    <ShopStage :item="lead?.item" :viewer="viewer" class-name="h-52">
      <KunChip
        v-if="lead?.duration_days"
        size="sm"
        color="info"
        variant="solid"
        class-name="absolute top-3 right-3"
      >
        {{ lead.duration_days }} 天
      </KunChip>
    </ShopStage>

    <div class="flex flex-1 flex-col p-5">
      <h3 class="text-foreground leading-snug font-semibold">{{ title }}</h3>
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
        <ShopPrice :amount="offer.price" class-name="text-xl" />
        <KunChip v-if="owned" color="success" size="md">
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:check" class="size-4" />
            已拥有
          </span>
        </KunChip>
        <KunButton
          v-else
          :color="action.disabled ? 'default' : 'primary'"
          :variant="action.disabled ? 'flat' : 'solid'"
          :disabled="action.disabled"
          @click="emit('buy')"
        >
          {{ action.label }}
        </KunButton>
      </div>
    </div>
  </KunCard>
</template>
