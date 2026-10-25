<script setup lang="ts">
import { KUN_FOOTER_COLUMNS, KUN_FOOTER_ORG } from '~/config/footer'
</script>

<template>
  <footer class="text-default-500 space-y-2.5 text-center text-xs">
    <nav
      class="grid grid-cols-[repeat(3,auto)] justify-between gap-x-2"
      aria-label="鲲 Galgame 与 NextMoe 站点"
    >
      <div
        v-for="(column, index) in KUN_FOOTER_COLUMNS"
        :key="index"
        class="flex flex-col items-center gap-1"
      >
        <KunLink
          v-for="site in column"
          :key="site.link"
          size="sm"
          :to="site.link"
          target="_blank"
          color="default"
          underline="hover"
          class-name="text-default-500 whitespace-nowrap"
        >
          {{ site.label }}
        </KunLink>
      </div>
    </nav>

    <div class="flex flex-wrap items-center justify-center gap-1.5">
      <a
        :href="kungal.domain.main"
        target="_blank"
        class="flex items-center"
        :aria-label="kungal.titleShort"
      >
        <KunImage
          class="size-5 rounded-md"
          src="/favicon.webp"
          :alt="kungal.titleShort"
        />
      </a>
      <span>© 2023 ~ Forever! {{ kungal.titleShort }}</span>
    </div>

    <div class="flex flex-wrap items-center justify-center gap-1">
      <KunLink
        :to="kungal.github"
        size="sm"
        target="_blank"
        rel="noopener noreferrer"
      >
        GitHub
      </KunLink>
      <span>开源 - 来加入</span>
      <KunLink
        :to="KUN_FOOTER_ORG.link"
        size="sm"
        target="_blank"
        rel="noopener noreferrer"
      >
        {{ KUN_FOOTER_ORG.label }}
      </KunLink>
      <span>吧</span>
    </div>
  </footer>
</template>
