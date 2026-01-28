export type QRMatrix = boolean[][]

// Minimal QR Code generator for our invite code use-case.
// Version: 1, Error correction: M, Mask: 0, Mode: Byte.
const VERSION = 1
const SIZE = 21 // 4*VERSION + 17
const ECC_LEVEL_BITS_M = 0b00
const MASK_PATTERN = 0
const DATA_CODEWORDS = 16
const ECC_CODEWORDS = 10

export function makeInviteCodeQRMatrix(text: string): QRMatrix {
  const bytes = utf8Encode(text)
  // Version 1-M, byte mode max is 14 bytes.
  if (bytes.length > 14) throw new Error('二维码内容过长')
  const codewords = makeCodewordsV1MByte(bytes)
  return makeMatrixV1(codewords, ECC_LEVEL_BITS_M, MASK_PATTERN)
}

function utf8Encode(input: string): number[] {
  const out: number[] = []
  for (let i = 0; i < input.length; i++) {
    const c = input.charCodeAt(i)
    if (c < 0x80) {
      out.push(c)
      continue
    }
    if (c < 0x800) {
      out.push(0xc0 | (c >> 6))
      out.push(0x80 | (c & 0x3f))
      continue
    }
    // surrogate pair
    if (c >= 0xd800 && c <= 0xdbff && i + 1 < input.length) {
      const n = input.charCodeAt(i + 1)
      if (n >= 0xdc00 && n <= 0xdfff) {
        const cp = 0x10000 + ((c - 0xd800) << 10) + (n - 0xdc00)
        out.push(0xf0 | (cp >> 18))
        out.push(0x80 | ((cp >> 12) & 0x3f))
        out.push(0x80 | ((cp >> 6) & 0x3f))
        out.push(0x80 | (cp & 0x3f))
        i++
        continue
      }
    }
    out.push(0xe0 | (c >> 12))
    out.push(0x80 | ((c >> 6) & 0x3f))
    out.push(0x80 | (c & 0x3f))
  }
  return out
}

function makeCodewordsV1MByte(dataBytes: number[]): number[] {
  const bits: number[] = []
  const capBits = DATA_CODEWORDS * 8

  appendBits(bits, 0b0100, 4) // byte mode
  appendBits(bits, dataBytes.length, 8) // count (v1..9: 8 bits)
  for (const b of dataBytes) appendBits(bits, b & 0xff, 8)

  const remaining = capBits - bits.length
  if (remaining > 0) appendBits(bits, 0, Math.min(4, remaining))
  while (bits.length % 8 !== 0) bits.push(0)

  const data: number[] = []
  for (let i = 0; i < bits.length; i += 8) {
    let v = 0
    for (let j = 0; j < 8; j++) v = (v << 1) | (bits[i + j] || 0)
    data.push(v)
  }

  const pad = [0xec, 0x11]
  for (let i = 0; data.length < DATA_CODEWORDS; i++) {
    data.push(pad[i % 2])
  }

  const ecc = rsCompute(data, ECC_CODEWORDS)
  return data.concat(ecc)
}

function appendBits(out: number[], value: number, length: number) {
  for (let i = length - 1; i >= 0; i--) out.push((value >> i) & 1)
}

