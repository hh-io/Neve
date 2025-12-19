---
trigger: model_decision
description: 适用于 Neve 记账系统：1. 修改 web 目录样式、组件或 pnpm 构建配置时；2. 涉及 server/web/data 中任何金额计算、财务逻辑或数据处理时。确保执行响应式适配及财务高精度要求。
globs: **/*
---

---
project: Neve
priority: high
---
## Neve Project Specs

1. **Package Management**:
   - 强制使用 `pnpm`。严禁生成 `package-lock.json` 或 `yarn.lock`。
   - 任何依赖安装或脚本运行指令必须以 `pnpm` 为准。

2. **Responsive-First Styling**:
   - 样式开发遵循“移动端优先”或“完全同步适配”原则。
   - **强制要求**：在提交任何 UI 组件或 CSS 修改时，必须同时提供相应的 Media Queries 或响应式布局方案（如 Tailwind 的 `sm:`, `md:` 前缀），确保在 iOS/Android 浏览器上的可用性。

3. **Data Integrity (Neve Core)**:
   - 考虑到记账系统的本质，涉及金额计算的代码必须使用高精度方案（如 `decimal.js` 或 `BigInt`），严禁直接使用浮点数进行财务运算。