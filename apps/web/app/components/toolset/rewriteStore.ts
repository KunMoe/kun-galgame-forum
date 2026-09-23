import type { ToolsetCreate } from '#shared/utils/api/schemas'

export type UpdateFormType = Required<ToolsetCreate> & { toolset_id: string }

export const toolsetUpdateForm = reactive<UpdateFormType>({
  toolset_id: '',
  title: '',
  content_markdown: '',
  interface_language: 'zh-cn',
  platform: 'windows',
  toolset_type: 'emulator',
  release_channel: 'stable',
  homepage_urls: [] as string[],
  aliases: [] as string[]
})
