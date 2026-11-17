<script setup lang="ts">
import { kunLayoutItem, type KunLayoutItem } from '~/constants/layout'

const agentItem: KunLayoutItem = {
  name: 'agentChat',
  icon: 'lucide:bot-message-square',
  router: '/chat',
  label: 'AI 对话'
}
const items = useRuntimeConfig().public.agentEnabled
  ? [agentItem, ...kunLayoutItem]
  : kunLayoutItem

const expandedItems = ref<string[]>(
  kunLayoutItem
    .filter((item) => item.children && !item.isCollapse)
    .map((item) => item.name)
)

const toggleExpand = (itemName: string) => {
  const index = expandedItems.value.indexOf(itemName)
  if (index === -1) {
    expandedItems.value.push(itemName)
  } else {
    expandedItems.value.splice(index, 1)
  }
}
</script>

<template>
  <div class="flex flex-col justify-center gap-1 py-3">
    <KunLayoutSideBarNavItem
      v-for="item in items"
      :key="item.name"
      :item="item"
      :expanded-items="expandedItems"
      @toggle-expand="toggleExpand"
    />
  </div>
</template>
