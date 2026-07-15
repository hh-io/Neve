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
  // Phase 4 组件全部改为 <script setup lang="ts"> 后,升级为 recommended
  // (含模板风格/属性顺序等规则),block-lang 默认强制全量 TS。
  pluginVue.configs['flat/recommended'],
  vueTsConfigs.recommended,
  skipFormatting,
  {
    name: 'app/rules',
    rules: {
      // 本项目模板统一用 camelCase 自定义 props/事件(activeTab / dailyData 等)。
      // 关掉连字符化两条规则:①保持与代码约定一致;②Vue 3 自定义事件名大小写敏感,
      // 若把 @update:activeTab 改成 @update:active-tab 会与 emit 的 update:activeTab 失配、
      // 断开 Tab 切换(见 REFACTOR_PLAN Phase 1 落地记录)。
      'vue/attribute-hyphenation': 'off',
      'vue/v-on-event-hyphenation': 'off',
    },
  },
)
