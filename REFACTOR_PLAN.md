# Neve 前端重构执行计划

> 本文档是一份**可直接执行**的重构规划,由项目审计产出。执行者(Claude Opus)按 Phase 顺序落地,
> 每个 Phase 是一个独立可提交的单元(中文 conventional commit)。执行前必读根目录 `CLAUDE.md`
> 的"正确性约定"一节——本次重构**不改变任何记账口径与后端统计逻辑**。

## 执行进度

| Phase | 状态 | 说明 |
|---|---|---|
| Phase 1 工具链与类型契约 | ✅ 已完成 | commit `bdd4289`;见下方"Phase 1 落地记录" |
| Phase 2 设计系统重建 | ✅ 已完成 | 新 token 体系 + legacy 别名共存;三主题目检通过 |
| Phase 3 基础设施重构 | ✅ 已完成 | lucide 图标统一 + 数据/主题/Toast 单例;prop 移除见落地记录 |
| Phase 4 组件逐个重写 | ⬜ 未开始 | |
| Phase 5 后端小清理 | ✅ 已完成 | commit `e661ed1`;顺带清理了只写不读的 `s.ledger` 字段 |
| Phase 6 文档同步 | ⬜ 未开始 | |

### Phase 1 落地记录(与原计划的偏差,下个会话必读)

- **依赖已升级到最新**(用户要求):`vue@3.5.39`、`echarts@6.1.0`、`vite@8.1.4`、
  `vue-echarts@8`。**TypeScript 刻意停在 5.9**——TS 7(tsgo 原生版)移除了 `./lib/tsc`
  子路径,会让 `vue-tsc` 崩溃,**不要升 TS 到 7**。
- **Vite 8 底层是 Rolldown**(非 Rollup):分包在 `vite.config.ts` 用
  `rollupOptions.output.codeSplitting: { groups }`(旧 `manualChunks` / `advancedChunks` 会报弃用)。
- **echarts 升 6** 顺带修复了原本 `vue-echarts@8` 要求 echarts^6 却装着 echarts 5 的 peer 不匹配。
- **ESLint 现用 `flat/essential`**(只查正确性,不含模板风格/排序规则),且 **`vue/block-lang` 关闭**
  (见 `eslint.config.js` 注释)。原因:组件仍是 JS `<script setup>`。
  **Phase 4 收尾时**再:①把组件全改成 `<script setup lang="ts">`;②ESLint 升级为 `flat/recommended`;
  ③删掉 `vue/block-lang` 覆盖。⚠️ 切勿现在用 `flat/recommended` 对遗留 JS 模板跑 `--fix`——
  它会把 `@update:activeTab` 误改成 kebab,而 Vue 3 自定义事件名大小写敏感,会断开 Tab 切换。
- `themeVersion.value;` 的响应式触碰惯例已统一改为 `void themeVersion.value;`(满足 no-unused-expressions,
  语义与行为不变,Phase 4 保留)。
- **可视化目检待补**:执行环境浏览器扩展常离线,Phase 1 只完成了 `make build` 全绿 + API 契约核对,
  echarts 5→6 / vite 7→8 的三主题图表**人工目检尚未完成**,下个会话开工前建议先跑
  `NEVE_DATA_DIR=./data.example TZ=Asia/Shanghai ./neve` 目检一遍作为基线。

## 0. 审计结论与决策记录

| 议题 | 决策 | 理由 |
|---|---|---|
| 迁移 Bun | **否决** | 运行时是 Go 二进制,Node 仅构建期使用;仅能加速依赖安装(3 个依赖),换来 Vite 兼容风险与工具链迁移成本,零运行时收益 |
| JS → TS | **采纳(渐进式)** | 单独迁移收益不足;但与 UI 组件重写合并执行边际成本≈0。核心收益:固化 `/api/analytics` 契约类型,防前后端脱节 |
| UI 重构 | **采纳** | 借鉴 Linear(结构/层级/克制)与 Revolut(金融数字表达),详见 Phase 2 |
| 引入 Pinia / Vue Router / 动画库 / 日期库 | **否决** | 单次 fetch + props 下发无需状态库;Tab 切换无需路由;CSS transition 够用;日期逻辑简单 |
| 引入 lucide-vue-next | **采纳** | 根治双图标系统与 `v-html` 注入 SVG 的问题 |
| 引入 ESLint + Prettier | **采纳** | 前端目前零 lint,与提交前 lint 的工作流约定冲突 |
| ECharts | **保留** | 日历热力图等有真实使用,已做 chunk 分离;更换图表库是纯成本 |
| 后端重写 | **否决** | 定点金额/软失败/口径统一/带测试,是项目质量最高的部分;仅做 Phase 5 的小清理 |
| 前端推倒重写(换框架) | **否决** | 问题在表现层腐化(内联样式/图标混乱/职责混杂),不在 Vue 本身;结构性重构即可 |

