<script setup lang="ts">
import type { CatalogIntro } from '#shared/utils/api/schemas'
import { galgameImageSourceLabel } from '~/constants/galgameImageSource'

interface IntroRow {
  lang: string
  intro: string
}

const props = defineProps<{
  modelValue: unknown
  disabled?: boolean
  upstream?: CatalogIntro[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: unknown]
}>()

// catalog speaks BCP-47-ish tags; the editor speaks the forum's locale keys.
const LANGS = [
  { value: 'zh-Hans', textValue: '简体中文', locale: 'zh-cn' },
  { value: 'zh-Hant', textValue: '繁体中文', locale: 'zh-tw' },
  { value: 'ja', textValue: '日本語', locale: 'ja-jp' },
  { value: 'en', textValue: 'English', locale: 'en-us' }
] as const

const rows = computed<IntroRow[]>(() =>
  Array.isArray(props.modelValue) ? (props.modelValue as IntroRow[]) : []
)

const ownTextOf = (lang: string) =>
  rows.value.find((row) => row.lang === lang)?.intro

// The intro the work page shows for a language when it is not this forum's
// own hand-written one. Catalog keeps it under its source; writing the
// language here adds this forum's version, which then wins on display.
const shownElsewhere = (lang: string) =>
  props.upstream?.find(
    (i) => i.locale === lang && !(i.data_source === 'curated' && !i.is_machine)
  )

const textOf = (lang: string) =>
  ownTextOf(lang) ?? shownElsewhere(lang)?.value ?? ''

const isPrefilled = (lang: string) =>
  ownTextOf(lang) === undefined && Boolean(shownElsewhere(lang))

const active = ref<string>(
  LANGS.find((l) => textOf(l.value).trim())?.value ?? 'zh-Hans'
)

const activeLocale = computed<Language>(
  () =>
    (LANGS.find((l) => l.value === active.value)?.locale ?? 'zh-cn') as Language
)

const items = computed(() =>
  LANGS.map((lang) => ({
    value: lang.value,
    textValue: lang.textValue,
    own: ownTextOf(lang.value) !== undefined,
    prefilled: isPrefilled(lang.value)
  }))
)

const activeSource = computed(() => {
  const intro = shownElsewhere(active.value)
  if (!intro || !isPrefilled(active.value)) {
    return null
  }
  return {
    label: galgameImageSourceLabel(intro.data_source ?? ''),
    isMachine: intro.is_machine
  }
})

const setText = (lang: string, text: string) => {
  const kept = new Map(rows.value.map((row) => [row.lang, row.intro]))
  const elsewhere = shownElsewhere(lang)?.value.trim()
  const isUnchangedPrefill =
    !kept.has(lang) && elsewhere !== undefined && text.trim() === elsewhere
  if (text.trim() && !isUnchangedPrefill) {
    kept.set(lang, text)
  } else {
    kept.delete(lang)
  }
  emit(
    'update:modelValue',
    LANGS.filter((l) => kept.has(l.value)).map((l) => ({
      lang: l.value,
      intro: kept.get(l.value) ?? ''
    }))
  )
}

const clearActive = () => setText(active.value, '')

const ownCount = computed(() => items.value.filter((i) => i.own).length)
const prefilledCount = computed(
  () => items.value.filter((i) => i.prefilled).length
)
</script>

<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <KunTab
        :model-value="active"
        :items="items"
        orientation="horizontal"
        variant="underlined"
        color="primary"
        size="sm"
        @update:model-value="(value) => (active = value)"
      >
        <template #tab="{ item }">
          {{ item.textValue }}
          <KunBadge v-if="item.own" variant="dot" color="success" size="sm" />
          <KunBadge
            v-else-if="item.prefilled"
            variant="dot"
            color="default"
            size="sm"
          />
        </template>
      </KunTab>

      <span class="text-default-400 text-xs">
        本站版本 {{ ownCount }} 种
        <template v-if="prefilledCount">
          · 数据源 {{ prefilledCount }} 种
        </template>
      </span>
    </div>

    <p v-if="activeSource" class="text-default-500 text-xs">
      当前显示的是 {{ activeSource.label
      }}{{ activeSource.isMachine ? '（机器翻译）' : '' }}
      的介绍。直接修改并提交后, 将以本站版本覆盖显示, 原文仍会保留;
      之后清空本站版本即可恢复显示原文。不修改则不会提交。
      <template v-if="activeSource.isMachine">
        修改后的文本会作为人工介绍保存, 不再随每日机器翻译更新。
      </template>
    </p>

    <KunMilkdownDualEditorProvider
      v-if="!disabled"
      :key="active"
      :value-markdown="textOf(active)"
      :language="activeLocale"
      :disable-image="true"
      placeholder="支持 Markdown。可参考官方页面、DLsite 的 ストーリー、Bangumi，也可以自行撰写"
      @set-markdown="(text: string) => setText(active, text)"
    >
      <KunButton
        v-if="ownTextOf(active) !== undefined"
        variant="light"
        color="danger"
        size="sm"
        @click="clearActive"
      >
        {{
          shownElsewhere(active) ? '清空本站版本, 恢复数据源介绍' : '清空该语言'
        }}
      </KunButton>
    </KunMilkdownDualEditorProvider>

    <p v-else class="text-default-500 text-sm whitespace-pre-wrap">
      {{ textOf(active) || '（空）' }}
    </p>
  </div>
</template>
