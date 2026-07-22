import { ref, computed, watch } from 'vue'
import { bumpThemeVersion } from './useThemeColor'

export type ThemeMode = 'light' | 'dark' | 'system'

const THEME_KEY = 'neve-theme'
const VALID: ThemeMode[] = ['light', 'dark', 'system']

// 模块级单例:全局唯一主题状态
const themeMode = ref<ThemeMode>('system')

const prefersDark = window.matchMedia('(prefers-color-scheme: dark)')

// system 跟随系统偏好,其余直出对应主题类;.app-layout 与 <html> 都会带上该类
const themeClass = computed(() => {
  if (themeMode.value === 'system') {
    return prefersDark.matches ? 'theme-dark' : 'theme-light'
  }
  return `theme-${themeMode.value}`
})

function applyTheme(): void {
  const html = document.documentElement
  html.classList.remove('theme-light', 'theme-dark')
  html.classList.add(themeClass.value)
  // canvas 不解析 CSS 变量,通知 ECharts 图表重新取实际颜色值(见 useThemeColor)
  bumpThemeVersion()
}

let initialized = false
function initTheme(): void {
  if (initialized) return
  initialized = true

  const saved = localStorage.getItem(THEME_KEY)
  if (saved && VALID.includes(saved as ThemeMode)) {
    themeMode.value = saved as ThemeMode
  }
  applyTheme()

  watch(themeClass, applyTheme)
  watch(themeMode, (mode) => localStorage.setItem(THEME_KEY, mode))
  prefersDark.addEventListener('change', () => {
    if (themeMode.value === 'system') applyTheme()
  })
}

export function useTheme() {
  initTheme()
  return { themeMode, themeClass }
}
