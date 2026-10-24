<script setup lang="ts">
import type { KunLayoutItem } from '~/constants/layout'

const props = defineProps<{
  item: KunLayoutItem
  expandedItems: string[]
}>()

defineEmits<{
  'toggle-expand': [itemName: string]
}>()

const route = useRoute()

const isExpanded = computed(() => props.expandedItems.includes(props.item.name))
</script>

<template>
  <div class="w-full">
    <template v-if="item.children">
      <div class="flex w-full flex-col">
        <KunButton
          variant="light"
          @click="$emit('toggle-expand', item.name)"
          :class-name="cn(isExpanded ? 'mb-1' : '')"
        >
          <KunIcon
            v-if="item.icon"
            class="text-foreground mr-3 flex items-center justify-center text-xl"
            :name="item.icon"
          />
          <span class="text-foreground text-sm sm:text-base">
            {{ item.label }}
          </span>

          <KunIcon
            :name="isExpanded ? 'lucide:chevron-down' : 'lucide:chevron-right'"
            class="ml-auto"
          />
        </KunButton>

        <div
          v-show="isExpanded"
          class="ml-8 flex flex-col gap-1 transition-all"
        >
          <KunLayoutSideBarNavItem
            v-for="child in item.children"
            :key="child.name"
            :item="child"
            :expanded-items="expandedItems"
            @toggle-expand="$emit('toggle-expand', $event)"
          />
        </div>
      </div>
    </template>

    <KunButton
      v-else
      class-name="justify-start w-full"
      :variant="route.path === item.router ? 'flat' : 'light'"
      :href="item.router"
      :target="item.external ? '_blank' : undefined"
    >
      <span
        :class="
          cn(
            'flex items-center justify-center gap-2 text-xl',
            route.path === item.router ? '' : 'text-foreground'
          )
        "
      >
        <KunIcon v-if="item.icon" :name="item.icon" />
        <span class="text-sm sm:text-base">{{ item.label }}</span>
      </span>

      <span
        v-if="item.hint"
        :class="[
          'ml-auto text-xs',
          item.hintColor === 'danger'
            ? 'text-danger'
            : item.hintColor === 'secondary'
              ? 'text-secondary'
              : 'text-primary'
        ]"
      >
        {{ item.hint }}
      </span>
    </KunButton>
  </div>
</template>
