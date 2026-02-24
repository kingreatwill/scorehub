import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

export default defineConfig(() => {
  const platform = String(process.env.UNI_PLATFORM || '')
  const isMpWeixin = platform === 'mp-weixin'

  return {
    plugins: [uni()],
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
