const fs = require('node:fs')
const path = require('node:path')

const repoRoot = path.resolve(__dirname, '..', '..', '..')
const sourceDir = path.join(repoRoot, 'frontend', 'miniapp', 'dist', 'build', 'h5')
const targetDir = path.join(repoRoot, 'backend', 'assets', 'h5')
const keepFile = '.gitkeep'

if (!fs.existsSync(sourceDir)) {
  console.error(`H5 build output not found: ${sourceDir}`)
  console.error('Run `uni build -p h5` first.')
  process.exit(1)
}

fs.mkdirSync(targetDir, { recursive: true })

for (const entry of fs.readdirSync(targetDir)) {
  if (entry === keepFile) continue
  fs.rmSync(path.join(targetDir, entry), { recursive: true, force: true })
}

fs.cpSync(sourceDir, targetDir, { recursive: true, force: true })

const dsStorePath = path.join(targetDir, 'static', '.DS_Store')
if (fs.existsSync(dsStorePath)) {
  fs.rmSync(dsStorePath, { force: true })
}

console.log(`Synced H5 build to backend assets: ${targetDir}`)
