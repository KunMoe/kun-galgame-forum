<script setup lang="ts">
import { reactive } from 'vue'
import { TOPIC_MINI_APPS, type TopicMiniAppKey } from './registry'
import { TOPIC_MINI_APP_SECTIONS } from './sections'
import type { HeldMiniApp } from '~/store/temp/topic/miniAppHandoff'

const props = defineProps<{
  topicId: number
  isTopicAdmin: boolean
}>()

const createOpen = reactive(
  Object.fromEntries(TOPIC_MINI_APPS.map((app) => [app.key, false]))
) as Record<TopicMiniAppKey, boolean>
const createForm = reactive<
  Partial<Record<TopicMiniAppKey, HeldMiniApp['form']>>
>({})

const handoff = useTempTopicMiniAppHandoffStore()
const held = computed(() =>
  props.isTopicAdmin ? handoff.heldFor(String(props.topicId)) : []
)
const retrying = ref<HeldMiniApp | null>(null)

const openCreate = (key: TopicMiniAppKey, item: HeldMiniApp | null = null) => {
  retrying.value = item
  createForm[key] = item?.form
  createOpen[key] = true
}

const onCreated = (key: TopicMiniAppKey) => {
  if (retrying.value?.key === key) {
    handoff.release(retrying.value)
    retrying.value = null
  }
}

const discard = async (item: HeldMiniApp) => {
  const ok = await useComponentMessageStore().alert(
    `确定放弃这个${item.key === 'poll' ? '投票' : '抽奖'}吗？`,
    '放弃后填写的内容不会保留。'
  )
  if (ok) {
    handoff.release(item)
  }
}

const appOf = (key: TopicMiniAppKey) =>
  TOPIC_MINI_APPS.find((app) => app.key === key)!
</script>

<template>
  <div class="space-y-3">
    <KunCard
      v-if="held.length"
      :is-hoverable="false"
      :is-transparent="false"
      content-class="space-y-3"
    >
      <div class="flex items-center gap-2">
        <KunIcon name="lucide:circle-alert" class="text-warning-500" />
        <h2 class="text-sm font-semibold">
          这些投票或抽奖在发布话题时没有创建成功
        </h2>
      </div>
      <p class="text-default-500 text-xs">
        填写的内容还在, 修改后可以重新创建。刷新页面后将不再保留
      </p>
      <div
        v-for="(item, index) in held"
        :key="index"
        class="border-default-200 flex items-center gap-3 rounded-xl border p-3"
      >
        <KunIcon
          :name="appOf(item.key).icon"
          class="text-default-500 shrink-0"
        />
        <span class="min-w-0 flex-1 truncate text-sm">
          {{ appOf(item.key).label }} · {{ item.form.title }}
        </span>
        <KunButton
          size="sm"
          variant="flat"
          :color="appOf(item.key).badgeColor"
          @click="openCreate(item.key, item)"
        >
          重新创建
        </KunButton>
        <KunButton
          size="sm"
          variant="light"
          color="default"
          @click="discard(item)"
        >
          放弃
        </KunButton>
      </div>
    </KunCard>

    <component
      :is="TOPIC_MINI_APP_SECTIONS[app.key]"
      v-for="app in TOPIC_MINI_APPS"
      :key="app.key"
      v-model:is-create-open="createOpen[app.key]"
      :topic-id="topicId"
      :is-topic-admin="isTopicAdmin"
      :create-form="createForm[app.key]"
      @created="onCreated(app.key)"
    />

    <TopicMiniappPanel
      v-if="isTopicAdmin"
      @create="(key: TopicMiniAppKey) => openCreate(key)"
    />
  </div>
</template>
