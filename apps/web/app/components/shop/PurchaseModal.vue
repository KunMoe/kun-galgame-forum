<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { ShopOffer, ShopRedeemCode } from '#shared/utils/api/schemas'
import { useIdempotencyKey } from '~/composables/useIdempotencyKey'

const props = defineProps<{
  offer: ShopOffer | null
  viewer: KunUser
  balance: number
}>()

const open = defineModel<boolean>({ required: true })
const emit = defineEmits<{ purchased: []; stale: [] }>()

const api = useApiClient()
const orderKey = useIdempotencyKey()
const { moemoepoint } = storeToRefs(usePersistUserStore())
const accountProfile = `${useRuntimeConfig().public.oauthFrontendUrl}/profile`

const submitting = ref(false)
const codes = ref<ShopRedeemCode[]>([])

watch(open, (v) => {
  if (v) {
    codes.value = []
  } else if (codes.value.length) {
    emit('purchased')
  }
})

const lead = computed(() => props.offer?.rewards[0])
const title = computed(
  () => props.offer?.rewards.map((r) => r.item.display_name).join(' + ') ?? ''
)
const isCode = computed(() => lead.value?.item.item_type === 'redeem_code')
const after = computed(() => props.balance - (props.offer?.price ?? 0))
const term = computed(() => {
  if (isCode.value) return '付款后立即发放一张兑换码'
  return lead.value?.duration_days
    ? `有效期 ${lead.value.duration_days} 天，再次购买会顺延`
    : '永久拥有'
})

const confirm = async () => {
  const offer = props.offer
  if (!offer || submitting.value) return
  submitting.value = true
  const body = { offer_id: offer.id }
  const result = await settle(
    api.POST('/me/shop/orders', {
      params: {
        header: { 'Idempotency-Key': orderKey.take('/me/shop/orders', body) }
      },
      body
    })
  )
  submitting.value = false

  if (!result.ok) {
    const { problem } = result
    // A 4xx is final and the server keeps its answer under this key, so a
    // later attempt at the same offer needs a new one; a 5xx or a lost
    // connection may have charged, and only the same key replays it safely.
    if (problem.status >= 400 && problem.status < 500) {
      orderKey.clear()
      emit('stale')
    }
    if (problem.code === 'ALREADY_EXISTS') {
      useMessage('你已经永久拥有这件物品了', 'warn')
      return
    }
    reportProblem(problem)
    return
  }

  orderKey.clear()
  moemoepoint.value = result.data.moemoepoint
  const issued = result.data.order.redeem_codes
  if (issued.length) {
    codes.value = issued
    return
  }
  const type = lead.value?.item.item_type
  const hint =
    type === 'profile_about'
      ? '去账号中心的个人资料里写主页介绍吧'
      : '去「我的物品」换上它吧'
  useMessage(`已购买「${title.value}」，${hint}`, 'success')
  open.value = false
  emit('purchased')
}
</script>

<template>
  <KunModal
    v-model="open"
    :title="codes.length ? '购买成功' : '确认购买'"
    size="sm"
    :is-dismissable="!submitting"
  >
    <div v-if="codes.length" class="space-y-5">
      <p class="text-default-500 text-sm">
        这是你的「{{ title }}」兑换码，之后也能在「我的物品」里找到它。
      </p>
      <div
        v-for="c in codes"
        :key="c.code"
        class="border-default-200 flex flex-col items-center gap-2 rounded-xl border p-5"
      >
        <KunCopy
          :text="c.code"
          variant="light"
          color="default"
          class-name="font-mono text-base"
        />
        <span v-if="c.expiry_date" class="text-default-500 text-xs">
          {{ c.expiry_date }} 前有效（日本时间）
        </span>
      </div>
      <p v-if="lead?.item.description" class="text-default-500 text-sm">
        {{ lead.item.description }}
      </p>
      <div class="flex justify-end">
        <KunButton @click="open = false">好的</KunButton>
      </div>
    </div>

    <div v-else-if="offer" class="space-y-5">
      <ShopStage
        :item="lead?.item"
        :viewer="viewer"
        decoration="always"
        class-name="h-48 rounded-xl"
      />
      <div>
        <p class="text-foreground text-lg font-semibold">{{ title }}</p>
        <p class="text-default-500 mt-1 text-sm">{{ term }}</p>
      </div>

      <dl
        class="divide-default-200 border-default-200 divide-y border-y text-sm"
      >
        <div class="flex items-center justify-between py-2.5">
          <dt class="text-default-500">价格</dt>
          <dd><ShopPrice :amount="offer.price" /></dd>
        </div>
        <div class="flex items-center justify-between py-2.5">
          <dt class="text-default-500">当前余额</dt>
          <dd><ShopPrice :amount="balance" class-name="font-normal" /></dd>
        </div>
        <div class="flex items-center justify-between py-2.5">
          <dt class="text-default-500">购买后余额</dt>
          <dd>
            <ShopPrice
              :amount="after"
              :class-name="after < 0 ? 'text-danger-600' : ''"
            />
          </dd>
        </div>
      </dl>

      <p v-if="isCode" class="text-warning-600 text-xs leading-relaxed">
        兑换码发出后无法收回，所以购买后不能退款。
      </p>
      <p
        v-else-if="lead?.item.item_type === 'profile_about'"
        class="text-default-500 text-xs leading-relaxed"
      >
        主页介绍在
        <KunLink
          :href="accountProfile"
          target="_blank"
          rel="noopener"
          underline="hover"
          size="xs"
        >
          账号中心
        </KunLink>
        编辑，所有站点的个人主页都会显示。
      </p>

      <div class="flex justify-end gap-2">
        <KunButton variant="light" color="default" @click="open = false">
          取消
        </KunButton>
        <KunButton :loading="submitting" :disabled="after < 0" @click="confirm">
          确认购买
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
