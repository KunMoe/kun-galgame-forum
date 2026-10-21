// ENTITY_DELETED carries why and when as extension members, which the shared
// problem settle() does not keep; read them off the raw error body.
export interface EntityDeletion {
  reason: string
  at: string | null
}

const REASON_LABELS: Record<string, string> = {
  not_a_work: '这个条目不是一部作品',
  out_of_scope: '这部作品不在收录范围内',
  spam_or_test: '这是垃圾或测试条目',
  other: '其他原因'
}

export const entityDeletion = (error: unknown): EntityDeletion | null => {
  if (
    !error ||
    typeof error !== 'object' ||
    (error as { code?: unknown }).code !== 'ENTITY_DELETED'
  ) {
    return null
  }
  const { deleted_reason: reason, deleted_at: at } = error as {
    deleted_reason?: unknown
    deleted_at?: unknown
  }
  return {
    reason: typeof reason === 'string' ? reason : 'other',
    at: typeof at === 'string' ? at : null
  }
}

export const deletionReasonLabel = (reason: string): string =>
  REASON_LABELS[reason] ?? REASON_LABELS.other!
