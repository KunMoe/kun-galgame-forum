import { defineStore } from 'pinia'
import { ref } from 'vue'
import { settingsCookieStorage } from '~/utils/settingsCookie'

export const usePersistGalgameCardStore = defineStore(
  'KUNGalgameCardDisplay',
  () => {
    const showPlatform = ref(true)
    const showRating = ref(true)
    const showViewLike = ref(true)
    const showUpdateTime = ref(true)
    const showNsfwBadge = ref(false)
    const showCompany = ref(true)
    // Shows whichever title the reader did not pick under the one they did —
    // the work's own title by default, the Chinese one under 优先显示日语原名.
    // Still named for the Japanese title it used to be: the key is persisted,
    // so renaming it resets the preference for everyone who has ever changed it.
    const showJapaneseName = ref(false)
    const isOpenInNewTab = ref(false)

    const reset = () => {
      showPlatform.value = true
      showRating.value = true
      showViewLike.value = true
      showUpdateTime.value = true
      showNsfwBadge.value = false
      showCompany.value = true
      showJapaneseName.value = false
      isOpenInNewTab.value = false
    }

    return {
      showPlatform,
      showRating,
      showViewLike,
      showUpdateTime,
      showNsfwBadge,
      showCompany,
      showJapaneseName,
      isOpenInNewTab,
      reset
    }
  },
  {
    persist: { storage: settingsCookieStorage() }
  }
)
