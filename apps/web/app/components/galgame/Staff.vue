<script setup lang="ts">
import type { WorkCreditGroup } from '#shared/utils/api/schemas'

const props = defineProps<{
  credits: WorkCreditGroup[]
}>()

const nameOf = useCatalogName()

const COLLAPSED_GROUPS = 6

const isExpanded = ref(false)
const isCollapsible = computed(() => props.credits.length > COLLAPSED_GROUPS)
const visibleStaff = computed(() =>
  isCollapsible.value && !isExpanded.value
    ? props.credits.slice(0, COLLAPSED_GROUPS)
    : props.credits
)
const hiddenCount = computed(() => props.credits.length - COLLAPSED_GROUPS)
</script>

<template>
  <div v-if="credits.length" class="space-y-3">
    <KunHeader
      name="制作人员"
      description="该 Galgame 的剧本, 原画, 音乐, 声优等制作人员名单, 数据来自 NextMoe 目录的署名图谱"
      scale="h2"
    />

    <dl class="space-y-4">
      <div
        v-for="group in visibleStaff"
        :key="group.role_key"
        class="grid grid-cols-1 gap-x-4 gap-y-1 sm:grid-cols-[6rem_1fr]"
      >
        <dt class="text-default-500 pt-0.5 text-sm font-medium">
          {{ group.display_name }}
        </dt>
        <dd class="flex flex-wrap items-baseline gap-x-4 gap-y-1.5">
          <span
            v-for="person in group.people"
            :key="person.id"
            class="text-base"
          >
            <KunLink
              :to="`/galgame/staff/${person.id}`"
              underline="none"
              class-name="text-default-800 hover:text-primary"
            >
              {{ nameOf(person).name }}
            </KunLink>
            <span
              v-if="person.voiced_characters.length"
              class="text-default-400 text-sm"
            >
              （{{ person.voiced_characters.join(' / ') }}）
            </span>
          </span>
        </dd>
      </div>
    </dl>

    <KunButton
      v-if="isCollapsible"
      variant="flat"
      color="primary"
      size="sm"
      @click="isExpanded = !isExpanded"
    >
      <KunIcon
        :name="isExpanded ? 'lucide:chevron-up' : 'lucide:chevron-down'"
      />
      {{ isExpanded ? '收起制作人员' : `展开其余 ${hiddenCount} 项职位` }}
    </KunButton>
  </div>
</template>
