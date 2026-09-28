<script setup lang="ts">
import type { ReviewAuthor, ReviewSubject } from '#shared/utils/api/schemas'

defineProps<{
  kind: string
  subject: ReviewSubject | null
  author: ReviewAuthor | null
}>()

const emit = defineEmits<{
  showAuthorHistory: [author: ReviewAuthor]
}>()

const STATE: Record<
  ReviewSubject['state'],
  { label: string; color: 'success' | 'warning' | 'default' }
> = {
  visible: { label: '正常可见', color: 'success' },
  hidden: { label: '已隐藏', color: 'warning' },
  deleted: { label: '已删除（原文保留）', color: 'default' },
  gone: { label: '已不存在或无法读取', color: 'default' }
}
</script>

<template>
  <div class="space-y-2">
    <span class="text-default-600 text-sm font-medium">被审核内容</span>

    <p v-if="!subject" class="text-default-400 text-sm">
      论坛无法读取这类内容，请参考下方的判定依据与举报记录
    </p>

    <template v-else>
      <div class="flex flex-wrap items-center gap-2 text-sm">
        <KunChip :color="STATE[subject.state].color" size="xs">
          {{ STATE[subject.state].label }}
        </KunChip>
        <span v-if="subject.parent_title || subject.parent_path">
          <span class="text-default-500">位于</span>
          <KunLink
            v-if="subject.parent_path"
            :to="subject.parent_path"
            target="_blank"
            class-name="inline"
          >
            {{ subject.parent_title ?? subject.parent_path }}
          </KunLink>
          <span v-else>{{ subject.parent_title }}</span>
        </span>
        <KunTime
          v-if="subject.authored_at"
          class="text-default-400 ml-auto text-xs"
          :time="subject.authored_at"
          type="datetime"
          show-year
        />
      </div>

      <p v-if="subject.title" class="font-medium break-words">
        {{ subject.title }}
      </p>

      <div
        v-if="subject.content.children.length"
        class="border-default-200 max-h-80 overflow-y-auto rounded-lg border p-3"
      >
        <ContentDocument :document="subject.content" />
      </div>
    </template>

    <template v-if="author">
      <span class="text-default-600 text-sm font-medium">
        {{ kind === 'user' ? '被审核用户' : '作者' }}
      </span>
      <AdminModerationAuthor
        :author="author"
        @show-history="emit('showAuthorHistory', author)"
      />
    </template>
  </div>
</template>
