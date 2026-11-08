<script setup lang="ts">
import type { CollectionSummary } from '#shared/utils/api/schemas'

withDefaults(
  defineProps<{
    collection: Pick<
      CollectionSummary,
      'view_count' | 'item_count' | 'follower_count'
    >
    size?: 'sm' | 'md' | 'lg'
  }>(),
  { size: 'sm' }
)
</script>

<template>
  <div
    :class="
      cn(
        'text-default-500 flex flex-wrap items-center',
        size === 'sm' && 'gap-3 text-xs',
        size === 'md' && 'gap-4 text-sm',
        size === 'lg' && 'gap-5 text-base'
      )
    "
  >
    <span class="inline-flex items-center gap-1 leading-none" title="浏览">
      <KunIcon
        name="lucide:eye"
        :class="size === 'sm' ? 'size-3.5' : 'size-4'"
      />
      {{ formatNumber(collection.view_count) }}
      <span v-if="size === 'lg'">浏览</span>
    </span>
    <span class="inline-flex items-center gap-1 leading-none" title="游戏">
      <KunIcon
        name="lucide:gamepad-2"
        :class="size === 'sm' ? 'size-3.5' : 'size-4'"
      />
      {{ formatNumber(collection.item_count) }}
      <span v-if="size === 'lg'">个游戏</span>
    </span>
    <span class="inline-flex items-center gap-1 leading-none" title="关注">
      <KunIcon
        name="lucide:users"
        :class="size === 'sm' ? 'size-3.5' : 'size-4'"
      />
      {{ formatNumber(collection.follower_count ?? 0) }}
      <span v-if="size === 'lg'">人关注</span>
    </span>
  </div>
</template>
