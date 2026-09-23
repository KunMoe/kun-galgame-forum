// @vitest-environment nuxt
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import { imageToken, uploadImage, uploadWorkEditImage } from './uploadImage'
import { useKunEditorAdapters } from '~/composables/useKunEditorAdapters'

const { api, reportProblem } = vi.hoisted(() => ({
  api: { POST: vi.fn(), GET: vi.fn() },
  reportProblem: vi.fn()
}))

mockNuxtImport('useApiClient', () => () => api)
mockNuxtImport('reportProblem', () => reportProblem)

const hash = 'ab'.repeat(32)
const image = {
  url: `https://image.test/ab/ab/${hash}.webp`,
  hash,
  width: 640,
  height: 360,
  thumbhash: null,
  sexual: null
}

const created = () => ({
  data: image,
  response: new Response(JSON.stringify(image), {
    status: 201,
    headers: { 'content-type': 'application/json' }
  })
})

const limited = () => {
  const body = { code: 'IMAGE_DAILY_LIMIT_REACHED', status: 429, errors: [] }
  return {
    error: body,
    response: new Response(JSON.stringify(body), {
      status: 429,
      headers: { 'content-type': 'application/problem+json' }
    })
  }
}

type PostInit = { body: FormData }

const lastCall = () => {
  const [path, init] = api.POST.mock.calls.at(-1) as [string, PostInit]
  return { path, form: init.body }
}

beforeEach(() => {
  api.POST.mockReset()
  reportProblem.mockReset()
})

const file = new File(['png'], 'pic.png', { type: 'image/png' })

describe('uploadImage', () => {
  it('sends the file and its purpose to /images', async () => {
    api.POST.mockResolvedValueOnce(created())
    expect(await uploadImage(file, 'message', 'pic.png')).toStrictEqual(image)
    const { path, form } = lastCall()
    expect(path).toBe('/images')
    expect(form.get('purpose')).toBe('message')
    expect(form.get('file')).toBeInstanceOf(Blob)
  })

  it('reports the problem and returns null', async () => {
    api.POST.mockResolvedValueOnce(limited())
    expect(await uploadImage(file, 'content')).toBeNull()
    expect(reportProblem).toHaveBeenCalledOnce()
  })
})

describe('uploadWorkEditImage', () => {
  it('sends the file and its preset to /work-edit-images', async () => {
    api.POST.mockResolvedValueOnce(created())
    expect(await uploadWorkEditImage(file, 'screenshot')).toStrictEqual(image)
    const { path, form } = lastCall()
    expect(path).toBe('/work-edit-images')
    expect(form.get('preset')).toBe('screenshot')
  })
})

describe('the editor upload adapter', () => {
  it('inserts the /image/<hash> token, never the URL', async () => {
    api.POST.mockResolvedValueOnce(created())
    const { uploadImage: upload } = useKunEditorAdapters()
    expect(await upload!(file)).toBe(imageToken(hash))
    expect(lastCall().form.get('purpose')).toBe('content')
  })

  it('throws when the upload fails, so the editor drops its placeholder', async () => {
    api.POST.mockResolvedValueOnce(limited())
    const { uploadImage: upload } = useKunEditorAdapters()
    await expect(upload!(file)).rejects.toThrow()
  })
})
