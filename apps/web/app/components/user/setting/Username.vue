<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const api = useApiClient()
const userStore = usePersistUserStore()

const inputValue = ref('')
const renameCost = useRenameCost()

const renameCostText = computed(() => {
  if (renameCost.value === null) return '改名需要消耗萌萌点。'
  if (renameCost.value === 0) return '改名免费。'
  return `改名需要 ${renameCost.value} 个萌萌点。`
})

const handleChangeUsername = async () => {
  const next = inputValue.value.trim()
  if (!isValidName(next)) {
    useMessage(10122, 'warn')
    return
  }
  if (next === userStore.name) {
    useMessage('新用户名与当前用户名相同', 'warn')
    return
  }

  const result = await settle(
    api.PATCH('/me/profile', {
      body: { name: next }
    })
  )

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(10124, 'success')
  userStore.name = next
  inputValue.value = ''
}
</script>

<template>
  <KunCard :is-hoverable="false" content-class="space-y-3">
    <div>
      <span class="text-xl">更改用户名</span>
      <p class="text-default-500 text-sm">
        用户名为 1~17 位任意字符, 全局唯一。{{ renameCostText }}当前:
        {{ userStore.name }}
      </p>
    </div>

    <KunInput type="text" v-model="inputValue" placeholder="输入新用户名" />

    <div class="flex justify-end">
      <KunButton @click="handleChangeUsername">确定更改</KunButton>
    </div>
  </KunCard>
</template>
