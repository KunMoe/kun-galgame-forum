<script setup lang="ts">
import type { Poll, PollCreate, PollPatch } from '#shared/utils/api/schemas'
import { usePoll } from '~/composables/topic/usePoll'
import { createPollSchema, updatePollSchema } from '~/validations/topic-poll'
import { TOPIC_POLL_VISIBILITY_OPTIONS } from '~/constants/topic'
import { closesAtFromPicker, deadlineToPicker } from '../miniapp/deadline'
import {
  clonePlain,
  pendingPollIssue
} from '~/composables/topic/pendingMiniApps'
import type { PollFormData } from './types'

const props = defineProps<{
  topicId?: string
  initialData?: Poll
  initialForm?: PollFormData
  saveOnly?: boolean
}>()

const emits = defineEmits<{
  refresh: []
  save: [form: PollFormData]
}>()

const isModalOpen = defineModel<boolean>({ required: true })

const isEditing = computed(
  () => !!props.initialData || (props.saveOnly && !!props.initialForm)
)
const { createPoll, updatePoll } = usePoll(() => props.topicId ?? '')
const isLoading = ref(false)

const getInitialFormData = (): PollFormData => {
  if (props.initialForm) {
    return clonePlain(props.initialForm)
  }
  const initial = props.initialData
  if (initial) {
    return {
      title: initial.title,
      description: initial.description,
      options: initial.options.map((option) => ({
        id: option.id,
        text: option.text,
        _status: 'existing'
      })),
      choice_type: initial.choice_type,
      min_choice: initial.min_choice,
      max_choice: initial.max_choice,
      closes_at: deadlineToPicker(initial.closes_at),
      result_visibility: initial.result_visibility,
      is_anonymous: initial.is_anonymous,
      can_change_vote: initial.can_change_vote
    }
  }

  return {
    title: '',
    description: '',
    options: [{ text: '' }, { text: '' }],
    choice_type: 'single',
    min_choice: 1,
    max_choice: 1,
    closes_at: undefined,
    result_visibility: 'always',
    is_anonymous: false,
    can_change_vote: false
  }
}

const formData = reactive<PollFormData>(getInitialFormData())

watch(
  () => isModalOpen.value,
  (isOpen) => {
    if (isOpen) {
      Object.assign(formData, getInitialFormData())
    }
  }
)

watch(
  () => formData.choice_type,
  (choiceType) => {
    if (choiceType === 'single') {
      formData.min_choice = 1
      formData.max_choice = 1
    }
  }
)

const liveOptions = computed(() =>
  formData.options.filter((option) => option._status !== 'deleted')
)

const addOption = () => {
  if (liveOptions.value.length >= 20) {
    useMessage('最多添加20个选项', 'warn')
    return
  }
  formData.options.push({ text: '', _status: 'new' })
}

const removeOption = (index: number) => {
  if (liveOptions.value.length <= 2) {
    useMessage('至少需要2个选项', 'warn')
    return
  }
  const option = formData.options[index]
  if (!option) {
    return
  }
  if (option.id) {
    option._status = 'deleted'
  } else {
    formData.options.splice(index, 1)
  }
}

const warnFirstIssue = (message: string) => {
  useMessage(formatKunZodIssue(JSON.parse(message)[0]), 'warn')
}

const commonFields = (closesAt: string | undefined) => ({
  title: formData.title,
  description: formData.description,
  choice_type: formData.choice_type,
  min_choice: formData.min_choice,
  max_choice: formData.max_choice,
  closes_at: closesAt,
  result_visibility: formData.result_visibility,
  is_anonymous: formData.is_anonymous,
  can_change_vote: formData.can_change_vote
})

const plannedOptionChanges = (initial: Poll) => {
  const storedText = new Map(
    initial.options.map((option) => [option.id, option.text])
  )
  return {
    add: formData.options
      .filter((option) => !option.id && option._status !== 'deleted')
      .map((option) => ({ text: option.text })),
    update: formData.options
      .filter(
        (option) =>
          option.id &&
          option._status !== 'deleted' &&
          storedText.get(option.id) !== option.text
      )
      .map((option) => ({ option_id: option.id!, text: option.text })),
    remove: formData.options
      .filter((option) => option.id && option._status === 'deleted')
      .map((option) => option.id!)
  }
}

const createBody = (closesAt: string | undefined) => ({
  ...commonFields(closesAt),
  options: liveOptions.value.map((option) => ({ text: option.text }))
})

const submitSaveOnly = (closesAt: string | undefined) => {
  const parsed = createPollSchema.safeParse(createBody(closesAt))
  if (!parsed.success) {
    warnFirstIssue(parsed.error.message)
    return false
  }
  const issue = pendingPollIssue(formData)
  if (issue) {
    useMessage(issue, 'warn')
    return false
  }
  emits('save', clonePlain(formData))
  return true
}

const submitCreate = async (closesAt: string | undefined) => {
  const body = createBody(closesAt)
  const parsed = createPollSchema.safeParse(body)
  if (!parsed.success) {
    warnFirstIssue(parsed.error.message)
    return false
  }
  const result = await createPoll({
    ...body,
    closes_at: closesAt ?? null
  } satisfies PollCreate)
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  useMessage('创建投票成功', 'success')
  return true
}