### 现状问题清单(重构要消灭的东西)

1. **217 处内联 `style="..."`**,集中在 `tabs/*.vue`(OverviewTab 74 处、TrendsTab 69 处、SpendingTab 35 处),设计系统形同虚设。
2. **两套图标系统并存**:`components/icons/*.vue`(SFC 组件)与 `composables/icons.js`(SVG 字符串 + 12 个文件里的 `v-html`)。
3. **App.vue 职责混杂**:主题管理、数据加载、Toast 三种逻辑挤在一起,且 `<style>` 块里放着侧边栏统计卡的样式。
4. **图表色板硬编码**:如 `OverviewTab.vue` 的 `pieColors = ['#C27B7B', ...]`,不随三主题切换。
5. **无类型契约**:后端 `analytics.go` 十几个 struct 输出的 JSON,前端全靠 `?.` 和 `|| 0` 防御。
6. **无前端 lint/format**。
7. 后端存在前端从未调用的冗余端点(`/api/summary`、`/api/transactions`、`/api/accounts`)与无实际作用的 CORS `*` 中间件(开发经 Vite 代理、生产同源)。

---

## Phase 1:工具链与类型契约(不动任何 UI) ✅ 已完成(commit `bdd4289`)

**目标**:建立 TS、lint、类型契约三件基础设施,全部绿灯后再开始重构。

1. `web/` 引入 TypeScript:
   - `pnpm add -D typescript vue-tsc @vue/tsconfig`
   - 新建 `web/tsconfig.json`,继承 `@vue/tsconfig/tsconfig-dom.json`,开启 `strict`,
     `allowJs: true`(允许 JS/TS 混存,支撑渐进迁移)。
   - `vite.config.js` → `vite.config.ts`,`main.js` → `main.ts`。
2. 新建 `web/src/types/api.ts`:**逐字段对照** `server/parser/analytics.go` 与 `server/parser/parser.go`
   的 struct JSON tag,定义 `Analytics`、`Summary`、`Transaction`、`Posting`、`CategoryAmount`、
   `MonthlyData`、`DailyData`、`WeeklyData`、`TagStats`、`PayeeStats`、`WeekdayStats`、
   `CategoryTrend`、`LiabilityStats`、`IncomeSource`、`ParseIssue`、`BalanceCheck` 等接口。
   注意:后端 `Amount` 序列化为**元的数字**(number),`kind` 是字符串字面量联合类型
   `'expense' | 'income' | 'transfer' | 'opening' | 'mixed'`。
3. composables 转 TS:`useCategories.js`、`useFormatters.js`、`useThemeColor.js`、`icons.js`
   (icons.js 在 Phase 3 会被删除,可暂不迁移)。
4. 引入 ESLint(flat config)+ `eslint-plugin-vue` + Prettier;
   `package.json` 增加 `lint`、`format`、`typecheck`(`vue-tsc --noEmit`)三个 script。
5. Makefile:`build-web` 前置执行 `pnpm run lint && pnpm run typecheck`;`make help` 同步更新。

**验收**:`make build` 全绿;浏览器行为与重构前完全一致。 → ✅ `make build` 已全绿(人工目检待补,见上方落地记录)
**提交**:`chore: 前端引入 TypeScript 与 ESLint 工具链,固化 API 契约类型` → ✅ 已提交 `bdd4289`

---

## Phase 2:设计系统重建(只改 token 与全局样式)

**目标**:重写 `web/src/styles/variables.css` 为新 token 体系,三主题(light/dark/geek)全部保留。
本 Phase 只建立体系并让现有语义类(`components.css` 等)映射到新 token,不逐个改组件。

### 2.1 借鉴 Linear 的部分(结构与克制)

- **Surface 阶梯代替阴影**:每主题定义 `--canvas`(页面底)、`--surface-1`(卡片)、
  `--surface-2`(悬停/嵌套面板)、`--surface-3`(下拉/浮层),配 `--hairline`、`--hairline-strong`
  两级 1px 发丝线。深色主题**完全去掉 box-shadow**,层级只靠 surface 提升 + 发丝线
  (浅色主题可保留极轻的 shadow-sm)。
