<script setup lang="ts">
useKunSeoMeta({
  title: '话题列表',
  description: `${kungal.titleShort} 话题列表, Galgame 讨论, Galgame 攻略, 长文评价, 新作资讯, 游戏运行, 开发技术, ACGN, 日常杂谈`
})
</script>

<template>
  <TopicShell active="">
    <TopicContainer />
  </TopicShell>
</template>
