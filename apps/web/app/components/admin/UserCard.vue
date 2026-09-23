<script setup lang="ts">
const props = defineProps<{
  user: SearchResultUser
}>()

const STAT_LABELS: { key: keyof AdminUserContentStats; label: string }[] = [
  { key: 'topics', label: '话题' },
  { key: 'replies', label: '回复' },
  { key: 'topic_comments', label: '话题评论' },
  { key: 'ratings', label: '评分' },
  { key: 'resources', label: '资源' },
  { key: 'websites', label: '收录网站 (转交保留)' },
  { key: 'toolsets', label: '工具' },
  { key: 'toolset_resources', label: '工具资源' },
  { key: 'community_posts', label: '社区评论' },
  { key: 'polls', label: '投票' },
  { key: 'lotteries', label: '抽奖' },
  { key: 'quizzes', label: '题目' },
  { key: 'collections', label: '收藏夹' },
  { key: 'drafts', label: '草稿' },
  { key: 'todos', label: '待办' },
  { key: 'chat_messages', label: '私聊消息' },
  { key: 'messages', label: '通知消息' },
  { key: 'interactions', label: '互动' }
]

const stats = ref<AdminUserContentStats | null>(null)
const isLoading = ref(false)
const purged = ref(false)

const isPermOpen = ref(false)

const loadStats = async () => {
  isLoading.value = true
  stats.value =
    (await kunFetch<AdminUserContentStats>(
      `/admin/user/${props.user.id}/content-stats`
    )) ?? null
  isLoading.value = false
}

const handlePurge = async () => {
  if (!stats.value || !stats.value.total) {
    return
  }
  const s = stats.value
  const breakdown = STAT_LABELS.filter((item) => s[item.key])
    .map((item) => `${item.label} ${s[item.key]}`)
    .join(' / ')
  const confirmed = await useComponentMessageStore().alert(
    `确认清除用户 ${props.user.name} 的全部内容吗`,
    `🚨 将永久删除该用户在本站的 ${s.total} 项内容: ${breakdown} (含其下全部嵌套回复、私聊会话与关联数据)。此操作不可撤销, 仅用于清理广告与 spam 账号, 请谨慎使用!`
  )
  if (!confirmed) {
    return
  }

  isLoading.value = true
  const deleted = await kunFetch<AdminUserContentStats>(
    `/admin/user/${props.user.id}/content`,
    { method: 'DELETE' }
  )
  isLoading.value = false

  if (deleted) {
    stats.value = deleted
    purged.value = true
    useMessage(
      `已清除用户 ${props.user.name} 的 ${deleted.total} 项内容`,
      'success'
    )
  }
}
</script>

<template>
  <div
    class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3"
  >
    <div class="flex items-center justify-between gap-3">
      <KunUserChip :user="user" />

      <div class="flex shrink-0 items-center gap-2">
        <KunButton size="sm" variant="flat" @click="isPermOpen = true">
          <KunIcon name="lucide:shield-check" />
          权限调整
        </KunButton>
        <KunButton
          v-if="!stats"
          size="sm"
          variant="flat"
          @click="loadStats"
          :loading="isLoading"
          :disabled="isLoading"
        >
          查看内容
        </KunButton>
      </div>
    </div>

    <AdminPermissionUserPanel v-model="isPermOpen" :user="user" />

    <div v-if="user.bio" class="text-default-500 line-clamp-2 text-sm">
      {{ user.bio }}
    </div>

    <template v-if="stats">
      <div class="flex flex-wrap gap-2 text-sm">
        <KunChip
          v-for="item in STAT_LABELS"
          :key="item.key"
          size="sm"
          variant="flat"
          :color="stats[item.key] ? 'primary' : 'default'"
        >
          {{ item.label }} {{ stats[item.key] }}
        </KunChip>
      </div>

      <div class="flex items-center justify-between">
        <span class="text-default-700 text-sm">
          {{
            purged
              ? `已清除 ${stats.total} 项内容（社区评论 ${stats.community_posts_purged ?? 0} / 互动 ${stats.community_reactions_deleted ?? 0}）`
              : `共 ${stats.total} 项内容`
          }}
        </span>

        <KunButton
          color="danger"
          @click="handlePurge"
          :loading="isLoading"
          :disabled="isLoading || purged || !stats.total"
        >
          一键清除全部内容
        </KunButton>
      </div>
    </template>
  </div>
</template>
