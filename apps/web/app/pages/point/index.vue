<script setup lang="ts">
definePageMeta({
  pageTransition: { name: 'kun-page', mode: 'out-in' }
})

useKunSeoMeta({
  title: '萌萌点说明',
  description:
    '萌萌点是什么、平时怎么赚、又会在哪里花掉——用大白话讲清楚本站的萌萌点收支规则。'
})
</script>

<template>
  <div class="min-w-0">
    <PointContainer />
  </div>
</template>
