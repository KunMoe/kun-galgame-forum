<script setup lang="ts">
import type { Section } from '#shared/utils/api/schemas'

const props = defineProps<{ section: Section }>()

// useState, not a bare Math.random(): the pick has to survive into the client
// payload, or SSR and hydration draw two different mascots.
const pick = useState(`kun-section-mascot-${props.section.section}`, () =>
  Math.random()
)

const ICON_COLOR: Record<Section['color'], string> = {
  default: 'text-default-500',
  primary: 'text-primary',
  secondary: 'text-secondary',
  success: 'text-success',
  warning: 'text-warning',
  danger: 'text-danger'
}

const mascot = computed(() => {
  const list = props.section.mascots
  return list.length ? list[Math.floor(pick.value * list.length)] : null
})
const hasArt = computed(() => !!props.section.banner || !!mascot.value)
const isAboutOpen = ref(false)
</script>

<template>
  <header
    class="border-default-200 bg-default-50 overflow-hidden rounded-2xl border"
  >
    <div class="relative">
      <img
        v-if="section.banner"
        :src="section.banner.url"
        alt=""
        aria-hidden="true"
        class="absolute inset-0 size-full object-cover"
      />
      <img
        v-if="mascot"
        :src="mascot.url"
        alt=""
        aria-hidden="true"
        decoding="async"
        fetchpriority="low"
        class="pointer-events-none absolute right-2 bottom-0 h-28 w-auto select-none sm:right-4 sm:h-40"
      />
      <!-- 铁律 #1 sanctioned exception (d): the section header's banner scrim,
           opaque behind the title on the left and fading out to the right so
           the picture shows. Do not remove it in a no-gradient sweep. -->
      <div
        v-if="hasArt"
        class="from-default-50/95 via-default-50/85 to-default-50/60 sm:from-default-50 sm:via-default-50/90 sm:to-default-50/10 absolute inset-0 bg-gradient-to-r"
        aria-hidden="true"
      />

      <div
        :class="
          cn('relative space-y-3 p-4 sm:p-5', mascot && 'sm:min-h-44 sm:pr-48')
        "
      >
        <div class="flex flex-wrap items-center gap-2">
          <h1 class="flex items-center gap-2 text-2xl font-bold">
            <KunIcon
              :name="section.icon_name"
              :class="cn('size-6', ICON_COLOR[section.color])"
            />
            {{ section.label }}
          </h1>
          <KunChip v-if="section.is_archived" size="sm">已归档</KunChip>
          <KunChip
            v-if="section.moemoepoint_on_create < 0"
            size="sm"
            color="warning"
          >
            发布消耗 {{ -section.moemoepoint_on_create }} 萌萌点
          </KunChip>
        </div>

        <div class="flex items-start gap-2">
          <p class="text-default-600 line-clamp-1 flex-1 text-sm">
            {{ section.description }}
          </p>
          <KunButton
            size="xs"
            variant="light"
            color="primary"
            class-name="shrink-0"
            @click="isAboutOpen = true"
          >
            分区介绍
          </KunButton>
        </div>

        <div
          class="text-default-500 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm"
        >
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:newspaper" class="size-4" />
            {{ formatNumber(section.topic_count) }} 个话题
          </span>
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:eye" class="size-4" />
            {{ formatNumber(section.view_count) }} 次浏览
          </span>
          <span
            v-if="section.is_hidden_from_feeds"
            class="flex items-center gap-1"
          >
            <KunIcon name="lucide:eye-off" class="size-4" />
            不出现在主页与全部话题
          </span>
        </div>

        <KunLink
          v-if="section.latest_topic"
          :to="`/topic/${section.latest_topic.id}`"
          underline="none"
          class-name="bg-default-100/80 hover:bg-default-200/80 flex w-fit max-w-full items-center gap-2 rounded-lg px-3 py-1.5 text-sm transition-colors"
        >
          <span class="text-default-500 shrink-0">最新</span>
          <span class="line-clamp-1">{{ section.latest_topic.title }}</span>
          <KunTime
            :time="section.latest_topic.created_at"
            type="relative"
            class="text-default-500 shrink-0 text-xs"
          />
        </KunLink>
      </div>
    </div>

    <KunModal
      v-model="isAboutOpen"
      :aria-label="`${section.label}分区介绍`"
      inner-class-name="max-w-lg"
    >
      <div class="space-y-3">
        <h2 class="flex items-center gap-2 text-xl font-bold">
          <KunIcon
            :name="section.icon_name"
            :class="cn('size-5', ICON_COLOR[section.color])"
          />
          {{ section.label }}
        </h2>
        <p class="text-default-600 text-sm leading-relaxed">
          {{ section.description }}
        </p>
      </div>
    </KunModal>
  </header>
</template>
