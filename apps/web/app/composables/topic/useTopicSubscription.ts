import { settle } from '#shared/utils/api/problem'
import type {
  TopicNotificationLevel,
  TopicSubscription
} from '#shared/utils/api/schemas'

const inflight = new Map<string, Promise<void>>()

export const TOPIC_NOTIFICATION_OPTIONS: {
  value: TopicNotificationLevel
  icon: string
  label: string
  description: string
}[] = [
  {
    value: 'watching',
    icon: 'lucide:bell-ring',
    label: '关注',
    description: '每条新回复都会通知你'
  },
  {
    value: 'normal',
    icon: 'lucide:bell',
    label: '默认',
    description: '只在有人 @ 你或评论你的回复时通知'
  },
  {
    value: 'muted',
    icon: 'lucide:bell-off',
    label: '静音',
    description: '不再收到这个话题的任何回复通知'
  }
]

const LEVEL_MESSAGE: Record<TopicNotificationLevel, string> = {
  watching: '已关注这个话题, 有新回复时会通知你',
  normal: '已恢复默认, 只在有人 @ 你时通知',
  muted: '已静音这个话题'
}

export const useTopicSubscription = (
  topicId: MaybeRefOrGetter<string | number>
) => {
  const api = useApiClient()
  const { id: currentUserId } = storeToRefs(usePersistUserStore())
  const entries = useState<Record<string, TopicSubscription>>(
    'topic-subscription',
    () => ({})
  )
  const topic = computed(() => String(toValue(topicId)))
  const key = computed(() => `${currentUserId.value}:${topic.value}`)
  const subscription = computed(() => entries.value[key.value] ?? null)
  const busy = ref(false)

  const store = (entryKey: string, next: TopicSubscription) => {
    entries.value[entryKey] = next
  }

  const load = async () => {
    if (!currentUserId.value || subscription.value) {
      return
    }
    const entryKey = key.value
    const pending =
      inflight.get(entryKey) ??
      settle(
        api.GET('/topics/{topic_id}/subscription', {
          params: { path: { topic_id: topic.value } }
        })
      )
        .then((result) => {
          if (result.ok) {
            store(entryKey, result.data)
          }
        })
        .finally(() => inflight.delete(entryKey))
    inflight.set(entryKey, pending)
    await pending
  }

  const setLevel = async (level: TopicNotificationLevel) => {
    if (!currentUserId.value) {
      useAuthModal().open()
      return
    }
    if (busy.value || subscription.value?.notification_level === level) {
      return
    }
    busy.value = true
    const entryKey = key.value
    const result = await settle(
      api.PUT('/topics/{topic_id}/subscription', {
        params: { path: { topic_id: topic.value } },
        body: { notification_level: level }
      })
    )
    busy.value = false
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    store(entryKey, result.data)
    useMessage(LEVEL_MESSAGE[level], 'success')
  }

  const markRead = async (floor: number) => {
    const entryKey = key.value
    const result = await settle(
      api.PUT('/topics/{topic_id}/subscription/read-marker', {
        params: { path: { topic_id: topic.value } },
        body: { floor }
      })
    )
    if (result.ok) {
      store(entryKey, result.data)
    }
  }

  return { subscription, busy, load, setLevel, markRead }
}
