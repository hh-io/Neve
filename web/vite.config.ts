import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../server/static',
    emptyOutDir: true,
    // 产物 embed 进二进制、只在本机/局域网访问,echarts 单块 ~730 kB 的传输成本可忽略,
    // 故抬高阈值消掉常态警告;仍留余量,真膨胀了会重新报出来。
    chunkSizeWarningLimit: 800,
    rollupOptions: {
      output: {
        // Vite 8 底层为 Rolldown,分包用其原生 codeSplitting:
        // ECharts 与 Vue 核心各自成块,按内容哈希独立缓存。
        // 注:六个 tab 在 App.vue 里都是静态 import,echarts 块随首屏同步加载,
        // 并非懒加载;首屏概览的饼图/日历热力图本来就需要它。
        codeSplitting: {
          groups: [
            { name: 'echarts', test: /[\\/]node_modules[\\/](echarts|zrender|vue-echarts)[\\/]/ },
            { name: 'vue-vendor', test: /[\\/]node_modules[\\/]@?vue[\\/]/ },
          ],
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
