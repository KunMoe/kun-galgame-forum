<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import type { KunTabItem } from '@kungal/ui-vue'
import { topicHiddenByMeta } from '~/constants/topic'
import { settle } from '#shared/utils/api/problem'
import type { AdminTopic, HiddenTopicSummary } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

definePageMeta({
  middleware: 'permission',
  permissions: ['topic.view_hidden']
})

useKunDisableSeo('隐藏话题管理')

type PurgeCountKey =
  | 'reply_count'
  | 'comment_count'
  | 'poll_count'
  | 'lottery_count'
  | 'drawn_lottery_count'
  | 'favorite_count'

const PURGE_COUNT_LABELS: { key: PurgeCountKey; label: string }[] = [
  { key: 'reply_count', label: '回复' },
  { key: 'comment_count', label: '评论' },
  { key: 'poll_count', label: '投票' },
  { key: 'lottery_count', label: '抽奖' },
  { key: 'drawn_lottery_count', label: '已开奖抽奖' },
  { key: 'favorite_count', label: '收藏' }
]

const HIDDEN_BY_TABS: KunTabItem[] = [
  { value: 'all', textValue: '全部' },
  { value: 'author', textValue: '作者隐藏' },
  { value: 'moderator', textValue: '管理员隐藏' },
  { value: 'trust', textValue: '风纪隐藏' }
]

type HiddenBy = NonNullable<HiddenTopicSummary['hidden_by']>

const canDeleteTopic = useCan('topic.delete_any')
const api = useApiClient()

const activeFilter = ref('all')
const searchQuery = ref('')

const pageData = reactive({
  page: 1,
  limit: 30,
  hidden_by: '' as HiddenBy | '',
  q: ''
})

watch(activeFilter, (value) => {
  pageData.hidden_by = value === 'all' ? '' : (value as HiddenBy)
  pageData.page = 1
})

watchDebounced(
  searchQuery,
  (value) => {
    pageData.q = value.trim()
    pageData.page = 1
  },
  { debounce: 500, maxWait: 1000 }
)

const { data, status, refresh } = await useApi(
  () =>
    `admin-hidden-topics:${pageData.page}:${pageData.limit}:${pageData.hidden_by}:${pageData.q}`,
  (client, { signal }) =>
    client.GET('/admin/hidden-topics', {
      params: {
        query: {
          page: pageData.page,
          limit: pageData.limit,
          ...(pageData.hidden_by ? { hidden_by: pageData.hidden_by } : {}),
          ...(pageData.q ? { q: pageData.q } : {})
        }
      },
      signal
    })
)

const topics = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total ?? 0)

const isPurgeOpen = ref(false)
const isLoadingStats = ref(false)
const isPurging = ref(false)
const purgeTarget = ref<AdminTopic | null>(null)

const purgeTotal = computed(() => {
  const target = purgeTarget.value
  if (!target) {
    return 0
  }
  // drawn_lottery_count is a subset of lottery_count; summing both counts them twice
  return PURGE_COUNT_LABELS.reduce(
    (sum, item) =>
      item.key === 'drawn_lottery_count' ? sum : sum + target[item.key],
    0
  )
})

const openPurge = async (topic: HiddenTopicSummary) => {
  isPurgeOpen.value = true
  isLoadingStats.value = true
  purgeTarget.value = null
  const result = await settle(
    api.GET('/admin/topics/{topic_id}', {
      params: { path: { topic_id: topic.id } }
    })
  )
  isLoadingStats.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    isPurgeOpen.value = false
    return
  }
  purgeTarget.value = result.data
}

const handlePurge = async () => {
  const target = purgeTarget.value
  if (!target) {
    return
  }
  isPurging.value = true
  const result = await settle(
    api.DELETE('/admin/topics/{topic_id}', {
      params: { path: { topic_id: target.id } }
    })
  )
  isPurging.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  isPurgeOpen.value = false
  purgeTarget.value = null
  useMessage(`已彻底删除话题《${target.title}》`, 'success')
  await refresh()
}
</script>

