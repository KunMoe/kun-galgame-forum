<script setup lang="ts">
import {
  topicSortItem,
  userSortItem,
  rankingPageTabs,
  rankingPageMetaData
} from '~/constants/ranking'
import {
  topicRankingPageData,
  userRankingPageData
} from '~/components/ranking/pageData'
import type { KunSelectOption } from '@kungal/ui-vue'

const route = useRoute()

const activeTab = computed(() => {
  const segment = route.path.split('/').filter(Boolean)[1]
  return segment === 'topic' || segment === 'user' ? segment : 'galgame'
})

const currentSortItems = computed(() => {
  switch (activeTab.value) {
    case 'topic':
      return topicSortItem
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
    <div v-if="activeTab !== 'galgame'" class="space-y-3">
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
