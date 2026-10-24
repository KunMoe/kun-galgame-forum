<script setup lang="ts">
import { kunSidebarRail } from '~/constants/layout'

const { hasReleaseToday } = useGalgameReleaseToday()
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
      variant="light"
      color="secondary"
      href="/galgame-quiz"
      aria-label="Galgame 题库"
      class-name="text-secondary-600 h-auto w-full flex-col gap-1 py-2"
    >
      <KunIcon name="lucide:library-big" class="text-xl" />
      <span class="text-[11px] leading-none">Gal 题库</span>
    </KunButton>
  </nav>
</template>
