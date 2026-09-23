<script setup lang="ts">
import {
  FRIEND_LINK_CATEGORIES,
  FRIEND_LINK_STATE_CHIP
} from '~/constants/friendLink'

const { links } = await useFriendLinks()

const utmLink = useUtmLink()

const groups = computed(() =>
  FRIEND_LINK_CATEGORIES.map((category) => ({
    label: category.label,
    links: links.value.filter(
      (link) => link.friend_link_category === category.key
    )
  })).filter((group) => group.links.length > 0)
)
</script>

<template>
  <div class="space-y-6 pb-12">
    <KunHeader
      name="友情链接"
      description="我们非常重视合作, 因为在我们网站的建设过程中, 受到了来自世界各地朋友的帮助, 我们与下面的网站都是朋友关系, 我们衷心的为能和这些网站成为朋友感到荣幸"
    >
      <template #endContent>
        <b class="text-default-700">
          如果下列网站出现了任何的付费行为, 请随时联系我们撤销友情链接!
        </b>
      </template>
    </KunHeader>

    <template v-for="(group, index) in groups" :key="index">
      <h2 class="mb-4 text-2xl font-bold">
        {{ group.label }}
      </h2>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KunCard
          :is-transparent="false"
          v-for="friend in group.links"
          :key="friend.id"
          :href="utmLink(friend.url)"
          target="_blank"
          rel="noopener noreferrer"
        >
          <div class="mb-2 flex items-center gap-2">
            <span class="text-lg font-bold">
              {{ friend.title }}
            </span>
            <KunChip
              v-if="FRIEND_LINK_STATE_CHIP[friend.state]"
              :color="FRIEND_LINK_STATE_CHIP[friend.state]!.color"
            >
              {{ FRIEND_LINK_STATE_CHIP[friend.state]!.label }}
            </KunChip>
          </div>
          <div class="text-default-600 mb-3 text-sm">
            {{
              friend.description.length > 107
                ? `${truncateRunes(friend.description, 107)}...`
                : friend.description
            }}
          </div>
          <KunImage
            v-if="friend.banner"
            :src="friend.banner.url"
            class="h-auto w-full rounded-md"
          />
        </KunCard>
      </div>
    </template>

    <div class="flex flex-col items-center justify-center gap-3">
      <KunLink underline="always" to="/doc/contact">
        <h3 class="text-primary text-xl font-bold">加入我们</h3>
      </KunLink>

      <p class="text-default-600 text-sm">
        要加入我们, 请加入我们的群组, 提供您的网站链接
      </p>
    </div>
  </div>
</template>
