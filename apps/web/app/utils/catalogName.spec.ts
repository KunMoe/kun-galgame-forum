import { describe, expect, it } from 'vitest'
import { catalogNameText as pickWorkName } from './catalogName'

const work = {
  display_name: '紅殻のパンドラ',
  latin: 'Koukaku no Pandora',
  localized: {
    en: { value: 'Pandora', is_machine: true },
    'zh-Hans': { value: '红壳的潘多拉', is_machine: false }
  }
}

describe('catalogNameText', () => {
  it('prefers the simplified Chinese name, then the original, then latin', () => {
    expect(pickWorkName(work, false)).toBe('红壳的潘多拉')
    expect(pickWorkName({ ...work, localized: {} }, false)).toBe(
      '紅殻のパンドラ'
    )
    expect(
      pickWorkName({ display_name: '', latin: 'Romaji', localized: {} }, false)
    ).toBe('Romaji')
  })

  it('falls back through zh then zh-Hant, never to other languages', () => {
    const hant = {
      ...work,
      localized: {
        'zh-Hant': { value: '紅殼', is_machine: false },
        en: { value: 'x', is_machine: false }
      }
    }
    expect(pickWorkName(hant, false)).toBe('紅殼')
  })

  it('puts the original name first when the reader prefers it', () => {
    expect(pickWorkName(work, true)).toBe('紅殻のパンドラ')
  })
})
