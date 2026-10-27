<script setup lang="ts">
defineProps<{
  kind: 'direct' | 'netdisk'
}>()
</script>

<template>
  <KunInfo
    color="primary"
    variant="flat"
    :title="
      kind === 'direct'
        ? '直链下载由 LetMoe·一起萌 提供'
        : '这些网盘链接由 LetMoe·一起萌 用户发布'
    "
  >
    <p class="text-sm">
      以下资源来自
      <GalgameResourceLetmoeLink />
      ，与鲲 Galgame
      论坛是同一个账号、同一个作品编号。下载失败、链接失效、文件损坏或合成版打不开，请到对应资源的
      LetMoe 页面「报错与维护」反馈，不要在本站报告，本站无法处理 LetMoe
      的资源。
    </p>
  </KunInfo>
</template>
