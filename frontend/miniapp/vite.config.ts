import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

export default defineConfig({
  plugins: [uni()],
  build: {
    // Avoid top-level identifier mangling to prevent name collisions after transpilation/minification
    // in the WeChat runtime (e.g. `n` from `require()` being shadowed by hoisted `var n`).
    minify: 'terser',
    terserOptions: {
      mangle: {
        toplevel: false,
      },
    },
  },
})
