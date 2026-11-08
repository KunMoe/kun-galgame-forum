<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type { CollectionChoice } from '#shared/utils/api/schemas'

const props = defineProps<{
  modelValue: boolean
  workId: number
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [payload: { favorited: boolean }]
}>()

const { name: myName } = usePersistUserStore()
const api = useApiClient()
const { isBlurred } = useContentStance()

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const collections = ref<CollectionChoice[]>([])
const selected = ref<Set<string>>(new Set())
const held = ref<Set<string>>(new Set())
const notes = ref<Record<string, string>>({})
const savedNotes = ref<Record<string, string>>({})
const noteErrors = ref<Record<string, string>>({})
const pending = ref(false)
const loaded = ref(false)
const saving = ref(false)
const createOpen = ref(false)

const applyList = (items: CollectionChoice[], preserveDrafts: boolean) => {
  collections.value = items
  held.value = new Set(items.filter((c) => c.viewer?.has_work).map((c) => c.id))
  const stored: Record<string, string> = {}
  for (const c of items) {
    stored[c.id] = c.work_note ?? ''
  }
  savedNotes.value = stored
  if (!preserveDrafts) {
    selected.value = new Set(held.value)
    notes.value = { ...stored }
    noteErrors.value = {}
    return
  }
  notes.value = { ...stored, ...notes.value }
}

const load = async (preserveDrafts = false) => {
  // A failed picker read once saved collection_ids: [] and stripped the work
  // from every folder. Save stays off until a list read succeeds.
  if (!preserveDrafts) {
    loaded.value = false
  }
  pending.value = !preserveDrafts
  const result = await settle(
    api.GET('/me/works/{work_id}/collections', {
      params: {
        path: { work_id: String(props.workId) },
        query: { limit: 100 }
      }
    })
  )
  pending.value = false
  if (!result.ok) {
    loaded.value = false
    collections.value = []
    reportProblem(result.problem)
    return
  }
  loaded.value = true
  applyList(result.data.items, preserveDrafts)
}

watch(
  () => [isOpen.value, props.workId] as const,
  ([open, workId]) => {
    if (open && workId > 0) {
      createOpen.value = false
      void load()
    }
  }
)

