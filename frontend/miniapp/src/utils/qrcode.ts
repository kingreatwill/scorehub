import qrcode from 'qrcode-generator'

export type QRMatrix = boolean[][]

export function makeInviteCodeQRMatrix(text: string): QRMatrix {
  return makeTextQRMatrix(text)
}

export function makeTextQRMatrix(text: string): QRMatrix {
  const content = String(text || '').trim()
  if (!content) throw new Error('二维码内容为空')

  const qr = qrcode(0, 'M')
  qr.addData(content, 'Byte')
  qr.make()

  const n = qr.getModuleCount()
  return Array.from({ length: n }, (_, row) =>
    Array.from({ length: n }, (_, col) => qr.isDark(row, col)),
  )
}
