<script setup lang="ts">
defineProps<{ sources: KunNewsSource[] }>()
</script>

<template>
  <KunCard :is-transparent="false" padding="md" content-class="gap-4">
    <div class="flex items-center gap-2">
      <KunIcon name="lucide:handshake" class="text-primary text-lg" />
      <h2 class="font-medium">情报来源</h2>
    </div>

    <div v-for="(source, index) in sources" :key="source.key" class="space-y-2">
      <KunDivider v-if="index" class="mb-4" />

      <div class="flex items-center gap-2">
        <KunAvatar
          v-if="source.publisher"
          :user="source.publisher"
          size="sm"
          is-navigation
          class-name="shrink-0"
        />
        <KunLink
          v-if="source.homepage_url"
          :href="source.homepage_url"
          target="_blank"
          rel="noopener"
          color="default"
          underline="hover"
          class-name="font-medium"
        >
          {{ source.name }}
        </KunLink>
        <span v-else class="font-medium">{{ source.name }}</span>
      </div>

      <p v-if="source.attribution" class="text-default-400 text-xs leading-5">
        {{ source.attribution }}
      </p>

      <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
        <KunLink
          v-if="source.homepage_url"
          :href="source.homepage_url"
          target="_blank"
          rel="noopener"
          color="primary"
          size="sm"
          underline="hover"
          is-show-anchor-icon
        >
          站点主页
        </KunLink>
        <KunLink
          v-if="source.column_url"
          :href="source.column_url"
          target="_blank"
          rel="noopener"
          color="primary"
          size="sm"
          underline="hover"
          is-show-anchor-icon
        >
          专栏
        </KunLink>
      </div>
    </div>
  </KunCard>
</template>
