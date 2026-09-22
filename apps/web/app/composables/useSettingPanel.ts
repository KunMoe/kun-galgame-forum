export const useSettingPanel = () => {
  const { showKUNGalgamePanel } = storeToRefs(useTempSettingStore())
  const activeTab = useState('kun-setting-panel-tab', () => 'appearance')

  const open = (tab?: string) => {
    if (tab) activeTab.value = tab
    showKUNGalgamePanel.value = true
  }

  return { activeTab, open }
}