- **单一稀缺强调色**:`--accent` 每主题一个(light 保留现有低饱和青 `#5B9A9A` 一系,geek 保留
  Indigo 一系),只用于:品牌标识、主按钮、focus ring、选中态。**不做**第二强调色。
- **圆角体系**(替换现有 8/12/16/20):`--radius-sm: 6px`(chips/badge 内嵌小件)、
  `--radius-md: 8px`(按钮、输入框)、`--radius-lg: 12px`(卡片)、`--radius-xl: 16px`(图表面板)、
  `--radius-full`(药丸)。
- **eyebrow 标签规范**:卡片小标题(如"净资产"、"本月收入")统一 12–13px / 500 / `letter-spacing: 0.4px`
  / `--text-tertiary`,与大数字形成 Linear 式的"分类标签 vs 内容"对比。
- **focus ring**:统一 `outline: 2px solid color-mix(in srgb, var(--accent) 50%, transparent)`。
- **交易列表 = changelog-row 模式**:行间只用 `--hairline` 底边分隔,无卡片嵌套、无斑马纹。

### 2.2 借鉴 Revolut 的部分(金融数字表达)

- **金额数字排版**:新增 token `--font-numeric`,所有金额展示统一
  `font-variant-numeric: tabular-nums`;大额展示(概览四卡的主数值)用
  28–32px / 500 / `line-height 1.1` / `letter-spacing -0.02em` 的紧排大数字。
- **药丸筛选 chips**:交易页/收支页的筛选器统一为 `--radius-full` 药丸,选中态 = surface 提升
  (`--surface-2` + `--text-primary`),非选中 = 透明 + `--text-tertiary`。
- **语义色纪律**:income/expense/warning/info 四组语义色**只出现在数据上**
  (金额文字、图表、进度条、badge),按钮与导航等 chrome 一律用 `--accent` 或中性色。
- **移动端触控**:底部导航与所有可点目标 ≥44px。
- **卡片留白**:统一卡片内边距 token `--card-pad: 24px`,移动端 16px。

### 2.3 图表色板 token 化

- 每主题定义 `--chart-1` … `--chart-8`(分类色板,从现有 `pieColors` 出发,按主题调明度/饱和),
  以及 `--chart-income`、`--chart-expense`(映射语义色)。
- `useThemeColor.js` 保持现有机制(canvas 不解析 CSS 变量,必须 `getThemeColor()` 取实值 +
  `themeVersion` 触发重算)——**这是 CLAUDE.md 的硬约定,不得绕开**。

**验收**:三主题切换正常,现有页面无明显视觉回归(允许 token 值微调带来的色差)。 → ✅ light/dark/geek 逐一目检通过(演示数据),token 均正确解析。
**提交**:`feat: 重建设计 token 体系(surface 阶梯/发丝线/数字排版/图表色板)`

### Phase 2 落地记录(Phase 4 必读)

- **兼容策略=新旧 token 共存**:`variables.css` 每个主题类同时导出新 token
  (`--canvas`/`--surface-1..3`/`--hairline`/`--hairline-strong`/`--accent`/`--accent-hover`/
  `--accent-subtle`/`--chart-1..8`/`--chart-income`/`--chart-expense`)与 legacy 别名
  (`--bg-*`/`--border*`/`--brand-*`/`--income` 等),二者指向同一底色。**未删任何旧名**——
  现有组件与 `components.css` 零改动继续工作。Phase 4 逐组件切到新名后,可回收 legacy 别名。
- **圆角 token 值已换**:`--radius-sm/md/lg/xl` 从 8/12/16/20 改为 6/8/12/16(全局收紧),
  组件无需改动即生效。
- **深色/geek 阴影 token 置为 `transparent`**:层级只靠 surface 阶梯 + 发丝线;浅色保留极轻阴影。
  刷新按钮/logo 的硬编码 glow 不受影响(它们不走 `--shadow`)。
- **新增全局件**:`base.css` 加了 `:focus-visible` 统一焦点环(`color-mix` + `--accent`)、
  `.eyebrow`(12px/500/0.4px letter-spacing/tertiary)、`.tabular-nums` 工具类;
  `components.css` 的 `.stat-value`/`.transaction-amount` 已接 `--font-numeric` + `tabular-nums`。
- **`--card-pad: 24px`** 已定义但**尚未接入**(旧 `.stat-card` 仍用 `--space-6`);Phase 4 重排卡片时切换。
- **图表色板仍是组件硬编码**:`OverviewTab` 的 `pieColors`、`SpendingTab`/`TrendsTab` 的 `color:[...]`
  尚未改用 `--chart-*`——这是 **Phase 4** 的接线工作(需配合 `getThemeColor()` + `themeVersion`)。
