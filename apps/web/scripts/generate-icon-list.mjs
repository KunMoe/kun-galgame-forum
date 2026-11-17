#!/usr/bin/env node
/**
 * Scans the app (apps/web/app) for
 * icon names used in `KunIcon` / `Icon` components and generates
 * `apps/web/lib/icon.ts` with two exports:
 *
 *   - ICON_NAMES: every `<collection>:<icon>` literal found
 *   - ICON_COLLECTIONS: the unique set of `<collection>` prefixes
 *
 * The output is consumed by `nuxt.config.ts`:
 *
 *   import { ICON_COLLECTIONS, ICON_NAMES } from './lib/icon'
 *   icon: {
 *     mode: 'svg',
 *     serverBundle: { collections: ICON_COLLECTIONS },
 *     clientBundle: { icons: ICON_NAMES, scan: false }
 *   }
 *
 * KunUI's own component icons are NOT scanned here: the published
 * @kungal/ui-vue renders <KunIcon> from its own bundled in-memory registry
 * (it never hits @nuxt/icon for them), so the forum's @nuxt/icon bundle only
 * needs the icons the forum itself references in apps/web/app.
 *
 * The collection allowlist is derived from the installed `@iconify-json/*`
 * packages (app package.json), so only bundleable collections are
 * emitted (a stray `foo:bar` CSS/time string can't leak in) and the list stays
 * correct when a collection is added or removed. Icons referenced from a
 * collection that ISN'T installed are reported as warnings — they cannot be
 * bundled or served locally; install the matching @iconify-json/* package (or
 * remove the usage).
 *
 * Both static (`name="lucide:x"`) and ternary (`:name="cond ? 'lucide:a' : 'lucide:b'"`)
 * usages are picked up. Dynamic names that come from variables (e.g.
 * `:name="item.icon"`) cannot be extracted statically — add those to
 * MANUAL_ICONS below.
 *
 * Usage:
 *   node scripts/generate-icon-list.mjs
 *   npm run icons     # via package.json
 */

