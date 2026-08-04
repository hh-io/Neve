# Neve 前端(`web/`)

样式、图表、安全区的正确性约定。整体架构与后端口径见根目录 `CLAUDE.md`。

- **ECharts 颜色**:canvas 不解析 CSS 变量,option 里必须用
  `getThemeColor('--xxx')` 取实际值,并在 computed 中引用 `themeVersion.value`
  以响应主题切换(见 `useThemeColor.ts`)。图表色板走 `--chart-1..8` /
  `--chart-income` / `--chart-expense` token(热力图为顺序标度,例外保留绿渐变)。
- **前端组件禁止内联 `style="..."`**:颜色/间距/圆角/字号一律走 design token
  (`variables.css` 的 surface/hairline/accent/chart 系列 + `--space-*`/`--radius-*`);
  仅**真正的运行时值**(进度条宽度、数据驱动的 tag 色、交错动画延时)可用 `:style` 绑定。
  组件样式写 `<style scoped>` 或提炼进 `styles/components.css` 共享类
  (如 `.panel`/`.filter-pill`/`.empty-state`)。**scoped 不穿透子组件**:父组件里定义、
  子组件模板复用的类必须放共享层,否则子组件那份渲染成裸元素。
- **交易明细页是两层 sticky**:筛选行钉 `top: var(--safe-top)`,日期分组头钉
  `top: var(--tx-day-top)`(= `--safe-top` + `--tx-filters-h`,两者都定义在 `.tx-layout`,
  后者是实测的筛选行高度 58px,改控件尺寸/内边距要同步改;安全区偏移见下条),
  右侧日历也停在同一条线上。筛选行**不能用负 margin 往上提**换取紧凑:它的顶边要与
  右列日历卡顶边同线,`padding-block: var(--space-3)` 又正好让搜索框/药丸的中心落在
  日历卡标题中心上(两者都是「顶边 + 12 + 控件半高」),动其一即破坏这两处对齐。列表**不做固定高度 + 内滚**——那会造出第二个滚动容器,
  边界处滚轮行为割裂,且一屏只剩八九条。共享类 `.section-card` 的 `overflow: hidden`
  会让它变成滚动容器、使内部 sticky 相对它定位而失效,故该页覆盖成 `overflow: clip`
  (同样裁圆角,但不建立滚动容器)。页面滚动是 document 级(`.main-content` 无 overflow),
  sticky 的 top 才能直接相对视口。
- **安全区一律走 `--safe-top`/`--safe-bottom`/`--safe-left`/`--safe-right`,不直接写
  `env()`**(定义在 `variables.css`,默认 0,由 `@supports` 在支持 `env()` 时覆盖成实际值)。
  `index.html` 带 `viewport-fit=cover`:iOS 加到主屏后以 standalone 运行,系统**不给底部
  Home 指示条留安全区**,而不加 cover 时 `env(safe-area-inset-*)` 恒返回 0——拿不到值就没法
  补偿,故只能铺满物理屏幕后自己让位。代价是顶部的自动退让也一并撤销,于是
  `.main-content` 的 `padding-top`、交易页两层 sticky 的 `top` 都要显式加 `--safe-top`,
  否则内容/筛选行会钉进状态栏。Safari 里这两个值为 0(浏览器 chrome 已占位),
  `calc()` 自然退化成原值,**双端同一份 CSS**。
  **底栏让位靠加高整条 `.mobile-nav`**(`height: var(--mobile-nav-total)`)**而非只加
  `padding-bottom`**:高度固定时变大的下内边距会把 24px 图标 + 文字挤出 48px 内容盒,
  standalone 下文字直接被裁。栏高与 `.main-content` 的 `padding-bottom` **共用
  `--mobile-nav-total`**(= `--mobile-nav-height` + `--safe-bottom`),横屏收窄底栏也只改
  `--mobile-nav-height` 这一个源,两处不会漂。同理 `padding-block` 不写 `padding`
  简写——简写会清掉基础规则里的左右安全区退让。
  **左右安全区必须写在与断点无关的基础规则上**(`.sidebar` / `.main-content` 本体,
  而非 `≤768` 的移动端块):iPhone **横屏逻辑宽度约 874px > 768**,走的是**桌面布局**分支
  (侧边栏出现、底栏隐藏),挂在移动端媒体查询里的左右退让在真机横屏时一行都不生效
  ——这是真机踩出来的,别再挂错层。侧边栏是贴边固定元素,故 `width` 加上 `--safe-left`
  且 `padding-left` 同值(border-box 下内容区仍是 `--sidebar-width`),`.main-content`
  的 `margin-left` 要跟着含进去;刘海落哪一侧由旋转方向决定,两侧都要让。
  主屏图标只认 `apple-touch-icon.png`(iOS 不吃 SVG,也不读 `manifest.icons`),
  缺了会拿页面截图当图标;PNG 由 `neve.svg` 渲染,重生成命令见 README。
- **图表祖先链上的 flex 子项必须 `min-width: 0`**(`.main-content` / `.tx-main`):
  ECharts 的 canvas 带固定 inline 宽度,是**不可收缩内容**,而 flex 子项默认
  `min-width: auto` 会把自己的 min-content 顶成 canvas 的宽度。后果是**变宽能跟随、
  变窄回不去**——手机横屏把图表撑到 874px,转回竖屏时容器被 canvas 顶住无法收缩,
  `ResizeObserver`(`<v-chart autoresize>`)看到宽度没变就不再 resize,形成自锁,
  「横屏一次就再也回不去」,内容右侧被切出屏幕。`.main-content` 是 `.app-layout`(flex)
  的子项,钉 0 后整条链解锁(实测只需这一处,内层随 canvas 一起收敛;卡片层再加是噪声)。
  排查手法:把 `.app-layout` 的 `width` 钉成竖屏宽度,看 `.main-content` 是否跟着变窄
  ——比改窗口尺寸精确,浏览器窗口在全屏态下 resize 不生效。
  **别用 `*:has(.echarts)` 这种通配 `:has()` 兜底**:实测会触发全量样式重算,渲染进程
  直接卡死 45s 以上。
- **分类中文映射只有一份**:`web/src/composables/useCategories.ts` 的 `categoryLabels`。
