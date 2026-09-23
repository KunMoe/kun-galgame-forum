<script setup lang="ts">
import { KUN_DLSITE_ANNOUNCE_TOPIC_ID } from '~/constants/dlsite'

const props = defineProps<{
  workId: string
  purchaseUrl?: string
  couponUrl?: string
}>()

const hasOffer = computed(() => Boolean(props.purchaseUrl))

const perks = [
  { icon: 'lucide:ticket', label: '每 3 个月 ¥400 优惠券' },
  { icon: 'lucide:gift', label: '每月 20 张 ¥1000 优惠券' },
  { icon: 'lucide:coins', label: '12% 返还点数 → 买游戏分享给大家' }
]
</script>

<template>
  <KunInfo
    v-if="hasOffer"
    color="primary"
    variant="bordered"
    title="支持正版 · 与 DLsite 官方合作"
  >
    <div class="space-y-3">
      <div class="flex flex-wrap gap-1.5">
        <KunChip
          v-for="perk in perks"
          :key="perk.label"
          color="primary"
          variant="flat"
          size="sm"
        >
          <span class="flex items-center gap-1">
            <KunIcon :name="perk.icon" />
            {{ perk.label }}
          </span>
        </KunChip>
      </div>

      <p class="text-default-600 text-sm">
        通过本站链接购买, DLsite 给本站的
        <span class="text-default-800 font-medium">全部分成都用于回馈用户</span>
        —— 鲲 Galgame 不从这次合作中获得任何收益。
      </p>
      <p class="text-default-500 text-xs">
        DLsite 无法把分成折算成个人折扣, 因此这 12% 以点数形式汇入论坛公共账户,
        用于购买游戏并分享给所有人 —— 本站不留一分。
      </p>

      <div class="flex flex-wrap items-center gap-2">
        <KunButton
          :href="purchaseUrl"
          target="_blank"
          rel="noopener noreferrer"
          color="primary"
          size="sm"
          class-name="gap-1.5"
        >
          <KunIcon name="lucide:shopping-cart" />
          前往 DLsite 购买正版
        </KunButton>

        <KunButton
          v-if="couponUrl"
          :href="couponUrl"
          target="_blank"
          rel="noopener noreferrer"
          variant="light"
          color="primary"
          size="sm"
          class-name="gap-1.5"
        >
          <KunIcon name="lucide:ticket" />
          领取优惠券
        </KunButton>

        <KunLink
          v-if="KUN_DLSITE_ANNOUNCE_TOPIC_ID"
          size="sm"
          :to="`/topic/${KUN_DLSITE_ANNOUNCE_TOPIC_ID}`"
        >
          合作详情
        </KunLink>
      </div>
    </div>
  </KunInfo>

  <KunInfo v-else color="danger" variant="bordered" title="补票提示">
    <p class="text-sm">
      Galgame 厂商制作游戏不易, 很多厂商如今都在炒冷饭, 可见经济并不宽裕。
      如果条件允许, 可前往
      <KunLink size="sm" :to="`/galgame/${workId}`" class-name="inline">
        Galgame 详情
      </KunLink>
      中的制作商部分进行正版补票, 感谢您对 Galgame 业界做出的贡献。
    </p>
  </KunInfo>
</template>
