import { settle } from '#shared/utils/api/problem'
import type { Settings, SettingsPatch } from '#shared/utils/api/schemas'

type Field = keyof Settings & keyof SettingsPatch

// Each settings section edits its own keys of the one /admin/settings
// resource, so saving a section never resends another section's values.
export const useAdminSettingsForm = <K extends Field>(
  initial: Pick<Settings, K>
) => {
  const api = useApiClient()
  const { data, problem, pending, refresh } = useKunApi(
    'admin-settings',
    (client, { signal }) => client.GET('/admin/settings', { signal })
  )
  const keys = Object.keys(initial) as K[]
  const form = reactive({ ...initial }) as Pick<Settings, K>
  watch(
    data,
    (value) => {
      if (!value) return
      for (const key of keys) {
        form[key] = value[key]
      }
    },
    { immediate: true, flush: 'sync' }
  )

  const isSaving = ref(false)
  const save = async () => {
    const body: Partial<SettingsPatch> = {}
    for (const key of keys) {
      const value = form[key]
      body[key] = (
        typeof initial[key] === 'number' ? Number(value) : value
      ) as never
    }
    isSaving.value = true
    const result = await settle(api.PATCH('/admin/settings', { body }))
    isSaving.value = false
    if (!result.ok) {
      reportProblem(result.problem)
      return
    }
    useMessage('设置已保存, 其它服务实例会在 30 秒内生效', 'success')
    await refresh()
  }

  return { data, problem, pending, form, isSaving, save }
}
