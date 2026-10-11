<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const api = useApiClient()

const isSignInRequired = ref(false)
const isReady = ref(false)
const isSaving = ref(false)

const load = async () => {
  const result = await settle(api.GET('/me/profile-settings'))
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  isSignInRequired.value = result.data.is_sign_in_required
  isReady.value = true
}

const onToggle = async (value: boolean) => {
  if (isSaving.value) {
    return
  }
  const previous = isSignInRequired.value
  isSignInRequired.value = value
  isSaving.value = true
  const result = await settle(
    api.PUT('/me/profile-settings', { body: { is_sign_in_required: value } })
  )
  isSaving.value = false
  if (!result.ok) {
    isSignInRequired.value = previous
    reportProblem(result.problem)
    return
  }
  isSignInRequired.value = result.data.is_sign_in_required
}

onMounted(load)
</script>

<template>
  <KunCard :is-hoverable="false" content-class="space-y-3">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <span class="text-xl">主页仅登录用户可见</span>
        <p class="text-default-500 text-sm">
          开启后，未登录的访客打开你的主页和主页里的话题、回复、资源、评分等列表时，会看到需要登录的提示。登录的用户不受影响，你在各处发布的内容也照常显示。
        </p>
      </div>
      <KunSwitch
        :model-value="isSignInRequired"
        :disabled="!isReady || isSaving"
        @update:model-value="onToggle"
      />
    </div>
  </KunCard>
</template>
