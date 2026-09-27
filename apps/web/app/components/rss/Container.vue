<script setup lang="ts">
import { TOPIC_SECTION_OPTIONS } from '~/constants/topic'

const config = useRuntimeConfig()
const baseUrl = computed(() => config.public.KUN_GALGAME_URL || '')
const includeNsfw = ref(false)
const section = ref('')
const lane = ref('')

const nsfwParams = computed(() =>
  includeNsfw.value ? { include_nsfw: true as const } : {}
)
const topicParams = computed(() => ({
  ...nsfwParams.value,
  ...(section.value ? { section: section.value } : {})
}))
const newsParams = computed(() => (lane.value ? { lane: lane.value } : {}))

const sectionOptions = [
  { value: '', label: '全部分区' },
  ...TOPIC_SECTION_OPTIONS
]

const laneOptions = [
  { value: '', label: '全部' },
  { value: 'news', label: '情报' },
  { value: 'column', label: '专栏' }
]
</script>

<template>
  <div class="min-h-[calc(100dvh-6rem)] space-y-6">
    <KunHeader
      name="RSS 订阅"
      description="订阅鲲 Galgame 论坛的最新内容。所有订阅源每 5 分钟更新一次, 支持 RSS 2.0, Atom 与 JSON Feed 三种格式"
    />

    <KunSwitch
      v-model="includeNsfw"
      label="包含 NSFW 内容"
      description="默认只包含全年龄内容。开启后, 支持的订阅源会加上 include_nsfw=1"
    />

    <RssFeedCard
      title="话题"
      description="论坛最新发布的话题"
      path="/rss/topic"
      :params="topicParams"
    >
      <div class="max-w-xs">
        <KunSelect v-model="section" :options="sectionOptions" />
      </div>
    </RssFeedCard>

    <RssFeedCard
      title="Galgame 新资源"
      description="每条对应一份新发布的下载资源, 同一部作品出了新资源也会推送"
      path="/rss/galgame"
      :params="nsfwParams"
    />

    <RssFeedCard
      title="Gal 情报"
      description="论坛转载的 Galgame 情报与专栏"
      path="/rss/news"
      :params="newsParams"
    >
      <div class="max-w-xs">
        <KunSelect v-model="lane" :options="laneOptions" />
      </div>
    </RssFeedCard>

    <RssFeedCard
      title="新工具"
      description="论坛最新收录的 Galgame 工具"
      path="/rss/toolset"
    />

    <KunCard :is-hoverable="false">
      <div class="space-y-3">
        <h2 class="text-xl">单部 Galgame 的资源</h2>
        <p class="text-default-500">
          在 Galgame 页面的「Galgame 资源链接」区域点击 RSS
          按钮即可复制这部作品的订阅地址, 地址格式为
          {{ baseUrl }}/rss/galgame/{作品 ID}.xml。已经有资源的作品才能订阅。
        </p>
      </div>
    </KunCard>

    <KunCard :is-hoverable="false">
      <div class="space-y-3">
        <h2 class="text-xl">用户订阅</h2>
        <p class="text-default-500">
          {{ baseUrl }}/rss/user/{用户 ID}/topic.xml 订阅某位用户发布的话题,
          {{ baseUrl }}/rss/user/{用户 ID}/galgame.xml 订阅某位用户发布的
          Galgame 资源。
        </p>
      </div>
    </KunCard>
  </div>
</template>
