<script setup lang="ts">
import type { LetmoeResourceDownload } from '#shared/utils/api/schemas'
import { formatFileSize } from '#shared/utils/format'
import { resourceLanguageLabel } from '~~/shared/utils/galgameResourceVocab'
import {
  SUPPORTED_LANGUAGE_MAP,
  SUPPORTED_TYPE_MAP
} from '~/components/galgame/patch/constant'

type Mirror = LetmoeResourceDownload['mirrors'][number]
type Composite = LetmoeResourceDownload['composites'][number]

const props = defineProps<{
  hosted: Mirror[]
  composites: Composite[]
  baseLanguages: string[]
  baseSize: string
}>()

// letmoe's own order: a human translation first, then machine, then the rest.
const TYPE_RANK = [
  'manual',
  'machine_polishing',
  'machine',
  'fix',
  'mod',
  'save'
]
const rankOf = (c: Composite) =>
  Math.min(
    TYPE_RANK.length,
    ...c.patch_types.map((t) => {
      const i = TYPE_RANK.indexOf(t)
      return i < 0 ? TYPE_RANK.length : i
    })
  )
const isTranslation = (c: Composite) =>
  c.patch_types.some((t) => TYPE_RANK.slice(0, 3).includes(t))

const sorted = computed(() =>
  [...props.composites].sort(
    (a, b) => rankOf(a) - rankOf(b) || Number(a.id) - Number(b.id)
  )
)

const joinLabels = (tokens: string[], map: (t: string) => string) =>
  tokens.map(map).join(' / ')
const baseLanguageText = computed(() =>
  joinLabels(props.baseLanguages, resourceLanguageLabel)
)
const patchLanguageText = (c: Composite) =>
  joinLabels(c.patch_languages, (t) => SUPPORTED_LANGUAGE_MAP[t] ?? t)

const options = computed(() => [
  ...props.hosted.map((m, i) => ({
    value: `m${m.id}`,
    label: `原版${baseLanguageText.value ? ` · ${baseLanguageText.value}` : ''}${props.hosted.length > 1 ? ` #${i + 1}` : ''}`,
    description: ['不含补丁', props.baseSize, m.note ?? '']
      .filter(Boolean)
      .join(' · ')
  })),
  ...sorted.value.map((c) => ({
    value: `c${c.id}`,
    label: `${patchLanguageText(c) || '补丁'}合成版 · ${c.patch_name || '补丁'}`,
    description: [
      joinLabels(c.patch_types, (t) => SUPPORTED_TYPE_MAP[t] ?? t),
      formatFileSize(c.bytes),
      c.patch_publisher && `补丁作者 ${c.patch_publisher}`
    ]
      .filter(Boolean)
      .join(' · ')
  }))
])

const defaultChoice = () => {
  const best = sorted.value.find(isTranslation)
  return best ? `c${best.id}` : (options.value[0]?.value ?? '')
}
const selected = ref(defaultChoice())

const currentComposite = computed(() =>
  sorted.value.find((c) => `c${c.id}` === selected.value)
)
const currentMirror = computed(() =>
  props.hosted.find((m) => `m${m.id}` === selected.value)
)
const currentUrl = computed(
  () => currentComposite.value?.url ?? currentMirror.value?.url ?? ''
)
const currentSize = computed(() =>
  currentComposite.value
    ? formatFileSize(currentComposite.value.bytes)
    : props.baseSize
)
</script>

<template>
  <div class="space-y-3">
    <KunRadioGroup
      v-if="options.length > 1"
      v-model="selected"
      :options="options"
      variant="card"
      aria-label="选择直链下载的版本"
    />

    <KunInfo
      :color="currentComposite ? 'success' : 'primary'"
      variant="flat"
      :title="
        currentComposite
          ? `本体已打好「${currentComposite.patch_name}」，解压即可游玩`
          : '游戏原版，不含任何补丁'
      "
    >
      <div class="space-y-3">
        <p v-if="currentComposite" class="text-sm">
          补丁归原作者所有, 来自
          <KunLink
            size="sm"
            target="_blank"
            :to="`https://www.moyu.moe/resource/${currentComposite.moyu_resource_id}`"
          >
            鲲 Galgame 补丁
          </KunLink>
          。合成版打不开时可以下载原版, 再自行打补丁。
        </p>
        <div class="flex flex-wrap items-center gap-2">
          <KunButton
            :href="currentUrl"
            target="_blank"
            rel="noopener noreferrer"
            :color="currentComposite ? 'success' : 'primary'"
            variant="solid"
            :disabled="!currentUrl"
          >
            <KunIcon name="lucide:download" />
            直链下载{{ currentSize ? `（${currentSize}）` : '' }}
          </KunButton>
          <KunCopy
            v-if="currentUrl"
            :text="currentUrl"
            name="复制链接"
            size="sm"
            variant="flat"
            color="default"
          />
        </div>
      </div>
    </KunInfo>
  </div>
</template>
