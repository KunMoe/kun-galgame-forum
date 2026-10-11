<script setup lang="ts">
import type { KunTabItem } from '@kungal/ui-vue'
import { getGalgameIntroLanguageName } from '~/constants/galgame'
import type { CatalogIntro } from '#shared/utils/api/schemas'
import { orderCatalogIntros } from '#shared/utils/catalogName'
import { galgameImageSourceLabel } from '~/constants/galgameImageSource'

const props = defineProps<{
  intros: CatalogIntro[]
}>()

const ordered = computed(() => orderCatalogIntros(props.intros))

const tabs = computed<KunTabItem[]>(() =>
  ordered.value.map((intro) => ({
    textValue: getGalgameIntroLanguageName(intro.locale),
    value: intro.locale
  }))
)

const language = ref(ordered.value[0]?.locale ?? '')

watch(ordered, (rows) => {
  if (!rows.some((row) => row.locale === language.value)) {
    language.value = rows[0]?.locale ?? ''
  }
})

const current = computed(() =>
  props.intros.find((intro) => intro.locale === language.value)
)

const currentSource = computed(() => {
  const source = current.value?.data_source
  if (!source) {
    return ''
  }
  return source === 'curated' ? '本站编辑' : galgameImageSourceLabel(source)
})

const currentText = computed(() =>
  markdownToText(current.value?.value ?? '', { preserveNewlines: true })
)
</script>

<template>
  <div class="space-y-3">
    <KunHeader
      name="游戏介绍"
      description="介绍来自 VNDB、DLsite、Bangumi 等数据源或本站编辑, 都可以在「编辑资料」中修改"
      scale="h2"
    >
      <template #endContent>
        <KunTab
          v-if="tabs.length > 1"
          :items="tabs"
          v-model="language"
          size="sm"
          variant="underlined"
        />
      </template>
    </KunHeader>

    <div v-if="!current" class="bg-primary/20 text-primary rounded-lg p-3">
      暂无简介, 欢迎贡献
    </div>

    <template v-else>
      <div
        v-if="current.is_machine || currentSource"
        class="text-default-500 text-sm"
      >
        <span v-if="currentSource">来源: {{ currentSource }}</span>
        <span v-if="current.is_machine">
          {{ currentSource ? ' · ' : '' }}本段简介由机器翻译生成,
          与原文可能有出入
        </span>
      </div>

      <p class="text-default-700 pt-3 whitespace-pre-line">
        {{ currentText }}
      </p>
    </template>
  </div>
</template>
