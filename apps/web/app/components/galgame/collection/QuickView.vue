<script setup lang="ts">
import { useMounted } from '@vueuse/core'

const props = defineProps<{
  collectionId: string
  title: string
  defaultSort: CollectionEntrySortValue
}>()

const {
  quick,
  quick_sort: quickSort,
  set
} = useQueryState({
  quick: queryString(),
  quick_sort: queryString()
})

const mounted = useMounted()
const open = computed({
  get: () => mounted.value && quick.value === '1',
  set: (value: boolean) =>
    set(value ? { quick: '1' } : { quick: '', quick_sort: '' })
})

const opened = ref(false)
watch(
  open,
  (value) => {
    if (value) {
      opened.value = true
    }
  },
  { immediate: true }
)

const sort = computed<CollectionEntrySortValue>({
  get: () =>
    COLLECTION_ENTRY_SORTS.find((o) => o.value === quickSort.value)?.value ??
    props.defaultSort,
  set: (value) => {
    quickSort.value = value === props.defaultSort ? '' : value
  }
})

const show = () => {
  open.value = true
}

defineExpose({ show })
</script>

<template>
  <KunDrawer
    v-model="open"
    placement="right"
    responsive
    size="xl"
    inner-class-name="md:w-[min(72rem,calc(100vw-3rem))] max-md:h-[90dvh]"
    :title="title"
  >
    <GalgameCollectionQuickViewGrid
      v-if="opened"
      v-model:sort="sort"
      :collection-id="collectionId"
    />
  </KunDrawer>
</template>
