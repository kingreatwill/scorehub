import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

export default defineConfig(() => {
  const platform = String(process.env.UNI_PLATFORM || '')
  const isMpWeixin = platform === 'mp-weixin'
  const now = new Date()
  const buildDate = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`

  return {
    plugins: [uni()],
    define: {
      'import.meta.env.VITE_SCOREHUB_BUILD_DATE': JSON.stringify(buildDate),
    },
    build: {
      // WeChat runtime + async/generator transpilation can be sensitive to identifier mangling.
      // Disabling mangle for mp-weixin avoids rare "ref becomes undefined" runtime issues.
      minify: 'terser',
      terserOptions: {
        mangle: isMpWeixin ? false : { toplevel: false },
      },
    },
  }
})
