<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    item: KunNewsItem
    source: KunNewsSource | undefined
    size?: 'sm' | 'md'
  }>(),
  { size: 'sm' }
)

const isWide = computed(() => props.size === 'md')
const sourceName = computed(() => props.source?.display_name ?? '合作站点')
const detailPath = computed(() =>
  props.item.has_body ? `/news/${props.item.id}` : ''
)
const titleClass = computed(() =>
  cn(
    'hover:text-primary line-clamp-2 wrap-anywhere break-normal transition-colors',
    isWide.value
      ? 'text-base font-semibold sm:text-lg'
      : 'text-sm font-medium sm:text-base'
  )
)
</script>

<template>
  <KunCard is-hoverable padding="md" content-class="gap-0">
    <div class="grid grid-cols-[auto_1fr] gap-y-1.5">
      <div class="col-start-2 row-start-1 flex items-start gap-2">
        <KunChip
          v-if="item.lane === 'column'"
          size="sm"
          color="secondary"
          variant="flat"
          class="mt-0.5 shrink-0"
        >
          专栏
        </KunChip>
        <KunLink
          v-if="detailPath"
          :to="detailPath"
          color="default"
          underline="none"
          :class-name="titleClass"
        >
          {{ item.title }}
        </KunLink>
        <!-- rel="noopener" is load-bearing, not decoration: the default for an
             outbound link is `noopener noreferrer`, and noreferrer strips the
             Referer header, so every click reached the partner's analytics as
             direct traffic. Bringing them referred traffic is the condition they
             gave us. Needs @kungal/ui-vue >= 2.24.0, where rel replaces the
             default instead of adding to it. -->
        <KunLink
          v-else-if="item.source_url"
          :href="item.source_url"
          target="_blank"
          rel="noopener"
          color="default"
          underline="none"
          :class-name="titleClass"
        >
          {{ item.title }}
        </KunLink>
        <span v-else :class="titleClass">{{ item.title }}</span>
      </div>

      <p
        :class="
          cn(
            'text-default-500 col-start-1 col-end-3 row-start-2 text-sm sm:col-start-2',
            isWide ? 'line-clamp-3 sm:leading-6' : 'line-clamp-2'
          )
        "
      >
        {{ item.preview }}
      </p>

      <div
        class="text-default-400 col-start-1 col-end-3 row-start-3 flex flex-wrap items-center gap-x-2 gap-y-1 pt-1 text-xs sm:col-start-2"
      >
        <span>{{ formatTimeDifference(item.published_at) }}</span>
        <span aria-hidden="true">·</span>
        <span class="truncate">{{ sourceName }}</span>
        <template v-if="item.submitter">
          <span aria-hidden="true">·</span>
          <KunUserChip
            :user="toKunUser(item.submitter)"
            size="sm"
            is-navigation
          />
        </template>
        <KunLink
          v-if="detailPath"
          :to="detailPath"
          color="primary"
          size="sm"
          underline="hover"
          class-name="ml-auto shrink-0"
        >
          阅读全文
        </KunLink>
        <KunLink
          v-else-if="item.source_url"
          :href="item.source_url"
          target="_blank"
          rel="noopener"
          color="primary"
          size="sm"
          underline="hover"
          is-show-anchor-icon
          class-name="ml-auto shrink-0"
        >
          阅读原文
        </KunLink>
      </div>
    </div>
  </KunCard>
</template>
