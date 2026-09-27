<script setup lang="ts">
useKunSeoMeta({
  title: 'RSS 订阅',
  description: `${kungal.titleShort}的 RSS 订阅: 新话题, 新发布的 Galgame 下载资源, Gal 情报与新工具, 也可以只订阅某一部 Galgame 或某一位用户, 支持 RSS 2.0, Atom 与 JSON Feed`
})
</script>

<template>
  <RssContainer />
</template>
