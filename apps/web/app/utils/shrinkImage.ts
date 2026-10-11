const WEBP = 'image/webp'
const JPEG = 'image/jpeg'
const UNSHRINKABLE_TYPES = new Set(['image/gif', 'image/svg+xml'])

const TARGET_RATIO = 0.9
const MAX_CANVAS_EDGE = 16_384
const SHRINK_LADDER = [
  { pixels: 16_000_000, quality: 0.9 },
  { pixels: 16_000_000, quality: 0.8 },
  { pixels: 9_000_000, quality: 0.8 },
  { pixels: 5_000_000, quality: 0.75 },
  { pixels: 2_500_000, quality: 0.7 }
]

export type ImageShrinkStep = {
  width: number
  height: number
  quality: number
}

export type ImageShrinkEncoder = (
  step: ImageShrinkStep,
  type: string
) => Promise<Blob | null>

export const isShrinkableImage = (
  file: { size: number; type: string },
  limit: number
) =>
  file.size > limit &&
  file.type.startsWith('image/') &&
  !UNSHRINKABLE_TYPES.has(file.type)

export const shrinkTargetBytes = (limit: number) =>
  Math.floor(limit * TARGET_RATIO)

export const shrinkSteps = (
  width: number,
  height: number
): ImageShrinkStep[] => {
  const steps: ImageShrinkStep[] = []
  for (const { pixels, quality } of SHRINK_LADDER) {
    const scale = Math.min(
      1,
      Math.sqrt(pixels / (width * height)),
      MAX_CANVAS_EDGE / Math.max(width, height)
    )
    const step = {
      width: Math.max(1, Math.round(width * scale)),
      height: Math.max(1, Math.round(height * scale)),
      quality
    }
    const last = steps.at(-1)
    if (
      !last ||
      last.width !== step.width ||
      last.height !== step.height ||
      last.quality !== step.quality
    ) {
      steps.push(step)
    }
  }
  return steps
}

export const renameImageForType = (filename: string, type: string) => {
  const extension = type === JPEG ? 'jpg' : type.replace(/^image\//, '')
  const base = filename.replace(/\.[^./\\]*$/, '')
  return `${base || 'image'}.${extension}`
}

export const encodeFirstFit = async (
  steps: ImageShrinkStep[],
  targetBytes: number,
  encode: ImageShrinkEncoder
): Promise<Blob | null> => {
  let type = WEBP
  for (const step of steps) {
    let blob = await encode(step, type)
    if (blob && blob.type !== type && type === WEBP) {
      type = JPEG
      blob = await encode(step, type)
    }
    if (blob && blob.type === type && blob.size <= targetBytes) {
      return blob
    }
  }
  return null
}

const paint = (
  context: CanvasRenderingContext2D | OffscreenCanvasRenderingContext2D | null,
  bitmap: ImageBitmap,
  step: ImageShrinkStep,
  type: string
) => {
  if (!context) {
    return false
  }
  if (type === JPEG) {
    context.fillStyle = '#fff'
    context.fillRect(0, 0, step.width, step.height)
  }
  context.imageSmoothingQuality = 'high'
  context.drawImage(bitmap, 0, 0, step.width, step.height)
  return true
}

const canvasEncoder =
  (bitmap: ImageBitmap): ImageShrinkEncoder =>
  async (step, type) => {
    try {
      if (typeof OffscreenCanvas !== 'undefined') {
        const canvas = new OffscreenCanvas(step.width, step.height)
        if (!paint(canvas.getContext('2d'), bitmap, step, type)) {
          return null
        }
        return await canvas.convertToBlob({ type, quality: step.quality })
      }
      const canvas = document.createElement('canvas')
      canvas.width = step.width
      canvas.height = step.height
      if (!paint(canvas.getContext('2d'), bitmap, step, type)) {
        return null
      }
      return await new Promise<Blob | null>((resolve) =>
        canvas.toBlob(resolve, type, step.quality)
      )
    } catch {
      return null
    }
  }

const decodeImage = (file: Blob) =>
  createImageBitmap(file, { imageOrientation: 'from-image' }).catch(() =>
    createImageBitmap(file)
  )

export const shrinkImageToFit = async (
  file: Blob,
  limit: number
): Promise<Blob | null> => {
  let bitmap: ImageBitmap
  try {
    bitmap = await decodeImage(file)
  } catch {
    return null
  }
  try {
    return await encodeFirstFit(
      shrinkSteps(bitmap.width, bitmap.height),
      shrinkTargetBytes(limit),
      canvasEncoder(bitmap)
    )
  } finally {
    bitmap.close()
  }
}
