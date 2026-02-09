import { createSSRApp } from 'vue'
import App from './App.vue'
import { applyNavigationBarTheme, applyTabBarTheme, bootstrapTheme, syncCurrentPageCustomTabBar } from './utils/theme'

export function createApp() {
  bootstrapTheme()
  const app = createSSRApp(App)
  app.mixin({
    onLoad() {
      applyNavigationBarTheme()
      applyTabBarTheme()
      syncCurrentPageCustomTabBar(undefined, this)
    },
    onShow() {
      applyNavigationBarTheme()
      applyTabBarTheme()
      syncCurrentPageCustomTabBar(undefined, this)
    },
    onReady() {
      syncCurrentPageCustomTabBar(undefined, this)
    },
  } as any)
  return { app }
}
