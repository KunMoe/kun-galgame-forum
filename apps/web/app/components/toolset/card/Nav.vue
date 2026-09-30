<script setup lang="ts">
import { watchDebounced } from '@vueuse/core'
import {
  kunGalgameToolsetTypeOptions,
  kunGalgameToolsetLanguageOptions,
  kunGalgameToolsetPlatformOptions,
  kunGalgameToolsetVersionOptions,
  kunGalgameToolsetSortFieldOptions,
  KUN_GALGAME_TOOLSET_TYPE_MAP,
  KUN_GALGAME_TOOLSET_LANGUAGE_MAP,
  KUN_GALGAME_TOOLSET_PLATFORM_MAP,
  KUN_GALGAME_TOOLSET_VERSION_MAP,
  KUN_GALGAME_TOOLSET_SORT_FIELD_MAP
} from '~/constants/toolset'

const { type, platform, version, sortField, sortOrder, query } =
  useToolsetFilters()

const keyword = ref(query.value)
watchDebounced(
  keyword,
  (value) => {
    query.value = value.trim()
  },
  { debounce: 250 }
)
watch(query, (value) => {
  if (value !== keyword.value.trim()) {
    keyword.value = value
  }
})
</script>

<template>
  <KunInput
    v-model="keyword"
    type="text"
    placeholder="模糊名字查询"
    class-name="col-span-2 lg:col-span-1"
  />
  <div
    class="flex w-full shrink-0 flex-wrap items-center justify-between gap-3 rounded-lg border border-transparent sm:flex-nowrap"
  >
    <div class="grid w-full grid-cols-2 gap-3 lg:grid-cols-4">
      <KunSelect
        :model-value="type"
        :options="kunGalgameToolsetTypeOptions"
        @set="(newVal) => (type = newVal as typeof type)"
      >
        {{ KUN_GALGAME_TOOLSET_TYPE_MAP[type] || type }}
      </KunSelect>

      <KunSelect
        :model-value="platform"
        :options="kunGalgameToolsetPlatformOptions"
        @set="(newVal) => (platform = newVal as typeof platform)"
      >
        {{ KUN_GALGAME_TOOLSET_PLATFORM_MAP[platform] || platform }}
      </KunSelect>

      <KunSelect
        :model-value="version"
        :options="kunGalgameToolsetVersionOptions"
        @set="(newVal) => (version = newVal as typeof version)"
      >
        {{ KUN_GALGAME_TOOLSET_VERSION_MAP[version] || version }}
      </KunSelect>

      <KunSelect
        :model-value="sortField"
        :options="kunGalgameToolsetSortFieldOptions"
        @set="(value) => (sortField = value as 'created')"
      >
        <span>{{ KUN_GALGAME_TOOLSET_SORT_FIELD_MAP[sortField] }}</span>
      </KunSelect>
    </div>

    <div class="flex items-center gap-2">
      <KunButton
        :is-icon-only="true"
        :variant="sortOrder === 'desc' ? 'flat' : 'light'"
        size="lg"
        @click="sortOrder = 'desc'"
      >
        <KunIcon class="text-inherit" name="lucide:arrow-down" />
      </KunButton>

      <KunButton
        :is-icon-only="true"
        :variant="sortOrder === 'asc' ? 'flat' : 'light'"
        size="lg"
        @click="sortOrder = 'asc'"
      >
        <KunIcon class="text-inherit" name="lucide:arrow-up" />
      </KunButton>
    </div>
  </div>
</template>
