<script setup lang="ts">
import type { WebsiteTag } from '#shared/utils/api/schemas'

const props = defineProps<{
  tags: WebsiteTag[]
  length?: number
  isNav?: boolean
}>()

const sortedTags = computed(() => {
  if (!props.tags || !props.tags.length) {
    return []
  }

  const res = [...props.tags].sort((a, b) => {
    if (a.level !== b.level) {
      return b.level - a.level
    }

    const aIsLegendary = a.slug.endsWith('0')
    const bIsLegendary = b.slug.endsWith('0')

    if (aIsLegendary && !bIsLegendary) {
      return -1
    }
    if (!aIsLegendary && bIsLegendary) {
      return 1
    }

    return 0
  })

  return props.length ? res.slice(0, props.length) : res
})

const tagColor = (tag: WebsiteTag) => {
  const level = tag.level

  if (tag.slug.endsWith('0')) {
    return 'warning'
  }

  if (level >= 10) {
    return 'primary'
  } else if (level >= 5) {
    return 'secondary'
  } else if (level >= 0) {
    return 'success'
  } else if (level >= -5) {
    return 'danger'
  } else {
    return 'default'
  }
}

const tagVariant = (tag: WebsiteTag) => {
  const level = tag.level

  if (tag.slug.endsWith('0')) {
    return 'shadow'
  }
  if (level >= 10) {
    return 'flat'
  } else if (level >= 5) {
    return 'flat'
  } else if (level >= 0) {
    return 'flat'
  } else if (level >= -5) {
    return 'solid'
  } else {
    return 'flat'
  }
}

const handleClick = async (slug: string) => {
  if (props.isNav) {
    await navigateTo(`/website-tag/${slug}`)
  }
}
</script>

<template>
  <template v-for="tag in sortedTags" :key="tag.id">
    <KunTooltip :text="`价值: ${tag.level}`">
      <KunChip
        @click="handleClick(tag.slug)"
        :variant="tagVariant(tag)"
        :color="tagColor(tag)"
        :class-name="isNav ? 'cursor-pointer' : ''"
      >
        {{ tag.label }}
      </KunChip>
    </KunTooltip>
  </template>
</template>
