<script setup lang="ts">
import {
  topicSortItem,
  galgameSortItem,
  userSortItem,
  rankingPageTabs,
  rankingPageMetaData
} from '~/constants/ranking'
import {
  topicRankingPageData,
  galgameRankingPageData,
  userRankingPageData
} from '~/components/ranking/pageData'
import type { KunSelectOption } from '@kungal/ui-vue'

const rankingTabs = new Set(rankingPageTabs.map((tab) => tab.value))

const activeTab = computed(() => {
  const segment = useRoute().path.split('/').filter(Boolean).pop()
  return segment && rankingTabs.has(segment) ? segment : 'user'
})

const currentSortItems = computed(() => {
  switch (activeTab.value) {
    case 'topic':
      return topicSortItem
    case 'galgame':
      return galgameSortItem
    case 'user':
    default:
      return userSortItem
  }
})

const sortOptions = computed(() => {
  return currentSortItems.value.map((item) => ({
    value: item.sort,
    label: item.label,
    icon: item.icon
  }))
})
</script>

<template>
  <div class="space-y-3">
    <div class="space-y-3">
      <KunHeader
        :name="rankingPageMetaData[activeTab]!.title"
        :description="rankingPageMetaData[activeTab]!.description"
      />

      <div class="flex items-center justify-between">
        <KunTab
          :model-value="activeTab"
          :items="rankingPageTabs"
          variant="underlined"
          color="primary"
        />

        <div class="w-48">
          <KunSelect
            v-if="activeTab === 'topic'"
            v-model="topicRankingPageData.sort"
            :options="
              sortOptions as KunSelectOption<typeof topicRankingPageData.sort>[]
            "
          />
          <KunSelect
            v-if="activeTab === 'galgame'"
            v-model="galgameRankingPageData.sort"
            :options="
              sortOptions as KunSelectOption<
                typeof galgameRankingPageData.sort
              >[]
            "
          />
          <KunSelect
            v-if="activeTab === 'user'"
            v-model="userRankingPageData.sort"
            :options="
              sortOptions as KunSelectOption<typeof userRankingPageData.sort>[]
            "
          />
        </div>
      </div>
    </div>

    <NuxtPage />
  </div>
</template>