import { readFile, readdir, mkdir, writeFile } from 'node:fs/promises'
import { dirname, join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const ROOT = join(__dirname, '..') // apps/web
// Only the app is scanned now. KunUI moved to the published @kungal/ui-vue,
// whose <KunIcon> renders from its OWN bundled in-memory registry (it never
// hits @nuxt/icon for its components' icons), so the forum's @nuxt/icon bundle
// only needs the icons the forum itself references.
const SCAN_DIRS = [join(ROOT, 'app')]
const OUTPUT_FILE = join(ROOT, 'lib/icon.ts')
const PACKAGE_JSONS = [join(ROOT, 'package.json')]

const SCAN_EXTENSIONS = new Set(['.vue', '.ts', '.tsx', '.js', '.mjs'])
const SKIP_DIRS = new Set(['node_modules', '.nuxt', '.output', 'dist', '.git'])

// Icons referenced indirectly (e.g. through props passed at runtime) that the
// regex below cannot find. Add manually if you spot a missing icon at runtime.
// The agent's default suggestion groups and upload options name their icons
// in the persona response; a site's own may name others, which load at runtime.
const MANUAL_ICONS = [
  'lucide:search',
  'lucide:book-open',
  'lucide:globe',
  'lucide:image',
  'lucide:file-text',
  'lucide:file'
]

// Icon references are always quoted string literals — `name="lucide:x"`,
// `:name="cond ? 'a:b' : 'c:d'"`, or map values like `icon: 'lucide:x'`.
// Matching only quoted `collection:icon` tokens avoids false positives from
// time strings (`12:34`), URLs (`https:`) and bare prose. The collection must
// start with a letter; the icon name MAY start with a digit (e.g.
// `svg-spinners:90-ring-with-bg`).
const ICON_PATTERN = /["']([a-z][a-z0-9-]{1,30}):([a-z0-9][a-z0-9-]{0,60})["']/g

// Tailwind variant prefixes (and the Vue `update:` event) also look like
// `prefix:rest` and can appear as standalone quoted tokens in conditional
// `:class` bindings. They are never icon collections — listed only to keep the
// "missing collection" warning quiet. The generated list is unaffected: it
// only ever includes INSTALLED @iconify-json collections.
const NON_ICON_PREFIXES = new Set([
  'sm',
  'md',
  'lg',
  'xl',
  '2xl',
  'dark',
  'light',
  'hover',
  'focus',
  'active',
  'visited',
  'target',
  'focus-within',
  'focus-visible',
  'group-hover',
  'group-focus',
  'peer-hover',
  'peer-focus',
  'peer-checked',
  'disabled',
  'enabled',
  'checked',
  'indeterminate',
  'default',
  'required',
  'valid',
  'invalid',
  'placeholder-shown',
  'autofill',
  'read-only',
  'empty',
  'open',
  'first',
  'last',
  'only',
  'odd',
  'even',
  'motion-safe',
  'motion-reduce',
  'contrast-more',
  'print',
  'portrait',
  'landscape',
  'rtl',
  'ltr',
  'before',
  'after',
  'placeholder',
  'file',
  'marker',
  'selection',
  'backdrop',
  'has',
  'not',
  'group',
  'peer',
  'aria',
  'data',
  'supports',
  'min',
  'max',
  'update'
])

// Read the installed @iconify-json/* packages from the app manifest.
// These are the only collections that can actually be bundled/served.
const installedCollections = async () => {
  const set = new Set()
  for (const p of PACKAGE_JSONS) {
    let pkg
    try {
      pkg = JSON.parse(await readFile(p, 'utf8'))
    } catch {
      continue
    }
    const deps = { ...pkg.dependencies, ...pkg.devDependencies }
    for (const dep of Object.keys(deps)) {
      const m = dep.match(/^@iconify-json\/(.+)$/)
      if (m) set.add(m[1])
    }
  }
  return set
}

async function* walk(dir) {
  let entries
  try {
    entries = await readdir(dir, { withFileTypes: true })
  } catch {
    return
  }
  for (const entry of entries) {
    if (SKIP_DIRS.has(entry.name)) continue
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      yield* walk(full)
    } else if (entry.isFile()) {
      const dot = entry.name.lastIndexOf('.')
      if (dot >= 0 && SCAN_EXTENSIONS.has(entry.name.slice(dot))) {
        yield full
      }
    }
  }
}

async function main() {
  const known = await installedCollections()
  const found = new Set(MANUAL_ICONS)
  const missingDeps = new Map() // collection -> Set(name) referenced but not installed
  let scanned = 0

  for (const dir of SCAN_DIRS) {
    for await (const file of walk(dir)) {
      scanned++
      const content = await readFile(file, 'utf8')
      for (const match of content.matchAll(ICON_PATTERN)) {
        const [, collection, icon] = match
        // Pure-numeric "icon" parts are domain strings (e.g. `'galgame:1207'`),
        // never real icons — real Iconify names always contain a letter.
        if (/^\d+$/.test(icon)) continue
        const name = `${collection}:${icon}`
        if (known.has(collection)) {
          found.add(name)
        } else if (!NON_ICON_PREFIXES.has(collection)) {
          // A collection that isn't installed (and isn't a Tailwind/Vue prefix)
          // — a likely missing @iconify-json dependency.
          if (!missingDeps.has(collection))
            missingDeps.set(collection, new Set())
          missingDeps.get(collection).add(name)
        }
      }
    }
  }

  const names = [...found].sort()
  const collections = [...new Set(names.map((n) => n.split(':')[0]))].sort()

  const body = render(names, collections)
  await mkdir(dirname(OUTPUT_FILE), { recursive: true })
  await writeFile(OUTPUT_FILE, body)

  const rel = relative(ROOT, OUTPUT_FILE)
  console.log(
    `Scanned ${scanned} files (app), found ${names.length} icons across ${collections.length} collections`
  )
  console.log(`Wrote ${rel}`)

  if (missingDeps.size) {
    console.warn(
      '\n⚠ Referenced icons from NOT-installed collections (broken — install @iconify-json/<collection> or remove the usage):'
    )
    for (const [collection, set] of [...missingDeps].sort()) {
      console.warn(
        `  @iconify-json/${collection}: ${[...set].sort().join(', ')}`
      )
    }
  }
}

function render(names, collections) {
  const namesLines = names.map((n) => `  '${n}'`).join(',\n')
  const collectionsLines = collections.map((c) => `  '${c}'`).join(',\n')
  return `/**
 * Auto-generated by scripts/generate-icon-list.mjs.
 * Do not edit by hand — run \`npm run icons\` after updating icon usage.
 */
export const ICON_NAMES: string[] = [
${namesLines}
]

export const ICON_COLLECTIONS: string[] = [
${collectionsLines}
]
`
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
