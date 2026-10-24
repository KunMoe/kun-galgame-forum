<script setup lang="ts">
import type { KunRadioOption } from '@kungal/ui-vue'
import type { Topic, TopicSectionRequest } from '#shared/utils/api/schemas'
import { settle } from '#shared/utils/api/problem'

const props = defineProps<{ topic: Topic }>()

const api = useApiClient()
const { id: currentUserId } = storeToRefs(usePersistUserStore())
const { sections, sectionLabel } = useTopicSections()

const isAuthor = computed(
  () =>
    !!currentUserId.value &&
    String(currentUserId.value) === props.topic.author.id
)
const open = ref(false)
const isLoading = ref(false)
const isSubmitting = ref(false)
const latest = ref<TopicSectionRequest | null>(null)
const target = ref('')
const reason = ref('')

const STATE_TEXT: Record<TopicSectionRequest['state'], string> = {
  pending: '等待管理员处理',
  approved: '管理员已同意',
  rejected: '管理员没有同意'
}

const options = computed(() =>
  sections.value
    .filter((s) => !s.is_archived && s.section !== props.topic.section)
    .map<KunRadioOption<string>>((s) => ({
      value: s.section,
      label: s.label,
      icon: s.icon_name
    }))
)

const load = async () => {
  isLoading.value = true
  const result = await settle(
    api.GET('/topics/{topic_id}/section-requests', {
      params: { path: { topic_id: props.topic.id } }
    })
  )
  isLoading.value = false
  latest.value = result.ok ? (result.data.items[0] ?? null) : null
}

const openModal = async () => {
  open.value = true
  target.value = ''
  reason.value = ''
  await load()
}

const submit = async () => {
  if (!target.value) {
    useMessage('请选择希望移到的分区', 'warn')
    return
  }
  if (!reason.value.trim()) {
    useMessage('请简单说明理由', 'warn')
    return
  }
  isSubmitting.value = true
  const result = await settle(
    api.POST('/topics/{topic_id}/section-requests', {
      params: { path: { topic_id: props.topic.id } },
      body: {
        section: target.value as TopicSectionRequest['to_section'],
        reason: reason.value
      }
    })
  )
  isSubmitting.value = false
  if (!result.ok) {
    reportProblem(result.problem)
    return
  }
  latest.value = result.data
  useMessage('申请已提交, 管理员处理后会通知你', 'success')
}
</script>

<template>
  <span v-if="isAuthor">
    <KunButton size="xs" variant="light" color="default" @click="openModal">
      <KunIcon name="lucide:arrow-right-left" class="text-inherit" />
      申请调整分区
    </KunButton>

    <KunModal
      v-model="open"
      aria-label="申请调整分区"
      inner-class-name="max-w-xl"
    >
      <div class="space-y-4">
        <h2 class="text-xl font-bold">申请调整分区</h2>
        <p class="text-default-500 text-sm">
          话题现在在「{{
            sectionLabel(topic.section)
          }}」。如果你认为它属于别的分区, 选择分区并说明理由,
          管理员处理后会通知你。从消耗萌萌点的分区移出会退还萌萌点
        </p>

        <KunLoading :loading="isLoading">
          <div
            v-if="latest?.state === 'pending'"
            class="bg-default-100 rounded-xl p-4 text-sm"
          >
            已申请移到「{{ sectionLabel(latest.to_section) }}」,
            {{ STATE_TEXT[latest.state] }}
          </div>

          <form v-else class="space-y-4" @submit.prevent="submit">
            <p v-if="latest" class="text-default-500 text-xs">
              上次申请移到「{{ sectionLabel(latest.to_section) }}」:
              {{ STATE_TEXT[latest.state] }}
            </p>
            <div class="space-y-2">
              <KunRadioGroup
                v-model="target"
                :options="options"
                aria-label="分区"
                variant="card"
                orientation="horizontal"
                size="sm"
                hide-indicator
              />
            </div>
            <KunTextarea
              v-model="reason"
              label="理由"
              :rows="3"
              :maxlength="200"
              placeholder="例如: 我写清了报错和尝试过的步骤, 应该属于游戏运行"
            />
            <div class="flex justify-end gap-2">
              <KunButton variant="light" color="default" @click="open = false">
                取消
              </KunButton>
              <KunButton type="submit" :loading="isSubmitting"
                >提交申请</KunButton
              >
            </div>
          </form>
        </KunLoading>
      </div>
    </KunModal>
  </span>
</template>
