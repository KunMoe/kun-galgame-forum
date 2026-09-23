<script setup lang="ts">
import { useSortable, moveArrayElement } from '@vueuse/integrations/useSortable'
import type { FriendLink, FriendLinkCategory } from '#shared/utils/api/schemas'
import { FRIEND_LINK_STATE_CHIP } from '~/constants/friendLink'

const props = defineProps<{
  category: FriendLinkCategory
  label: string
  links: FriendLink[]
}>()

const emits = defineEmits<{
  add: [category: FriendLinkCategory]
  edit: [link: FriendLink]
  remove: [link: FriendLink]
  reorder: [category: FriendLinkCategory, ids: string[]]
}>()

const items = ref<FriendLink[]>([...props.links])
watch(
  () => props.links,
  (value) => {
    items.value = [...value]
  }
)

const listEl = ref<HTMLElement | null>(null)
useSortable(listEl, items, {
  animation: 150,
  handle: '.friend-drag-handle',
  onUpdate: (e) => {
    if (e.oldIndex == null || e.newIndex == null) return
    moveArrayElement(items, e.oldIndex, e.newIndex)
    nextTick(() =>
      emits(
        'reorder',
        props.category,
        items.value.map((item) => item.id)
      )
    )
  }
})
</script>

<template>
  <div class="space-y-3">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-bold">
        {{ label }}
        <span class="text-default-400 text-sm">({{ items.length }})</span>
      </h2>
      <KunButton size="sm" color="primary" @click="emits('add', category)">
        <KunIcon name="lucide:plus" />
        添加友链
      </KunButton>
    </div>

    <div ref="listEl" class="space-y-3">
      <KunCard
        v-for="friend in items"
        :key="friend.id"
        :is-hoverable="false"
        :is-transparent="false"
        padding="sm"
      >
        <div class="flex items-center gap-3">
          <KunIcon
            name="lucide:grip-vertical"
            class="friend-drag-handle text-default-400 shrink-0 cursor-grab active:cursor-grabbing"
          />
          <KunImage
            v-if="friend.banner"
            :src="friend.banner.url"
            class="border-default-200 h-10 w-16 shrink-0 rounded border object-cover"
          />
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="truncate font-medium">{{ friend.title }}</span>
              <KunChip
                v-if="FRIEND_LINK_STATE_CHIP[friend.state]"
                :color="FRIEND_LINK_STATE_CHIP[friend.state]!.color"
              >
                {{ FRIEND_LINK_STATE_CHIP[friend.state]!.label }}
              </KunChip>
            </div>
            <a
              :href="friend.url"
              target="_blank"
              rel="noopener noreferrer"
              class="text-default-500 block truncate text-xs hover:underline"
            >
              {{ friend.url }}
            </a>
          </div>
          <KunButton
            size="sm"
            variant="light"
            :is-icon-only="true"
            @click="emits('edit', friend)"
          >
            <KunIcon name="lucide:pencil" />
          </KunButton>
          <KunButton
            size="sm"
            variant="light"
            color="danger"
            :is-icon-only="true"
            @click="emits('remove', friend)"
          >
            <KunIcon name="lucide:trash-2" />
          </KunButton>
        </div>
      </KunCard>
    </div>

    <div v-if="!items.length" class="text-default-400 py-6 text-center text-sm">
      暂无友链, 点击右上角添加
    </div>
  </div>
</template>