const submitPatch = async (initial: Poll, closesAt: string | undefined) => {
  const optionChanges = plannedOptionChanges(initial)
  const parsed = updatePollSchema.safeParse({
    ...commonFields(closesAt),
    option_changes: optionChanges
  })
  if (!parsed.success) {
    warnFirstIssue(parsed.error.message)
    return false
  }
  const touchesOptions =
    optionChanges.add.length +
      optionChanges.update.length +
      optionChanges.remove.length >
    0
  const body: PollPatch = {
    ...commonFields(closesAt),
    closes_at: closesAt ?? null,
    ...(touchesOptions ? { option_changes: optionChanges } : {})
  }
  const result = await updatePoll(initial.id, body)
  if (!result.ok) {
    reportProblem(result.problem)
    return false
  }
  useMessage('保存投票成功', 'success')
  return true
}

const submitText = computed(() => {
  if (props.saveOnly) {
    return isEditing.value ? '保存' : '添加到话题'
  }
  return isEditing.value ? '保存更改' : '发布投票'
})

const handleSubmit = async () => {
  if (isLoading.value) {
    return
  }
  const closesAt = closesAtFromPicker(formData.closes_at)
  if (props.saveOnly) {
    if (submitSaveOnly(closesAt)) {
      isModalOpen.value = false
    }
    return
  }
  isLoading.value = true
  const initial = props.initialData
  const ok = initial
    ? await submitPatch(initial, closesAt)
    : await submitCreate(closesAt)
  isLoading.value = false
  if (!ok) {
    return
  }
  emits('refresh')
  isModalOpen.value = false
}
</script>

<template>
  <KunModal
    :is-dismissable="false"
    v-model="isModalOpen"
    inner-class-name="max-w-3xl"
  >
    <form @submit.prevent="handleSubmit">
      <h2 class="mb-3 text-2xl font-bold">
        {{ isEditing ? '编辑投票' : '创建投票' }}
      </h2>

      <p class="text-default-500 mb-6 text-sm">
        目前阶段, 话题下方投票最多 30 个, 每个投票最多 20 个选项
      </p>

      <div class="flex flex-col gap-4">
        <KunInput v-model="formData.title" label="投票标题" required />
        <KunTextarea
          v-model="formData.description"
          label="补充描述 (可选)"
          auto-grow
          :rows="2"
        />

        <div>
          <label class="mb-1 block text-sm font-medium">选项设置</label>
          <div class="flex flex-col gap-2">
            <template v-for="(option, index) in formData.options" :key="index">
              <div
                v-if="option._status !== 'deleted'"
                class="flex items-center gap-2"
              >
                <KunInput
                  v-model="option.text"
                  :placeholder="`选项 ${index + 1}`"
                  class-name="flex-grow"
                />
                <KunButton
                  variant="light"
                  color="danger"
                  :is-icon-only="true"
                  @click="removeOption(index)"
                >
                  <KunIcon name="lucide:trash-2" />
                </KunButton>
              </div>
            </template>
          </div>
          <KunButton
            variant="light"
            size="sm"
            class-name="mt-2"
            @click="addOption"
          >
            <KunIcon name="lucide:plus" class="mr-1" />
            增加选项
          </KunButton>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="mb-1 block text-sm font-medium">投票类型</label>
            <div class="flex gap-4">
              <KunCheckBox
                :model-value="formData.choice_type === 'single'"
                label="单选"
                color="primary"
                @update:model-value="formData.choice_type = 'single'"
              />
              <KunCheckBox
                :model-value="formData.choice_type === 'multiple'"
                label="多选"
                color="primary"
                @update:model-value="formData.choice_type = 'multiple'"
              />
            </div>
          </div>

          <div
            v-if="formData.choice_type === 'multiple'"
            class="grid grid-cols-2 gap-2"
          >
            <KunInput
              v-model.number="formData.min_choice"
              label="至少选"
              type="number"
              :min="1"
            />
            <KunInput
              v-model.number="formData.max_choice"
              label="至多选"
              type="number"
              :min="formData.min_choice"
              :max="liveOptions.length"
            />
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <KunDatePicker v-model="formData.closes_at" label="截止日期 (可选)" />
          <KunSelect
            v-model="formData.result_visibility"
            label="结果可见性"
            :options="TOPIC_POLL_VISIBILITY_OPTIONS"
          />
        </div>

        <div class="flex flex-col gap-2">
          <KunSwitch v-model="formData.is_anonymous" label="匿名投票" />
          <KunSwitch v-model="formData.can_change_vote" label="允许修改投票" />
        </div>
      </div>

      <div class="mt-8 flex justify-end gap-3">
        <KunButton variant="light" @click="isModalOpen = false">取消</KunButton>
        <KunButton type="submit" color="primary" :loading="isLoading">
          {{ submitText }}
        </KunButton>
      </div>
    </form>
  </KunModal>
</template>
