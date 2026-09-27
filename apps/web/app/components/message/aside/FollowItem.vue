<script setup lang="ts"></script>

<template>
  <KunLink
    color="default"
    underline="none"
    class-name="hover:bg-primary/20 flex cursor-pointer flex-nowrap gap-3 rounded-lg p-2 transition-colors hover:opacity-80"
    to="/message/follow"
  >
    <div
      class="bg-default-100 flex h-12 w-12 shrink-0 items-center justify-center rounded-full"
    >
      <KunIcon name="lucide:bell" class="text-default-500 text-xl" />
    </div>
    <div class="flex w-full flex-col justify-center">
      <span class="font-bold">关注的话题与评论区</span>
      <div class="flex items-center justify-between text-sm">
        <span class="text-default-500 line-clamp-1">
          管理你关注的话题和评论区
        </span>
      </div>
    </div>
  </KunLink>
</template>
