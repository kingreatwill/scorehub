const fs = require('node:fs')
const path = require('node:path')

const nextRootArg = process.argv[2]
if (!nextRootArg) {
  console.error('Usage: node scripts/set-miniprogram-root.cjs <miniprogramRoot>')
  process.exit(1)
}

const nextRoot = nextRootArg.endsWith('/') ? nextRootArg : `${nextRootArg}/`
const configPath = path.resolve(__dirname, '..', 'project.config.json')

let config
try {
  config = JSON.parse(fs.readFileSync(configPath, 'utf8'))
} catch (e) {
  console.error(`Failed to read ${configPath}`)
  throw e
}

if (config.miniprogramRoot === nextRoot) process.exit(0)
config.miniprogramRoot = nextRoot
fs.writeFileSync(configPath, JSON.stringify(config, null, 2) + '\n')

