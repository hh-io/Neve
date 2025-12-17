import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: "../server/static",
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          // Separate ECharts into its own chunk (lazy loaded)
          'echarts': ['echarts', 'vue-echarts'],
          // Separate Vue core
          'vue-vendor': ['vue'],
        },
      },
    },
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
