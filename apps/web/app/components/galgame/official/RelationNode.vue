<script setup lang="ts">
import { KUN_GALGAME_OFFICIAL_TREE_ROLE_MAP } from '~/constants/galgameOfficial'

const props = defineProps<{
  official: GalgameOfficialRelationNode
  role?: string
  isCurrent?: boolean
}>()

const logoSrc = computed(() =>
  props.official.logo ? withImageVariant(props.official.logo, 'mini') : ''
)

const meta = computed(() =>
  [
    props.role ? (KUN_GALGAME_OFFICIAL_TREE_ROLE_MAP[props.role] ?? '') : '',
    props.official.work_count > 0 ? `${props.official.work_count} 部` : ''
  ]
    .filter(Boolean)
    .join(' · ')
)
</script>

<template>
  <KunCard
    :is-transparent="false"
    :is-hoverable="!isCurrent"
    :href="isCurrent ? undefined : taxonomyDetailPath('official', official.id)"
    padding="sm"
    :class-name="cn('w-full', isCurrent && 'border-primary-500 bg-primary-50')"
    content-class="flex-row items-center gap-2"
  >
    <div v-if="logoSrc" class="bg-default-100 shrink-0 rounded-md p-1">
      <KunImage
        :src="logoSrc"
        :alt="`${official.name} logo`"
        loading="lazy"
        object-fit="contain"
        class-name="size-7"
      />
    </div>

    <div class="min-w-0 flex-1">
      <p
        :class="
          cn(
            'line-clamp-2 text-sm font-medium break-words',
            isCurrent ? 'text-primary-600' : 'text-foreground'
          )
        "
        :title="official.name"
      >
        {{ official.name }}
      </p>
      <p v-if="meta" class="text-default-500 text-xs">{{ meta }}</p>
    </div>
  </KunCard>
</template>
