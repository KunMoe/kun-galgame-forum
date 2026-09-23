<script setup lang="ts">
import { userSortItem } from '~/constants/ranking'
import { toKunUser } from '~/utils/userRef'
import { RANKING_LIMIT, userRankingPageData, getRankClasses } from './pageData'

const { data } = await useApi(
  () => `ranking-users:${userRankingPageData.sort}`,
  (client, { signal }) =>
    client.GET('/rankings/users', {
      params: {
        query: { sort: userRankingPageData.sort, limit: RANKING_LIMIT }
      },
      signal
    })
)

const icon = computed(
  () =>
    userSortItem.find((i) => i.sort === userRankingPageData.sort)?.icon ?? ''
)
</script>

<template>
  <ul v-if="data" class="space-y-3">
    <li v-for="(entry, index) in data.items" :key="entry.member.id">
      <KunLink
        color="default"
        underline="none"
        :to="`/user/${entry.member.id}`"
        :class-name="
          cn(
            'relative flex items-center gap-3 rounded-xl border p-3 transition-colors',
            getRankClasses(index)
          )
        "
      >
        <RankingMedal :index="index" />

        <KunAvatar
          :user="toKunUser(entry.member)"
          size="lg"
          :is-navigation="false"
        />
        <div class="flex-1 overflow-hidden">
          <h3 class="flex items-center gap-2 font-semibold">
            <span>{{ toKunUser(entry.member).name }}</span>
            <div class="flex shrink-0 items-center gap-2 sm:hidden">
              <KunIcon :name="icon" class="text-primary h-5 w-5" />
              <span class="text-default-500 text-sm font-normal">
                {{ entry.metric_value }}
              </span>
            </div>
          </h3>
          <p class="text-default-500 truncate text-sm">
            {{ entry.bio || '这只笨蛋萝莉居然不写签名! 八嘎!' }}
          </p>
        </div>

        <div class="hidden shrink-0 items-center gap-2 sm:flex">
          <KunIcon :name="icon" class="text-primary h-5 w-5" />
          <span class="text-lg font-medium">{{ entry.metric_value }}</span>
        </div>
      </KunLink>
    </li>
  </ul>
</template>
