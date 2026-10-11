import { fieldMessage, problemMessage } from '#shared/utils/api/message'
import type { ClientProblem, FieldError } from '#shared/utils/api/problem'
import { KUN_FIELD_LABELS } from './kunZodError'

// fieldMessage renders only the localized reason, so two field errors on one
// form arrived as "内容过短" and "选择的项太少" with nothing saying which field.
const labelOf = (error: FieldError): string => {
  const key =
    error.parameter ??
    error.header ??
    error.pointer
      ?.split('/')
      .filter((part) => part && !/^\d+$/.test(part))
      .pop()
  if (!key) {
    return ''
  }
  return KUN_FIELD_LABELS[key] ?? key
}

const SERVER_FAULT_MESSAGE_DURATION = 10_000

export const reportProblem = (problem: ClientProblem) => {
  if (import.meta.server) {
    return
  }
  if (problem.status >= 500 && problem.requestId) {
    useMessage(
      `${problemMessage(problem)}（错误编号 ${problem.requestId}）`,
      'error',
      SERVER_FAULT_MESSAGE_DURATION
    )
  } else {
    useMessage(problemMessage(problem), 'error')
  }
  for (const error of problem.errors) {
    const label = labelOf(error)
    const text = fieldMessage(error)
    useMessage(label ? `${label}：${text}` : text, 'error')
  }
}
