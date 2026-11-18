import type { EnvValue, EnvVariable } from '#shared/utils/api/schemas'

type ChipColor = 'default' | 'primary' | 'success' | 'warning' | 'danger'

export const KUN_ENV_KIND: Record<
  EnvVariable['variable_kind'],
  { label: string; color: ChipColor }
> = {
  bootstrap: { label: '启动接线', color: 'danger' },
  topology: { label: '拓扑', color: 'default' },
  secret: { label: '密钥', color: 'warning' },
  plain: { label: '普通', color: 'primary' }
}

export const KUN_ENV_SOURCE: Record<
  NonNullable<EnvVariable['resolved_from']>,
  string
> = {
  env: '环境变量',
  store: '配置中心',
  default: '代码默认'
}

export const KUN_ENV_ISSUE: Record<
  EnvVariable['issues'][number],
  { label: string; hint: string; color: ChipColor }
> = {
  panel_redundant: {
    label: '面板值可删除',
    hint: '环境变量与存储值一致, 从 Dokploy 删掉后重新部署',
    color: 'success'
  },
  panel_overrides_store: {
    label: '面板值覆盖了配置中心',
    hint: '两者不一致, 生效的是环境变量。确认哪个对, 删面板值或改存储值',
    color: 'danger'
  },
  restart_pending: {
    label: '待重启生效',
    hint: '存储值在 API 启动后改过, 重启 kungal-api 后生效',
    color: 'warning'
  }
}

export const KUN_ENV_CHANGE: Record<string, string> = {
  set: '写入',
  delete: '删除',
  import: '导入'
}

export const envValueText = (value: EnvValue | null | undefined) => {
  if (!value) return '—'
  if (!value.is_set) return '(空)'
  if (value.plain !== null) return value.plain
  return `${value.fingerprint ?? ''} ${value.hint ?? ''}`.trim()
}
