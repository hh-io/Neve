<div align="center">

<img src="web/public/neve.svg" width="88" alt="Neve logo">

# Neve

**基于 Beancount 纯文本复式记账的个人财务可视化系统**

数据即文件 · 单文件部署 · AI 无感记账

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Built with Claude Code](https://img.shields.io/badge/Built%20with-Claude%20Code-D97757?logo=claude&logoColor=white)](https://claude.com/claude-code)

</div>

---

## 简介

Neve 是一个轻量级的个人/家庭财务可视化系统，围绕 [Beancount](https://beancount.github.io/) 纯文本账本格式构建：Go 后端解析 `.bean` 账本并输出统计，Vue 3 前端展示，前端构建产物 embed 进 Go，最终产出一个单文件二进制 `./neve`。

它解决三件事：

- **移动端无感记账** — iPhone 快捷指令拍照上传账单，AI 视觉识别后自动写入 iCloud 中的账本文件；
- **优雅的数据可视化** — Apple 风格 Dashboard，收支趋势、分类占比、商户排行一目了然；
- **数据所有权** — 账本是纯文本，永远不被任何平台锁定。

> 本项目由作者与 [Claude Code](https://claude.com/claude-code) 结对开发完成——从架构设计、解析器实现、前端组件到部署脚本，均为人机协作的产物。详见[「与 Claude Code 共同开发」](#与-claude-code-共同开发)。

## 特性

- **记账正确性校验** — 借贷平衡、账户定义、`balance` 断言真实对账；脏数据显式报错并跳过，不污染统计
- **定点金额运算** — 金额以「分」为单位的 int64 运算，无浮点误差
- **AI 无感记账** — 上传账单图片立即返回，识别 / 预校验 / 入账 / 推送全部异步完成；AI 输出必须通过解析器预校验才允许落盘
- **交易口径后端唯一计算** — 支出 / 收入 / 转账 / 手续费 / 退款冲减由后端统一分类，前端只做展示
- **负债待还看板** — 信用卡 / 白条等按账单日余额快照自动汇总本期应还，卡内免息分期（如 24 期免息）只需配置一次即自动按月出账、扣减未出账尾款，房贷等固定分期按生效日期配置月供，全局倒计时与逾期提醒
- **精心调校的亮暗双主题** — 手写 CSS 变量设计 token 体系，无 UI 库依赖
- **单文件部署** — `make build` 产出一个二进制，无数据库、后端唯一依赖 Gin

## 截图

<!-- TODO: 补充截图（建议：Dashboard 亮/暗主题、支出分析、趋势图表） -->

## 快速开始

### 环境要求

| 工具 | 版本 |
|------|------|
| [Go](https://go.dev/dl/) | >= 1.26 |
| [Node.js](https://nodejs.org/) | >= 20.19（Vite 8 要求） |
| [pnpm](https://pnpm.io/) | >= 8 |

### 安装与运行

```bash
git clone https://github.com/hh-io/Neve.git
cd Neve

make deps      # 安装前后端依赖
make build     # 前端(lint+typecheck) → 测试 → 后端,产出 ./neve
./neve         # 访问 http://localhost:8080
```

首次运行没有账本时，可将 `NEVE_DATA_DIR` 指向仓库自带的演示数据 `data.example/`。

### 开发模式

```bash
make dev-server   # 终端 1:后端 (localhost:8080)
make dev          # 终端 2:前端热重载 (localhost:5173,/api 代理到 8080)
make test         # 后端单元测试 (go test -race)
```

## 配置

全部通过环境变量配置：

```bash
NEVE_DATA_DIR=/path/to/beancount/data   # Beancount 数据目录 (默认 ./data)
NEVE_PORT=8080                          # HTTP 服务端口 (默认 8080)

# 以下为无感记账入口,四者齐备才启用 /api/inbox,详见「无感记账」
NEVE_INBOX_TOKEN=<随机串>               # /api/inbox 的 Bearer 令牌
NEVE_AI_PROVIDER=claude                 # claude | gemini
NEVE_AI_API_KEY=<key>                   # 对应提供商的 API Key
NEVE_AI_MODEL=                          # claude 留空默认 claude-opus-4-8;gemini 必填
NEVE_BARK_URL=https://api.day.app/<key> # Bark 推送地址 (可选)
```

部署时密钥统一放 gitignore 的 `deploy/local.env`（模板见 `deploy/local.env.example`）。

## 架构

```
iOS 快捷指令上传账单图片 ── POST /api/inbox (Bearer 鉴权,立即 202)
  └─ 异步:AI 视觉识别 → parser 预校验(失败回喂修正一次)
      → 追加 iCloud 的 data/inbox.bean → 刷新缓存 → Bark 推送结果

server/parser  解析 main.bean(include 展开)+ 校验 → 统计并缓存
GET /api/analytics  一次性输出全部数据 → Vue 前端各 Tab 直接消费
```

```
Neve/
├── Makefile             # 构建自动化
├── data/                # Beancount 账本 (iCloud 软链接,不入库)
├── data.example/        # 演示数据 (入库,结构同 data/)
├── deploy/              # launchd / cloudflared 部署模板
├── server/              # Go 后端
│   ├── api/             #   路由、analytics 缓存、无感记账端点
│   ├── ai/              #   AI 视觉客户端 (claude/gemini 原生 HTTP) + 提示词
│   └── parser/          #   Beancount 解析器、定点金额、统计分析
└── web/                 # Vue 3 + TypeScript 前端
    └── src/
        ├── components/  #   Tab 页面、图表、布局组件
        ├── composables/ #   模块级单例 (analytics/theme/toast/budgets/debts)
        └── styles/      #   CSS 变量设计 token (亮/暗双主题)
```

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/analytics` | 完整分析数据（摘要/图表/全量交易/解析问题/对账结果），前端一次拉取 |
| `POST` | `/api/refresh` | 重新解析账本并重建缓存（5 秒限流） |
| `GET` | `/api/budgets` | 获取预算 |
| `POST` | `/api/budgets` | 保存预算（原子写 `budgets.json`） |
| `GET` | `/api/debts` | 负债待还配置 + 实时计算的还款报告（账期快照/剩余待还/倒计时） |
| `POST` | `/api/debts` | 保存待还配置（校验后原子写 `debts.json`，响应附重算报告） |
| `POST` | `/api/inbox` | 无感记账：上传账单图片立即 202，识别与入账异步完成（Bearer 鉴权，未配置时 404） |

> 仅支持 CNY 单币种；非 CNY、借贷不平衡、未 open 账户的交易会被跳过并记入 `parseIssues`。

## 无感记账

iPhone 快捷指令只做一次图片上传（约 1–2 秒），其余全部在服务端后台完成：

1. 从 `main.bean` 实时提取账户列表拼提示词（无需手工维护账户清单）
2. 调用 AI 视觉识别（Claude / Gemini，原生 HTTP，不引入 SDK 依赖）
3. 解析器预校验，失败回喂 AI 修正一次
4. 追加 `data/inbox.bean` 并刷新缓存
5. Bark 推送结果；识别失败不污染账本，原始输出与图片留档 `data/failed/<时间戳>/`

请求格式（快捷指令「获取 URL 内容」发 POST，附 `Authorization: Bearer $NEVE_INBOX_TOKEN`）：

```json
{ "image": "<base64>", "mime": "image/jpeg", "text": "可选补充说明" }
```

快捷指令搭建说明见 `shortcut/`。

## 部署

### macOS 后台服务 (launchd)

服务与日志轮转配置以模板形式放在 `deploy/`，由 make 按本机路径渲染安装：

```bash
make install-service    # 渲染 plist 到 ~/Library/LaunchAgents (只写文件,不启动)
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.neve.server.plist
make install-logrotate  # 日志轮转 (需 sudo)
tail -f ~/Library/Logs/neve.log
```

> 日期按服务器本地时区归属月份/星期，plist 模板已通过 `TZ` 显式钉死记账时区（默认 `Asia/Singapore`，按需修改），避免系统时区切换导致归属漂移。

### Cloudflare Tunnel（无感记账需要）

让 iPhone 在任意网络下访问 `/api/inbox`。ingress **只放行 `/api/inbox` 一条路径**，无鉴权端点绝不暴露公网：

```bash
brew install cloudflared && cloudflared tunnel login
cloudflared tunnel create neve            # tunnel UUID 填入 deploy/local.env
make install-tunnel                       # 渲染 config.yml + 常驻 LaunchAgent
cloudflared tunnel route dns neve inbox.your-domain.com
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.cloudflared.tunnel.plist
```

> 隧道用本地托管模式（config.yml + credentials），不要用仪表盘 token 连接器——路径级 ingress 限制需要留在版本化的本地配置里。没有 Cloudflare 托管域名时可改用 Tailscale 直连。

## 与 Claude Code 共同开发

这个项目是与 [Claude Code](https://claude.com/claude-code) 深度协作开发的实践：

- **全程结对** — 从 Beancount 解析器、定点金额类型、交易分类算法，到前端设计 token 体系与图表组件，绝大部分代码由 Claude Code 编写，作者负责需求定义、方案取舍与代码审查；
- **约定沉淀** — 项目的正确性约定（定点运算、软失败、AI 预校验等）沉淀在 [`CLAUDE.md`](CLAUDE.md) 中，作为每次协作的上下文与红线；
- **联合署名** — 提交历史中保留 `Co-Authored-By: Claude` 署名，如实记录人机协作过程。

如果你也想体验这种开发方式，可以从 [Claude Code 文档](https://code.claude.com/docs)开始。

## 贡献

欢迎提交 Issue 和 Pull Request。提交前请：

1. 阅读 [`CLAUDE.md`](CLAUDE.md) 中的正确性约定（改代码前必读）；
2. 运行 `make build` 确保 lint、类型检查与测试全部通过；
3. 提交信息使用 conventional commit 格式（`type: 描述`）。

## 致谢

- [Beancount](https://beancount.github.io/) — 纯文本复式记账的基石
- [Claude Code](https://claude.com/claude-code) — 本项目的共同开发者

## 开源协议

[MIT](LICENSE) © hh
