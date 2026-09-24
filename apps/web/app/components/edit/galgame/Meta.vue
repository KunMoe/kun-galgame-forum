<script setup lang="ts">
import { kunGalgameOriginalLanguageOptions } from '~/constants/galgame'

const { age_limit, original_language, release_date, release_date_tba } =
  storeToRefs(usePersistEditGalgameStore())

const ageLimitOptions = [
  { value: 'all', label: '全年龄 (本游戏不含成人内容)' },
  { value: 'r18', label: 'R18 (本游戏含成人内容)' }
] as const

// Catalog keeps a closed set of original languages and has none for "others".
const originalLanguageOptions = kunGalgameOriginalLanguageOptions.filter(
  (option) => option.value !== 'others'
)
</script>

<template>
  <div class="space-y-2">
    <h2 class="text-xl">游戏基础信息</h2>
    <p class="text-default-500 text-sm">
      默认假设是日语原版、含成人内容的视觉小说。如果游戏与默认不同，请在此调整。
    </p>
    <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
      <KunSelect
        v-model="age_limit"
        label="年龄分级"
        :options="ageLimitOptions"
      />
      <KunSelect
        v-model="original_language"
        label="原始语言"
        :options="originalLanguageOptions"
      />
    </div>
    <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
      <KunInput v-model="release_date" type="date" label="发售日期 (可留空)" />
      <KunSwitch v-model="release_date_tba" label="发售日期待定 (TBA)" />
    </div>
  </div>
</template>
