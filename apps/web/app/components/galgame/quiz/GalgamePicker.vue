<script setup lang="ts">
import {
  usePersistQuizGalgameStore,
  type RecentQuizGalgame
} from '~/store/modules/edit/quizGalgame'
import type { WorkRef } from '#shared/utils/api/schemas'
import { KUN_QUIZ_WORK_LIMIT } from '~/constants/galgame-quiz'

const props = defineProps<{
  modelValue: string[]
  initialSelected?: RecentQuizGalgame[]
}>()
const emits = defineEmits<{ 'update:modelValue': [value: string[]] }>()

const store = usePersistQuizGalgameStore()
const { recent } = storeToRefs(store)
const workName = useWorkName()

const selectedList = ref<RecentQuizGalgame[]>([])

const emitIds = () =>
  emits(
    'update:modelValue',
    selectedList.value.map((g) => String(g.id))
  )

const toRecent = (work: WorkRef): RecentQuizGalgame => ({
  id: work.id,
  name: workName(work),
  coverUrl: work.cover?.url,
  thumbhash: work.cover?.thumbhash ?? undefined,
  isNsfw: work.is_nsfw
})

const pick = (game: RecentQuizGalgame) => {
  const id = String(game.id)
  if (selectedList.value.some((g) => String(g.id) === id)) return
  if (selectedList.value.length >= KUN_QUIZ_WORK_LIMIT) {
    useMessage('最多关联 20 部作品', 'warn')
    return
  }
  const next = { ...game, id }
  selectedList.value.push(next)
  emitIds()
  store.add(next)
}

const pickWork = (work: WorkRef) => pick(toRecent(work))

const remove = (id: string) => {
  selectedList.value = selectedList.value.filter((g) => String(g.id) !== id)
  emitIds()
}

watch(
  () => props.modelValue,
  (v) => {
    if (!v || v.length === 0) selectedList.value = []
  }
)
watch(
  () => props.initialSelected,
  (v) => {
    if (v && v.length) {
      selectedList.value = v.map((g) => ({ ...g, id: String(g.id) }))
    }
  },
  { immediate: true }
)
</script>

<template>
  <div class="space-y-2">
    <label class="text-sm font-medium">关联 Galgame（可选, 最多 20 部）</label>

    <div v-if="selectedList.length" class="space-y-2">
      <div
        v-for="g in selectedList"
        :key="g.id"
        class="border-default-200 flex items-center gap-3 rounded-lg border p-2"
      >
        <div class="bg-default-100 h-12 w-9 shrink-0 overflow-hidden rounded">
          <KunImage
            v-if="g.coverUrl"
            :src="g.coverUrl"
            :thumbhash="g.thumbhash"
            width="36"
            height="48"
            object-fit="cover"
            class-name="h-full w-full"
          />
        </div>
        <div class="min-w-0 flex-1">
          <p class="truncate font-medium">{{ g.name }}</p>
          <p v-if="g.isNsfw" class="text-danger text-xs">NSFW</p>
        </div>
        <KunButton
          :is-icon-only="true"
          variant="light"
          size="sm"
          @click="remove(String(g.id))"
        >
          <KunIcon name="lucide:x" />
        </KunButton>
      </div>
    </div>

    <GalgameSearchAutocomplete
      v-if="selectedList.length < KUN_QUIZ_WORK_LIMIT"
      :exclude-ids="selectedList.map((g) => String(g.id))"
      placeholder="输入游戏名搜索并关联"
      @select="pickWork"
    />

    <div v-if="recent.length" class="space-y-1">
      <span class="text-default-400 text-xs">最近关联</span>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="g in recent"
          :key="g.id"
          type="button"
          class="border-default-200 hover:border-primary flex items-center gap-2 rounded-lg border p-1 pr-2 transition-colors"
          @click="pick({ ...g, id: String(g.id) })"
        >
          <div class="bg-default-100 h-8 w-6 shrink-0 overflow-hidden rounded">
            <KunImage
              v-if="g.coverUrl"
              :src="g.coverUrl"
              :thumbhash="g.thumbhash"
              width="24"
              height="32"
              object-fit="cover"
              class-name="h-full w-full"
            />
          </div>
          <span class="max-w-[10rem] truncate text-sm">{{ g.name }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
