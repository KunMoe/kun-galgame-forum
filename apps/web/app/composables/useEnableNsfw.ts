export const useEnableNsfw = () => {
  const { isSignedIn, setAnonymousNsfw } = useContentStance()
  const { open: openSettingPanel } = useSettingPanel()

  return () => {
    if (isSignedIn.value) {
      openSettingPanel('content')
      return
    }
    setAnonymousNsfw(true)
    location.reload()
  }
}
