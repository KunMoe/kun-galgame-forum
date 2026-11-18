<script setup lang="ts">
import { kunSidebarRail } from '~/constants/layout'

const { hasReleaseToday } = useGalgameReleaseToday()
const { agentEnabled } = useSiteFeatures()
</script>

<template>
  <nav class="flex flex-col items-center gap-1">
    <KunLayoutSideBarRailGroup
      v-for="group in kunSidebarRail"
      :key="group.name"
      :group="group"
    />

    <KunButton
      variant="light"
      color="default"
      href="/galgame-calendar"
      aria-label="Galgame 发售月历"
      :class-name="
        cn(
          'h-auto w-full flex-col gap-1 py-2',
          hasReleaseToday ? 'text-primary' : 'text-foreground'
        )
      "
    >
      <KunIcon name="lucide:calendar-days" class="text-xl" />
      <span class="text-[11px] leading-none">
        {{ hasReleaseToday ? '有新作发售' : '发售月历' }}
      </span>
    </KunButton>

    <KunButton
      v-if="agentEnabled"
      variant="light"
      color="primary"
      href="/chat"
      aria-label="AI 帮找"
      class-name="text-primary-600 h-auto w-full flex-col gap-1 py-2"
    >
      <KunIcon name="lucide:bot-message-square" class="text-xl" />
      <span class="text-[11px] leading-none">AI 帮找</span>
    </KunButton>
  </nav>
</template>
