<script setup lang="ts">
import type { Work } from '#shared/utils/api/schemas'
import { toKunUser } from '~/utils/userRef'

const galgame = inject<Work>('galgame')
const contributors = computed(() =>
  (galgame?.contributors ?? []).map((user) => toKunUser(user))
)
</script>

<template>
  <div v-if="contributors.length" class="flex flex-wrap items-center gap-1">
    <KunTooltip v-for="user in contributors" :key="user.id" :text="user.name">
      <KunAvatar :user="user" size="sm" />
    </KunTooltip>
  </div>
</template>
