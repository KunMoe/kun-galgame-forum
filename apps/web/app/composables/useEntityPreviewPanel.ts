import type { EntityTarget } from '#shared/utils/api/schemas'

export const useEntityPreviewPanel = () => {
  const state = useState('kun-entity-preview-panel', () => ({
    isOpen: false,
    target: null as EntityTarget | null,
    blurCovers: false
  }))

  const open = (target: EntityTarget, blurCovers = false) => {
    state.value = { isOpen: true, target, blurCovers }
  }

  const close = () => {
    state.value.isOpen = false
  }

  return { state, open, close }
}
