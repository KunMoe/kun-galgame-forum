<script setup lang="ts">
import type { UserContent, UserSearchHit } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'
import { toKunUser } from '~/utils/userRef'

const props = defineProps<{
  user: UserSearchHit
}>()

const api = useApiClient()
const kunUser = computed(() => toKunUser(props.user))

type CountKey = {
  [K in keyof UserContent]: K extends `${string}_count` ? K : never
}[keyof UserContent]

const STAT_LABELS: { key: Exclude<CountKey, 'total_count'>; label: string }[] =
  [
    { key: 'topic_count', label: '话题' },
    { key: 'reply_count', label: '回复' },
    { key: 'topic_comment_count', label: '话题评论' },
    { key: 'rating_count', label: '评分' },
    { key: 'resource_count', label: '资源' },
    { key: 'website_count', label: '收录网站 (转交保留)' },
    { key: 'toolset_count', label: '工具' },
    { key: 'toolset_resource_count', label: '工具资源' },
    { key: 'community_post_count', label: '社区评论' },
    { key: 'poll_count', label: '投票' },
    { key: 'lottery_count', label: '抽奖' },
    { key: 'quiz_count', label: '题目' },
    { key: 'collection_count', label: '收藏夹' },
    { key: 'draft_count', label: '草稿' },
    { key: 'todo_count', label: '待办' },
    { key: 'chat_message_count', label: '私聊消息' },
    { key: 'message_count', label: '通知消息' },
    { key: 'interaction_count', label: '互动' }
  ]

const content = ref<UserContent | null>(null)
const isLoading = ref(false)
const purgedTotal = ref<number | null>(null)

const isPermOpen = ref(false)
const isPurgeOpen = ref(false)

const countText = (value: number | null) => (value === null ? '?' : value)

const breakdown = computed(() => {
  const c = content.value
  if (!c) {
    return ''
  }
  return STAT_LABELS.filter((item) => c[item.key])
    .map((item) => `${item.label} ${c[item.key]}`)
    .join(' / ')
})

const loadContent = async () => {
  isLoading.value = true
  const result = await settle(
    api.GET('/admin/user-contents/{user_id}', {
      params: { path: { user_id: props.user.id } }
    })
  )
  isLoading.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  content.value = result.data
}

const handlePurged = async () => {
  const total = content.value?.total_count ?? 0
  purgedTotal.value = total
  useMessage(`已清除用户 ${kunUser.value.name} 的 ${total} 项内容`, 'success')
  await loadContent()
}
</script>

<template>
  <div
    class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3"
  >
    <div class="flex items-center justify-between gap-3">
      <UserHoverCard :user-id="kunUser.id">
        <KunUserChip :user="kunUser" />
      </UserHoverCard>

      <div class="flex shrink-0 items-center gap-2">
        <KunButton size="sm" variant="flat" @click="isPermOpen = true">
          <KunIcon name="lucide:shield-check" />
          权限调整
        </KunButton>
        <KunButton
          v-if="!content"
          size="sm"
          variant="flat"
          @click="loadContent"
          :loading="isLoading"
          :disabled="isLoading"
        >
          查看内容
        </KunButton>
      </div>
    </div>

    <AdminPermissionUserPanel v-model="isPermOpen" :user="kunUser" />

    <div v-if="user.bio" class="text-default-500 line-clamp-2 text-sm">
      {{ user.bio }}
    </div>

    <template v-if="content">
      <div class="flex flex-wrap gap-2 text-sm">
        <KunChip
          v-for="item in STAT_LABELS"
          :key="item.key"
          size="sm"
          variant="flat"
          :color="content[item.key] ? 'primary' : 'default'"
        >
          {{ item.label }} {{ countText(content[item.key]) }}
        </KunChip>
      </div>

      <div class="flex items-center justify-between gap-3">
        <span class="text-default-700 text-sm">
          {{
            purgedTotal !== null
              ? `已清除 ${purgedTotal} 项内容`
              : content.is_protected
                ? '该用户是版主或管理员, 不能清除其内容'
                : `共 ${content.total_count} 项内容`
          }}
        </span>

        <KunButton
          color="danger"
          @click="isPurgeOpen = true"
          :loading="isLoading"
          :disabled="
            isLoading ||
            content.is_protected ||
            (!content.total_count && !content.community_post_count)
          "
        >
          一键清除全部内容
        </KunButton>
      </div>

      <AdminUserPurgeDialog
        v-model="isPurgeOpen"
        :user-id="user.id"
        :name="kunUser.name"
        :content="content"
        :breakdown="breakdown"
        @purged="handlePurged"
      />
    </template>
  </div>
</template>
