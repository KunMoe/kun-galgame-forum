<script setup lang="ts">
const route = useRoute()
const year = Number(route.params.year)
const month = Number(route.params.month)

definePageMeta({
  validate: (route) => {
    const year = Number(route.params.year)
    const month = Number(route.params.month)
    return (
      Number.isInteger(year) &&
      year >= 1970 &&
      year <= 9999 &&
      Number.isInteger(month) &&
      month >= 1 &&
      month <= 12
    )
  }
})

useKunSeoMeta({
  title: `${year} 年 ${month} 月 Galgame 业界新闻 - 新作情报、发售消息与专栏存档`,
  description: `${year} 年 ${month} 月 Galgame 业界新闻存档, 收录月幕 Galgame、Galgame 批评等合作站点授权转载的新作情报、发售消息与深度专栏, 以及 NextMoe 用户投稿的情报, 可按发布日期筛选。`
})
</script>

<template>
  <NewsMonthContainer :year="year" :month="month" />
</template>
