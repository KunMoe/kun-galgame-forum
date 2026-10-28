<script setup lang="ts">
import type { EntityTarget } from '#shared/utils/api/schemas'
import {
  ENTITY_ICON,
  entityHref,
  entityKind,
  entityTitle
} from '~/utils/entityTarget'

const props = defineProps<{
  target: EntityTarget
  inert?: boolean
}>()

const { showKUNGalgamePreferOriginalName } = storeToRefs(
  usePersistSettingsStore()
)

const CHIP_CLASS =
  'kun-content-entity-chip bg-default-100 inline-flex max-w-[min(100%,20em)] items-center gap-1 rounded-md px-1.5 align-bottom text-[0.92em] leading-normal'

const kind = computed(() => entityKind(props.target))
const isUnavailable = computed(
  () => props.target.object === 'unavailable_entity'
)
const label = computed(() => {
  const title = entityTitle(
    props.target,
    showKUNGalgamePreferOriginalName.value
  )
  return props.target.object === 'user' ? `@${title}` : title
})
</script>

<template>
  <span
    v-if="inert"
    :title="label"
    :class="cn(CHIP_CLASS, isUnavailable ? 'text-default-500' : 'text-primary')"
  >
    <KunIcon
      :name="isUnavailable ? 'lucide:eye-off' : ENTITY_ICON[kind]"
      class="size-[1.05em] shrink-0"
    />
    <span class="truncate">{{ label }}</span>
  </span>
  <KunLink
    v-else
    :to="entityHref(target)"
    :title="label"
    :color="isUnavailable ? 'default' : 'primary'"
    underline="none"
    :class-name="cn(CHIP_CLASS, 'hover:bg-default-200 transition-colors')"
  >
    <KunIcon
      :name="isUnavailable ? 'lucide:eye-off' : ENTITY_ICON[kind]"
      class="size-[1.05em] shrink-0"
    />
    <span class="truncate">{{ label }}</span>
  </KunLink>
</template>

<style scoped>
.kun-content-entity-chip {
  font-weight: 400;
  text-decoration: none;
}
</style>
