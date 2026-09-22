<script setup lang="ts">
const { showKUNGalgameBackLoli } = storeToRefs(usePersistSettingsStore())
const { showKUNGalgamePanel } = storeToRefs(useTempSettingStore())

const settingTabs = [
  { value: 'appearance', textValue: '外观', icon: 'lucide:palette' },
  { value: 'background', textValue: '背景', icon: 'lucide:image' },
  { value: 'galgame', textValue: 'Galgame', icon: 'lucide:gamepad-2' },
  { value: 'feed', textValue: '主页', icon: 'lucide:house' },
  { value: 'content', textValue: '内容', icon: 'lucide:shield-alert' },
  { value: 'general', textValue: '通用', icon: 'lucide:settings-2' }
]
const { activeTab } = useSettingPanel()
</script>

<template>
  <KunModal
    :model-value="showKUNGalgamePanel"
    @update:model-value="(value) => (showKUNGalgamePanel = value)"
    inner-class-name="w-[92vw] sm:max-w-3xl"
  >
    <div class="space-y-4">
      <div class="flex items-center gap-2 text-lg">
        <span>设置面板</span>

        <KunTooltip class-name="flex" text="设置面板帮助" position="bottom">
          <KunLink
            to="/doc/setting-panel-help"
            color="default"
            class="hover:text-primary"
          >
            <KunIcon name="lucide:circle-help" />
          </KunLink>
        </KunTooltip>
      </div>

      <div class="sm:hidden">
        <KunTab
          v-model="activeTab"
          :items="settingTabs"
          variant="light"
          color="primary"
          size="sm"
          scrollable
        />
      </div>

      <div class="flex flex-col gap-4 sm:flex-row sm:items-start">
        <div class="hidden shrink-0 sm:block sm:w-24">
          <KunTab
            v-model="activeTab"
            :items="settingTabs"
            orientation="vertical"
            variant="underlined"
            color="primary"
            full-width
          />
        </div>

        <div class="min-w-0 flex-1 space-y-4 sm:min-h-75">
          <div v-show="activeTab === 'appearance'" class="space-y-4">
            <KunSettingPanelComponentsMode />
            <KunSettingPanelComponentsConfigItems />
          </div>

          <div v-show="activeTab === 'background'">
            <KunSettingPanelComponentsBackground />
          </div>

          <div v-show="activeTab === 'galgame'">
            <KunSettingPanelComponentsGalgame />
          </div>

          <div v-show="activeTab === 'feed'">
            <KunSettingPanelComponentsFeedTabs />
          </div>

          <div v-show="activeTab === 'content'">
            <KunSettingPanelComponentsNSFW />
          </div>

          <div v-show="activeTab === 'general'" class="space-y-5">
            <div class="flex items-start justify-between gap-4">
              <div class="space-y-0.5">
                <p class="text-default-700 font-medium">显示琥珀</p>
                <p class="text-default-500 text-sm">
                  在网站右下角显示这只可爱的看板娘琥珀；关闭后页面角落不再出现她。
                </p>
              </div>
              <KunSwitch v-model="showKUNGalgameBackLoli" class="shrink-0" />
            </div>

            <KunSettingPanelComponentsReset />
          </div>
        </div>

        <KunSettingPanelComponentsLoli />
      </div>
    </div>
  </KunModal>
</template>
