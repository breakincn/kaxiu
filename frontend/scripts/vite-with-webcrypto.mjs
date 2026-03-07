import { webcrypto } from 'node:crypto'
import { createRequire } from 'node:module'
import { pathToFileURL } from 'node:url'
import path from 'node:path'

if (!globalThis.crypto || typeof globalThis.crypto.getRandomValues !== 'function') {
  globalThis.crypto = webcrypto
}

const require = createRequire(import.meta.url)
const nodeCrypto = require('node:crypto')
if (typeof nodeCrypto.getRandomValues !== 'function' && typeof webcrypto.getRandomValues === 'function') {
  nodeCrypto.getRandomValues = webcrypto.getRandomValues.bind(webcrypto)
}

const viteCliPath = path.resolve(process.cwd(), 'node_modules/vite/bin/vite.js')
await import(pathToFileURL(viteCliPath).href)
