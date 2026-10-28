<script setup lang="ts">
import { ENTITY_MENU_TYPES } from '~/composables/useEntityEditorAdapters'
import { ENTITY_KINDS } from '~/utils/entityTarget'
import {
  entityUrl,
  type EntityLinkRef,
  type EntityRefType
} from '~/utils/entityUrl'

const { state, finish } = useEntityPicker()
const { searchIds, ratingIdsOfWork } = useEntitySearch()

const isOpen = computed({
  get: () => state.value.isOpen,
  set: (open) => {
    if (!open) {
      finish(null)
    }
  }
})

const type = computed({
  get: () => state.value.type,
  set: (next: EntityRefType) => {
    state.value.type = next
  }
})
const typeTabs = ENTITY_MENU_TYPES.map((t) => ({
  value: t,
  textValue: ENTITY_KINDS[t].noun.replace('Galgame ', '')
}))

const query = ref('')
const ratingWork = ref<EntityLinkRef | null>(null)
const results = ref<EntityLinkRef[]>([])
const status = ref<'idle' | 'loading' | 'done'>('idle')
const active = ref(0)

const searchType = computed<EntityRefType>(() =>
  type.value === 'rating' && !ratingWork.value ? 'work' : type.value
)
const noun = computed(() => ENTITY_KINDS[type.value].noun)
const placeholder = computed(() =>
  type.value === 'rating' && !ratingWork.value
    ? '先搜索要引用评分的 Galgame'
    : `搜索${noun.value}`
)

let controller: AbortController | null = null
let timer: ReturnType<typeof setTimeout> | undefined

const run = async () => {
  controller?.abort()
  controller = new AbortController()
  const { signal } = controller
  const q = query.value.trim()
  if (ratingWork.value) {
    status.value = 'loading'
    const ids = await ratingIdsOfWork(ratingWork.value.id, signal)
    if (signal.aborted) return
    results.value = ids.map((id) => ({
      type: 'rating',
      id,
      url: entityUrl('rating', id)
    }))
  } else if (!q) {
    results.value = []
    status.value = 'idle'
    return
  } else {
    status.value = 'loading'
    const kind = searchType.value
    const ids = await searchIds(kind, q, signal)
    if (signal.aborted) return
    results.value = ids.map((id) => ({
      type: kind,
      id,
      url: entityUrl(kind, id)
    }))
  }
  active.value = 0
  status.value = 'done'
}

const schedule = () => {
  clearTimeout(timer)
  timer = setTimeout(run, 250)
}

watch(query, schedule)
watch(type, () => {
  ratingWork.value = null
  query.value = ''
  results.value = []
  status.value = 'idle'
})
watch(
  () => state.value.isOpen,
  (open) => {
    if (open) {
      query.value = ''
      ratingWork.value = null
      results.value = []
      status.value = 'idle'
    } else {
      controller?.abort()
      clearTimeout(timer)
    }
  }
)
onBeforeUnmount(() => {
  controller?.abort()
  clearTimeout(timer)
})

const choose = (ref: EntityLinkRef) => {
  if (type.value === 'rating' && ref.type === 'work') {
    ratingWork.value = ref
    void run()
    return
  }
  finish(ref)
}

const onKeydown = (e: KeyboardEvent) => {
  const n = results.value.length
  if (!n) return
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    active.value = (active.value + 1) % n
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    active.value = (active.value - 1 + n) % n
  } else if (e.key === 'Enter' && !e.isComposing) {
    e.preventDefault()
    choose(results.value[active.value]!)
  }
}

const listRef = ref<HTMLElement | null>(null)
watch(active, (i) => {
  listRef.value
    ?.querySelector<HTMLElement>(`[data-index="${i}"]`)
    ?.scrollIntoView({ block: 'nearest' })
})
</script>

<template>
  <KunModal
    v-model="isOpen"
    size="lg"
    initial-focus="input"
    :return-focus="false"
    :title="`插入${noun}卡片`"
    inner-class-name="w-[96vw] max-w-2xl"
  >
    <div class="flex flex-col gap-3" @keydown="onKeydown">
      <KunTab
        v-model="type"
        :items="typeTabs"
        variant="underlined"
        size="sm"
        class-name="overflow-x-auto scrollbar-hide"
      />

      <div v-if="ratingWork" class="flex items-center gap-2">
        <KunButton size="sm" variant="light" @click="ratingWork = null">
          <KunIcon name="lucide:arrow-left" />
          重新选择作品
        </KunButton>
        <span class="text-default-500 text-sm">选择要插入的评分</span>
      </div>
      <div v-else>
        <KunInput
          v-model="query"
          is-clearable
          :placeholder="placeholder"
          aria-controls="kun-entity-picker-results"
          :aria-activedescendant="
            results.length ? `kun-entity-option-${active}` : undefined
          "
        />
      </div>

      <div
        id="kun-entity-picker-results"
        ref="listRef"
        role="listbox"
        :aria-label="`${noun}搜索结果`"
        class="scrollbar-hide flex h-[min(56dvh,520px)] flex-col gap-2 overflow-y-auto p-1"
      >
        <template v-if="status === 'loading'">
          <div
            v-for="i in 3"
            :key="i"
            class="bg-default-100 h-36 animate-pulse rounded-lg"
          />
        </template>
        <template v-else-if="results.length">
          <div
            v-for="(item, index) in results"
            :id="`kun-entity-option-${index}`"
            :key="`${item.type}:${item.id}`"
            :data-index="index"
            role="option"
            :aria-selected="index === active"
            :class="
              cn(
                'cursor-pointer rounded-lg outline-offset-2 transition-shadow',
                index === active && 'outline-primary outline-2'
              )
            "
            @mouseenter="active = index"
            @click="choose(item)"
          >
            <KunMilkdownEntityCardView
              :entity="item"
              :selected="index === active"
            />
          </div>
        </template>
        <p
          v-else-if="status === 'done'"
          class="text-default-500 py-10 text-center text-sm"
        >
          {{ ratingWork ? '这部作品还没有评分' : `没有找到相关的${noun}` }}
        </p>
        <p v-else class="text-default-500 py-10 text-center text-sm">
          输入关键词开始搜索
        </p>
      </div>

      <div
        class="text-default-500 flex flex-wrap items-center justify-between gap-2 text-xs"
      >
        <span>
          {{ state.display === 'card' ? '将以卡片插入' : '将以行内链接插入' }}
        </span>
        <span class="hidden items-center gap-1 sm:flex">
          <KunKbd>↑</KunKbd><KunKbd>↓</KunKbd> 选择 <KunKbd>Enter</KunKbd> 插入
          <KunKbd>Esc</KunKbd> 关闭
        </span>
      </div>
    </div>
  </KunModal>
</template>
