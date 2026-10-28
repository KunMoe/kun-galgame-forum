<script setup lang="ts">
import { TOPIC_TOC } from '~/composables/topic/useTopicTOC'
import {
  useCenterActiveToc,
  useTopicRailPhase
} from '~/composables/topic/useTopicRail'

const props = defineProps<{
  user: {
    id: number
    name: string
    avatar: string
    moemoepoint: number
  }
}>()

const user = computed(() => props.user)
const { id: currentUserId } = storeToRefs(usePersistUserStore())
const isSelf = computed(() => currentUserId.value === user.value.id)

const { headings, activeIds } = inject(TOPIC_TOC)!

const headingItems = computed(() =>
  headings.value.filter((h) => h.type === 'heading')
)
const replyItems = computed(() =>
  headings.value.filter((h) => h.type === 'reply')
)

const { isPastPoll, isPastReplyTool, scrollToReplyTool } = useTopicRailPhase()

const railRef = ref<HTMLElement | null>(null)
useCenterActiveToc(railRef, activeIds)
</script>

<template>
  <div
    ref="railRef"
    class="sticky top-20 hidden max-h-[calc(100dvh-6rem)] w-52 shrink-0 flex-col gap-3 overflow-hidden lg:flex"
  >
    <div
      class="grid shrink-0 transition-all duration-500 ease-out"
      :class="
        isPastPoll ? 'grid-rows-[0fr] opacity-0' : 'grid-rows-[1fr] opacity-100'
      "
    >
      <div class="flex min-h-0 flex-col items-center gap-3 overflow-hidden">
        <TopicDetailAuthorCard :user="user" />

        <div
          v-if="headingItems.length"
          class="scrollbar-hide max-h-[30vh] w-full overflow-y-auto"
        >
          <h3 class="mb-2 text-sm font-medium">文章目录</h3>
          <TopicDetailTocSection
            :items="headingItems"
            :active-ids="activeIds"
          />
        </div>
      </div>
    </div>

    <div v-if="replyItems.length" class="flex min-h-0 flex-1 flex-col">
      <Transition
        enter-active-class="transition-all duration-300 ease-out"
        enter-from-class="-translate-y-1 opacity-0"
        leave-active-class="transition-all duration-200 ease-in"
        leave-to-class="-translate-y-1 opacity-0"
      >
        <div v-if="isPastPoll" class="mb-2 flex shrink-0 items-center gap-1">
          <KunLink
            :to="`/user/${user.id}`"
            :aria-label="user.name"
            underline="none"
            color="default"
            class-name="hover:bg-default-100 flex min-w-0 flex-1 items-center gap-2.5 rounded-lg p-1.5 transition-colors"
          >
            <KunAvatar
              :user="user"
              :is-navigation="false"
              size="original"
              class-name="size-10 shrink-0"
              image-class-name="size-10"
            />
            <div class="flex min-w-0 flex-col">
              <span class="truncate text-base font-medium">{{
                user.name
              }}</span>
              <span class="text-secondary flex items-center gap-1 text-sm">
                <KunIcon class="text-inherit" name="lucide:lollipop" />
                {{ user.moemoepoint }}
              </span>
            </div>
          </KunLink>
          <UserFollowButton v-if="!isSelf" :user-id="String(user.id)" compact />
        </div>
      </Transition>

      <Transition
        enter-active-class="transition-all duration-300 ease-out"
        enter-from-class="-translate-y-1 opacity-0"
        leave-active-class="transition-all duration-200 ease-in"
        leave-to-class="-translate-y-1 opacity-0"
      >
        <div v-if="isPastReplyTool" class="shrink-0 pb-2">
          <KunButton
            class-name="w-full justify-center"
            size="sm"
            variant="flat"
            @click="scrollToReplyTool"
          >
            <KunIcon name="lucide:arrow-up-to-line" />
            回到回复顶部
          </KunButton>
        </div>
      </Transition>

      <div class="scrollbar-hide min-h-0 flex-1 overflow-y-auto">
        <h3 class="mb-2 text-sm font-medium">回复列表</h3>
        <TopicDetailTocSection :items="replyItems" :active-ids="activeIds" />
      </div>
    </div>
  </div>
</template>
