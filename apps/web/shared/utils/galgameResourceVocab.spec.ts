// @vitest-environment node
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import {
  EMULATOR_RUNTIME_OPTIONS,
  RUNTIME_OPTIONS
} from './galgameResourceVocab'

const spec = JSON.parse(
  readFileSync(
    fileURLToPath(
      new URL('../../../api/openapi/kungal-v1.json', import.meta.url)
    ),
    'utf8'
  )
) as { components: { schemas: { ResourceRuntime: { enum: string[] } } } }

describe('RUNTIME_OPTIONS', () => {
  it('lists the published runtime vocabulary in its order', () => {
    expect(RUNTIME_OPTIONS.map((o) => o.value)).toEqual(
      spec.components.schemas.ResourceRuntime.enum
    )
  })

  it('puts Tyranor Next and YukiHub right under 安卓直装', () => {
    expect(RUNTIME_OPTIONS.slice(1, 4)).toEqual([
      { value: 'native-and', label: '安卓直装' },
      { value: 'tyranor-next', label: 'Tyranor Next' },
      { value: 'yukihub', label: 'YukiHub' }
    ])
  })

  it('counts YukiHub among the emulators', () => {
    expect(EMULATOR_RUNTIME_OPTIONS.map((o) => o.value)).toContain('yukihub')
  })
})
