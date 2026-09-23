<script setup lang="ts">
import type { ReportExpireStatus } from '~/composables/useReportResourceExpired'

defineProps<{ status: ReportExpireStatus }>()
</script>

<template>
  <div
    v-if="status !== 'idle'"
    class="space-y-2 rounded-lg border p-3 text-sm"
    :class="{
      'border-primary/20 bg-primary/5':
        status === 'checking' || status === 'alive',
      'border-success/20 bg-success/5':
        status === 'expired' || status === 'unchecked',
      'border-danger/20 bg-danger/5': status === 'error'
    }"
  >
    <div v-if="status === 'error'" class="text-danger flex items-center gap-2">
      <KunIcon name="lucide:circle-alert" class="shrink-0" />
      <span>操作失败, 请稍后重试</span>
    </div>

    <template v-else>
      <div class="flex items-center gap-2">
        <KunIcon
          v-if="status === 'checking'"
          name="lucide:loader-circle"
          class="text-primary shrink-0 animate-spin"
        />
        <KunIcon
          v-else
          name="lucide:circle-check"
          class="text-success shrink-0"
        />
        <span
          :class="status === 'checking' ? 'text-primary' : 'text-default-700'"
        >
          <template v-if="status === 'checking'"
            >正在检测链接有效性, 请稍候…</template
          >
          <template v-else-if="status === 'alive'"
            >检测完成: 链接仍然有效</template
          >
          <template v-else-if="status === 'unchecked'"
            >检测完成: 未能核验链接</template
          >
          <template v-else>检测完成: 链接已失效</template>
        </span>
      </div>

      <div class="flex items-center gap-2">
        <KunIcon
          v-if="status === 'checking'"
          name="lucide:circle-dashed"
          class="text-default-300 shrink-0"
        />
        <KunIcon
          v-else-if="status === 'alive'"
          name="lucide:minus"
          class="text-default-400 shrink-0"
        />
        <KunIcon
          v-else
          name="lucide:circle-check"
          class="text-success shrink-0"
        />
        <span
          :class="
            status === 'checking'
              ? 'text-default-400'
              : status === 'alive'
                ? 'text-default-500'
                : 'text-default-700'
          "
        >
          <template v-if="status === 'checking'">等待标记失效</template>
          <template v-else-if="status === 'alive'"
            >无需标记, 链接仍可访问</template
          >
          <template v-else-if="status === 'unchecked'"
            >已按失效处理, 已通知发布者</template
          >
          <template v-else>已标记为失效, 已通知发布者</template>
        </span>
      </div>
    </template>
  </div>
</template>
