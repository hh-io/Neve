import { ref } from 'vue'
import type { Analytics } from '../types/api'
import { showToast } from './useToast'

// 模块级单例:一次 fetch 后经各 Tab 直接消费,替代 App.vue → Tab 的 prop 钻透
const analytics = ref<Analytics | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

async function fetchAnalytics(): Promise<Analytics> {
  const response = await fetch('/api/analytics')
  if (!response.ok) throw new Error('Failed to fetch analytics')
  return response.json()
}

function toMessage(e: unknown): string {
  return e instanceof Error ? e.message : String(e)
}

// 首屏加载:失败只落 error,不弹 toast(此时页面本就显示错误态)
async function load(): Promise<void> {
  loading.value = true
  try {
    analytics.value = await fetchAnalytics()
  } catch (e) {
    error.value = toMessage(e)
  } finally {
    loading.value = false
  }
}

async function refresh(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const response = await fetch('/api/refresh', { method: 'POST' })
    if (response.status === 429) {
      const body = await response.json().catch(() => null)
      const wait = Math.ceil(body?.retryAfter ?? 5)
      showToast(`刷新过于频繁,请 ${wait} 秒后再试`, 'error')
      // 服务端有缓存数据,仍拉取一次保证页面可用(如初次加载失败后的重试)
      if (!analytics.value) analytics.value = await fetchAnalytics()
      return
    }
    if (!response.ok) {
      const body = await response.json().catch(() => null)
      throw new Error(body?.error?.message || `HTTP ${response.status}`)
    }
    analytics.value = await fetchAnalytics()
    showToast('数据刷新成功', 'success')
  } catch (e) {
    error.value = toMessage(e)
    showToast('刷新失败: ' + error.value, 'error')
  } finally {
    loading.value = false
  }
}

export function useAnalytics() {
  return { analytics, loading, error, load, refresh }
}
