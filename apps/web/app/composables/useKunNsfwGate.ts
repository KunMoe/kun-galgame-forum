import type { ApiResult } from '#shared/utils/api/problem'

// Being signed in used to be enough on its own, which showed every NSFW
// detail page to a reader who had never asked for one. The account's stance
// decides now; 模糊 lets the page through and masks the imagery instead.
// The gate was also decided once at setup, and setup can run before the data
// lands (a hydration key miss): it never closed and an NSFW work rendered for
// a hide-stance account (2026-09-24). So it follows the data and the stance.
export const useKunNsfwGate = (isNsfw: () => boolean | undefined) => {
  const { allowsNsfw } = useContentStance()
  const revealed = ref(false)
  return {
    gated: computed(() => !!isNsfw() && !allowsNsfw.value && !revealed.value),
    // What a revealed page's own lists ask with: an adult tag or trait has no
    // works or characters at include_nsfw=false, only a 404 or 400.
    includeNsfw: computed(() => allowsNsfw.value || revealed.value),
    reveal: () => {
      revealed.value = true
    }
  }
}

// An adult tag or trait answers 404 to a reader who hides adult content. Asked
// again with include_nsfw it tells an adult record from a missing one, so the
// page can offer the gate the work pages do instead of answering 404.
export const readWithNsfwProbe = async <T>(
  read: (includeNsfw: boolean) => Promise<ApiResult<T>>,
  allowsNsfw: boolean
): Promise<ApiResult<T>> => {
  const first = await read(allowsNsfw)
  if (first.ok || allowsNsfw || first.problem.status !== 404) {
    return first
  }
  const probe = await read(true)
  return probe.ok ? probe : first
}
