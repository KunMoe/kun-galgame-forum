import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ToolsetResourceDraftFields = {
  link_url: string
  size_label: string
  extraction_code: string
  archive_password: string
  note: string
}

export const emptyToolsetResourceDraft = (): ToolsetResourceDraftFields => ({
  link_url: '',
  size_label: '',
  extraction_code: '',
  archive_password: '',
  note: ''
})

export const usePersistEditToolsetResourceStore = defineStore(
  'KUNGalgameEditToolsetResource',
  () => {
    const toolsetId = ref('')
    const mode = ref<'file' | 'link'>('file')
    const fields = ref<ToolsetResourceDraftFields>(emptyToolsetResourceDraft())

    const reset = () => {
      toolsetId.value = ''
      mode.value = 'file'
      fields.value = emptyToolsetResourceDraft()
    }

    const keep = (id: string) => {
      if (toolsetId.value !== id) {
        reset()
        toolsetId.value = id
      }
    }

    return { toolsetId, mode, fields, reset, keep }
  },
  {
    persist: {
      storage: piniaPluginPersistedstate.localStorage()
    }
  }
)
