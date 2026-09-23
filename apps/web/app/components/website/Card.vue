<script setup lang="ts">
import { KUN_WEBSITE_STATUS_CHIP } from '~/constants/galgameWebsite'
import type { WebsiteSummary } from '#shared/utils/api/schemas'

const props = defineProps<{
  website: WebsiteSummary
}>()

const statusChip = computed(() => KUN_WEBSITE_STATUS_CHIP[props.website.state])

const iconSrc = computed(
  () => props.website.icon?.url ?? props.website.external_icon_url ?? ''
)

const scoreColor = computed(() => {
  const score = props.website.score
  if (score > 200) {
    return 'text-warning-500'
  }
  if (score > 100) {
    return 'text-success-600'
  }
  if (score > 0) {
    return 'text-default-700'
  }
  return 'text-danger-600'
})
</script>

<template>
  <KunCard
    :is-transparent="false"
    :href="`/website/${website.host}`"
    class-name="group"
    content-class="space-y-3"
  >
    <div class="flex items-start space-x-4">
      <div class="flex-shrink-0">
        <KunImage
          :src="iconSrc"
          :alt="website.title"
          :class="
            cn(
              'h-12 w-12 rounded-2xl object-cover',
              website.state === 'closed' && 'grayscale'
            )
          "
        />
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <h3
            class="group-hover:text-primary-500 text-default-900 truncate text-lg font-semibold transition-colors"
          >
            {{ website.title }}
          </h3>
          <KunChip
            v-if="statusChip"
            :color="statusChip.color"
            class-name="shrink-0"
          >
            {{ statusChip.label }}
          </KunChip>
        </div>
        <p class="text-default-500 truncate font-mono text-sm">
          {{ website.host }}
        </p>
      </div>
    </div>

    <p class="text-default-600 line-clamp-2 text-sm leading-relaxed">
      {{ website.description }}
    </p>

    <div class="text-default-500 flex items-center justify-between text-sm">
      <span>网站价值精算值</span>
      <span :class="cn('font-bold', scoreColor)">
        {{ website.score }}
      </span>
    </div>
  </KunCard>
</template>
