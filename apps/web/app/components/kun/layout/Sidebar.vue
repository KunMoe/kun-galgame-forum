<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    className?: string
    forceExpanded?: boolean
  }>(),
  { className: '', forceExpanded: false }
)

const isCollapsed = computed(() => !props.forceExpanded)

const links = [
  {
    name: 'GitHub',
    icon: 'lucide:github',
    to: kungal.github,
    target: '_blank',
    tooltip: 'GitHub 仓库地址'
  },
  {
    name: 'RSS',
    icon: 'lucide:rss',
    to: '/rss',
    tooltip: '话题, Galgame 资源与 Gal 情报 RSS 订阅'
  },
  {
    name: 'Telegram',
    icon: 'ph:telegram-logo',
    to: kungal.domain.telegram_group,
    target: '_blank',
    tooltip: '加入 Telegram 交流群'
  }
]
</script>

<template>
  <div
    :class="
      cn(
        'scrollbar-hide bg-content1 border-kun shadow-kun-sm fixed z-20 flex h-full shrink-0 -translate-x-1 flex-col justify-between rounded-none border-r p-0 transition-all duration-300 sm:backdrop-blur-[var(--kun-background-blur)]',
        isCollapsed ? 'w-20' : 'w-3xs overflow-y-scroll',
        // Mobile drawer (force-expanded) is a popup over a scrim → opaque, like
        // the other menus (see .kun-sidebar-drawer in styles/tailwindcss.css).
        forceExpanded && 'kun-sidebar-drawer',
        className
      )
    "
    @click.stop
  >
    <div class="space-y-3 p-3">
      <template v-if="!isCollapsed">
        <KunBrand :name="kungal.titleShort" />
        <NuxtLink to="/galgame-quiz">
          <div
            class="bg-secondary/10 text-secondary-600 hover:bg-secondary/20 flex items-center gap-2 rounded-lg px-3 py-2 text-xs transition-colors"
          >
            <KunIcon name="lucide:party-popper" class="shrink-0 text-base" />
            <span>Galgame 题库上线啦, 快来出题和回答问题吧</span>
          </div>
        </NuxtLink>
      </template>
      <template v-else>
        <KunLink
          class-name="flex justify-center items-center gap-0"
          underline="none"
          to="/"
        >
          <KunImage
            class="size-12 rounded-2xl"
            src="/favicon.webp"
            :alt="kungal.titleShort"
          />
        </KunLink>
      </template>

      <KunLayoutSidebarNSFWToggle :is-collapsed="isCollapsed" />

      <Transition name="sidebar-switch" mode="out-in">
        <template v-if="!isCollapsed">
          <KunLayoutSideBarNav />
        </template>
        <template v-else>
          <KunLayoutSideBarRail />
        </template>
      </Transition>
    </div>

    <div>
      <template v-if="!isCollapsed">
        <KunLayoutSideBarExternal />

        <div class="flex w-full justify-between px-7 py-6">
          <KunLink
            v-for="item in links"
            :key="item.name"
            underline="none"
            color="default"
            class-name="flex-col gap-0"
            :to="item.to"
            :target="item.target as '_blank'"
          >
            <KunIcon class="icon" :name="item.icon" />
            <span class="text-xs">{{ item.name }}</span>
          </KunLink>
        </div>
      </template>

      <template v-else>
        <div class="flex flex-col items-center gap-2 px-3 pb-4">
          <KunTooltip
            v-for="item in links"
            :key="item.name"
            :text="item.tooltip"
            position="right"
          >
            <KunButton
              :is-icon-only="true"
              variant="light"
              color="default"
              class-name="flex-col gap-0"
              :href="item.to"
              :target="item.target as '_blank'"
              :title="item.name"
            >
              <KunIcon class="icon text-xl" :name="item.icon" />
            </KunButton>
          </KunTooltip>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.sidebar-switch-enter-active,
.sidebar-switch-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}
.sidebar-switch-enter-from,
.sidebar-switch-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