function makeMatrixV1(codewords: number[], eccLevelBits: number, maskPattern: number): QRMatrix {
  const n = SIZE
  const modules: boolean[][] = Array.from({ length: n }, () => Array.from({ length: n }, () => false))
  const isFunction: boolean[][] = Array.from({ length: n }, () => Array.from({ length: n }, () => false))

  const setFunctionModule = (r: number, c: number, v: boolean) => {
    if (r < 0 || c < 0 || r >= n || c >= n) return
    modules[r][c] = v
    isFunction[r][c] = true
  }

  const placeFinder = (row: number, col: number) => {
    for (let r = -1; r <= 7; r++) {
      for (let c = -1; c <= 7; c++) {
        const rr = row + r
        const cc = col + c
        if (rr < 0 || cc < 0 || rr >= n || cc >= n) continue

        // separator
        if (r === -1 || r === 7 || c === -1 || c === 7) {
          setFunctionModule(rr, cc, false)
          continue
        }

        const isBorder = r === 0 || r === 6 || c === 0 || c === 6
        const isCenter = r >= 2 && r <= 4 && c >= 2 && c <= 4
        setFunctionModule(rr, cc, isBorder || isCenter)
      }
    }
  }

  placeFinder(0, 0)
  placeFinder(0, n - 7)
  placeFinder(n - 7, 0)

  // timing patterns
  for (let i = 8; i < n - 8; i++) {
    const v = i % 2 === 0
    setFunctionModule(6, i, v)
    setFunctionModule(i, 6, v)
  }

  // reserve format info (will be written later)
  for (let i = 0; i < 15; i++) {
    if (i < 6) setFunctionModule(i, 8, false)
    else if (i < 8) setFunctionModule(i + 1, 8, false)
    else setFunctionModule(n - 15 + i, 8, false)

    if (i < 8) setFunctionModule(8, n - i - 1, false)
    else if (i === 8) setFunctionModule(8, 7, false)
    else setFunctionModule(8, 15 - i - 1, false)
  }

  // data placement
  const totalBits = codewords.length * 8
  const getBit = (i: number): boolean => {
    if (i < 0 || i >= totalBits) return false
    const b = codewords[(i / 8) | 0]
    return ((b >> (7 - (i % 8))) & 1) === 1
  }

  let bitIndex = 0
  let row = n - 1
  let dir = -1
  for (let col = n - 1; col > 0; col -= 2) {
    if (col === 6) col--
    for (;;) {
      for (let cc = col; cc >= col - 1; cc--) {
        if (isFunction[row][cc]) continue
        let v = getBit(bitIndex++)
        if (maskPattern === 0) {
          if (((row + cc) & 1) === 0) v = !v
        }
        modules[row][cc] = v
      }
      row += dir
      if (row < 0 || row >= n) {
        row -= dir
        dir = -dir
        break
      }
    }
  }

  // format info (includes fixed dark module at (n-8, 8))
  const format = makeFormatBits(eccLevelBits, maskPattern)
  for (let i = 0; i < 15; i++) {
    const v = ((format >> i) & 1) === 1
    if (i < 6) setFunctionModule(i, 8, v)
    else if (i < 8) setFunctionModule(i + 1, 8, v)
    else setFunctionModule(n - 15 + i, 8, v)

    if (i < 8) setFunctionModule(8, n - i - 1, v)
    else if (i === 8) setFunctionModule(8, 7, v)
    else setFunctionModule(8, 15 - i - 1, v)
  }
  setFunctionModule(n - 8, 8, true)

  return modules
}

function makeFormatBits(eccLevelBits: number, maskPattern: number): number {
  const data = ((eccLevelBits & 0x3) << 3) | (maskPattern & 0x7)
  let v = data << 10
  const g = 0b10100110111
  while (msb(v) >= msb(g)) {
    v ^= g << (msb(v) - msb(g))
  }
  const bits = ((data << 10) | v) ^ 0b101010000010010 // 0x5412
  return bits & 0x7fff
}

function msb(v: number): number {
  let n = -1
  while (v > 0) {
    v >>= 1
    n++
  }
  return n
}

// --- Reed-Solomon (GF(256), primitive 0x11d) ---
const gfExp: number[] = []
const gfLog: number[] = []

initGF()

function initGF() {
  if (gfExp.length) return
  gfExp.length = 512
  gfLog.length = 256
  let x = 1
  for (let i = 0; i < 255; i++) {
    gfExp[i] = x
    gfLog[x] = i
    x <<= 1
    if (x & 0x100) x ^= 0x11d
  }
  for (let i = 255; i < 512; i++) gfExp[i] = gfExp[i - 255]
}

function gfMul(a: number, b: number): number {
  if (a === 0 || b === 0) return 0
  return gfExp[gfLog[a] + gfLog[b]]
}

function rsGeneratorPoly(degree: number): number[] {
  let poly = [1]
  for (let i = 0; i < degree; i++) {
    poly = polyMul(poly, [1, gfExp[i]])
  }
  return poly
}

function polyMul(a: number[], b: number[]): number[] {
  const out = Array(a.length + b.length - 1).fill(0)
  for (let i = 0; i < a.length; i++) {
    for (let j = 0; j < b.length; j++) {
      out[i + j] ^= gfMul(a[i], b[j])
    }
  }
  return out
}

function rsCompute(data: number[], eccLen: number): number[] {
  const gen = rsGeneratorPoly(eccLen) // length eccLen+1
  const ecc = Array(eccLen).fill(0)
  for (const b of data) {
    const factor = (b ^ ecc[0]) & 0xff
    for (let i = 0; i < eccLen - 1; i++) ecc[i] = ecc[i + 1]
    ecc[eccLen - 1] = 0
    for (let i = 0; i < eccLen; i++) {
      ecc[i] ^= gfMul(gen[i + 1], factor)
    }
  }
  return ecc.map((x) => x & 0xff)
}

