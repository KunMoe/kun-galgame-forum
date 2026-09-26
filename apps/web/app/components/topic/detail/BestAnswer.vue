<script setup lang="ts">
import { scrollPage } from '../_helper'
import type { Reply } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

defineProps<{
  bestAnswer: Reply
}>()
</script>

<template>
  <div
    :class="
      cn(
        'bg-success-500/20 text-success-700 dark:text-success-300 cursor-pointer',
        'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold'
      )
    "
    @click="scrollPage(bestAnswer.floor)"
  >
    <KunUserChip :user="toKunUser(bestAnswer.author)" />
    {{ `在 #${bestAnswer.floor} 发布了最佳答案` }}
  </div>
</template>
