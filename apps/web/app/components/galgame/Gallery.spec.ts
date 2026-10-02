// @vitest-environment nuxt
import { beforeEach, describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import GalgameGallery from './Gallery.vue'
import type { WorkScreenshot } from '#shared/utils/api/schemas'

const shot = (
  hash: string,
  sexual: 'safe' | 'suggestive' | 'explicit' | null,
  site = 'vndb'
): WorkScreenshot => ({
  object: 'work_screenshot',
  image: {
    hash,
    url: `https://cdn.test/${hash}.webp`,
    width: 1280,
    height: 720,
    thumbhash: null,
    sexual
  },
  caption: '',
  site,
  sort_order: 0
})

const screenshots = [
  shot('safe', 'safe'),
  shot('ungraded', null),
  shot('suggestive', 'suggestive'),
  shot('explicit', 'explicit')
]

const rendered = (html: string) =>
  ['safe', 'ungraded', 'suggestive', 'explicit'].filter((hash) =>
    html.includes(`https://cdn.test/${hash}`)
  )

describe('GalgameGallery', () => {
  beforeEach(() => {
    const user = usePersistUserStore()
    user.id = 0
    user.setContentStance(false, 'hide')
    usePersistSettingsStore().showKUNGalgameContentLimit = 'sfw'
  })

  it('renders no graded screenshot for a reader who hides adult content', async () => {
    const wrapper = await mountSuspended(GalgameGallery, {
      props: { screenshots }
    })
    expect(rendered(wrapper.html())).toEqual(['safe', 'ungraded'])
    expect(wrapper.text()).toContain('2 张成人向图片已按您的内容设置隐藏')
  })

  // The filter once offered a checkbox per grade to exactly this reader. A
  // ticked one was saved with the settings and outranked the stance.
  it('ignores a grade opt-in left over in the saved settings', async () => {
    const settings = usePersistSettingsStore() as unknown as {
      showKUNGalgameGallerySexualLevels: number[]
    }
    settings.showKUNGalgameGallerySexualLevels = [1, 2]
    const wrapper = await mountSuspended(GalgameGallery, {
      props: { screenshots }
    })
    expect(rendered(wrapper.html())).toEqual(['safe', 'ungraded'])
  })

  it('renders every screenshot once NSFW is on', async () => {
    usePersistSettingsStore().showKUNGalgameContentLimit = 'nsfw'
    const wrapper = await mountSuspended(GalgameGallery, {
      props: { screenshots }
    })
    expect(rendered(wrapper.html())).toEqual([
      'safe',
      'ungraded',
      'suggestive',
      'explicit'
    ])
    expect(wrapper.text()).not.toContain('已按您的内容设置隐藏')
  })

  it('hides them from a signed-in account whose stance is hide', async () => {
    const user = usePersistUserStore()
    user.id = 7
    user.setContentStance(true, 'hide')
    usePersistSettingsStore().showKUNGalgameContentLimit = 'nsfw'
    const wrapper = await mountSuspended(GalgameGallery, {
      props: { screenshots }
    })
    expect(rendered(wrapper.html())).toEqual(['safe', 'ungraded'])
  })
})
