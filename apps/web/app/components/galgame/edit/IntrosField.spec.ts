// @vitest-environment nuxt
import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { CatalogIntro } from '#shared/utils/api/schemas'
import IntrosField from './IntrosField.vue'

const EditorStub = {
  name: 'KunMilkdownDualEditorProvider',
  props: ['valueMarkdown'],
  emits: ['set-markdown'],
  template: '<div class="editor">{{ valueMarkdown }}<slot /></div>'
}

const intro = (
  locale: string,
  data_source: string,
  value: string,
  is_machine = false
): CatalogIntro => ({ locale, data_source, value, is_machine })

const mount = (modelValue: unknown, upstream: CatalogIntro[]) =>
  mountSuspended(IntrosField, {
    props: { modelValue, upstream },
    global: { stubs: { KunMilkdownDualEditorProvider: EditorStub } }
  })

const lastEmitted = (wrapper: Awaited<ReturnType<typeof mount>>) =>
  wrapper.emitted('update:modelValue')?.at(-1)?.[0]

describe('IntrosField', () => {
  const own = [{ lang: 'en', intro: 'EN own' }]
  const upstream = [
    intro('en', 'curated', 'EN own'),
    intro('ja', 'dlsite', 'JA from dlsite')
  ]

  it('opens on the imported text of a language this forum has not written', async () => {
    const wrapper = await mount(own, upstream)
    const editor = wrapper.findComponent(EditorStub)
    expect(editor.props('valueMarkdown')).toBe('JA from dlsite')
    expect(wrapper.text()).toContain('DLsite')
  })

  it('submits nothing for an untouched prefill', async () => {
    const wrapper = await mount(own, upstream)
    wrapper
      .findComponent(EditorStub)
      .vm.$emit('set-markdown', 'JA from dlsite\n')
    expect(lastEmitted(wrapper)).toEqual(own)
  })

  it('submits an edited prefill as this forum’s version', async () => {
    const wrapper = await mount(own, upstream)
    wrapper.findComponent(EditorStub).vm.$emit('set-markdown', 'JA fixed')
    expect(lastEmitted(wrapper)).toEqual([
      { lang: 'ja', intro: 'JA fixed' },
      { lang: 'en', intro: 'EN own' }
    ])
  })

  it('prefills a machine translation kept in the curated lane', async () => {
    const wrapper = await mount(
      [],
      [intro('zh-Hans', 'curated', '机器翻译的简介', true)]
    )
    expect(wrapper.findComponent(EditorStub).props('valueMarkdown')).toBe(
      '机器翻译的简介'
    )
    expect(wrapper.text()).toContain('机器翻译')
  })
})
