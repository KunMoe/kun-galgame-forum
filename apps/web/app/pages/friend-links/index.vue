<script setup lang="ts">
const { links } = await useFriendLinks()

const description = `${kungal.titleShort} 的友情链接, 收录 ${links.value
  .map((link) => link.title)
  .join('、')} 等 Galgame 相关的优质网站与社区。`

useKunSeoMeta({
  title: '友情链接网站',
  description,
  articleAuthor: links.value.map((link) => link.url)
})
</script>

<template>
  <FriendLinksContainer />
</template>
