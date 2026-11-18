<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { EnvVariable } from '#shared/utils/api/schemas'

const props = defineProps<{ variable: EnvVariable | null }>()
const emit = defineEmits<{ saved: [] }>()
const open = defineModel<boolean>({ default: false })

const api = useApiClient()
const value = ref('')
const note = ref('')
const isSaving = ref(false)

watch(open, (isOpen) => {
  if (isOpen) {
    const plain =
      props.variable?.stored?.plain ?? props.variable?.running?.plain
    value.value = plain ?? ''
    note.value = ''
  }
})

const isSecret = computed(() => props.variable?.variable_kind === 'secret')

const save = async () => {
  if (!props.variable || !value.value) return
  isSaving.value = true
  const result = await settle(
    api.PUT('/admin/settings/env/variables/{variable}', {
      params: { path: { variable: props.variable.variable } },
      body: { value: value.value, change_note: note.value || undefined }
    })
  )
  isSaving.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  useMessage('已存入配置中心, 重启 kungal-api 后生效', 'success')
  open.value = false
  emit('saved')
}
</script>

<template>
  <KunModal
    v-model="open"
    :is-dismissable="false"
    aria-label="编辑存储值"
    inner-class-name="max-w-xl"
  >
    <form v-if="variable" class="space-y-4" @submit.prevent="save">
      <div>
        <h2 class="font-mono text-lg font-semibold">
          {{ variable.variable }}
        </h2>
        <p class="text-default-500 text-sm">
          值加密存入配置中心, kungal-api
          下次启动时读取。环境变量里设了同名变量时, 生效的仍是环境变量
        </p>
      </div>
      <KunInput
        v-model="value"
        :type="isSecret ? 'password' : 'text'"
        :label="isSecret ? '新的值 (不会回显已存的密钥)' : '值'"
        required
      />
      <KunInput v-model="note" label="备注 (可选)" :maxlength="200" />
      <div class="flex justify-end gap-2">
        <KunButton variant="light" color="default" @click="open = false">
          取消
        </KunButton>
        <KunButton type="submit" :loading="isSaving" :disabled="!value">
          保存
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
