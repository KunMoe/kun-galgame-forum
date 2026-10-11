import { settle } from '#shared/utils/api/problem'
import {
  KUN_TAG_BLOCKS_MAX,
  type KunTagBlockMode,
  type KunTagBlockSet
} from '~/store/modules/tagBlocks'

const emptyBlocks = (): KunTagBlockSet => ({ tree: [], exact: [] })

const ascending = (ids: number[]) => [...ids].sort((a, b) => a - b).map(String)

const withMode = (
  set: KunTagBlockSet,
  id: number,
  mode: KunTagBlockMode | null
): KunTagBlockSet => {
  const next = {
    tree: set.tree.filter((x) => x !== id),
    exact: set.exact.filter((x) => x !== id)
  }
  if (mode) {
    next[mode].push(id)
  }
  return next
}

export const useTagBlocks = () => {
  const store = usePersistTagBlockStore()
  const userStore = usePersistUserStore()
  const api = useApiClient()

  const isSignedIn = computed(() => !!userStore.id)
  const isAccountLoaded = computed(
    () => store.accountUid !== 0 && store.accountUid === userStore.id
  )

  const blocks = computed<KunTagBlockSet>(() => {
    if (!isSignedIn.value) {
      return store.anonymous
    }
    return isAccountLoaded.value ? store.account : emptyBlocks()
  })

  const count = computed(
    () => blocks.value.tree.length + blocks.value.exact.length
  )

  const excludeQuery = computed(() => {
    const q: { exclude_tag_trees?: string[]; exclude_tag_ids?: string[] } = {}
    if (blocks.value.tree.length) {
      q.exclude_tag_trees = ascending(blocks.value.tree)
    }
    if (blocks.value.exact.length) {
      q.exclude_tag_ids = ascending(blocks.value.exact)
    }
    return q
  })

  const modeOf = (id: number): KunTagBlockMode | null => {
    if (blocks.value.tree.includes(id)) {
      return 'tree'
    }
    return blocks.value.exact.includes(id) ? 'exact' : null
  }

  const refresh = async () => {
    if (!isSignedIn.value) {
      return
    }
    const uid = userStore.id
    const result = await settle(api.GET('/me/blocked-tags'))
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    const next = emptyBlocks()
    for (const item of result.data.items) {
      next[item.mode].push(Number(item.id))
    }
    store.account = next
    store.accountUid = uid
  }

  const write = (next: KunTagBlockSet) => {
    if (isSignedIn.value) {
      store.account = next
      store.accountUid = userStore.id
    } else {
      store.anonymous = next
    }
  }

  const block = async (id: number, mode: KunTagBlockMode = 'tree') => {
    if (isSignedIn.value && !isAccountLoaded.value) {
      await refresh()
    }
    if (modeOf(id) === null && count.value >= KUN_TAG_BLOCKS_MAX) {
      useMessage(`最多屏蔽 ${KUN_TAG_BLOCKS_MAX} 个标签`, 'warn')
      return false
    }
    if (isSignedIn.value) {
      const result = await settle(
        api.PUT('/me/blocked-tags/{tag_id}', {
          params: { path: { tag_id: String(id) } },
          body: { mode }
        })
      )
      if (!result.ok) {
        reportProblem(result.problem)
        return false
      }
    }
    write(withMode(blocks.value, id, mode))
    return true
  }

  const unblock = async (id: number) => {
    if (isSignedIn.value) {
      if (!isAccountLoaded.value) {
        await refresh()
      }
      const result = await settle(
        api.DELETE('/me/blocked-tags/{tag_id}', {
          params: { path: { tag_id: String(id) } }
        })
      )
      if (!result.ok) {
        reportProblem(result.problem)
        return false
      }
    }
    write(withMode(blocks.value, id, null))
    return true
  }

  return {
    isSignedIn,
    blocks,
    count,
    excludeQuery,
    modeOf,
    refresh,
    block,
    unblock
  }
}
