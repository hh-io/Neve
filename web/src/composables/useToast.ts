import { ref, readonly } from 'vue'

export type ToastType = 'success' | 'error'

interface ToastState {
  show: boolean
  message: string
  type: ToastType
}

// 模块级单例:任意组件调用 showToast 都作用于同一个 toast,AppToast 自行订阅
const state = ref<ToastState>({ show: false, message: '', type: 'success' })
let timer: ReturnType<typeof setTimeout> | null = null

export function showToast(message: string, type: ToastType = 'success', duration = 3000): void {
  state.value = { show: true, message, type }
  if (timer) clearTimeout(timer)
  timer = setTimeout(() => {
    state.value.show = false
  }, duration)
}

export function useToast() {
  return { toast: readonly(state), showToast }
}
