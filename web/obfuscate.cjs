#!/usr/bin/env node
// Post-build JS obfuscation for production
// Run: node obfuscate.cjs

const fs = require('fs')
const path = require('path')
const JavaScriptObfuscator = require('javascript-obfuscator')

const distDir = path.join(__dirname, 'dist', 'assets')

const obfuscationOptions = {
  compact: true,
  controlFlowFlattening: false,
  deadCodeInjection: false,
  debugProtection: false,
  debugProtectionInterval: 0,
  disableConsoleOutput: false,
  identifierNamesGenerator: 'hexadecimal',
  log: false,
  numbersToExpressions: false,
  renameGlobals: false,
  selfDefending: false,
  simplify: true,
  splitStrings: true,
  splitStringsChunkLength: 15,
  stringArray: true,
  stringArrayEncoding: [],
  stringArrayIndexShift: true,
  stringArrayRotate: true,
  stringArrayShuffle: true,
  stringArrayWrappersCount: 1,
  stringArrayWrappersChainedCalls: false,
  stringArrayWrappersParametersMaxCount: 2,
  stringArrayWrappersType: 'function',
  stringArrayThreshold: 0.35,
  transformObjectKeys: false,
  unicodeEscapeSequence: false,
}

function obfuscateFile(filePath) {
  const code = fs.readFileSync(filePath, 'utf-8')
  const result = JavaScriptObfuscator.obfuscate(code, obfuscationOptions)
  fs.writeFileSync(filePath, result.getObfuscatedCode(), 'utf-8')
}

function shouldObfuscate(fileName) {
  // Keep Vue runtime, app bootstrap, and heavy vendor chunks stable and fast.
  // Business route chunks remain obfuscated.
  if (/^(index|echarts|vue|element|vendor|rolldown-runtime)-.*\.js$/.test(fileName)) return false
  return true
}

function main() {
  if (!fs.existsSync(distDir)) {
    console.error('dist/assets not found, run vite build first')
    process.exit(1)
  }

  const files = fs.readdirSync(distDir).filter(f => f.endsWith('.js'))
  const targets = files.filter((file) => {
    if (!shouldObfuscate(file)) return false
    return fs.statSync(path.join(distDir, file)).size <= 180 * 1024
  })
  const skipped = files.filter(f => !targets.includes(f))
  console.log(`Obfuscating ${targets.length} JS files...`)
  if (skipped.length) {
    console.log(`Skipping runtime/vendor chunks: ${skipped.join(', ')}`)
  }

  for (const file of targets) {
    const filePath = path.join(distDir, file)
    const before = fs.statSync(filePath).size
    obfuscateFile(filePath)
    const after = fs.statSync(filePath).size
    console.log(`  ✓ ${file} (${(before / 1024).toFixed(0)}KB → ${(after / 1024).toFixed(0)}KB)`)
  }

  console.log(`Done. ${targets.length} files obfuscated, ${skipped.length} skipped.`)
}

main()
