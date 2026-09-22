import { describe, expect, it } from 'vitest'
import { foldContentStance } from './contentStance'

describe('foldContentStance', () => {
  it.each([
    ['the backfilled majority: blur stored, never attested', false, 'blur'],
    ['show stored, never attested', false, 'show'],
    ['hide stored, never attested', false, 'hide'],
    ['attested but the claim never arrived', true, undefined],
    ['attested with a value this build does not know', true, 'peek'],
    ['attested and hiding', true, 'hide'],
    ['no claims at all', undefined, undefined]
  ])('%s folds to hide', (_why, confirmed, display) => {
    expect(foldContentStance(confirmed, display)).toBe('hide')
  })

  it('only an attested account reaches blur or show', () => {
    expect(foldContentStance(true, 'blur')).toBe('blur')
    expect(foldContentStance(true, 'show')).toBe('show')
  })
})