- ⚠️ **CSS 注释里禁止出现 `*/` 字符序列**(如写 `--border*/--brand-*`):postcss-import 会把它
  当成注释提前闭合并报 `Unknown word`。本次已踩过一次,注释统一改用"bg / border 系列"这类措辞。

---

## Phase 3:基础设施重构(状态、图标、App.vue 拆分)

1. **图标统一**:`pnpm add lucide-vue-next`。删除 `components/icons/` 整个目录与
   `composables/icons.js`,全部 `v-html` 图标替换为 lucide 组件(按需具名导入,可 tree-shake)。
   分类图标映射(Food→`Utensils` 等)集中放 `composables/useCategoryIcon.ts` 一份。
2. **数据层抽离**:新建 `composables/useAnalytics.ts`——模块级单例
   (`const analytics = ref<Analytics | null>(null)` 置于模块作用域),封装 `fetchAnalytics`、
   `refresh`(含 429 处理)、`loading`、`error`。各 Tab 直接 `useAnalytics()` 取数,
   **移除 App.vue → 各 Tab 的 `analytics` prop 钻透**。不引入 Pinia。
3. **主题抽离**:新建 `composables/useTheme.ts`,收编 App.vue 里的 themeMode/themeClass/
   applyTheme/localStorage/系统偏好监听逻辑(保留对 `bumpThemeVersion()` 的调用)。
4. **Toast 抽离**:`composables/useToast.ts` 模块级单例,`AppToast` 自行订阅。
5. App.vue 瘦身为纯布局壳(侧边栏 + 头部 + Tab 分发 + 全局组件),其 `<style>` 里的
   `.stats-card` 系列样式移入 `AppSidebar.vue`。
6. 顺手删除模板里未实现功能的 FAB 按钮(`.fab`,当前无任何点击行为)。

**验收**:功能与重构前一致;`grep -rn 'v-html' web/src` 仅剩 0 处图标用法。 → ✅ v-html 归零;三主题 × 六 Tab + 刷新 toast 目检通过。
**提交**:`refactor: 统一图标为 lucide,抽离数据/主题/Toast 组合式函数`

### Phase 3 落地记录(Phase 4 必读)

- **图标包用 `@lucide/vue`(不是 `lucide-vue-next`)**:后者已废弃(`npm view` 显示
  `Please use @lucide/vue instead`,latest 是占位的 1.0.0)。当前装 `@lucide/vue@1.24.0`。
  `color` prop → svg 的 `stroke`;`:size` → width/height。与旧 SFC 图标 `:color`/`:size` 同接口,drop-in。
- **两套旧图标系统已删**:`composables/icons.js`(v-html 字符串)与 `components/icons/`(SFC + IconBase)
  整个目录删除。`v-html` 图标用法全归零。
- **图标名冲突要 alias**:多个 tab 从 `echarts/charts` 具名导入 `PieChart`/`LineChart`,
  与 lucide 同名。lucide 侧统一 alias:`PieChart as PieChartIcon`、`LineChart as LineChartIcon`
  (见 OverviewTab/SpendingTab/TrendsTab)。新增图标时注意查重。
- **两个中央映射模块**:
  - `composables/useCategoryIcon.ts`:分类 → lucide 组件(键对齐 `useCategories.ts` 的 `categoryLabels`);
    `TransactionList`/`TransactionsTab` 用 `<component :is="getCategoryIcon(cat)">` 渲染。
  - `composables/navItems.ts`:主导航项(携带 lucide 组件),`AppSidebar`/`MobileNav` 共用。
- **tree 图标靠 CSS 上色**:`AccountsTab` 的 `.icon-* :deep(svg){stroke:...}` 决定颜色/尺寸,
  故这些 `<component :is>` **不传** color/size 让 CSS 接管。
- **三个模块级单例 composable**:`useAnalytics`(fetch/refresh/429/loading/error)、
  `useTheme`(themeMode/themeClass/系统偏好监听,内部 `initTheme` 幂等)、
  `useToast`(`showToast` 全局可调,`AppToast` 自订阅、不再收 props)。
