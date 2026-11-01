<script setup lang="ts">
import type { SiteRating } from '#shared/utils/api/schemas'
import { ratingSiteLabel } from '~/constants/ratedRanking'
import {
  formatSiteScore,
  formatSiteValue,
  formatTopShare,
  formatVotes
} from './format'

defineProps<{ ratings: SiteRating[] }>()
</script>

<template>
  <div class="border-default-200 overflow-x-auto rounded-lg border">
    <table class="w-full min-w-120 text-xs tabular-nums">
      <thead class="bg-default-100 text-default-500">
        <tr>
          <th class="px-3 py-2 text-left font-medium">站点</th>
          <th class="px-3 py-2 text-right font-medium">原分</th>
          <th class="px-3 py-2 text-right font-medium">估计 (90% 区间)</th>
          <th class="px-3 py-2 text-right font-medium">中位数</th>
          <th class="px-3 py-2 text-right font-medium">票数</th>
          <th class="px-3 py-2 text-right font-medium">站内排位</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="r in ratings"
          :key="r.site"
          class="border-default-200 border-t"
        >
          <td class="text-default-700 px-3 py-2 font-medium">
            {{ ratingSiteLabel(r.site) }}
          </td>
          <td class="px-3 py-2 text-right">
            <span class="text-foreground font-semibold">
              {{ formatSiteScore(r.published_score, r.scale_max) }}
            </span>
            <span class="text-default-400">/{{ r.scale_max }}</span>
          </td>
          <td class="text-default-600 px-3 py-2 text-right">
            {{ formatSiteValue(r, r.estimate) }}
            <span class="text-default-400">
              ({{ formatSiteValue(r, r.lower) }} –
              {{ formatSiteValue(r, r.upper) }})
            </span>
          </td>
          <td class="text-default-600 px-3 py-2 text-right">
            {{ r.median === null ? '—' : formatSiteValue(r, r.median) }}
          </td>
          <td class="text-default-600 px-3 py-2 text-right">
            {{ formatVotes(r.vote_count) }}
          </td>
          <td class="text-default-600 px-3 py-2 text-right">
            {{ formatTopShare(r.percentile) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
