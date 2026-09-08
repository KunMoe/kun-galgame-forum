<script setup lang="ts">
const props = defineProps<{
  modelValue: boolean
  galgameId: number
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [payload: { favorited: boolean }]
}>()

const { name: myName } = usePersistUserStore()

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const collections = ref<MyCollectionForGalgame[]>([])
const selected = ref<Set<number>>(new Set())
const pending = ref(false)
const saving = ref(false)
const createOpen = ref(false)

const load = async (preserveSelection = false) => {
  pending.value = true
  const res = await kunFetch<{ collections: MyCollectionForGalgame[] }>(
    `/galgame/${props.galgameId}/collections/mine`
  )
  pending.value = false
  collections.value = res?.collections ?? []
  if (!preserveSelection) {
    selected.value = new Set(
      collections.value.filter((c) => c.contains).map((c) => c.id)
    )
  }
}

watch(
  () => isOpen.value,
  (open) => {
    if (open) {
      createOpen.value = false
      load()
    }
  }
)

const toggle = (id: number) => {
  const next = new Set(selected.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  selected.value = next
}

const onCreated = async (newId?: number) => {
  await load(true)
  if (newId) {
    const next = new Set(selected.value)
    next.add(newId)
    selected.value = next
  }
}

const save = async () => {
  saving.value = true
  const ids = [...selected.value]
  const result = await kunFetch<string>(
    `/galgame/${props.galgameId}/collections`,
    { method: 'PUT', body: { collection_ids: ids } }
  )
  saving.value = false
  if (!result) {
    return
  }
  useMessage(10569, 'success')
  emits('saved', { favorited: ids.length > 0 })
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
              :to="`/galgame/collection/${c.id}`"
              target="_blank"
              color="default"
              underline="none"
              class-name="text-default-400 hover:text-primary flex shrink-0 items-center rounded-md p-1.5"
            >
              <KunIcon name="lucide:external-link" />
            </KunLink>
          </KunTooltip>
        </div>

        <KunNull
          v-if="!collections.length"
          description="还没有收藏夹，点击新建一个吧"
        />
      </div>

      <div class="flex justify-end gap-3">
        <KunButton variant="light" color="danger" @click="isOpen = false">
          取消
        </KunButton>
        <KunButton color="primary" :loading="saving" @click="save">
          保存
        </KunButton>
      </div>
    </div>
  </KunModal>

  <GalgameCollectionEditModal
    v-model="createOpen"
    mode="create"
    @saved="onCreated"
  />
</template>
