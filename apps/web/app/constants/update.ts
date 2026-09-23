import type { KunTabItem } from '@kungal/ui-vue'
import type {
  TodoProject,
  TodoState,
  UpdateLogChangeType
} from '#shared/utils/api/schemas'

export const kunUpdateLogTabItem: KunTabItem[] = [
  {
    value: 'history',
    textValue: '更新历史',
    href: '/update/history'
  },
  {
    value: 'todo',
    textValue: '待办列表',
    href: '/update/todo'
  }
]

export const KUN_UPDATE_LOG_CHANGE_TYPES = [
  'feat',
  'perf',
  'fix',
  'style',
  'mod',
  'chore',
  'sec',
  'refactor',
  'docs',
  'test'
] as const satisfies readonly UpdateLogChangeType[]

export const KUN_UPDATE_LOG_CHANGE_TYPE_LABEL: Record<
  UpdateLogChangeType,
  string
> = {
  feat: '增加功能',
  perf: '性能优化',
  fix: '错误修复',
  style: '样式修改',
  mod: '功能更改',
  chore: '其它修改',
  sec: '安全提升',
  refactor: '代码重构',
  docs: '文档修改',
  test: '测试用例'
}

export const KUN_TODO_PROJECTS = [
  'forum',
  'patch'
] as const satisfies readonly TodoProject[]

export const KUN_TODO_PROJECT_LABEL: Record<TodoProject, string> = {
  forum: '论坛',
  patch: '补丁站'
}

export const KUN_TODO_STATES = [
  'pending',
  'in_progress',
  'done',
  'discarded'
] as const satisfies readonly TodoState[]

export const KUN_TODO_STATE_LABEL: Record<TodoState, string> = {
  pending: '待处理',
  in_progress: '进行中',
  done: '已完成',
  discarded: '已废弃'
}

export const KUN_TODO_STATE_ICON: Record<TodoState, string> = {
  pending: 'lucide:circle-divide',
  in_progress: 'lucide:loader',
  done: 'lucide:check',
  discarded: 'lucide:x'
}

export const KUN_TODO_STATE_TEXT_CLASS: Record<TodoState, string> = {
  pending: 'text-default',
  in_progress: 'text-primary',
  done: 'text-success',
  discarded: 'text-danger'
}

export const kunTodoStateOfLegacyStatus = (
  status: number
): TodoState | undefined => KUN_TODO_STATES[status]
