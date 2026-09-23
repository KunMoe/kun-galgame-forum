<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    modelValue: string[]
    nsfw?: string[]
    locked?: string[]
    previewUrls?: string[]
    label?: string
    description?: string
    max?: number
  }>(),
  {
    nsfw: () => [],
    locked: () => [],
    previewUrls: () => [],
    label: '图片',
    description: '',
    max: 9
  }
)

const emits = defineEmits<{
  'update:modelValue': [value: string[]]
  'update:nsfw': [value: string[]]
}>()

const MAX_FILE_SIZE = 10 * 1024 * 1024

interface Slot {
  hash: string
  url: string
  nsfw: boolean
  locked: boolean
}

// Hashes the image service graded as adult. The author can add marks but not
// take these off, so they are kept apart from the marks that get submitted.
const machineLocked = ref<string[]>([])
const isLocked = (hash: string) =>
  props.locked.includes(hash) || machineLocked.value.includes(hash)

// The hash is what gets submitted, but only the upload response carries the URL
// to preview it with, so the pairing is remembered here instead of being
// re-derived every time the parent hands the list back.
const urlByHash = new Map<string, string>()
const slots = ref<Slot[]>([])

watch(
  [
    () => props.modelValue,
    () => props.previewUrls,
    () => props.nsfw,
    () => props.locked,
    machineLocked
  ],
  ([hashes, urls, nsfw]) => {
    hashes.forEach((hash, index) => {
      const url = urls[index]
      if (url && !urlByHash.has(hash)) {
        urlByHash.set(hash, url)
      }
    })
    slots.value = hashes.map((hash) => ({
      hash,
      url: urlByHash.get(hash) ?? '',
      nsfw: nsfw.includes(hash),
      locked: isLocked(hash)
    }))
  },
  { immediate: true, deep: true }
)

const uploadingCount = ref(0)
const isDropping = ref(false)
const dragIndex = ref(-1)

const remaining = computed(
  () => props.max - slots.value.length - uploadingCount.value
)
const isFull = computed(() => remaining.value <= 0)
const markedCount = computed(
  () => slots.value.filter((s) => s.nsfw || s.locked).length
)
const lockedCount = computed(() => slots.value.filter((s) => s.locked).length)

const commit = () => {
  emits(
    'update:modelValue',
    slots.value.map((s) => s.hash)
  )
  emits(
    'update:nsfw',
    slots.value.filter((s) => s.nsfw).map((s) => s.hash)
  )
}

const uploadOne = async (file: File) => {
  const image = await uploadImage(file, 'content', file.name)
  if (!image) {
    return null
  }
  urlByHash.set(image.hash, image.url)
  // A brand new image has no grade yet — the grader is nightly — so this only
  // ever fires when the upload deduplicated onto an image already graded.
  const locked = image.sexual === 'explicit'
  if (locked && !machineLocked.value.includes(image.hash)) {
    machineLocked.value = [...machineLocked.value, image.hash]
  }
  return { hash: image.hash, url: image.url, nsfw: false, locked }
}

const addFiles = async (files: File[]) => {
  const accepted = files.filter((file) => file.type.startsWith('image/'))
  if (!accepted.length) {
    return
  }
  const oversized = accepted.find((file) => file.size > MAX_FILE_SIZE)
  if (oversized) {
    useMessage(`${oversized.name} 超过 10MB, 请压缩后再上传`, 'warn')
    return
  }
  if (accepted.length > remaining.value) {
    useMessage(
      `最多 ${props.max} 张图片, 只能再添加 ${remaining.value} 张`,
      'warn'
    )
    return
  }

  uploadingCount.value += accepted.length
  try {
    const uploaded = await Promise.all(accepted.map(uploadOne))
    const ok = uploaded.filter((item) => item !== null)
    if (ok.length < accepted.length) {
      useMessage(`${accepted.length - ok.length} 张图片上传失败`, 'error')
    }
    if (!ok.length) {
      return
    }
    slots.value = [...slots.value, ...ok]
    commit()
  } finally {
    uploadingCount.value -= accepted.length
  }
}

const removeAt = (index: number) => {
  slots.value.splice(index, 1)
  commit()
}

const toggleNsfw = (index: number) => {
  const slot = slots.value[index]
  if (!slot || slot.locked) {
    return
  }
  slot.nsfw = !slot.nsfw
  commit()
}

const handleDropFiles = (event: DragEvent) => {
  isDropping.value = false
  const files = Array.from(event.dataTransfer?.files ?? [])
  if (files.length) {
    addFiles(files)
  }
}

// Reordering happens on dragenter rather than on drop so the grid rearranges
// under the cursor and the author can see the cover image change before letting
// go.
const handleDragEnter = (index: number) => {
  if (dragIndex.value < 0 || dragIndex.value === index) {
    return
  }
  const moved = slots.value.splice(dragIndex.value, 1)[0]
  if (!moved) {
    return
  }
  slots.value.splice(index, 0, moved)
  dragIndex.value = index
}

const endReorder = () => {
  if (dragIndex.value < 0) {
    return
  }
  dragIndex.value = -1
  commit()
}
</script>

