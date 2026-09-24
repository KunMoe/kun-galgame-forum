import { defineVitestConfig } from '@nuxt/test-utils/config'

export default defineVitestConfig({
  test: {
    // Two environments coexist. This is the default, for pure ts/js utils;
    // component and composable specs opt in per file with the docblock
    // `// @vitest-environment nuxt`. That pragma is load-bearing — delete it
    // and the spec silently drops to happy-dom, where mountSuspended dies with
    // "Cannot read properties of undefined (reading 'vueApp')".
    environment: 'happy-dom',
    globals: true,
    // @nuxt/test-utils runs setupNuxt in its own beforeAll. On a loaded dev
    // machine (load ≈ 9.6) six happy-dom files failed at file level with
    // "Hook timed out in 10000ms"; the tests themselves never timed out.
    hookTimeout: 30_000,
    include: [
      'app/**/*.spec.ts',
      'shared/**/*.spec.ts',
      'server/**/*.spec.ts',
      'tests/**/*.spec.ts'
    ],
    exclude: ['**/node_modules/**', '**/dist/**', '**/.nuxt/**']
  }
})
