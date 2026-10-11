import { describe, expect, it } from 'vitest'
import {
  encodeFirstFit,
  isShrinkableImage,
  renameImageForType,
  shrinkSteps,
  shrinkTargetBytes,
  type ImageShrinkStep
} from './shrinkImage'

const MiB = 1024 * 1024
const LIMIT = 10 * MiB

describe('isShrinkableImage', () => {
  it('leaves an image at or under the limit alone', () => {
    expect(isShrinkableImage({ size: LIMIT, type: 'image/png' }, LIMIT)).toBe(
      false
    )
    expect(isShrinkableImage({ size: 1, type: 'image/jpeg' }, LIMIT)).toBe(
      false
    )
  })

  it('shrinks a raster image over the limit', () => {
    for (const type of [
      'image/png',
      'image/jpeg',
      'image/webp',
      'image/avif',
      'image/heic'
    ]) {
      expect(isShrinkableImage({ size: LIMIT + 1, type }, LIMIT)).toBe(true)
    }
  })

  it('never touches an animation, a vector, or a file that is not an image', () => {
    for (const type of ['image/gif', 'image/svg+xml', 'application/pdf', '']) {
      expect(isShrinkableImage({ size: LIMIT * 3, type }, LIMIT)).toBe(false)
    }
  })
})

describe('shrinkTargetBytes', () => {
  it('aims below the limit, leaving a margin', () => {
    expect(shrinkTargetBytes(LIMIT)).toBe(9_437_184)
    expect(shrinkTargetBytes(LIMIT)).toBeLessThan(LIMIT)
  })
})

describe('shrinkSteps', () => {
  it('walks a large photo down in area and quality', () => {
    expect(shrinkSteps(6000, 4000)).toEqual([
      { width: 4899, height: 3266, quality: 0.9 },
      { width: 4899, height: 3266, quality: 0.8 },
      { width: 3674, height: 2449, quality: 0.8 },
      { width: 2739, height: 1826, quality: 0.75 },
      { width: 1936, height: 1291, quality: 0.7 }
    ])
  })

  it('keeps a small image at its own size until the area cap reaches it', () => {
    expect(shrinkSteps(2000, 1500)).toEqual([
      { width: 2000, height: 1500, quality: 0.9 },
      { width: 2000, height: 1500, quality: 0.8 },
      { width: 2000, height: 1500, quality: 0.75 },
      { width: 1826, height: 1369, quality: 0.7 }
    ])
  })

  it('keeps a long strip inside the canvas edge limit and its aspect ratio', () => {
    const steps = shrinkSteps(1080, 30000)
    for (const step of steps) {
      expect(Math.max(step.width, step.height)).toBeLessThanOrEqual(16_384)
      expect(step.width * step.height).toBeLessThanOrEqual(16_000_000)
      expect(step.height / step.width).toBeCloseTo(30000 / 1080, 0)
    }
    expect(steps[0]).toEqual({ width: 590, height: 16384, quality: 0.9 })
  })

  it('never grows the image or raises the quality from one step to the next', () => {
    const steps = shrinkSteps(8000, 6000)
    steps.slice(1).forEach((step, index) => {
      const previous = steps[index]!
      expect(step.width * step.height).toBeLessThanOrEqual(
        previous.width * previous.height
      )
      expect(step.quality).toBeLessThanOrEqual(previous.quality)
    })
  })
})

describe('renameImageForType', () => {
  it('swaps the extension for the encoded type', () => {
    expect(renameImageForType('photo.PNG', 'image/webp')).toBe('photo.webp')
    expect(renameImageForType('scan.tiff', 'image/jpeg')).toBe('scan.jpg')
    expect(renameImageForType('a.b.png', 'image/webp')).toBe('a.b.webp')
  })

  it('adds one when the name has none', () => {
    expect(renameImageForType('image', 'image/webp')).toBe('image.webp')
    expect(renameImageForType('.png', 'image/jpeg')).toBe('image.jpg')
  })
})

describe('encodeFirstFit', () => {
  const steps: ImageShrinkStep[] = [
    { width: 400, height: 300, quality: 0.9 },
    { width: 400, height: 300, quality: 0.8 },
    { width: 200, height: 150, quality: 0.7 }
  ]
  const blob = (size: number, type: string) =>
    new Blob([new Uint8Array(size)], { type })

  it('stops at the first encode under the target', async () => {
    const calls: string[] = []
    const sizes = [500, 90, 10]
    const result = await encodeFirstFit(steps, 100, async (step, type) => {
      calls.push(`${type}@${step.quality}`)
      return blob(sizes[calls.length - 1]!, type)
    })
    expect(result?.size).toBe(90)
    expect(result?.type).toBe('image/webp')
    expect(calls).toEqual(['image/webp@0.9', 'image/webp@0.8'])
  })

  it('falls back to JPEG for good when the browser cannot encode WebP', async () => {
    const calls: string[] = []
    const result = await encodeFirstFit(steps, 100, async (step, type) => {
      calls.push(`${type}@${step.quality}`)
      const size = step.quality === 0.7 ? 50 : 500
      return blob(size, type === 'image/webp' ? 'image/png' : type)
    })
    expect(result?.type).toBe('image/jpeg')
    expect(calls).toEqual([
      'image/webp@0.9',
      'image/jpeg@0.9',
      'image/jpeg@0.8',
      'image/jpeg@0.7'
    ])
  })

  it('moves on when a canvas fails, and gives up after the last step', async () => {
    let count = 0
    const result = await encodeFirstFit(steps, 100, async (_step, type) => {
      count++
      return count === 1 ? null : blob(500, type)
    })
    expect(result).toBeNull()
    expect(count).toBe(steps.length)
  })
})
