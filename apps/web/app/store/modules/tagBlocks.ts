import { defineStore } from 'pinia'
import { ref } from 'vue'
import { settingsCookieStorage } from '~/utils/settingsCookie'

export type KunTagBlockMode = 'tree' | 'exact'

export interface KunTagBlockSet {
  tree: number[]
  exact: number[]
}

export const KUN_TAG_BLOCKS_MAX = 100

export const usePersistTagBlockStore = defineStore(
  'KUNGalgameTagBlocks',
  () => {
    const anonymous = ref<KunTagBlockSet>({ tree: [], exact: [] })
    const accountUid = ref(0)
    const account = ref<KunTagBlockSet>({ tree: [], exact: [] })
    return { anonymous, accountUid, account }
  },
  { persist: { storage: settingsCookieStorage() } }
)
