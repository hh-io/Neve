# Neve HTTP 层(`server/api/`)

路由、缓存发布、无感记账端点、公网自检的约定。统计口径见 `server/parser/CLAUDE.md`。

- **配置文件解析失败不得降级为空**(debts.json;新增配置文件照此办理):软失败策略只适用于账本
  (脏数据跳过一笔,其余照常),配置文件反过来——空配置在界面上与"从未配置过"无法区分,
  用户随手补一条再保存就把原文件**整份覆盖**,而配置没有第二份真源。故
  `loadDebtsConfig` 只把「文件不存在」当正常(回空),
  解析失败一律 5xx 显式报错;写盘路径再加一道 `quarantineCorrupt`——覆盖前若磁盘上那份
  已无法解析,先改名成 `<name>.corrupt-<时间戳>` 留档。`Refresh()` 是唯一例外
  (走 `longTermAccounts()`):账本刷新与配置无关,读不出来就退回全量净资产口径并记日志,
  不让一个坏配置卡死账本。
- **AI 输出必须过 parser 预校验才可落盘**:`inbox.go` 的 `validateCandidate`
  先经 `checkTransactionOnly` 拒绝任何非交易顶层行(open/include/option/散文——parser
  会静默忽略或如实执行它们,AI 补一行 open 即可绕过账户白名单),再在临时目录拼
  "真实 open 指令 + 候选交易"试解析,任何 issue 都拒绝写入并回喂 AI
  修正一次;识别提示词的账户列表由 `server/ai.ExtractAccounts` 从账本**原文**提取
  (保留行尾中文注释,parser 结构化数据会丢注释),不要再手工维护账户清单。
  **它会逐层展开 include**(去重 + 循环检测,读不到的 include 跳过):这份输出既是提示词里的
  账户白名单、又是 `validateCandidate` 临时账本里唯一的 open 来源,只扫 main.bean 的话,
  账户 open 在子文件里的用户会遇到「AI 看不见该账户 → 照着账本写反而 UNKNOWN_ACCOUNT
  → 两次尝试全废」。
- **识别提示词的唯一真源是 `server/ai/prompt.md`**(`{{DATE}}`/`{{ACCOUNTS}}` 运行时注入):
  快捷指令本身不再携带提示词,只上传图片到 `/api/inbox`,别在快捷指令侧复制一份。
- **已发布的 `s.analytics` 必须当只读**:`ApplyLongTermLiabilities` 是就地修改,
  `handleAnalytics` 锁内只取指针、脱锁才序列化,就地改会与正在编码 JSON 的请求竞争
  (`TestAnalyticsReadWhileDebtsSaved` 用 -race 锁定)。故 `handleSaveDebts` 走
  `WithLongTermLiabilities` 拿副本再换指针,前端 `useDebts.saveDebts` 再静默 `reload()` analytics。
  口径本身见 `server/parser/CLAUDE.md` 的净资产分层。
- **公网入口故障必须靠主动探测发现,被动等日志发现不了**(`health.go`):
  记账链路是「快捷指令 → Cloudflare edge → tunnel → 本进程」,前两段挂掉时服务端完全无感
  ——本机进程/端口/账本全正常,现象只是"没有请求进来",与"今天没记账"无法区分;而记账是
  单向 fire-and-forget,用户对成功的唯一感知是 Bark 推送,收不到推送时同样分不清这两者。
  实测 tunnel 曾静默断开 10.5 小时,期间每笔记账都石沉大海(请求没到服务端,连
  `data/failed/` 留档都没有),靠人工翻日志才发现。故 `StartHealthChecker` 定期从公网打
  自己的 `/api/inbox`,**拿 401 当唯一健康信号**:ingress 白名单只放行 `^/api/inbox$`,
  加健康端点就要放宽白名单、扩大暴露面;而无令牌请求由 `handleInbox` 在读 body 前返回 401,
  零副作用不占限流额度,且 401 只可能由本进程产生(edge 侧故障给的是 5xx/1033),
  拿到它就证明链路确实通到了应用层。`TestInboxUnauthorizedIsHealthSignal` 锁定这个契约
  ——鉴权改成 403 之类会让自检把正常入口全判成故障。
  连续 `healthFailThreshold` 次失败才告警(tunnel 重连、边缘节点切换有分钟级窗口,
  单次失败不值得推送),持续故障按 `healthAlertInterval` 节流,恢复时推一次并清零计数。
  三个状态字段只由单个 checker goroutine 读写,故不像备份告警那样需要锁。
