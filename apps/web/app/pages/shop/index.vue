<script setup lang="ts">
definePageMeta({
  pageTransition: { name: 'kun-page', mode: 'out-in' }
})

useKunSeoMeta({
  title: '萌萌点商店',
  description:
    '用萌萌点兑换 DLsite 优惠券、主页介绍、头像框等装扮。买到的东西属于你的 NextMoe 账号，在所有站点通用。'
})
</script>

<template>
  <div class="min-w-0">
    <ShopContainer />
  </div>
</template>
