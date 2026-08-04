# Neve 部署模板(`deploy/`)

模板由 `make install-service` / `install-tunnel` / `install-logrotate` 渲染安装,
密钥来自 gitignore 的 `local.env`。

- **Tunnel 强制走 HTTP/2 而非默认 QUIC**(`cloudflared-config.yml.in` 的
  `protocol: http2`):本机 Surge 以 TUN 模式运行,`udp-policy-not-supported-behaviour=reject`
  会把代理不支持的 UDP 直接丢弃,cloudflared 的 QUIC(UDP/7844)拨不出去,只反复报
  `no recent network activity` 并无限重试——即那次 10.5 小时静默中断的根因。
  HTTP/2 走 TCP/443 不受代理 UDP 能力影响,换网络环境也稳,对每天几笔图片上传无感知代价。
  注意 `--protocol` 已从 `cloudflared --help` 隐藏(2026.7.3 实测仍生效),升级后连不上先查这里;
  cloudflared 自带的启动 precheck 会打印 `suggested_protocol`,可用来复核。
- **stderr 只留"进程活不下去"的信号**:`main` 里 `log.SetOutput(os.Stdout)` 把应用日志
  与 gin 访问日志合到一条时间线(落 `neve.log`,时间格式统一成 `2006/01/02 15:04:05`),
  stderr 只剩启动期致命错误(走 `fatalLog`)与 Recovery 的 panic 栈(落 `neve.error.log`)。
  **`neve.error.log` 非空即真出事**是这个文件的全部价值,别往里写正常信息——用默认 stderr
  时它塞满了启动信息和"记账成功"的交易明细,真正的失败反被淹没,而 newsyslog 给它的
  轮转阈值(1MB)还比访问日志(5MB)小,信噪比最差的那份反倒裁得最快。
- Tunnel ingress 只放行 `/api/inbox`,无鉴权端点不暴露公网(自检依赖这条,见
  `server/api/CLAUDE.md`)。`NEVE_TUNNEL_HOSTNAME` 同时注入服务端,启用公网入口自检。
- 记账时区由 `com.neve.server.plist.in` 的 `TZ` 显式钉死(当前 `Asia/Singapore`),
  不跟随系统时区。
