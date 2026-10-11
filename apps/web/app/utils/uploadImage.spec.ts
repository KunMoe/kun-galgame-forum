// @vitest-environment nuxt
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import {
  imageToken,
  uploadImage,
  uploadNewsSubmissionImage,
  uploadWorkEditImage
} from './uploadImage'
import { useKunEditorAdapters } from '~/composables/useKunEditorAdapters'

const { api, reportProblem, useMessage, shrinkImageToFit } = vi.hoisted(() => ({
  api: { POST: vi.fn(), GET: vi.fn() },
  reportProblem: vi.fn(),
  useMessage: vi.fn(),
  shrinkImageToFit: vi.fn()
}))

vi.mock('./shrinkImage', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./shrinkImage')>()),
  shrinkImageToFit
}))

mockNuxtImport('useApiClient', () => () => api)
mockNuxtImport('reportProblem', () => reportProblem)
mockNuxtImport('useMessage', () => useMessage)

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
  useMessage.mockReset()
  shrinkImageToFit.mockReset()
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

describe('an image over 10 MiB', () => {
  const oversized = (name: string, type: string) =>
    new File([new Uint8Array(10 * 1024 * 1024 + 1)], name, { type })
  const sentFile = () => lastCall().form.get('file') as File

  it('goes up shrunk, named for its new type, and says so', async () => {
    const shrunk = new Blob([new Uint8Array(64)], { type: 'image/webp' })
    shrinkImageToFit.mockResolvedValueOnce(shrunk)
    api.POST.mockResolvedValueOnce(created())
    await uploadImage(oversized('big.png', 'image/png'), 'content', 'big.png')
    expect(sentFile().name).toBe('big.webp')
    expect(sentFile().type).toBe('image/webp')
    expect(sentFile().size).toBe(64)
    expect(useMessage).toHaveBeenCalledWith(
      '图片超过 10MB，已自动压缩后上传',
      'info'
    )
  })

  it('shrinks a work-edit screenshot too', async () => {
    shrinkImageToFit.mockResolvedValueOnce(
      new Blob([new Uint8Array(64)], { type: 'image/jpeg' })
    )
    api.POST.mockResolvedValueOnce(created())
    await uploadWorkEditImage(
      oversized('shot.png', 'image/png'),
      'screenshot',
      'shot.png'
    )
    expect(sentFile().name).toBe('shot.jpg')
  })

  it('goes up untouched when it is a GIF', async () => {
    api.POST.mockResolvedValueOnce(created())
    const gif = oversized('anim.gif', 'image/gif')
    await uploadImage(gif, 'content', 'anim.gif')
    expect(shrinkImageToFit).not.toHaveBeenCalled()
    expect(sentFile().size).toBe(gif.size)
    expect(useMessage).not.toHaveBeenCalled()
  })

  it('goes up untouched when the browser cannot shrink it', async () => {
    shrinkImageToFit.mockResolvedValueOnce(null)
    api.POST.mockResolvedValueOnce(created())
    const heic = oversized('photo.heic', 'image/heic')
    await uploadImage(heic, 'content', 'photo.heic')
    expect(sentFile().name).toBe('photo.heic')
    expect(sentFile().size).toBe(heic.size)
    expect(useMessage).not.toHaveBeenCalled()
  })

  it('is never shrunk when it is under the limit', async () => {
    api.POST.mockResolvedValueOnce(created())
    await uploadImage(file, 'content', 'pic.png')
    expect(shrinkImageToFit).not.toHaveBeenCalled()
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

describe('uploadNewsSubmissionImage', () => {
  it('sends only the file to /me/news-submission-images', async () => {
    api.POST.mockResolvedValueOnce(created())
    expect(await uploadNewsSubmissionImage(file)).toStrictEqual(image)
    const { path, form } = lastCall()
    expect(path).toBe('/me/news-submission-images')
    expect([...form.keys()]).toEqual(['file'])
    expect(form.get('file')).toBeInstanceOf(Blob)
  })

  it('reports the problem and returns null', async () => {
    api.POST.mockResolvedValueOnce(limited())
    expect(await uploadNewsSubmissionImage(file)).toBeNull()
    expect(reportProblem).toHaveBeenCalledOnce()
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
