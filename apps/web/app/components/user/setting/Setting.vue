<script setup lang="ts">
import type { UserProfile } from '#shared/utils/api/schemas'

defineProps<{
  user: UserProfile
}>()
</script>

<template>
  <div class="h-full w-full space-y-6">
    <UserSettingAvatar />
    <UserSettingBio :user="user" />
    <UserSettingUsername />
    <UserSettingEmail />
    <UserSettingPassword />
    <UserSettingProfileVisibility />
    <UserSettingActivityVisibility />
    <UserSettingPrivacySwitch
      field="is_play_records_public"
      anchor="play-records"
      title="公开我的游玩记录"
    >
      <p>
        你在鲲 Galgame 和 NextMoe
        各站标记的游玩状态（想玩、在玩、玩过、搁置、弃坑）和游玩时长，会显示在你个人主页的「游戏库」和「动态」里，所有人都能看到。
      </p>
      <p>关闭后只有你自己能看到。评分一直是公开的，不受这个开关影响。</p>
    </UserSettingPrivacySwitch>
    <UserSettingPrivacySwitch
      field="is_followed_companies_public"
      anchor="followed-companies"
      title="公开我关注的会社"
    >
      <p>
        打开后，你在鲲 Galgame 和 NextMoe
        各站关注的会社会显示在你个人主页的「关注 →
        会社」里，之后新关注的会社也会出现在你的动态中。
      </p>
      <p>默认不公开。关闭后只有你自己能看到，动态里的关注记录也会撤回。</p>
    </UserSettingPrivacySwitch>
    <KunCard :is-hoverable="false" content-class="space-y-4">
      <MessageNotificationPreference />
    </KunCard>
  </div>
</template>