<template>
  <div class="w-full space-y-4">
    <KunHeader
      name="隐藏话题管理"
      description="此处汇总全站被隐藏的话题, 包含作者自行隐藏、管理员隐藏与风纪处置隐藏三种来源。隐藏只是让话题从列表与搜索中消失, 作者与持有查看权限的管理人员仍可访问; 若要连同回复、评论、投票、抽奖与收藏一并抹除, 请使用彻底删除。"
    />

    <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
      <KunTab
        v-model="activeFilter"
        :items="HIDDEN_BY_TABS"
        variant="underlined"
        color="primary"
        size="sm"
        class-name="min-w-0 flex-1"
      />
      <!-- KunInput's root keeps w-full regardless of class-name (it lands on
           the inner input), so an unwrapped KunInput takes the whole flex row
           and crushes the KunTab beside it to zero width. -->
      <div class="w-full shrink-0 sm:w-72">
        <KunInput
          v-model="searchQuery"
          type="text"
          placeholder="输入话题标题以搜索"
        />
      </div>
    </div>

    <KunDivider />

    <KunLoading v-if="status === 'pending'" />

    <KunNull v-else-if="!topics.length" description="暂无被隐藏的话题" />

    <div v-else class="flex flex-col gap-3">
      <div
        v-for="topic in topics"
        :key="topic.id"
        class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 sm:flex-row sm:items-start"
      >
        <div class="min-w-0 flex-1 space-y-2">
          <div class="flex flex-wrap items-center gap-2">
            <KunLink :to="`/topic/${topic.id}`" target="_blank">
              {{ topic.title }}
            </KunLink>
            <KunChip
              size="xs"
              variant="flat"
              :color="topicHiddenByMeta(topic.hidden_by ?? '').color"
            >
              {{ topicHiddenByMeta(topic.hidden_by ?? '').label }}
            </KunChip>
          </div>

          <UserHoverCard :user-id="topic.author.id">
            <KunUserChip :user="toKunUser(topic.author)" size="xs" />
          </UserHoverCard>

          <div
            class="text-default-500 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm"
          >
            <span class="flex items-center gap-1">
              <KunIcon name="carbon:reply" class="size-4" />
              {{ topic.reply_count }}
            </span>
            <span class="flex items-center gap-1">
              <KunIcon name="lucide:eye-off" class="size-4" />
              隐藏于
              <KunTime :time="topic.bumped_at" type="datetime" show-year />
            </span>
            <span class="flex items-center gap-1">
              <KunIcon name="lucide:clock" class="size-4" />
              发布于
              <KunTime :time="topic.created_at" type="date" show-year />
            </span>
          </div>
        </div>

        <KunButton
          v-if="canDeleteTopic"
          size="sm"
          color="danger"
          variant="flat"
          class-name="shrink-0"
          @click="openPurge(topic)"
        >
          <KunIcon name="lucide:trash-2" />
          彻底删除
        </KunButton>
      </div>
    </div>

    <KunPagination
      v-if="total > pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(total / pageData.limit)"
      :is-loading="status === 'pending'"
    />

    <KunModal
      v-model="isPurgeOpen"
      role="alertdialog"
      title="彻底删除这个话题"
      description="删除后无法恢复, 话题及其下的全部内容都会从数据库中抹除, 已发放的萌萌点不会追回。"
      inner-class-name="w-full max-w-lg"
    >
      <KunLoading v-if="isLoadingStats" />

      <div v-else-if="purgeTarget" class="space-y-4">
        <KunInfo
          color="danger"
          variant="flat"
          icon="lucide:triangle-alert"
          title="这是不可撤销的操作"
          description="这不是隐藏, 也不是软删除。确认后话题连同下列内容一起从数据库消失, 没有任何恢复手段。"
        />

        <div class="space-y-1">
          <p class="font-medium break-words">{{ purgeTarget.title }}</p>
          <UserHoverCard :user-id="purgeTarget.author.id">
            <KunUserChip :user="toKunUser(purgeTarget.author)" size="xs" />
          </UserHoverCard>
        </div>

        <div class="flex flex-wrap gap-2 text-sm">
          <KunChip
            v-for="item in PURGE_COUNT_LABELS"
            :key="item.key"
            size="sm"
            variant="flat"
            :color="purgeTarget[item.key] ? 'danger' : 'default'"
          >
            {{ item.label }} {{ purgeTarget[item.key] }}
          </KunChip>
        </div>

        <p class="text-default-500 text-sm">
          共 {{ purgeTotal }} 项关联数据将一并删除。
        </p>

        <p
          v-if="purgeTarget.open_lottery_escrow > 0"
          class="text-default-500 text-sm"
        >
          进行中的抽奖托管着
          {{ purgeTarget.open_lottery_escrow }} 萌萌点, 删除时会退回给抽奖发起人。
        </p>

        <div class="flex justify-end gap-2">
          <KunButton
            variant="light"
            :disabled="isPurging"
            @click="isPurgeOpen = false"
          >
            取消
          </KunButton>
          <KunButton
            color="danger"
            :loading="isPurging"
            :disabled="isPurging"
            @click="handlePurge"
          >
            确认彻底删除
          </KunButton>
        </div>
      </div>
    </KunModal>
  </div>
</template>
