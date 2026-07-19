import { ref } from 'vue'
import { showToast } from './useToast'
import type { DebtsConfig, DebtsReport, DebtsResponse } from '../types/api'

function emptyConfig(): DebtsConfig {
  return { revolving: {}, installments: [] }
}

// 模块级单例:负债账单配置 + 后端现算的还款报告
// report 由后端唯一计算(账期快照/倒计时口径),前端不做本地推算
const config = ref<DebtsConfig>(emptyConfig())
const report = ref<DebtsReport | null>(null)
let loaded = false

// 首屏加载:服务端不可达时仅回退 localStorage 的配置备份;
// report 无法本地计算,保持 null 由 UI 提示"计算不可用"
async function loadDebts(): Promise<void> {
  if (loaded) return
  loaded = true
  try {
    const res = await fetch('/api/debts')
    if (res.ok) {
      const data: DebtsResponse = await res.json()
      config.value = data.config
      report.value = data.report
      return
    }
  } catch {
    // 网络错误与非 2xx 同样走下方本地备份
  }
  try {
    config.value = JSON.parse(localStorage.getItem('neve-debts') || '') as DebtsConfig
  } catch {
    config.value = emptyConfig()
  }
}

// 保存整份配置;成功后用响应回填(拿到后端规范化的 config 与重算的 report)。
// 返回是否成功,供编辑表单决定是否退出编辑态。
async function saveDebts(next: DebtsConfig): Promise<boolean> {
  localStorage.setItem('neve-debts', JSON.stringify(next))
  try {
    const res = await fetch('/api/debts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(next)
    })
    if (res.status === 429) {
      const body = await res.json().catch(() => null)
      const wait = Math.ceil(body?.retryAfter ?? 5)
      showToast(`保存过于频繁,请 ${wait} 秒后再试`, 'error')
      return false
    }
    if (res.status === 400) {
      const body = await res.json().catch(() => null)
      const detail = body?.details?.[0] ?? body?.error?.message ?? '配置不合法'
      showToast('保存失败: ' + detail, 'error')
      return false
    }
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data: DebtsResponse = await res.json()
    config.value = data.config
    report.value = data.report ?? report.value
    showToast('待还配置已保存')
    return true
  } catch (e) {
    showToast('待还配置保存失败: ' + (e instanceof Error ? e.message : String(e)), 'error')
    return false
  }
}

export function useDebts() {
  return { config, report, loadDebts, saveDebts }
}
