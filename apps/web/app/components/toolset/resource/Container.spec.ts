// @vitest-environment nuxt
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'
import { usePersistEditToolsetResourceStore } from '~/store/modules/edit/toolsetResource'
import ToolsetResourceContainer from './Container.vue'

const { api, message } = vi.hoisted(() => ({
  api: { GET: vi.fn(), POST: vi.fn(), PATCH: vi.fn(), DELETE: vi.fn() },
  message: vi.fn()
}))

mockNuxtImport('useApiClient', () => () => api)
mockNuxtImport('useMessage', () => message)

const RESUME_KEY = 'kungal:toolset-upload-resume:7'

const finishedUpload = {
  id: 'art-1',
  name: 'tool.7z',
  size: 2048,
  last_modified: 0,
  progress: 100,
  updated_at: 1,
  completed: true
}

const storedUploads = () =>
  JSON.parse(localStorage.getItem(RESUME_KEY) ?? '[]') as { id: string }[]

const problem = (status: number, body: Record<string, unknown>) => ({
  error: { status, request_id: 'req_01JZ0000000000000000000000', ...body },
  response: new Response(null, {
    status,
    headers: { 'content-type': 'application/problem+json' }
  })
})

const mountContainer = () =>
  mountSuspended(ToolsetResourceContainer, { props: { toolsetId: '7' } })

type Wrapper = Awaited<ReturnType<typeof mountContainer>>

const clickButton = async (wrapper: Wrapper, text: string) => {
  const button = wrapper.findAll('button').find((b) => b.text() === text)
  expect(button, text).toBeDefined()
  await button!.trigger('click')
  await flushPromises()
}

beforeEach(() => {
  localStorage.setItem(RESUME_KEY, JSON.stringify([finishedUpload]))
})

afterEach(() => {
  localStorage.clear()
  usePersistEditToolsetResourceStore().reset()
  Object.values(api).forEach((fn) => fn.mockReset())
  message.mockReset()
})

describe('ToolsetResourceContainer', () => {
  it('keeps a fresh upload across a failed publish and a reload', async () => {
    localStorage.clear()
    const upload = {
      object: 'toolset_upload',
      id: 'art-2',
      toolset_id: '7',
      filename: 'patch.zip',
      file_size: 4096,
      is_multipart: false,
      parts: [],
      uploaded_parts: [],
      created_at: '2026-10-11T00:00:00Z'
    }
    api.POST.mockResolvedValueOnce({
      data: {
        ...upload,
        state: 'pending',
        upload_url: 'https://s3.test/put',
        completed_at: null
      },
      response: new Response(null, { status: 201 })
    })
    api.PATCH.mockResolvedValueOnce({
      data: { ...upload, state: 'completed', completed_at: upload.created_at },
      response: new Response(null, { status: 200 })
    })
    api.POST.mockResolvedValueOnce(
      problem(500, { code: 'INTERNAL_ERROR', errors: [] })
    )
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => new Response(null, { status: 200 }))
    )

    const wrapper = await mountContainer()
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [new File([new Uint8Array(4096)], 'patch.zip')]
    })
    await input.trigger('change')
    await clickButton(wrapper, '确认上传')
    await clickButton(wrapper, '提交链接')
    vi.unstubAllGlobals()

    expect(api.POST.mock.calls[1]![1].body).toEqual({
      toolset_resource_type: 'file',
      artifact_id: 'art-2'
    })
    expect(storedUploads()).toEqual([
      expect.objectContaining({ id: 'art-2', completed: true })
    ])
    wrapper.unmount()

    const reloaded = await mountContainer()
    expect(reloaded.text()).toContain('已上传完成但尚未发布的文件「patch.zip」')
  })

  it('offers a finished upload again and publishes with it', async () => {
    api.POST.mockResolvedValueOnce({
      data: { id: '9' },
      response: new Response(null, { status: 201 })
    })
    const wrapper = await mountContainer()
    expect(wrapper.text()).toContain('已上传完成但尚未发布的文件「tool.7z」')

    await clickButton(wrapper, '用它发布')
    await clickButton(wrapper, '提交链接')

    const [path, init] = api.POST.mock.calls[0]!
    expect(path).toBe('/toolsets/{toolset_id}/resources')
    expect(init.body).toEqual({
      toolset_resource_type: 'file',
      artifact_id: 'art-1'
    })
    expect(storedUploads()).toEqual([])
    expect(wrapper.emitted('onSuccess')).toHaveLength(1)
  })

  it('keeps the finished upload when the publish fails on the server', async () => {
    api.POST.mockResolvedValueOnce(
      problem(500, { code: 'INTERNAL_ERROR', errors: [] })
    )
    const wrapper = await mountContainer()

    await clickButton(wrapper, '用它发布')
    await clickButton(wrapper, '提交链接')

    expect(storedUploads().map((u) => u.id)).toEqual(['art-1'])
    expect(String(message.mock.calls.at(-1)![0])).toContain(
      '错误编号 req_01JZ0000000000000000000000'
    )
  })

  it('drops the finished upload when the server no longer accepts it', async () => {
    api.POST.mockResolvedValueOnce(
      problem(422, {
        code: 'VALIDATION_FAILED',
        errors: [{ pointer: '/artifact_id', reason: 'UNKNOWN_REFERENCE' }]
      })
    )
    const wrapper = await mountContainer()

    await clickButton(wrapper, '用它发布')
    await clickButton(wrapper, '提交链接')

    expect(storedUploads()).toEqual([])
    expect(message).toHaveBeenCalledWith(
      '这个已上传的文件已失效, 请重新上传',
      'warn'
    )
    expect(wrapper.text()).not.toContain('tool.7z')
  })

  it('keeps the typed fields as a draft for the same toolset only', async () => {
    const wrapper = await mountContainer()
    await wrapper.find('textarea').setValue('needs the 2.0 runtime')
    wrapper.unmount()

    const again = await mountContainer()
    expect((again.find('textarea').element as HTMLTextAreaElement).value).toBe(
      'needs the 2.0 runtime'
    )
    again.unmount()

    const other = await mountSuspended(ToolsetResourceContainer, {
      props: { toolsetId: '8' }
    })
    expect((other.find('textarea').element as HTMLTextAreaElement).value).toBe(
      ''
    )
  })
})
