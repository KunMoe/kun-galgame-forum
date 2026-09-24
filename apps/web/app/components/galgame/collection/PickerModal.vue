<script setup lang="ts">
import { settle } from '#shared/utils/api/problem'
import type {
  CollectionSummary,
  CollectionVisibility
} from '#shared/utils/api/schemas'

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

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const collections = ref<CollectionSummary[]>([])
const selected = ref<Set<string>>(new Set())
const held = ref<Set<string>>(new Set())
const pending = ref(false)
const loaded = ref(false)
const saving = ref(false)
const createOpen = ref(false)

const applyList = (items: CollectionSummary[], preserveSelection: boolean) => {
  collections.value = items
  const nextHeld = new Set(
    items.filter((c) => c.viewer?.has_work).map((c) => c.id)
  )
  held.value = nextHeld
  if (!preserveSelection) {
    selected.value = new Set(nextHeld)
  }
}

const load = async (preserveSelection = false) => {
  // A failed picker read once saved collection_ids: [] and stripped the work
  // from every folder. Save stays off until a list read succeeds.
  if (!preserveSelection) {
    loaded.value = false
  }
  pending.value = true
  const result = await settle(
    api.GET('/me/collections', {
      params: {
        query: {
          work_id: String(props.workId),
          limit: 100
        }
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
  applyList(result.data.items, preserveSelection)
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

const save = async () => {
  if (!loaded.value) {
    return
  }
  saving.value = true
  const workId = String(props.workId)
  const diffs = collections.value.filter(
    (c) => selected.value.has(c.id) !== held.value.has(c.id)
  )
  const results = await Promise.all(
    diffs.map((c) => {
      const params = {
        params: {
          path: { collection_id: c.id, work_id: workId }
        }
      }
      return settle(
        selected.value.has(c.id)
          ? api.PUT('/collections/{collection_id}/works/{work_id}', params)
          : api.DELETE('/collections/{collection_id}/works/{work_id}', params)
      )
    })
  )
  saving.value = false
  const failed = results.find((result) => !result.ok)
  if (failed && !failed.ok) {
    reportProblem(failed.problem)
    await load()
    if (loaded.value) {
      emits('saved', { favorited: selected.value.size > 0 })
    }
    return
  }
  useMessage(10569, 'success')
  emits('saved', { favorited: selected.value.size > 0 })
  isOpen.value = false
}

const visibilityIcon = (v: CollectionVisibility) =>
  v === 'private' ? 'lucide:lock' : 'lucide:globe'
</script>

<template>
  <KunModal v-model="isOpen" inner-class-name="max-w-md">
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-xl font-bold">收藏到收藏夹</h2>
        <KunButton variant="light" size="sm" @click="createOpen = true">
          <KunIcon name="lucide:plus" />
          新建
        </KunButton>
      </div>

      <div v-if="pending" class="text-default-500 py-8 text-center text-sm">
        加载中...
      </div>

      <div v-else class="max-h-[45vh] space-y-1 overflow-y-auto">
        <div
          v-for="c in collections"
          :key="c.id"
          class="hover:bg-default-100 flex items-center gap-1 rounded-lg pr-1 transition-colors"
        >
          <button
            type="button"
            class="flex min-w-0 flex-1 items-center gap-2 px-3 py-2 text-left"
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
            <KunIcon
              :name="visibilityIcon(c.visibility)"
              class="text-default-400 shrink-0"
            />
            <span class="truncate">{{ collectionDisplayName(c, myName) }}</span>
            <span class="text-default-400 ml-auto shrink-0 text-sm">
              {{ c.item_count }}
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

        <KunNull v-if="!loaded" description="收藏夹读取失败，请稍后重试" />
        <KunNull
          v-else-if="!collections.length"
          description="还没有收藏夹，点击新建一个吧"
        />
      </div>

      <div class="flex justify-end gap-3">
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
  </KunModal>

  <GalgameCollectionEditModal
    v-model="createOpen"
    mode="create"
    :is-default="loaded && collections.length === 0"
    @saved="onCreated"
  />
</template>
