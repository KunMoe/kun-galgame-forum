<script setup lang="ts">
const userStore = usePersistUserStore()

const inputValue = ref('')

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

  const result = await kunFetch('/user/username', {
    method: 'PUT',
    body: { username: next }
  })

  if (result) {
    useMessage(10124, 'success')
    userStore.name = next
    inputValue.value = ''
  }
}
</script>

<template>
  <KunCard :is-hoverable="false" content-class="space-y-3">
    <div>
      <span class="text-xl">更改用户名</span>
      <p class="text-default-500 text-sm">
        用户名为 1~17 位任意字符, 全局唯一。改名需要 17 个萌萌点。当前:
        {{ userStore.name }}
      </p>
    </div>

    <KunInput type="text" v-model="inputValue" placeholder="输入新用户名" />

    <div class="flex justify-end">
      <KunButton @click="handleChangeUsername">确定更改</KunButton>
    </div>
  </KunCard>
</template>
