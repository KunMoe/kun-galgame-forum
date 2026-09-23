<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import { settle } from '#shared/utils/api/problem'
import type { UserSearchHit } from '#shared/utils/api/schemas'

definePageMeta({
  middleware: 'permission',
  permissions: ['user.purge_content']
})

useKunDisableSeo('用户内容管理')

const oauthUsersAdminURL = computed(
  () => `${useRuntimeConfig().public.oauthAdminUrl}/users`
)

const api = useApiClient()

const searchQuery = ref('')
const users = ref<UserSearchHit[]>([])
const isSearching = ref(false)

const handleSearch = async () => {
  const keywords = searchQuery.value.trim()
  if (!keywords) {
    users.value = []
    return
  }
  isSearching.value = true
  const result = await settle(
    api.GET('/search/users', {
      params: { query: { q: keywords, page: 1, limit: 12 } }
    })
  )
  isSearching.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  users.value = result.data.items
}

watchDebounced(() => searchQuery.value, handleSearch, {
  debounce: 500,
  maxWait: 1000
})
</script>

<template>
  <div>
    <KunHeader
      name="用户内容管理"
      description="此处用于管理用户在本站发布的内容: 搜索用户后, 可一键清除其在 kungal 的全部内容 (话题 / 回复 / 评论 / 评分 / 资源 / 网站 / 工具及一切互动), 主要用于清理广告与 spam 账号。单条内容的编辑与删除请直接在对应页面操作。"
    >
      <template #endContent>
        <KunInput
          v-model="searchQuery"
          type="text"
          placeholder="输入用户名以搜索用户"
        />
      </template>
    </KunHeader>

    <KunInfo
      color="info"
      icon="lucide:shield-alert"
      :title="`封禁 / 注销账号请前往 ${nextmoe.admin}`"
      :description="`账号本身的封禁、解封、注销 (匿名化) 与角色管理由 ${nextmoe.account} 集中处理，在那里操作会对所有站点 (kungal / 摸鱼 / 贴纸…) 同时生效；本页仅用于清理用户在本站发布的内容。`"
      class-name="mt-6"
    >
      <KunButton
        :href="oauthUsersAdminURL"
        target="_blank"
        size="sm"
        color="info"
        class-name="mt-2"
      >
        <KunIcon name="lucide:external-link" />
        前往 {{ nextmoe.admin }} 管理用户
      </KunButton>
    </KunInfo>

    <div class="mt-6 flex flex-col gap-3">
      <AdminUserCard v-for="user in users" :key="user.id" :user="user" />
    </div>

    <KunLoading v-if="isSearching" />
    <KunNull
      v-if="!isSearching && !users.length && searchQuery.trim()"
      description="未找到匹配的用户"
    />
  </div>
</template>
