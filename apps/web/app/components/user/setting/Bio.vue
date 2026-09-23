<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{
  user: UserInfo
}>()

const api = useApiClient()

const bioValue = ref('')
const user = computed(() => props.user)

const handleChangeBio = async () => {
  if (!bioValue.value.trim()) {
    return
  }

  if (bioValue.value.length > 107) {
    useMessage(10116, 'warn')
    return
  }

  const result = await settle(
    api.PATCH('/me/profile', {
      body: { bio: bioValue.value }
    })
  )

  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage(10117, 'success')
  bioValue.value = ''
}

onMounted(() => {
  bioValue.value = user.value.bio
})
</script>

<template>
  <KunCard :is-hoverable="false" content-class="space-y-3">
    <div>
      <span class="text-xl">更改签名</span>
      <p class="text-default-500 text-sm">
        签名的长度在 107 字之内, 签名暂时只能使用纯文本 (但是保留换行符),
        以后会加强的呜呜呜, 我一定不会咕咕咕的, 嗯
      </p>
    </div>

    <KunTextarea
      name="bio"
      placeholder="输入您的新签名，最大 107 个字符"
      :rows="5"
      v-model="bioValue"
    >
    </KunTextarea>

    <div class="flex justify-end">
      <KunButton @click="handleChangeBio">确定更改</KunButton>
    </div>
  </KunCard>
</template>
