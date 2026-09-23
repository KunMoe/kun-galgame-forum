import { describe, expect, it } from 'vitest'
import { workExternalId } from './workExternalRef'

describe('workExternalId', () => {
  it('takes the VNDB work anchor whatever the order', () => {
    for (const refs of [
      [
        { site: 'vndb', external_id: 'r22119' },
        { site: 'vndb', external_id: 'v10875' }
      ],
      [
        { site: 'vndb', external_id: 'v10875' },
        { site: 'vndb', external_id: 'r22119' }
      ]
    ]) {
      expect(workExternalId(refs, 'vndb')).toBe('v10875')
    }
  })

  it('keeps a release-only VNDB id and the first id of other sites', () => {
    expect(
      workExternalId([{ site: 'vndb', external_id: 'r69531' }], 'vndb')
    ).toBe('r69531')
    expect(
      workExternalId(
        [
          { site: 'bangumi', external_id: '111743' },
          { site: 'bangumi', external_id: '2' }
        ],
        'bangumi'
      )
    ).toBe('111743')
    expect(workExternalId([], 'erogamescape')).toBeUndefined()
  })
})
