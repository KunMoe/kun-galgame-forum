import type { UserPlaytimeTotal } from '#shared/utils/api/schemas'

export const useUserPlaytimeTotal = (userId: MaybeRefOrGetter<string>) => {
  const { allowsNsfw, stanceKey } = useContentStance()
  const { data } = useKunApi<UserPlaytimeTotal>(
    () => `user-playtime-total:${toValue(userId)}:${stanceKey.value}`,
    (api, { signal }) =>
      api.GET('/users/{user_id}/playtime-total', {
        params: {
          path: { user_id: toValue(userId) },
          query: { include_nsfw: allowsNsfw.value }
        },
        signal
      })
  )
  return computed(() => data.value?.total_playtime_minutes || null)
}

export const playtimeHeadline = (minutes: number) =>
  minutes < 60
    ? { value: String(minutes), unit: '分钟' }
    : {
        value: Math.floor(minutes / 60).toLocaleString('zh-CN'),
        unit: '小时'
      }
