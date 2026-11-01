<script setup lang="ts">
import { isRatingSite, ratedRankingPageMeta } from '~/constants/ratedRanking'

definePageMeta({
  key: 'ranking-rated',
  validate: (route) => !route.params.site || isRatingSite(route.params.site)
})

const route = useRoute()
const site = computed(() =>
  isRatingSite(route.params.site) ? route.params.site : null
)

useKunSeoMeta(() => ratedRankingPageMeta(site.value))
</script>

<template>
  <RankingRatedContainer :site="site" />
</template>
