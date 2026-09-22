<script setup lang="ts">
defineProps<{ noun: string }>()
const emit = defineEmits<{ reveal: [] }>()

const { isSignedIn } = useContentStance()
const { open } = useSettingPanel()
</script>

<template>
  <KunCard :is-hoverable="false" :is-transparent="false">
    <div class="flex flex-col items-start gap-3">
      <div class="flex items-center gap-2">
        <KunIcon name="lucide:shield-alert" class="text-danger size-5" />
        <p class="text-foreground font-medium">
          这个{{ noun }}含有 NSFW 内容
        </p>
      </div>

      <p class="text-default-500 text-sm">
        <template v-if="isSignedIn">
          您的账号当前将成人向内容设置为「隐藏」，改成「模糊」或「显示」后即可查看。
        </template>
        <template v-else>
          您需要点击确认以显示这个{{ noun }}。
        </template>
      </p>

      <KunButton v-if="isSignedIn" @click="open('content')">
        前往内容设置
      </KunButton>
      <KunButton v-else @click="emit('reveal')">确认显示</KunButton>
    </div>
  </KunCard>
</template>
