import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      // During build, /wails/runtime.js (used by generated v3 bindings)
      // resolves to the @wailsio/runtime npm package for type-checking.
      // At runtime in the Wails webview, this URL is served by the
      // embedded asset server directly — the alias is a build-time shim only.
      '/wails/runtime.js': '@wailsio/runtime',
    },
  },
})