- ⚠️ **偏差:Tab 的 `:analytics` prop 尚未移除**。计划 step 2 要求各 Tab 直接 `useAnalytics()`,
  但 Phase 4 本就逐个重写每个 Tab 的 `<script setup>` 成 TS,现在切一次、Phase 4 再切一次是双重改动。
  **决策**:数据层逻辑已全部抽进 `useAnalytics` 单例、App.vue 消费它(达成瘦身/数据层抽离目标);
  各 Tab 改用 `useAnalytics()` 取数、删掉 `:analytics`/`:transactions`/`:expenseByCategory` prop
  **并入 Phase 4 的逐组件重写**。`analytics` 单例 ref 与 App.vue 传下去的是同一对象,行为一致。
- **FAB 已删**:模板按钮 + `components.css`/`mobile.css` 的 `.fab` 死 CSS 一并清除。
- `.stats-card` 系列样式已从 App.vue 迁入 `AppSidebar.vue` 的 `<style scoped>`;
  App.vue 现只剩 `@keyframes spin`(加载态用)。顺手修掉了 `.stats-card:hover` 引用未定义
  `--shadow-sm` 的问题(改为只变 border-color)。

---

## Phase 4:组件逐个重写(TS 化 + 去内联样式 + 新设计语言)

**规则**(每个组件适用):

- `<script setup lang="ts">`,props 用 `types/api.ts` 的类型;
- 模板内**禁止** `style="..."`(动态绑定 `:style` 仅允许用于真正的运行时值,如进度条宽度);
- 样式写入组件 `<style scoped>` 或提炼进 `styles/components.css` 的共享类,全部引用 Phase 2 token;
- 金额一律 `tabular-nums`;卡片/列表/chips 遵循 Phase 2 规范;
- ECharts option 中的颜色一律 `getThemeColor()` + computed 引用 `themeVersion.value`,
  色板取 `--chart-*` token。

**执行顺序**(由外向内,每 1–2 个组件可单独提交):

1. 布局层:`AppSidebar`、`MobileNav`、`ThemeSwitcher`、`AppToast`、`IssuesBanner`
2. `TransactionList`(changelog-row 化)+ `TransactionsTab`(筛选器药丸化)
3. `OverviewTab`(四张统计卡按"eyebrow + 紧排大数字 + 发丝线分隔明细"重排)
4. `SpendingTab`、`MerchantRanking`、`PlatformRanking`
5. `TrendsTab`、`CategoryTrendChart`、`WeekdayChart`、`TransactionCalendar`
6. `AccountsTab`、`BudgetCard`(BudgetCard 的 fetch 迁入 `useAnalytics` 同级的
   `useBudgets.ts`,复用 429/错误处理模式)

**验收**:`grep -c 'style="' web/src/components -r` 趋近 0;三主题 × 桌面/移动逐 Tab 目检;
`pnpm run lint && pnpm run typecheck` 绿。
**提交**:按组件分批,`refactor: XXX 组件 TS 化并接入新设计系统`

---

## Phase 5:后端小清理(可选,低风险) ✅ 已完成(commit `e661ed1`)

1. 删除前端从未调用的 `/api/summary`、`/api/transactions`、`/api/accounts` 三个端点及其 handler
   (保留 `/api/analytics`、`/api/refresh`、`/api/budgets` 读写)。
2. 移除 `main.go` 的 `corsMiddleware`(开发走 Vite 代理、生产同源,均不产生跨域请求)。
3. `go test -race ./...` 全绿。

**提交**:`chore: 移除未使用的 API 端点与 CORS 中间件` → ✅ 已提交 `e661ed1`

---

## Phase 6:文档同步

1. 更新 `CLAUDE.md`:前端技术栈(TS/lucide/ESLint)、新增常用命令(`pnpm lint/typecheck`)、
   "关键文件"清单(新增 `types/api.ts`、`useAnalytics.ts`,移除 `icons.js`);
   在"正确性约定"追加:**前端组件禁止内联 style,颜色/间距/圆角必须走 design token**。
2. 更新 `README.md` 对应部分。

**提交**:`docs: 同步重构后的技术栈与约定`

---

## 全局红线(执行期间不得违反)

1. 不改 `server/parser/` 的任何统计口径与解析逻辑(Phase 5 只删 API 层代码)。
2. 前端不得从 postings 推断交易类型/金额——`processTransaction` 只派生展示字段的约定不变。
3. `getThemeColor()` + `themeVersion` 的 ECharts 主题机制不变。
4. 三主题必须全部存活,每个 Phase 结束目检一次。
5. 每个 Phase 结束:`make build` 全绿(含 go test、lint、typecheck)后才提交;
   开发验证可用 `NEVE_DATA_DIR=../data.example` 跑演示数据。
