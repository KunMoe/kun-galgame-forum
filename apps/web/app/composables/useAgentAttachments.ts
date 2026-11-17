import { agentProblemMessage } from '#shared/utils/api/agent'
import { formatFileSize } from '#shared/utils/format'
import { agentFileFromUpload } from '~/utils/agentUpload'

export interface AgentAttachment {
  key: string
  kind: AgentUploadKind
  filename: string
  status: 'uploading' | 'ready'
  upload: AgentUpload | null
}

export const useAgentAttachments = () => {
  const api = useAgentApi()
  const agent = useAgentStore()
  const items = ref<AgentAttachment[]>([])

  const maxFiles = computed(() => agent.persona?.uploads?.max_files ?? 0)
  const options = computed(() =>
    maxFiles.value > 0 ? (agent.persona?.uploads?.options ?? []) : []
  )
  const uploading = computed(() =>
    items.value.some((a) => a.status === 'uploading')
  )
  const files = computed(() =>
    items.value.flatMap((a) =>
      a.upload ? [agentFileFromUpload(a.upload)] : []
    )
  )

  const upload = async (a: AgentAttachment, file: File) => {
    const form = new FormData()
    form.append('file', file)
    const r = await api.post<AgentUpload>('/uploads', form)
    const item = items.value.find((x) => x.key === a.key)
    if (!item) return
    if (!r.ok) {
      items.value = items.value.filter((x) => x.key !== a.key)
      useMessage(`「${file.name}」${agentProblemMessage(r.problem)}`, 'warn')
      return
    }
    item.upload = r.data
    item.status = 'ready'
  }

  const add = (list: FileList | null, option: AgentUploadOption) => {
    for (const file of Array.from(list ?? [])) {
      if (items.value.length >= maxFiles.value) {
        useMessage(`一条消息最多带 ${maxFiles.value} 个文件`, 'warn')
        break
      }
      if (file.size > option.max_bytes) {
        useMessage(
          `「${file.name}」超过 ${formatFileSize(option.max_bytes)}`,
          'warn'
        )
        continue
      }
      const a: AgentAttachment = {
        key: crypto.randomUUID(),
        kind: option.kind,
        filename: file.name,
        status: 'uploading',
        upload: null
      }
      items.value.push(a)
      void upload(a, file)
    }
  }

  const remove = (key: string) => {
    items.value = items.value.filter((a) => a.key !== key)
  }

  const clear = () => {
    items.value = []
  }

  return { items, options, maxFiles, uploading, files, add, remove, clear }
}
