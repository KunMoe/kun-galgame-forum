<script setup lang="ts">
import type { Website } from '#shared/utils/api/schemas'

defineProps<{
  data: Website
}>()
</script>

<template>
  <KunCard :is-transparent="false" :is-hoverable="false" class-name="p-6">
    <h3 class="text-default-900 mb-4 text-lg font-semibold">统计信息</h3>
    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <span class="text-default-600">总访问量</span>
        <span class="text-default-900 font-semibold">
          {{ formatNumber(data.view_count) }}
        </span>
      </div>
      <div class="flex items-center justify-between">
        <span class="text-default-600">点赞数</span>
        <span class="text-default-900 font-semibold">
          {{ formatNumber(data.like_count) }}
        </span>
      </div>
      <div class="flex items-center justify-between">
        <span class="text-default-600">收藏数</span>
        <span class="text-default-900 font-semibold">
          {{ formatNumber(data.favorite_count) }}
        </span>
      </div>
      <div class="flex items-center justify-between">
        <span class="text-default-600">网站建立时间</span>
        <span class="text-default-900 font-semibold">
          {{ data.founded }}
        </span>
      </div>
      <div class="flex items-center justify-between">
        <span class="text-default-600">网站索引时间</span>
        <span class="text-default-900 font-semibold">
          <KunTime :time="data.created_at" type="datetime" show-year />
        </span>
      </div>
    </div>
  </KunCard>
</template>