<template>
  <div>
    <div class="mb-1 flex items-end justify-between gap-2">
      <label class="block text-sm font-medium">{{ label }}</label>
      <span class="text-default-400 text-xs tabular-nums">
        {{ slots.length }} / {{ max }}
      </span>
    </div>

    <div
      :class="
        cn(
          'border-default-200 rounded-lg border p-2 transition-colors',
          isDropping && 'border-primary bg-primary/5'
        )
      "
      @dragover.prevent="isDropping = true"
      @dragleave="isDropping = false"
      @drop.prevent="handleDropFiles"
    >
      <div class="flex flex-wrap gap-2">
        <div
          v-for="(slot, index) in slots"
          :key="slot.hash"
          :class="
            cn(
              'group border-default-200 relative size-24 cursor-grab overflow-hidden rounded-md border',
              (slot.nsfw || slot.locked) && 'border-danger',
              dragIndex === index && 'opacity-40'
            )
          "
          draggable="true"
          @dragstart="dragIndex = index"
          @dragenter="handleDragEnter(index)"
          @dragend="endReorder"
          @drop.stop.prevent="endReorder"
        >
          <KunImage
            v-if="slot.url"
            :src="slot.url"
            :alt="`${label} ${index + 1}`"
            class="size-full object-cover"
          />
          <div
            v-else
            class="bg-default-100 text-default-400 flex size-full flex-col items-center justify-center gap-1 px-1 text-center"
          >
            <KunIcon name="lucide:eye-off" class="size-4 text-inherit" />
            <span class="text-[10px] leading-tight">已按内容设置隐藏</span>
          </div>

          <span
            v-if="index === 0"
            class="absolute bottom-1 left-1 rounded bg-black/55 px-1.5 py-0.5 text-[10px] leading-none text-white"
          >
            主图
          </span>

          <!-- Marked and unmarked have to differ in shape, not only in colour:
               a red R18 pill on a red photo is not a state anyone can read. -->
          <button
            type="button"
            :disabled="slot.locked"
            :aria-pressed="slot.nsfw || slot.locked"
            :aria-label="`将第 ${index + 1} 张图片标记为成人内容`"
            :title="
              slot.locked
                ? '图片服务判定为成人内容, 无法取消'
                : slot.nsfw
                  ? '已标记为成人内容, 点击取消'
                  : '标记为成人内容'
            "
            :class="
              cn(
                'absolute right-1 bottom-1 flex items-center justify-center gap-0.5 rounded text-white ring-1 transition-colors',
                slot.nsfw || slot.locked
                  ? 'bg-danger px-1.5 py-0.5 text-[10px] leading-none font-semibold ring-white/70'
                  : 'size-6 bg-black/55 ring-transparent hover:bg-black/75',
                slot.locked && 'cursor-not-allowed'
              )
            "
            @click="toggleNsfw(index)"
          >
            <KunIcon
              v-if="slot.locked"
              name="lucide:lock"
              class="size-2.5 text-inherit"
            />
            <template v-if="slot.nsfw || slot.locked">R18</template>
            <KunIcon
              v-else
              name="lucide:eye-off"
              class="size-3.5 text-inherit"
            />
          </button>

          <button
            type="button"
            :aria-label="`移除第 ${index + 1} 张图片`"
            class="absolute top-1 right-1 flex size-6 items-center justify-center rounded-full bg-black/55 text-white transition-colors hover:bg-black/75"
            @click="removeAt(index)"
          >
            <KunIcon name="lucide:x" class="size-3.5 text-inherit" />
          </button>
        </div>

        <div
          v-for="n in uploadingCount"
          :key="`uploading-${n}`"
          class="border-default-200 bg-default-100 flex size-24 items-center justify-center rounded-md border"
        >
          <KunLoading size="sm" />
        </div>

        <div v-if="!isFull" class="relative size-24">
          <KunFileInput
            accept="image/*"
            multiple
            :max-size="MAX_FILE_SIZE"
            :show-file-name="false"
            class-name="absolute inset-0"
            @change="addFiles"
            @error-pick="(message: string) => useMessage(message, 'warn')"
          >
            <template #default="{ pick }">
              <button
                type="button"
                class="border-default-300 text-default-500 hover:border-primary hover:text-primary absolute inset-0 flex cursor-pointer flex-col items-center justify-center gap-1 rounded-md border border-dashed transition-colors"
                @click="pick"
              >
                <KunIcon name="lucide:image-plus" class="size-5 text-inherit" />
                <span class="text-xs">
                  {{ isDropping ? '松开上传' : '添加图片' }}
                </span>
              </button>
            </template>
          </KunFileInput>
        </div>
      </div>

      <p
        v-if="!slots.length && !uploadingCount"
        class="text-default-400 mt-2 text-xs"
      >
        可以直接把图片拖进这里, 支持一次选择多张。
      </p>
      <p v-else class="text-default-400 mt-2 text-xs">
        <template v-if="slots.length > 1">
          拖动可以调整顺序, 第一张会作为奖品卡片上的封面。
        </template>
        点图片右下角的图标可以把它标为成人内容 (R18), 未开启 NSFW
        显示的读者看不到它。
      </p>
      <p v-if="markedCount" class="text-danger mt-1 text-xs">
        已标记 {{ markedCount }} 张成人内容图片。
        <template v-if="lockedCount">
          其中 {{ lockedCount }} 张由图片服务判定, 无法取消。
        </template>
      </p>
    </div>

    <p v-if="description" class="text-default-500 mt-1 text-xs">
      {{ description }}
    </p>
  </div>
</template>
