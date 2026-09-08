<script setup lang="ts">
import type { KUN_ADMIN_PAGE_ROUTE_TYPE } from '~/constants/admin'

useKunDisableSeo('管理系统')

const route = useRoute()
const pageType = computed(() => {
  const routeType = route.path.split('/').pop()
  return routeType as KUN_ADMIN_PAGE_ROUTE_TYPE
})

const { items } = useAdminNav()

const adminNavItems = computed(() =>
  items.value.map((item) => ({
    value: item.to ?? item.router!,
    textValue: item.label,
    icon: item.icon
  }))
)

const goto = (value: string) =>
  navigateTo(value.startsWith('/') ? value : `/admin/${value}`)
</script>

<template>
  <div class="flex flex-col gap-3 sm:flex-row">
    <!-- The nav used to exist only as the `sm:block` column below, so a phone
         got no navigation at all: /admin lands on the first permitted page and
         there was nothing to switch with. A horizontal KunTab contains its own
         overflow, so all 11 entries scroll inside the strip. -->
    <div class="sm:hidden">
      <KunTab
        :model-value="pageType"
        :items="adminNavItems"
        variant="underlined"
        color="primary"
        @update:model-value="goto"
      />
    </div>

    <div class="hidden w-48 shrink-0 sm:block">
      <KunTab
        :model-value="pageType"
        :items="adminNavItems"
        orientation="vertical"
        variant="underlined"
        color="primary"
        size="lg"
        full-width
        @update:model-value="goto"
      />
    </div>

    <NuxtPage />
  </div>
</template>
