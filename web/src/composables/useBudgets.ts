import { ref } from 'vue'
import { showToast } from './useToast'

// 模块级单例:预算按分类的月度上限(元),服务端原子写 + localStorage 本地备份
const budgets = ref<Record<string, number>>({})
let loaded = false

// 首屏加载:服务端不可达或响应异常时回退 localStorage 备份
async function loadBudgets(): Promise<void> {
  if (loaded) return
  loaded = true
  try {
    const res = await fetch('/api/budgets')
    if (res.ok) {
      budgets.value = await res.json()
      return
    }
    // 服务端明确报错(如 budgets.json 损坏)必须让用户知道:下面回退的是本地备份,
    // 用户在此基础上改动并保存会覆盖服务端那份(服务端会先把损坏内容留档)
    const body = await res.json().catch(() => null)
    showToast('预算读取失败,已回退本地备份: ' + (body?.error?.message || `HTTP ${res.status}`), 'error')
  } catch {
    // 网络错误同样走下方本地备份
  }
  try {
    budgets.value = JSON.parse(localStorage.getItem('neve-budgets') || '{}')
  } catch {
    // 本地备份损坏时置空,避免 loadBudgets 变成 rejected promise
    budgets.value = {}
  }
}

// 写回服务端(复用 refresh 的 429/错误处理模式),同时落 localStorage 备份
async function saveBudgets(): Promise<void> {
  const val = budgets.value
  localStorage.setItem('neve-budgets', JSON.stringify(val))
  try {
    const res = await fetch('/api/budgets', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(val)
    })
    if (res.status === 429) {
      const body = await res.json().catch(() => null)
      const wait = Math.ceil(body?.retryAfter ?? 5)
      showToast(`保存过于频繁,请 ${wait} 秒后再试`, 'error')
      return
    }
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
  } catch (e) {
    showToast('预算保存失败: ' + (e instanceof Error ? e.message : String(e)), 'error')
  }
}

export function useBudgets() {
  return { budgets, loadBudgets, saveBudgets }
}
