<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'

type Field = 'is_play_records_public' | 'is_followed_companies_public'

const props = defineProps<{
  field: Field
  anchor: string
  title: string
}>()

const api = useApiClient()

const isPublic = ref(false)
const isReady = ref(false)
const isSaving = ref(false)

const load = async () => {
  const result = await settle(api.GET('/me/privacy-settings'))
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  isPublic.value = result.data[props.field]
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
    api.PATCH('/me/privacy-settings', { body: { [props.field]: value } })
  )
  isSaving.value = false
  if (!result.ok) {
    isPublic.value = previous
    reportProblem(result.problem)
    return
  }
  isPublic.value = result.data[props.field]
}

onMounted(load)
</script>

<template>
  <KunCard :id="anchor" :is-hoverable="false" content-class="space-y-3">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <span class="text-xl">{{ title }}</span>
        <div class="text-default-500 space-y-1 text-sm">
          <slot />
        </div>
      </div>
      <KunSwitch
        :model-value="isPublic"
        :disabled="!isReady || isSaving"
        @update:model-value="onToggle"
      />
    </div>
  </KunCard>
</template>
