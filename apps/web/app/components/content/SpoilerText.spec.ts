// @vitest-environment nuxt
import { afterEach, describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import type { VueWrapper } from '@vue/test-utils'
import { h } from 'vue'
import { renderToString } from 'vue/server-renderer'
import SpoilerText from './SpoilerText.vue'

let wrapper: VueWrapper | undefined

afterEach(() => {
  wrapper?.unmount()
})

describe('plain text with inline spoilers', () => {
  it('reveals only the clicked passage and supports keyboard reveal', async () => {
    wrapper = await mountSuspended(SpoilerText, {
      props: { content: '前文||秘密一||中间||秘密二||后文' }
    })
    const spoilers = wrapper.findAll('.kun-spoiler-hidden')
    expect(spoilers).toHaveLength(2)
    expect(spoilers[0]!.attributes('aria-expanded')).toBe('false')
    expect(wrapper.get('article').text()).toBe('前文秘密一中间秘密二后文')

    await spoilers[0]!.trigger('click')
    expect(wrapper.findAll('.kun-spoiler-hidden')).toHaveLength(1)
    expect(spoilers[0]!.attributes('aria-expanded')).toBe('true')
    expect(spoilers[0]!.classes()).not.toContain('text-transparent')
    expect(spoilers[1]!.classes()).toContain('kun-spoiler-hidden')

    await spoilers[1]!.trigger('keydown', { key: 'Enter' })
    expect(wrapper.find('.kun-spoiler-hidden').exists()).toBe(false)
  })

  it('keeps multiline text and treats HTML and markdown as literal text', async () => {
    const content =
      '<img src=x onerror=alert(1)> **普通文本**\n||第一行\n第二行||'
    wrapper = await mountSuspended(SpoilerText, { props: { content } })

    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('strong').exists()).toBe(false)
    expect(wrapper.get('article').element.textContent).toBe(
      '<img src=x onerror=alert(1)> **普通文本**\n第一行\n第二行'
    )
    expect(wrapper.get('.kun-spoiler-hidden').text()).toBe('第一行\n第二行')
    expect(wrapper.get('article').classes()).toContain('whitespace-pre-wrap')
  })

  it('leaves an unclosed marker as text', async () => {
    wrapper = await mountSuspended(SpoilerText, {
      props: { content: '普通文本 ||未闭合' }
    })
    expect(wrapper.text()).toBe('普通文本 ||未闭合')
    expect(wrapper.find('.kun-spoiler').exists()).toBe(false)
  })

  it('masks a changed review again after the old one was revealed', async () => {
    wrapper = await mountSuspended(SpoilerText, {
      props: { content: '前文||旧秘密||' }
    })
    await wrapper.get('.kun-spoiler-hidden').trigger('click')
    await wrapper.setProps({ content: '前文||新秘密||' })

    expect(wrapper.get('.kun-spoiler-hidden').text()).toBe('新秘密')
    expect(wrapper.get('.kun-spoiler-hidden').attributes('aria-expanded')).toBe(
      'false'
    )
  })

  it('renders the hidden state on the server before JavaScript runs', async () => {
    const html = await renderToString(
      h(SpoilerText, { content: '前||秘密||后' })
    )
    expect(html).toContain('kun-spoiler kun-spoiler-hidden')
    expect(html).not.toContain('||')
  })
})