const toggle = (id: string) => {
  const next = new Set(selected.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  selected.value = next
}

const onCreated = async (newId?: string) => {
  await load(true)
  if (newId) {
    const next = new Set(selected.value)
    next.add(newId)
    selected.value = next
  }
}

const coverOf = (c: CollectionChoice) => {
  const work = c.recent_work
  if (!work) {
    return ''
  }
  const art = work.is_nsfw && isBlurred.value ? work.cover_sfw : work.cover
  return art?.url ?? ''
}

const itemParams = (collectionId: string) => ({
  params: {
    path: { collection_id: collectionId, work_id: String(props.workId) }
  }
})

const saveNote = async (collectionId: string) => {
  const result = await settle(
    api.PATCH('/collections/{collection_id}/works/{work_id}', {
      ...itemParams(collectionId),
      body: { note: notes.value[collectionId] ?? '' }
    })
  )
  if (result.ok) {
    return true
  }
  const message = collectionFieldErrors(result.problem, '/note')['/note']
  if (message) {
    noteErrors.value = { ...noteErrors.value, [collectionId]: message }
  } else {
    reportProblem(result.problem)
  }
  return false
}

const save = async () => {
  if (!loaded.value) {
    return
  }
  saving.value = true
  noteErrors.value = {}
  const memberships = collections.value.filter(
    (c) => selected.value.has(c.id) !== held.value.has(c.id)
  )
  const results = await Promise.all(
    memberships.map(async (c) => {
      const result = await settle(
        selected.value.has(c.id)
          ? api.PUT(
              '/collections/{collection_id}/works/{work_id}',
              itemParams(c.id)
            )
          : api.DELETE(
              '/collections/{collection_id}/works/{work_id}',
              itemParams(c.id)
            )
      )
      return { id: c.id, result }
    })
  )
  const failed = results.find((r) => !r.result.ok)
  const added = new Set(
    results
      .filter((r) => r.result.ok && selected.value.has(r.id))
      .map((r) => r.id)
  )
  const noteTargets = collections.value.filter((c) => {
    if (!selected.value.has(c.id)) {
      return false
    }
    const draft = (notes.value[c.id] ?? '').trim()
    if (added.has(c.id)) {
      return draft !== ''
    }
    return held.value.has(c.id) && draft !== (savedNotes.value[c.id] ?? '')
  })
  const noteResults = await Promise.all(noteTargets.map((c) => saveNote(c.id)))
  saving.value = false

  if (failed && !failed.result.ok) {
    reportProblem(failed.result.problem)
  }
  if (failed || noteResults.includes(false)) {
    await load(true)
    if (loaded.value) {
      emits('saved', { favorited: held.value.size > 0 })
    }
    return
  }
  useMessage(10569, 'success')
  emits('saved', { favorited: selected.value.size > 0 })
  isOpen.value = false
}
</script>

<template>
  <KunModal
    v-model="isOpen"
    inner-class-name="max-w-2xl w-full"
    title="收藏到收藏夹"
  >
    <div class="space-y-4">
      <div class="flex items-center justify-between gap-3">
        <p class="text-default-500 text-sm">
          可以同时放进多个收藏夹，选中后还能写下这部作品的备注
        </p>
        <KunButton
          variant="flat"
          size="sm"
          class-name="shrink-0"
          @click="createOpen = true"
        >
          <KunIcon name="lucide:plus" />
          新建收藏夹
        </KunButton>
      </div>

      <div v-if="pending" class="space-y-2" aria-hidden="true">
        <KunSkeleton
          v-for="n in 3"
          :key="n"
          variant="rect"
          height="4.5rem"
          class-name="rounded-lg"
        />
      </div>

      <div v-else class="max-h-[60vh] space-y-2 overflow-y-auto pr-1">
        <div
          v-for="c in collections"
          :key="c.id"
          :class="
            cn(
              'rounded-lg border transition-colors',
              selected.has(c.id)
                ? 'border-primary/50 bg-primary/5'
                : 'border-default-200 hover:bg-default-100'
            )
          "
        >
          <div class="flex items-center gap-3 p-2">
            <button
              type="button"
              class="flex min-w-0 flex-1 items-center gap-3 text-left"
              :aria-pressed="selected.has(c.id)"
              @click="toggle(c.id)"
            >
              <span
                :class="
                  cn(
                    'flex size-5 shrink-0 items-center justify-center rounded border transition-colors',
                    selected.has(c.id)
                      ? 'border-primary bg-primary text-white'
                      : 'border-default-300'
                  )
                "
              >
                <KunIcon v-if="selected.has(c.id)" name="lucide:check" />
              </span>
              <span
                class="bg-default-100 flex h-14 w-10 shrink-0 items-center justify-center overflow-hidden rounded"
              >
                <img
                  v-if="coverOf(c)"
                  :src="coverOf(c)"
                  alt=""
                  class="size-full object-cover"
                />
                <KunIcon
                  v-else
                  name="lucide:folder-heart"
                  class="text-default-300"
                />
              </span>
              <span class="min-w-0 flex-1 space-y-0.5">
                <span class="flex items-center gap-1.5">
                  <span class="truncate font-medium">
                    {{ collectionDisplayName(c, myName) }}
                  </span>
                  <KunIcon
                    :name="collectionVisibilityMeta(c.visibility).icon"
                    class="text-default-400 shrink-0 text-xs"
                  />
                  <span
                    v-if="c.is_default"
                    class="text-default-400 shrink-0 text-xs"
                  >
                    默认
                  </span>
                </span>
                <span
                  v-if="c.description"
                  class="text-default-500 line-clamp-1 block text-xs"
                >
                  {{ c.description }}
                </span>
                <span class="text-default-400 block text-xs">
                  {{ c.item_count }} 个游戏
                  <template v-if="held.has(c.id)"> · 已收藏</template>
                </span>
              </span>
            </button>

            <KunTooltip text="查看收藏夹">
              <KunLink
                :to="`/collection/${c.id}`"
                target="_blank"
                color="default"
                underline="none"
                class-name="text-default-400 hover:text-primary flex shrink-0 items-center rounded-md p-1.5"
              >
                <KunIcon name="lucide:external-link" />
              </KunLink>
            </KunTooltip>
          </div>

          <div v-if="selected.has(c.id)" class="px-2 pb-2">
            <KunTextarea
              v-model="notes[c.id]"
              :placeholder="`为什么把它放进「${collectionDisplayName(c, myName)}」? (可选)`"
              :rows="2"
              auto-grow
              :maxlength="COLLECTION_NOTE_MAX"
              show-char-count
              :error="noteErrors[c.id] ?? ''"
            />
          </div>
        </div>

        <KunNull v-if="!loaded" description="收藏夹读取失败，请稍后重试" />
        <KunNull
          v-else-if="!collections.length"
          description="还没有收藏夹，点击新建一个吧"
        />
      </div>

      <div class="flex items-center justify-between gap-3">
        <span class="text-default-500 text-sm">
          已选 {{ selected.size }} 个收藏夹
        </span>
        <div class="flex gap-3">
          <KunButton variant="light" color="danger" @click="isOpen = false">
            取消
          </KunButton>
          <KunButton
            color="primary"
            :loading="saving"
            :disabled="!loaded"
            @click="save"
          >
            保存
          </KunButton>
        </div>
      </div>
    </div>
  </KunModal>

  <GalgameCollectionEditModal
    v-model="createOpen"
    mode="create"
    :is-default="loaded && collections.length === 0"
    @saved="onCreated"
  />
</template>
