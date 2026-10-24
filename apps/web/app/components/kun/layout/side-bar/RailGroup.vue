<script setup lang="ts">
import type { KunRailGroup } from '~/constants/layout'

const props = defineProps<{
  group: KunRailGroup
}>()

const route = useRoute()
const isLinkActive = (router: string) =>
  route.path === router || route.path.startsWith(`${router}/`)

const isGroupActive = computed(() => {
  if (props.group.router && isLinkActive(props.group.router)) return true
  return props.group.sections.some((s) =>
    s.items.some((i) => isLinkActive(i.router))
  )
})
</script>

<template>
  <KunButton
    v-if="!group.sections.length"
    variant="light"
    color="default"
    :href="group.router"
    :aria-label="group.label"
    :class-name="
      cn(
        'relative h-auto w-full flex-col gap-1 py-2',
        isGroupActive ? 'text-primary' : 'text-foreground'
      )
    "
  >
    <KunIcon :name="group.icon" class="text-xl" />
    <span class="text-[11px] leading-none">{{ group.label }}</span>
    <span
      v-if="group.hint"
      class="bg-secondary absolute top-0.5 right-0.5 rounded px-1 text-[9px] leading-tight text-white"
    >
      {{ group.hint }}
    </span>
  </KunButton>
  <KunPopover
    v-else
    position="right-start"
    trigger="hover"
    group="kun-sidebar-rail"
    :close-delay="150"
    :aria-label="group.label"
    opaque
    full-width
    inner-class="min-w-52 p-2"
  >
    <template #trigger>
      <KunButton
        variant="light"
        color="default"
        :href="group.router"
        :aria-label="group.label"
        :class-name="
          cn(
            'relative h-auto w-full flex-col gap-1 py-2',
            isGroupActive ? 'text-primary' : 'text-foreground'
          )
        "
      >
        <KunIcon :name="group.icon" class="text-xl" />
        <span class="text-[11px] leading-none">{{ group.label }}</span>
        <span
          v-if="group.hint"
          class="bg-secondary absolute top-0.5 right-0.5 rounded px-1 text-[9px] leading-tight text-white"
        >
          {{ group.hint }}
        </span>
      </KunButton>
    </template>

    <template
      v-for="(section, si) in group.sections"
      :key="`${group.name}-${si}`"
    >
      <div v-if="si > 0" class="border-default-200/60 my-1 border-t" />
      <p
        v-if="section.label"
        class="text-default-500 px-2 pt-1 pb-0.5 text-xs select-none"
      >
        {{ section.label }}
      </p>
      <KunLink
        v-for="link in section.items"
        :key="link.router"
        :to="link.router"
        :target="link.external ? '_blank' : undefined"
        underline="none"
        color="default"
        :class-name="
          cn(
            'hover:bg-default-100 flex items-center gap-2 rounded-md px-2 py-1.5 text-sm',
            isLinkActive(link.router)
              ? 'bg-accent text-primary'
              : 'text-foreground'
          )
        "
      >
        <KunIcon
          v-if="link.icon"
          :name="link.icon"
          class="shrink-0 text-base"
        />
        <span class="whitespace-nowrap">{{ link.label }}</span>
        <span v-if="link.hint" class="text-primary ml-auto text-xs">
          {{ link.hint }}
        </span>
      </KunLink>
    </template>
  </KunPopover>
</template>
