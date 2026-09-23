<script setup lang="ts">
import { KUN_DOC_CATEGORY_MAP } from '~/constants/doc'

const { docs: articles } = await useAllDocs('position_asc')
</script>

<template>
  <div class="min-h-[calc(100dvh-6rem)] space-y-6">
    <KunHeader
      name="Galgame 帮助文档"
      description="如果您在 Galgame 发布, Galgame 交流, Galgame 资源 等方面有任何的问题, 或者想要联系我们, 都可以查看此界面的帮助文档"
    />

    <div
      class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      v-if="articles.length"
    >
      <KunCard
        :is-transparent="false"
        v-for="post in articles"
        :key="post.id"
        :href="`/doc/${post.slug}`"
        content-class="space-y-3"
      >
        <div class="flex items-center gap-3 text-sm">
          <KunChip color="default">
            {{ KUN_DOC_CATEGORY_MAP[post.doc_category] }}
          </KunChip>

          <time :datetime="post.published_at" class="text-default-500">
            <KunTime :time="post.published_at" type="date" show-year />
          </time>
        </div>

        <div class="group relative h-full space-y-3">
          <img
            :alt="post.title"
            class="rounded-lg"
            :src="post.banner?.url || '/kungalgame.webp'"
            width="100%"
            height="100%"
          />

          <h2
            class="group-hover:text-primary line-clamp-2 text-lg leading-6 font-semibold"
          >
            {{ post.title }}
          </h2>

          <p class="text-default-500 line-clamp-3 text-sm leading-6">
            {{ post.description }}
          </p>
        </div>
      </KunCard>
    </div>

    <KunNull v-else description="暂时没有找到任何文档" />
  </div>
</template>
