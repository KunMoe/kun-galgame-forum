<script setup lang="ts">
import type { ShopItem } from '#shared/utils/api/schemas'
import { SHOP_STAGE_TINT } from '~/constants/shop'

const props = defineProps<{
  item: ShopItem | undefined
  viewer: KunUser
  decoration?: 'hover' | 'always'
  className?: string
}>()

const type = computed(() => props.item?.item_type ?? 'avatar_frame')

const framed = computed<KunUser>(() => {
  const art = props.item?.artwork
  return {
    ...props.viewer,
    avatarDecoration: art && {
      src: art.static_url,
      animatedSrc: art.animated_url ?? undefined
    }
  }
})
</script>

<template>
  <div
    :class="
      cn(
        'relative flex items-center justify-center px-5 py-6',
        SHOP_STAGE_TINT[type],
        className
      )
    "
  >
    <div
      v-if="type === 'profile_background'"
      class="bg-content1 shadow-kun-sm w-full max-w-80 overflow-hidden rounded-xl"
    >
      <ShopBanner :artwork="item?.artwork" />
      <div class="flex items-end gap-2 px-3 pb-3">
        <KunAvatar
          :user="viewer"
          size="md"
          :is-navigation="false"
          class="ring-content1 -mt-5 rounded-full ring-2"
        />
        <span class="text-foreground truncate text-xs font-medium">
          {{ viewer.name }}
        </span>
      </div>
    </div>

    <div
      v-else-if="type === 'profile_about'"
      class="bg-content1 shadow-kun-sm w-56 space-y-3 rounded-xl p-4"
    >
      <div class="flex items-center gap-2">
        <KunAvatar :user="viewer" size="sm" :is-navigation="false" />
        <span class="text-foreground truncate text-xs font-semibold">
          {{ viewer.name }}
        </span>
        <KunIcon
          name="lucide:notebook-pen"
          class="text-primary-500 ml-auto size-4 shrink-0"
        />
      </div>
      <div class="space-y-1.5">
        <p class="text-foreground text-xs font-semibold">关于我</p>
        <div class="bg-default-200 h-1.5 w-full rounded-full" />
        <div class="bg-default-200 h-1.5 w-11/12 rounded-full" />
        <div class="bg-default-200 h-1.5 w-3/5 rounded-full" />
      </div>
    </div>

    <ShopCoupon
      v-else-if="type === 'redeem_code'"
      :name="item?.display_name ?? ''"
    />

    <KunAvatar
      v-else
      :user="framed"
      size="original-sm"
      :decoration="decoration ?? 'hover'"
      :is-navigation="false"
    />

    <slot />
  </div>
</template>
