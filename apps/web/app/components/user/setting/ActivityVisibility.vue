<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const api = useApiClient()

const isHidden = ref(false)
const isReady = ref(false)
const isSaving = ref(false)

const load = async () => {
  const result = await settle(api.GET('/me/activity-settings'))
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  isHidden.value = result.data.is_hidden
  isReady.value = true
}

const onToggle = async (value: boolean) => {
  if (isSaving.value) {
    return
  }
  const previous = isHidden.value
  isHidden.value = value
  isSaving.value = true
  const result = await settle(
    api.PUT('/me/activity-settings', { body: { is_hidden: value } })
  )
  isSaving.value = false
  if (!result.ok) {
    isHidden.value = previous
    reportProblem(result.problem)
    return
  }
  isHidden.value = result.data.is_hidden
}

onMounted(load)
</script>

<template>
  <KunCard :is-hoverable="false" content-class="space-y-3">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <span class="text-xl">隐藏我的动态</span>
        <p class="text-default-500 text-sm">
          开启后，你在鲲 Galgame 和 NextMoe
          各站的发布、回复、评论等动态不会出现在任何人的「关注」动态里，关注你的人也不会再收到你的新发布通知，已经发出的这类通知会被撤回。你自己的主页不受影响。
        </p>
        <p class="text-default-500 text-sm">
          关闭后动态会立即恢复显示，但撤回的通知不会回来，隐藏期间的发布也不会补发通知。
        </p>
      </div>
      <KunSwitch
        :model-value="isHidden"
        :disabled="!isReady || isSaving"
        @update:model-value="onToggle"
      />
    </div>
  </KunCard>
</template>
