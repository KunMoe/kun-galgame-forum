<script setup lang="ts">
import { websiteFormSchema } from '~/validations/website'
import {
  KUN_WEBSITE_LANGUAGE_MAP,
  KUN_WEBSITE_NSFW_OPTIONS,
  KUN_WEBSITE_STATUS_OPTIONS
} from '~/constants/galgameWebsite'
import type { WebsiteForm } from './types'
import type { KunSelectOption } from '@kungal/ui-vue'

const props = defineProps<{
  modelValue: boolean
  initialData?: WebsiteForm
  isEditing: boolean
  iconPreviewUrl?: string
  hasExternalIcon?: boolean
  loading?: boolean
}>()

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [data: WebsiteForm]
}>()

const languageOptions = Object.entries(KUN_WEBSITE_LANGUAGE_MAP).map(
  ([value, label]) => ({ value, label })
)

const nsfwOptions = KUN_WEBSITE_NSFW_OPTIONS as KunSelectOption<string>[]

const statusOptions = KUN_WEBSITE_STATUS_OPTIONS as KunSelectOption<
  WebsiteForm['state']
>[]

const isModalOpen = computed({
  get: () => props.modelValue,
  set: (value) => emits('update:modelValue', value)
})

const newUrl = ref('')

const { data: categories } = useWebsiteCategories()
const categoryOptions = computed(() =>
  (categories.value ?? []).map((category) => ({
    value: category.id,
    label: category.label || category.slug
  }))
)

const { data: tags, status: tagStatus } = useWebsiteTags()

const emptyForm = (): WebsiteForm => ({
  host: '',
  title: '',
  description: '',
  icon_image_hash: '',
  website_category_id: categories.value?.[0]?.id ?? '',
  website_tag_ids: [],
  is_nsfw: false,
  state: 'normal',
  language: 'zh-cn',
  urls: [],
  founded: ''
})

const formData = reactive<WebsiteForm>(emptyForm())

const nsfwChoice = computed({
  get: (): string => (formData.is_nsfw ? 'nsfw' : 'sfw'),
  set: (value: string) => (formData.is_nsfw = value === 'nsfw')
})

watch(
  () => isModalOpen.value,
  (isOpen) => {
    if (isOpen) {
      Object.assign(
        formData,
        emptyForm(),
        structuredClone(toRaw(props.initialData ?? {}))
      )
    }
  }
)

const addUrl = () => {
  const url = newUrl.value.trim()
  if (url && !formData.urls.includes(url)) {
    formData.urls.push(url)
    newUrl.value = ''
  }
}

const removeUrl = (index: number) => {
  formData.urls.splice(index, 1)
}

// Closing on submit is the parent's job — it is the only side that knows
// whether the request came back. Self-closing here threw away a filled-in form
// every time the API rejected it.
const handleSubmit = () => {
  const result = websiteFormSchema.safeParse(formData)
  if (!result.success) {
    const message = JSON.parse(result.error.message)[0]
    useMessage(formatKunZodIssue(message), 'warn')
    return
  }
  if (!result.data.icon_image_hash && !props.hasExternalIcon) {
    useMessage('请上传网站图标', 'warn')
    return
  }
  emits('submit', result.data)
}
</script>

<template>
  <KunModal
    :is-dismissable="false"
    v-model="isModalOpen"
    :aria-label="isEditing ? '编辑网站' : '创建新网站'"
    inner-class-name="max-w-2xl"
  >
    <form @submit.prevent>
      <h2 class="mb-4 text-xl font-bold">
        {{ isEditing ? '编辑网站' : '创建新网站' }}
      </h2>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <KunInput v-model="formData.title" label="网站名称" required />
        <KunInput v-model="formData.founded" label="网站创建时间" />

        <div class="md:col-span-2">
          <KunCoverUpload
            v-model="formData.icon_image_hash"
            :preview-url="iconPreviewUrl ?? ''"
            label="网站图标"
          />
        </div>
        <KunInput
          v-model="formData.host"
          label="网站主域名"
          :placeholder="kungal.domain.main"
          class-name="md:col-span-2"
        />

        <KunTextarea
          v-model="formData.description"
          label="网站介绍"
          required
          auto-grow
          show-char-count
          :maxlength="1000"
          class-name="md:col-span-2"
        />

        <KunSelect
          v-model="formData.website_category_id"
          label="分类"
          :options="categoryOptions"
        />

        <KunSelect
          v-model="formData.state"
          label="网站状态"
          :options="statusOptions"
        />

        <KunSelect
          v-model="formData.language"
          label="语言"
          :options="languageOptions"
        />

        <KunSelect
          v-model="nsfwChoice"
          label="年龄限制"
          :options="nsfwOptions"
        />

        <div class="md:col-span-2">
          <label class="mb-1 block text-sm font-medium">
            可用地址 (可选, 带 https://)
          </label>
          <div class="flex gap-2">
            <KunInput
              v-model="newUrl"
              placeholder="https://www.kungal.com"
              class-name="flex-grow"
              @keydown.enter.prevent="addUrl"
            />
            <KunButton
              :is-icon-only="true"
              color="primary"
              @click="addUrl"
              class-name="shrink-0"
            >
              <KunIcon name="lucide:plus" />
            </KunButton>
          </div>
          <div v-if="formData.urls.length" class="mt-2 flex flex-wrap gap-2">
            <span
              v-for="(url, index) in formData.urls"
              :key="url"
              class="bg-default-100 flex items-center rounded-full px-3 py-1 text-sm"
            >
              {{ url }}
              <button
                type="button"
                class="text-default-500 hover:text-default-700 ml-2"
                @click="removeUrl(index)"
              >
                <KunIcon name="lucide:x" class="h-4 w-4" />
              </button>
            </span>
          </div>
        </div>

        <div class="md:col-span-2">
          <label class="mb-1 block text-sm font-medium">
            标签 (可选, 最多 20 个)
          </label>
          <div v-if="tagStatus === 'pending'" class="text-default-500 text-sm">
            正在加载标签...
          </div>

          <WebsiteModalTagSelector
            v-else-if="tags"
            :tags="tags"
            :tag-ids="formData.website_tag_ids"
            @update-ids="(value) => (formData.website_tag_ids = value)"
          />
        </div>
      </div>

      <div class="mt-6 flex justify-end gap-3">
        <KunButton
          variant="light"
          color="danger"
          :disabled="loading"
          @click="isModalOpen = false"
        >
          取消
        </KunButton>
        <KunButton color="primary" :loading="loading" @click="handleSubmit">
          {{ isEditing ? '保存更改' : '创建' }}
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
