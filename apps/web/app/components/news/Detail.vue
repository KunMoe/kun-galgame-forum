<script setup lang="ts">
const props = defineProps<{
  item: KunNewsItemDetail
  source: KunNewsSource | undefined
}>()

const sourceName = computed(() => props.source?.display_name ?? '合作站点')
const hasBody = computed(() => props.item.content.children.length > 0)
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-4">
    <KunLink to="/news" color="default" size="sm" underline="hover">
      <KunIcon name="lucide:arrow-left" class="mr-1" />
      返回 Galgame 情报
    </KunLink>

    <KunCard :is-hoverable="false" padding="lg" content-class="gap-6">
      <header class="space-y-3">
        <div class="flex flex-wrap items-center gap-2">
          <KunChip
            v-if="item.lane === 'column'"
            size="sm"
            color="secondary"
            variant="flat"
          >
            专栏
          </KunChip>
          <KunChip size="sm" variant="flat">{{ sourceName }}</KunChip>
        </div>

        <h1
          class="text-2xl font-bold tracking-tight break-normal wrap-anywhere sm:text-3xl"
        >
          {{ item.title }}
        </h1>

        <div
          class="text-default-500 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm"
        >
          <UserHoverCard v-if="item.submitter" :user-id="item.submitter.id">
            <KunUserChip
              :user="toKunUser(item.submitter)"
              size="sm"
              is-navigation
            />
          </UserHoverCard>
          <span class="flex items-center gap-1">
            <KunIcon name="lucide:calendar-days" />
            <KunTime :time="item.published_at" type="datetime" show-year />
          </span>
        </div>
      </header>

      <p
        class="text-default-600 border-primary border-l-2 pl-4 leading-7 whitespace-pre-line"
      >
        {{ item.preview }}
      </p>

      <ContentDocument v-if="hasBody" :document="item.content" />

      <div
        v-if="item.source_url || source?.attribution"
        class="border-default-200 space-y-2 border-t pt-4"
      >
        <!-- rel="noopener" keeps the Referer, which partners count as the
             click-through they asked for; see news/Card.vue. -->
        <KunLink
          v-if="item.source_url"
          :href="item.source_url"
          target="_blank"
          rel="noopener"
          color="primary"
          size="sm"
          underline="hover"
          is-show-anchor-icon
        >
          {{ hasBody ? '原文链接' : '阅读原文' }}
        </KunLink>
        <p v-if="source?.attribution" class="text-default-400 text-xs">
          {{ source.attribution }}
        </p>
      </div>
    </KunCard>
  </div>
</template>
