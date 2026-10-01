<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import { KUN_NEWS_SUBMISSION_STATE_MAP } from '~/constants/news'

const api = useApiClient()

const { items, hasMore, loadingMore, loadMore, problem, status } =
  await useCursorList<KunNewsSubmission>(
    'my-news-submissions',
    (client, cursor, { signal }) =>
      client.GET('/me/news-submissions', {
        params: { query: cursor ? { cursor } : {} },
        signal
      })
  )

const busy = ref<Record<string, boolean>>({})

const isEditable = (item: KunNewsSubmission) =>
  item.state === 'pending' || item.state === 'published'

const handleWithdraw = async (item: KunNewsSubmission) => {
  const ok = await useComponentMessageStore().alert(
    '确定撤回这条情报吗？',
    '撤回后它会从 Galgame 情报中移除，并且不能再编辑或重新发布。'
  )
  if (!ok) {
    return
  }
  busy.value = { ...busy.value, [item.id]: true }
  const result = await settle(
    api.PATCH('/me/news-submissions/{news_submission_id}', {
      params: { path: { news_submission_id: item.id } },
      body: { state: 'withdrawn' }
    })
  )
  busy.value = { ...busy.value, [item.id]: false }
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  items.value = items.value.map((row) =>
    row.id === item.id ? result.data : row
  )
  useMessage('已撤回', 'success')
}
</script>

<template>
  <div class="space-y-4">
    <KunHeader
      name="我的情报投稿"
      description="你投稿的 Gal 情报都在这里。待审核的情报由 NextMoe 审核员人工审核，通过后才会出现在 Galgame 情报中。"
    >
      <template #endContent>
        <div class="flex gap-2">
          <KunButton size="sm" href="/edit/news">发布情报</KunButton>
          <KunButton size="sm" variant="flat" href="/news">
            Galgame 情报
          </KunButton>
        </div>
      </template>
    </KunHeader>

    <KunInfo
      v-if="problem"
      color="danger"
      title="加载失败"
      description="暂时无法获取你的投稿，请稍后重试。"
    />

    <div v-else-if="items.length" class="flex flex-col gap-3">
      <KunCard
        v-for="item in items"
        :key="item.id"
        :is-hoverable="false"
        padding="md"
        content-class="gap-2"
      >
        <div class="flex flex-wrap items-center gap-2">
          <KunChip
            size="sm"
            variant="flat"
            :color="KUN_NEWS_SUBMISSION_STATE_MAP[item.state].color"
          >
            {{ KUN_NEWS_SUBMISSION_STATE_MAP[item.state].label }}
          </KunChip>
          <KunLink
            v-if="item.state === 'published'"
            :to="`/news/${item.id}`"
            color="default"
            underline="hover"
            class-name="font-medium wrap-anywhere"
          >
            {{ item.title }}
          </KunLink>
          <span v-else class="font-medium wrap-anywhere">
            {{ item.title }}
          </span>
        </div>

        <div class="flex items-start gap-3">
          <KunImage
            v-if="item.banner"
            :src="withImageVariant(item.banner.url, 'mini')"
            :thumbhash="item.banner.thumbhash ?? undefined"
            :alt="item.title"
            aspect-ratio="16/9"
            object-fit="cover"
            loading="lazy"
            class-name="w-28 shrink-0 overflow-hidden rounded-lg"
          />
          <p class="text-default-500 line-clamp-2 text-sm">
            {{ item.preview }}
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <span class="text-default-400 text-xs">
            {{ formatTimeDifference(item.published_at) }}
          </span>
          <div class="ml-auto flex gap-2">
            <KunButton
              v-if="isEditable(item)"
              size="sm"
              variant="flat"
              :href="`/edit/news/${item.id}`"
            >
              编辑
            </KunButton>
            <KunButton
              v-if="item.state === 'published'"
              size="sm"
              color="danger"
              variant="flat"
              :loading="busy[item.id]"
              :disabled="busy[item.id]"
              @click="handleWithdraw(item)"
            >
              撤回
            </KunButton>
          </div>
        </div>
      </KunCard>
    </div>

    <KunNull
      v-else-if="status !== 'pending'"
      description="你还没有投稿过情报"
    />

    <KunButton
      v-if="hasMore"
      variant="flat"
      :loading="loadingMore"
      @click="loadMore"
    >
      加载更多
    </KunButton>
  </div>
</template>
