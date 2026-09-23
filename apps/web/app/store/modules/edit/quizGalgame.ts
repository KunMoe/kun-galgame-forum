import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface RecentQuizGalgame {
  id: string
  name: string
  coverUrl?: string
  thumbhash?: string
  isNsfw?: boolean
}

const MAX_RECENT = 8

export const usePersistQuizGalgameStore = defineStore(
  'KUNGalgameQuizGalgame',
  () => {
    const recent = ref<RecentQuizGalgame[]>([])

    const add = (game: RecentQuizGalgame) => {
      const id = String(game.id)
      if (!id) {
        return
      }
      recent.value = [
        { ...game, id },
        ...recent.value.filter((g) => String(g.id) !== id)
      ].slice(0, MAX_RECENT)
    }

    const remove = (id: string) => {
      recent.value = recent.value.filter((g) => String(g.id) !== id)
    }

    return { recent, add, remove }
  },
  { persist: { storage: piniaPluginPersistedstate.localStorage() } }
)
