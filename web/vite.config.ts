import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../server/static',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        // Vite 8 底层为 Rolldown,分包用其原生 codeSplitting:
        // ECharts 单独成块(懒加载),Vue 核心单独成块。
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
