# Changelog

本项目所有值得记录的变更都记在此文件。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

## [UI 重设计] - 2026-07-23

将前端整体重设计为靛蓝 Linear 风格,建立全新设计 token 体系并逐页对齐设计稿。纯前端展示层与命名规范化改动,**未触及解析/统计逻辑、API 契约与数据流**。

### 新增

- 全新 CSS 变量设计 token 体系:surface 阶梯、发丝线、accent 与 `--chart-*` 色板,亮/暗双主题。
- 新「N」monogram Logo 组件 `AppLogo.vue`(inline SVG,`currentColor` 跟随容器)。
- 侧边栏与移动端底部导航共用配置 `navItems.ts`,底部导航用短名(`short`)避免窄栏挤压。
- Makefile 支持一条命令同时启动前后端开发服务器。

### 变更

- 概览/收支/趋势/账户/待还/交易全部 Tab 对齐设计稿,替换为更精美的图标。
- 消费日历热力图迁至概览页;交易日历加宽右栏并放大格子。
- 账户页重构为分组卡片,补全账户中文名映射(全路径 → 末段 → 原文的回退链)。
- 待还页重构为 2 列卡片网格。
- 账户命名规范化 `CMBC → CMB`(招商银行正确缩写),贯穿 `data.example`、`debts.json` 与 AI 提示词。

### 移除

- 移除 `stripe` 主题,回退为亮色 / 暗色 / 跟随系统三态;localStorage 中残留的 `stripe` 值经校验优雅回退到 `system`。

### 修复

- 5 处静态内联 `style` 提为 scoped 修饰类,符合「组件禁止内联 style」约定。

### 文档

- README / CLAUDE.md 对齐实际工具链版本要求:Go 1.26+(对齐 `go.mod`)、Node 20.19+(Vite 8 engines 要求)。
