<script setup lang="ts">
import type { UserCounts, UserProfile } from '#shared/utils/api/schemas'

const props = defineProps<{
  user: UserProfile
}>()
const user = computed(() => props.user)

const statsBlocks: { key: keyof UserCounts; label: string }[] = [
  { key: 'topic_count', label: '话题' },
  { key: 'poll_count', label: '话题投票' },
  { key: 'lottery_count', label: '话题抽奖' },
  { key: 'reply_count', label: '回复' },
  { key: 'topic_comment_count', label: '话题评论' },
  { key: 'published_galgame_count', label: 'Galgame' },
  { key: 'contributed_galgame_count', label: 'Galgame 贡献' },
  { key: 'community_comment_count', label: '评论' },
  { key: 'galgame_rating_count', label: 'Galgame 评分' },
  { key: 'galgame_resource_count', label: 'Galgame 资源' },
  { key: 'toolset_count', label: 'Galgame 工具' },
  { key: 'toolset_resource_count', label: 'Galgame 工具资源' }
]

const interactionBlocks: {
  key: keyof UserCounts
  label: string
  icon: string
  color: string
}[] = [
  {
    key: 'received_upvote_count',
    label: '被推',
    icon: 'lucide:sparkles',
    color: 'text-secondary'
  },
  {
    key: 'received_like_count',
    label: '被赞',
    icon: 'lucide:thumbs-up',
    color: 'text-primary'
  },
  {
    key: 'received_dislike_count',
    label: '被踩',
    icon: 'lucide:thumbs-down',
    color: 'text-default'
  }
]

const displayCount = (value: number | null) => (value === null ? '—' : value)

const infoList = [
  { label: '注册序号', value: (u: UserProfile) => u.id },
  {
    label: '今日发布话题',
    value: (u: UserProfile) => u.counts.topic_today_count
  },
  {
    label: '今日发布 Galgame',
    value: (u: UserProfile) => u.counts.published_galgame_today_count
  }
]
</script>

<template>
  <div v-if="user" class="w-full space-y-4">
    <KunCard :is-hoverable="false">
      <h3 class="text-default-500 mb-4 text-sm font-medium">内容统计</h3>
      <div
        class="grid grid-cols-3 gap-x-3 gap-y-5 sm:grid-cols-4 lg:grid-cols-6"
      >
        <div v-for="block in statsBlocks" :key="block.key">
          <div class="text-primary text-xl font-bold">
            {{ displayCount(user.counts[block.key]) }}
          </div>
          <div class="text-default-500 text-xs">{{ block.label }}</div>
        </div>
      </div>
    </KunCard>

    <div class="grid gap-4 md:grid-cols-2">
      <KunCard :is-hoverable="false">
        <h3 class="text-default-500 mb-4 text-sm font-medium">互动</h3>
        <div class="grid grid-cols-3 gap-3">
          <div
            v-for="block in interactionBlocks"
            :key="block.key"
            class="flex items-center gap-2"
          >
            <KunIcon
              :name="block.icon"
              class="shrink-0 text-2xl"
              :class="block.color"
            />
            <div class="min-w-0">
              <div class="text-lg font-semibold">
                {{ displayCount(user.counts[block.key]) }}
              </div>
              <div class="text-default-500 text-xs">{{ block.label }}</div>
            </div>
          </div>
        </div>
      </KunCard>

      <KunCard :is-hoverable="false">
        <h3 class="text-default-500 mb-2 text-sm font-medium">资料</h3>
        <div class="divide-default-200/60 divide-y">
          <div
            v-for="item in infoList"
            :key="item.label"
            class="flex items-center justify-between py-2 text-sm"
          >
            <span class="text-default-600">{{ item.label }}</span>
            <span class="font-medium">{{ item.value(user) }}</span>
          </div>
          <div class="flex items-center justify-between py-2 text-sm">
            <span class="text-default-600">注册时间</span>
            <span class="font-medium">
              <KunTime :time="user.created_at" type="datetime" show-year />
            </span>
          </div>
        </div>
      </KunCard>
    </div>

    <KunCard :is-hoverable="false">
      <h3 class="text-default-500 mb-2 text-sm font-medium">签名</h3>
      <p
        v-if="user.bio"
        class="text-default-700 text-sm break-words whitespace-pre-wrap"
      >
        {{ user.bio }}
      </p>
      <KunNull v-else :is-show-sticker="false" />
    </KunCard>
  </div>
</template>
