import pluginVue from 'eslint-plugin-vue'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,vue}'],
  },
  {
    name: 'app/files-to-ignore',
    ignores: ['**/dist/**', '**/node_modules/**'],
  },
  // 渐进式重构期用 essential(只查正确性,不含模板风格/排序规则),避免对尚未重写的
  // 遗留 JS 模板做有风险的自动重排(如把 @update:activeTab 误改成 kebab 会断开自定义事件)。
  // Phase 4 组件全部改为 TS 并统一风格后,再升级为 flat/recommended。
  pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,
  skipFormatting,
  {
    name: 'app/rules',
    rules: {
      // 渐进式 TS 迁移期:现有组件仍为 JS <script setup>。Phase 4 逐个改为 lang="ts" 后,
      // 再删除此覆盖以强制全量 TS。
      'vue/block-lang': 'off',
    },
  },
)
