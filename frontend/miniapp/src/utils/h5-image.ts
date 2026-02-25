const DEFAULT_MAX_BYTES = 200 * 1024
const DEFAULT_MAX_SIDE = 1600
const QUALITY_STEPS = [0.92, 0.86, 0.8, 0.74, 0.68, 0.62, 0.56, 0.5, 0.44, 0.38]

export type H5ImagePickOptions = {
  maxBytes?: number
  maxSide?: number
}

function normalizeMaxBytes(raw: number | undefined): number {
  const n = Number.parseInt(String(raw ?? ''), 10)
  if (!Number.isFinite(n) || n <= 0) return DEFAULT_MAX_BYTES
  return Math.max(50 * 1024, n)
}

function normalizeMaxSide(raw: number | undefined): number {
  const n = Number.parseInt(String(raw ?? ''), 10)
  if (!Number.isFinite(n) || n <= 0) return DEFAULT_MAX_SIDE
  return Math.max(256, n)
}

function estimateDataUrlBytes(dataUrl: string): number {
  const idx = dataUrl.indexOf(',')
  const base64 = idx >= 0 ? dataUrl.slice(idx + 1) : dataUrl
  const pad = base64.endsWith('==') ? 2 : base64.endsWith('=') ? 1 : 0
  return Math.floor((base64.length * 3) / 4) - pad
}

function chooseImagePath(): Promise<string> {
  return new Promise((resolve, reject) => {
    uni.chooseImage({
      count: 1,
      sizeType: ['compressed'],
      sourceType: ['album', 'camera'],
      success: (res: any) => {
        const path = String(res?.tempFilePaths?.[0] || res?.tempFiles?.[0]?.path || '').trim()
        if (!path) {
          reject(new Error('未读取到图片'))
          return
        }
        resolve(path)
      },
      fail: (err: any) => {
        reject(new Error(String(err?.errMsg || err?.message || '选择图片失败')))
      },
    } as any)
  })
}

function loadImage(blob: Blob): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(blob)
    const img = new Image()
    img.onload = () => {
      URL.revokeObjectURL(url)
      resolve(img)
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('图片解析失败'))
    }
    img.src = url
  })
}

async function compressBlobToDataUrl(blob: Blob, maxBytes: number, maxSide: number): Promise<string> {
  const img = await loadImage(blob)
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('图片处理失败')

  let width = Math.max(1, Math.round((img as any).naturalWidth || img.width || 1))
  let height = Math.max(1, Math.round((img as any).naturalHeight || img.height || 1))
  const maxDim = Math.max(width, height)
  if (maxDim > maxSide) {
    const scale = maxSide / maxDim
    width = Math.max(1, Math.round(width * scale))
    height = Math.max(1, Math.round(height * scale))
  }

  let best = ''
  for (let pass = 0; pass < 6; pass++) {
    canvas.width = width
    canvas.height = height
    ctx.clearRect(0, 0, width, height)
    // Convert to JPEG for predictable size; fill white background for transparent sources.
    ctx.fillStyle = '#FFFFFF'
    ctx.fillRect(0, 0, width, height)
    ctx.drawImage(img, 0, 0, width, height)

    for (const quality of QUALITY_STEPS) {
      const dataUrl = canvas.toDataURL('image/jpeg', quality)
      best = dataUrl
      if (estimateDataUrlBytes(dataUrl) <= maxBytes) return dataUrl
    }

    if (width <= 256 || height <= 256) break
    width = Math.max(256, Math.round(width * 0.85))
    height = Math.max(256, Math.round(height * 0.85))
  }

  if (best && estimateDataUrlBytes(best) <= maxBytes) return best
  throw new Error(`图片压缩后仍超过 ${Math.round(maxBytes / 1024)}KB，请换一张`)
}

async function fetchBlob(src: string): Promise<Blob> {
  const resp = await fetch(src)
  if (!resp.ok) throw new Error('读取图片失败')
  return await resp.blob()
}

export function isH5ImagePickCancelError(err: any): boolean {
  const msg = String(err?.message || err?.errMsg || err || '').toLowerCase()
  return msg.includes('cancel') || msg.includes('canceled') || msg.includes('取消')
}

export async function pickH5ImageAsDataUrl(options: H5ImagePickOptions = {}): Promise<string> {
  const maxBytes = normalizeMaxBytes(options.maxBytes)
  const maxSide = normalizeMaxSide(options.maxSide)

  // #ifndef H5
  throw new Error('仅支持 H5')
  // #endif

  // #ifdef H5
  const path = await chooseImagePath()
  const blob = await fetchBlob(path)
  return await compressBlobToDataUrl(blob, maxBytes, maxSide)
  // #endif
}

