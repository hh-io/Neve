# Neve - 个人记账系统

基于 Beancount 复式记账格式的个人/家庭财务管理系统。采用 **Go 后端 + Vue 3 前端** 架构，提供精美的 Dashboard 可视化界面。

## 核心特性

- 📊 Apple 风格 + 玻璃拟态设计的 Dashboard
- 📱 响应式布局，支持手机/平板/桌面
- 💰 净资产、收支统计、分类分析
- 📈 月度趋势图表 (ECharts)
- 🔄 手动刷新数据
- 🔒 支持 Cloudflare Access 认证

## 系统架构

```
iOS 快捷指令 → iCloud Drive (inbox.bean) → macOS Go 后端 → Vue 3 Dashboard
                                                ↓
                                        Cloudflared Tunnel → 公网
```

---

## 快速开始

### 环境要求

- **Go** >= 1.21
- **Node.js** >= 18
- **pnpm** >= 8

### 安装依赖

```bash
# 安装前端依赖
cd web && pnpm install
```

### 环境变量

| 变量            | 说明                   | 默认值   |
| --------------- | ---------------------- | -------- |
| `NEVE_DATA_DIR` | Beancount 数据目录路径 | `./data` |
| `NEVE_PORT`     | HTTP 服务端口          | `8080`   |

### 构建与运行

```bash
# 方式一：使用 Makefile
make build    # 构建前端 + 后端
./neve        # 运行

# 方式二：分步执行
cd web && pnpm run build    # 构建前端
cd server && go build -o ../neve .  # 构建后端
./neve

# 方式三：指定数据目录运行
NEVE_DATA_DIR=/path/to/data ./neve
```

### 开发模式

```bash
# 前端热重载 (需要后端同时运行)
cd web && pnpm run dev

# 后端开发
cd server && go run .
```

---

## 目录结构

```
Neve/
├── neve                    # 编译后的可执行文件 (12MB ARM64)
├── Makefile                # 构建脚本
├── README.md               # 项目说明
├── AGENTS.md               # AI Agent 专用文档
│
├── data/                   # Beancount 数据目录 (iCloud 同步)
│   ├── main.bean          # 入口文件 (账户定义 + include)
│   ├── inbox.bean         # 待整理流水 (iOS 快捷指令写入)
│   ├── balance.bean       # 初始余额 + 余额断言
│   └── 2025.bean          # 年度归档交易
│
├── server/                 # Go 后端
│   ├── main.go            # 入口 + HTTP 服务 + 静态文件
│   ├── go.mod
│   ├── parser/            # Beancount 解析器
│   │   ├── parser.go      # 文件解析 (open/transaction/balance)
│   │   └── analytics.go   # 数据分析 (净资产/分类/趋势)
│   ├── api/
│   │   └── handler.go     # REST API 处理器
│   └── static/            # 构建后的前端文件 (自动生成)
│
└── web/                    # Vue 3 前端
    ├── package.json
    ├── vite.config.js     # Vite 配置 (输出到 server/static)
    ├── index.html
    └── src/
        ├── main.js        # 入口
        ├── App.vue        # 主 Dashboard 组件
        └── styles/
            └── main.css   # 设计系统 (Apple Minimal + Glassmorphism)
```

---

## API 接口

### 基础信息

- **Base URL**: `http://localhost:8080`
- **Content-Type**: `application/json`

### 接口列表

| 方法 | 路径                | 说明                           |
| ---- | ------------------- | ------------------------------ |
| GET  | `/api/summary`      | 获取财务摘要 (净资产/本月收支) |
| GET  | `/api/analytics`    | 获取完整分析数据 (图表/列表)   |
| GET  | `/api/transactions` | 获取交易列表 (最近 100 条)     |
| GET  | `/api/accounts`     | 获取账户列表                   |
| POST | `/api/refresh`      | 刷新数据 (重新解析 .bean 文件) |

### 响应示例

```json
// GET /api/summary
{
  "netWorth": -26205.56,
  "totalAssets": 22431,
  "totalLiabilities": 48636.56,
  "monthIncome": 199,
  "monthExpense": 210,
  "monthBalance": -11,
  "lastUpdated": "2025-12-12T03:15:38+08:00"
}
```

---

## 页面路由

