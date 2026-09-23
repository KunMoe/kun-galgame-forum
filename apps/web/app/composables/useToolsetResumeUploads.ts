export type ToolsetPendingUpload = {
  id: string
  name: string
  size: number
  last_modified: number
  progress: number
  updated_at: number
}

export const useToolsetResumeUploads = (toolsetId: string) => {
  const key = `kungal:toolset-upload-resume:${toolsetId}`

  const read = (): ToolsetPendingUpload[] => {
    if (!import.meta.client) {
      return []
    }
    try {
      const raw = localStorage.getItem(key)
      const parsed = raw ? JSON.parse(raw) : []
      if (!Array.isArray(parsed)) {
        return []
      }
      return parsed.flatMap((item) => {
        if (!item || typeof item !== 'object') {
          return []
        }
        const record = item as Record<string, unknown>
        const id =
          typeof record.id === 'string'
            ? record.id
            : typeof record.artifact_uuid === 'string'
              ? record.artifact_uuid
              : ''
        if (!id) {
          return []
        }
        return [
          {
            id,
            name: String(record.name ?? ''),
            size: Number(record.size) || 0,
            last_modified: Number(record.last_modified) || 0,
            progress: Number(record.progress) || 0,
            updated_at: Number(record.updated_at) || 0
          }
        ]
      })
    } catch {
      return []
    }
  }

  const write = (items: ToolsetPendingUpload[]) => {
    if (!import.meta.client) {
      return
    }
    try {
      localStorage.setItem(key, JSON.stringify(items))
    } catch {
      // Quota exceeded or storage disabled: resume state is an optimisation, so
      // losing it must not fail the upload.
    }
  }

  const list = (): ToolsetPendingUpload[] =>
    read().sort((a, b) => b.updated_at - a.updated_at)

  const upsert = (record: ToolsetPendingUpload) => {
    write([...read().filter((p) => p.id !== record.id), record])
  }

  const setProgress = (uploadId: string, progress: number) => {
    const all = read()
    const record = all.find((p) => p.id === uploadId)
    if (!record) {
      return
    }
    record.progress = progress
    record.updated_at = Date.now()
    write(all)
  }

  const remove = (uploadId: string) => {
    write(read().filter((p) => p.id !== uploadId))
  }

  return { list, upsert, setProgress, remove }
}
