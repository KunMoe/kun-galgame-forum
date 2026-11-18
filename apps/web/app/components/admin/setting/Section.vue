<script setup lang="ts">
import type { ClientProblem } from '#shared/utils/api/problem'
import { problemMessage } from '#shared/utils/api/message'

defineProps<{
  title: string
  description: string
  problem: ClientProblem | null
  pending: boolean
  isReady: boolean
  isSaving: boolean
}>()

const emit = defineEmits<{ save: [] }>()
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-5"
  >
    <div>
      <h2 class="text-lg font-semibold">{{ title }}</h2>
      <p class="text-default-500 text-sm">{{ description }}</p>
    </div>

    <KunNull v-if="problem" :description="problemMessage(problem)" />
    <template v-else-if="isReady">
      <slot />
      <div class="flex justify-end">
        <KunButton :loading="isSaving" @click="emit('save')">保存</KunButton>
      </div>
    </template>
    <div v-else-if="pending" class="grid gap-4 sm:grid-cols-2">
      <KunSkeleton v-for="n in 2" :key="n" height="3.5rem" rounded="lg" />
    </div>
  </KunCard>
</template>