| 路径 | 说明                          |
| ---- | ----------------------------- |
| `/`  | Dashboard 主页 (SPA 单页应用) |

> 注：当前为单页应用，所有路由由 Vue Router 在前端处理

---

## 技术栈

### 后端

| 技术  | 版本  | 用途         |
| ----- | ----- | ------------ |
| Go    | 1.21+ | 后端语言     |
| Gin   | 1.9.x | HTTP 框架    |
| embed | -     | 嵌入静态文件 |

### 前端

| 技术        | 版本  | 用途     |
| ----------- | ----- | -------- |
| Vue         | 3.4.x | 前端框架 |
| Vite        | 5.x   | 构建工具 |
| ECharts     | 5.5.x | 图表库   |
| vue-echarts | 6.6.x | Vue 封装 |

### 数据格式

| 格式      | 说明                       |
| --------- | -------------------------- |
| Beancount | 纯文本复式记账格式 (.bean) |

---

## Beancount 数据说明

### 文件职责

| 文件           | 职责                                  |
| -------------- | ------------------------------------- |
| `main.bean`    | 账户定义 (`open` 指令) + 引入其他文件 |
| `inbox.bean`   | iOS 快捷指令写入的待整理交易          |
| `balance.bean` | 初始余额 + 定期余额断言               |
| `2025.bean`    | 年度归档的已核对交易                  |

### 账户命名规范

```
Assets:Bank:CMBC           # 资产 - 银行 - 招商
Assets:Cash:WeChat         # 资产 - 现金 - 微信
Liabilities:CreditCard:CMBC # 负债 - 信用卡 - 招商
Income:Salary              # 收入 - 工资
Expenses:Food:Coffee       # 支出 - 餐饮 - 咖啡
Equity:Opening-Balances    # 权益 - 初始余额
```

---

## 部署

### Cloudflared Tunnel 配置

```yaml
# ~/.cloudflared/config.yml
ingress:
  - hostname: neve.your-domain.com
    service: http://localhost:8080
  - service: http_status:404
```

### macOS 后台服务 (launchd)

#### 1. 安装服务

```bash
# 复制 plist 到 LaunchAgents
cp com.neve.server.plist ~/Library/LaunchAgents/

# 加载服务 (立即启动 + 开机自启)
launchctl load ~/Library/LaunchAgents/com.neve.server.plist
```

#### 2. 修改端口

编辑 `com.neve.server.plist` 中的环境变量：

```xml
<key>NEVE_PORT</key>
<string>9090</string>  <!-- 修改为你想要的端口 -->
```

然后重载服务 (见下方)。

#### 3. 代码修改后重载

```bash
# 停止服务
launchctl unload ~/Library/LaunchAgents/com.neve.server.plist

# 重新构建
make build

# 启动服务
launchctl load ~/Library/LaunchAgents/com.neve.server.plist
```

#### 4. 常用命令

```bash
# 查看服务状态
launchctl list | grep neve

# 查看日志
tail -f ~/Library/Logs/neve.log
tail -f ~/Library/Logs/neve.error.log

# 停止服务
launchctl unload ~/Library/LaunchAgents/com.neve.server.plist

# 启动服务
launchctl load ~/Library/LaunchAgents/com.neve.server.plist

# 卸载服务 (删除开机自启)
launchctl unload ~/Library/LaunchAgents/com.neve.server.plist
rm ~/Library/LaunchAgents/com.neve.server.plist
```

#### 5. plist 配置说明

| 配置项              | 说明                    |
| ------------------- | ----------------------- |
| `RunAtLoad`         | 加载时立即运行          |
| `KeepAlive`         | 崩溃后自动重启          |
| `WorkingDirectory`  | 工作目录 (用于相对路径) |
| `StandardOutPath`   | 标准输出日志路径        |
| `StandardErrorPath` | 错误日志路径            |

---

## 常见问题

### Q: 页面白屏？

检查浏览器控制台是否有 JS 错误，确保已运行 `pnpm run build` 构建前端。

### Q: 数据不显示？

1. 确认 `NEVE_DATA_DIR` 指向正确的 data 目录
2. 检查 `main.bean` 语法是否正确
3. 调用 `POST /api/refresh` 刷新数据

### Q: 如何添加交易？

在 `inbox.bean` 中按 Beancount 格式添加交易，然后点击"刷新数据"按钮。
