<script setup lang="ts">
const { data, problem, pending, form, isSaving, save } = useAdminSettingsForm({
  app_min_version: '0.1.0',
  app_latest_version: '0.1.0',
  app_release_notes: '',
  app_android_download_url: '',
  app_ios_download_url: '',
  app_windows_download_url: '',
  app_linux_download_url: '',
  app_android_package_url: '',
  app_android_package_size: 0,
  app_android_package_sha256: ''
})
</script>

<template>
  <AdminSettingSection
    title="App 版本"
    description="App 启动时读取的版本门。低于最低版本的安装会被强制更新, 所以最低版本不能高于最新版本"
    :problem="problem"
    :pending="pending"
    :is-ready="!!data"
    :is-saving="isSaving"
    @save="save"
  >
    <div class="grid gap-4 sm:grid-cols-2">
      <KunInput v-model="form.app_min_version" label="最低可用版本" />
      <KunInput v-model="form.app_latest_version" label="最新版本" />
    </div>
    <KunTextarea
      v-model="form.app_release_notes"
      label="最新版本的更新说明"
      :rows="4"
      :maxlength="2000"
    />

    <div class="space-y-2">
      <p class="text-sm font-medium">各平台下载地址</p>
      <p class="text-default-500 text-xs">
        留空时指向 https://www.kungal.com/app 下载页
      </p>
      <div class="grid gap-4 sm:grid-cols-2">
        <KunInput v-model="form.app_android_download_url" label="Android" />
        <KunInput v-model="form.app_ios_download_url" label="iOS" />
        <KunInput v-model="form.app_windows_download_url" label="Windows" />
        <KunInput v-model="form.app_linux_download_url" label="Linux" />
      </div>
    </div>

    <div class="space-y-2">
      <p class="text-sm font-medium">Android 应用内更新包</p>
      <p class="text-default-500 text-xs">
        最新版本的 .apk, 三项同时填写或同时留空。大小以字节计, SHA-256
        为小写十六进制 (sha256sum 的输出)。与最新版本不符的包会被 App 拒绝安装
      </p>
      <KunInput
        v-model="form.app_android_package_url"
        label=".apk 地址 (https)"
      />
      <div class="grid gap-4 sm:grid-cols-2">
        <KunInput
          v-model="form.app_android_package_size"
          type="number"
          label="文件大小 (字节)"
        />
        <KunInput v-model="form.app_android_package_sha256" label="SHA-256" />
      </div>
    </div>
  </AdminSettingSection>
</template>
