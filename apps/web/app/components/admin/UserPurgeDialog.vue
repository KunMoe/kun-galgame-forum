<script setup lang="ts">
import type { UserContent } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  userId: string
  name: string
  content: UserContent
  breakdown: string
}>()

const emit = defineEmits<{
  purged: []
}>()

const isOpen = defineModel<boolean>({ required: true })

const api = useApiClient()
const confirmId = ref('')
const isSubmitting = ref(false)

const oauthUsersAdminURL = computed(
  () => `${useRuntimeConfig().public.oauthAdminUrl}/users`
)

const isConfirmed = computed(
  () => confirmId.value.trim() === props.userId
)

watch(isOpen, (open) => {
  if (open) {
    confirmId.value = ''
  }
})

const handlePurge = async () => {
  if (!isConfirmed.value) {
    return
  }
  isSubmitting.value = true
  const result = await settle(
    api.DELETE('/admin/user-contents/{user_id}', {
      params: { path: { user_id: props.userId } }
    })
  )
  isSubmitting.value = false

  if (!result.ok) {
    if (result.problem.code === 'LOTTERY_DRAWN') {
      useMessage('该用户有抽奖正在开奖, 请稍后再试, 本次没有删除任何内容', 'warn')
      return
    }
    if (result.problem.code === 'SERVICE_UNAVAILABLE') {
      useMessage(
        '清除没有全部完成 (账号中心、社区或收藏夹暂时不可用), 请稍后重试, 重试是安全的',
        'error'
      )
      return
    }
    reportProblem(result.problem)
    return
  }
  isOpen.value = false
  emit('purged')
}
</script>

<template>
  <KunModal v-model="isOpen" inner-class-name="max-w-lg w-[94vw]">
    <div class="space-y-4">
      <div class="space-y-1">
        <span class="text-xl">清除用户 {{ name }} 的全部内容</span>
        <p class="text-default-500 text-sm">
          将删除该用户在本站的 {{ content.total_count }} 项内容:
          {{ breakdown }}, 以及挂在这些内容下的他人回复、私聊会话与关联数据。收录的网站会转交给站点, 不会删除。
        </p>
      </div>

      <KunInfo
        v-if="content.is_account_active"
        color="danger"
        icon="lucide:triangle-alert"
        title="该账号仍然可以登录和发帖"
        :description="`请先在 ${nextmoe.admin} 封禁或注销该账号, 再清除内容。否则清除之后, 这个账号还能继续发布新的内容。`"
      >
        <KunButton
          :href="oauthUsersAdminURL"
          target="_blank"
          size="sm"
          color="danger"
          class-name="mt-2"
        >
          <KunIcon name="lucide:external-link" />
          前往 {{ nextmoe.admin }}
        </KunButton>
      </KunInfo>
      <KunInfo
        v-else
        color="info"
        icon="lucide:user-x"
        title="该账号已封禁、已注销或不存在"
        description="清除之后该账号无法再产生新的内容。"
      />

      <p class="text-default-500 text-sm">
        本站被删除的内容会存档 30 天, 期间可由开发者恢复; 社区评论与收藏夹的清除无法恢复。
      </p>

      <KunInput
        v-model="confirmId"
        type="text"
        :label="`请输入用户 ID ${userId} 以确认`"
        :placeholder="userId"
        autocomplete="off"
      />

      <div class="flex justify-end gap-2">
        <KunButton variant="light" color="default" @click="isOpen = false">
          取消
        </KunButton>
        <KunButton
          color="danger"
          :loading="isSubmitting"
          :disabled="!isConfirmed || isSubmitting"
          @click="handlePurge"
        >
          确认清除
        </KunButton>
      </div>
    </div>
  </KunModal>
</template>
