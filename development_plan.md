## 项目背景

**我正在开发一个名为 Neve 的个人记账系统。 核心架构是：iOS 快捷指令 (前端录入) + iCloud Drive (数据同步) + macOS (后端解析/展示)。 该系统采用 Beancount (纯文本复式记账) 格式存储数据。**

### 【核心文件结构与定义】由以下四个核心文件组成,请务必理解它们各自的职责：

1. main.bean (入口索引文件)

- 作用： 整个账本的入口。
- 内容： 定义所有账户 (open 指令)、设置基础 Option、不直接包含交易数据。
- 逻辑： 它负责 include 其他三个文件,作为后端读取的唯一目标文件。

2. inbox.bean (收件箱/暂存区)

- 作用： iOS 快捷指令的写入目标,作为“待清洗数据的缓冲区”。
- 特性：
  - iOS 端通过“追加文本”的方式直接写入此文件。
  - 允许存在 Expenses:Unknown 等临时分类。
  - 数据未经人工核对,属于 Pending 状态。

3. balance.bean (余额管理)

- 作用： 系统的“锚点”。
- 内容：
  - 包含 Equity:Opening-Balances 初始化指令（定义系统启动时的资产负债现状）。
  - 包含定期的 balance 断言指令（用于核对账本与银行实际金额）。

4. 2025.bean (年度归档)

- 作用： 存放经过清洗、核对无误的历史交易数据。
- 逻辑： 这里的交易是“干净”的（Clean Data）。未来我会定期手动或脚本将 inbox.bean 的内容剪切归档到这里。

### 【账户命名规范】 系统严格遵循以下命名层级：

- **资产**：Assets:Bank:Name / Assets:Cash:App
- **负债**：Liabilities:CreditCard:Name
- **收入**：Income:Category (记为负数)
- **支出**：Expenses:Category:SubCategory
- **权益**：Equity:Opening-Balances

### 【开发目标】 请你进行技术选型

- **约束**： 最好保持低内存占用,低性能消耗, 易维护, 目前计划在 macOS 后台长期后台运行通过 Cloudflared 发布到公网(你有更好的方案可以提出)
- **目标**： 提供展示系统,图标分析,美观易读（统计总支出、分类占比、当前净资产等）并具有一定的财务分析能力。

> **请确认你已理解上述文件结构和业务逻辑。不合理的请提出沟通包括命名不合理的直接提出或者修改，如果合理请开始进行技术选型和实现。**

### 【Beancount 交易语法深度解析】

> 请严格按照以下逻辑理解和生成 .bean 格式的交易数据。每一笔交易（Transaction）由“表头”和“分录（Postings）”组成，必须满足复式记账平衡原则（所有分录金额相加为 0）。

#### 1. 通用结构说明

```
YYYY-MM-DD * "交易对手(Payee)" "描述(Narration)" #标签(可选)
  账户A    金额 货币
  账户B    金额 货币
  ...
```

- **YYYY-MM-DD**: 交易发生的日期。
- **\***: 这是一个固定标记，代表“已确认/Completed”的交易。
- **交易对手 (Payee)**: 比如 "Starbucks", "京东", "Google"。
- **描述 (Narration)**: 具体的商品或行为，如 "冰美式", "还款"。
- **账户 (Account)**: 必须是 main.bean 中定义过的标准账户名。
- **金额 (Amount)**: 必须保留两位小数。

#### 2. 核心场景与正负号规则

- **场景 A**：普通支出 (Standard Expense)
  - **逻辑**： 资产减少，支出增加。
  - **规则**： 支出账户为 正数 (+)，资产账户为 负数 (-)。

```
2025-12-04 * "Starbucks" "冰美式"
  Expenses:Food:Coffee        35.00 CNY ; [支出] 消费了35元
  Assets:Cash:WeChat         -35.00 CNY ; [资产] 微信里少了35元
```

- **场景 B**：信用卡/花呗消费 (Liability Spending)
  - **逻辑**： 负债增加（欠得更多了），支出增加。
  - **规则**： 支出账户为 正数 (+)，负债账户为 负数 (-)。

```
2025-12-04 * "Google" "Colab Pro Subscription"
  Expenses:Digital:Subscription        70.00 CNY ; [支出] 享受了70元的服务
  Liabilities:CreditCard:CMBC         -70.00 CNY ; [负债] 信用卡欠款增加了70元
```

- **场景 C**：收入 (Income)
  - **逻辑**： 资产增加，收入账户作为平衡项。
  - **规则**： 资产账户为 正数 (+)，收入账户为 负数 (-) (注意！Beancount 中收入必须记为负)。

```
2025-12-04 * "张三" "付费入群-会员费"
  Assets:Cash:WeChat                   199.00 CNY ; [资产] 收到钱，余额增加
  Income:Membership                   -199.00 CNY ; [收入] 来源记为负，以保持平衡
```

- **场景 D**：退款 (Refund/Contra-Expense)
  - **逻辑**： 之前买的东西退了，资产回来了，支出被冲销。
  - **规则**： 资产账户为 正数 (+)，支出账户为 负数 (-)。

```
2025-07-15 * "Uniqlo" "退货退款"
  Assets:Cash:Alipay          199.00 CNY ; [资产] 钱回到了支付宝
  Expenses:Shopping:Clothing -199.00 CNY ; [支出] 衣服没买成，支出减少
```

- **场景 E**：借贷还款 - 含利息 (Loan Repayment with Interest)
  - **逻辑**： 银行卡扣了一大笔钱，其中一部分是还本金，一部分是给机构的利息（消费）。
  - **规则**： 资产(总扣款)为 负，负债(本金)为 正，利息(支出)为 正。

```
2025-08-01 * "京东金融" "金条还款"
  Assets:Bank:CMBC             -3383.33 CNY ; [资产] 银行卡实际扣款总额
  Liabilities:JD:CLO            3333.33 CNY ; [负债] 欠款减少了这么多 (本金)
  Expenses:Financial:Interest     50.00 CNY ; [支出] 剩下的50块是利息成本
```

- **场景 F**：信用卡还款 - 含通道费 (Repayment with Service Fee)
  - **逻辑**： 我想还 5000，但微信多收了 5 块，我实际付了 5005。
  - **规则**： 资产(总扣款)为 负，负债(账单额)为 正，手续费(支出)为 正。

```
2025-08-01 * "微信" "信用卡还款"
  Assets:Cash:WeChat               -5005.00 CNY ; [资产] 微信实际少了5005
  Liabilities:CreditCard:CMBC       5000.00 CNY ; [负债] 但银行只认5000，债只消了5000
  Expenses:Financial:ServiceFee        5.00 CNY ; [支出] 5块钱被微信收走了
```

- **场景 G**：未知交易 (Unknown)
  - **逻辑**： 无法识别分类时，使用默认账户。

```
2025-07-16 * "未知商户" "从截图提取"
  Expenses:Unknown            50.00 CNY ; [待定] 暂时记在这里
  Assets:Cash:WeChat         -50.00 CNY
```

### 【数据生成指令】 在编写代码处理数据时：

1. 必须 保证 Amount 的代数和为 0。
2. 如果涉及手续费或利息，必须 拆分为 3 行（或更多）分录。
3. 日期格式固定为 YYYY-MM-DD。
4. 货币单位固定为 CNY。

**请确认你已完全理解上述 Beancount 语法结构。**
