<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const api = useApiClient()

const isPublic = ref(true)
const isReady = ref(false)
const isSaving = ref(false)

const load = async () => {
  const result = await settle(api.GET('/me/privacy-settings'))
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  isPublic.value = result.data.is_play_records_public
  isReady.value = true
}

const onToggle = async (value: boolean) => {
  if (isSaving.value) {
    return
  }
  const previous = isPublic.value
  isPublic.value = value
  isSaving.value = true
  const result = await settle(
    api.PATCH('/me/privacy-settings', {
      body: { is_play_records_public: value }
    })
  )
  isSaving.value = false
  if (!result.ok) {
    isPublic.value = previous
    reportProblem(result.problem)
    return
  }
  isPublic.value = result.data.is_play_records_public
}

onMounted(load)
</script>

<template>
  <KunCard id="play-records" :is-hoverable="false" content-class="space-y-3">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <span class="text-xl">公开我的游玩记录</span>
        <p class="text-default-500 text-sm">
          你在鲲 Galgame 和 NextMoe
          各站标记的游玩状态（想玩、在玩、玩过、搁置、抛弃）和游玩时长，会显示在你个人主页的「游戏库」里，所有人都能看到。
        </p>
        <p class="text-default-500 text-sm">
          关闭后只有你自己能看到。评分一直是公开的，不受这个开关影响。
        </p>
      </div>
      <KunSwitch
        :model-value="isPublic"
        :disabled="!isReady || isSaving"
        @update:model-value="onToggle"
      />
    </div>
  </KunCard>
</template>
