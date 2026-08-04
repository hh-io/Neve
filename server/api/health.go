package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	// healthCheckInterval 自检间隔。每次自检会在 access log 里留下一条 401,
	// 10 分钟一条的噪声可以接受,换来入口挂掉最多十几分钟就能发现。
	healthCheckInterval = 10 * time.Minute
	// healthStartupDelay 启动后首检的延迟。服务端与 cloudflared 是两个各自独立的
	// launchd 任务,拉起顺序无保证,进程刚起来就检大概率误报。
	healthStartupDelay = 2 * time.Minute
	// healthFailThreshold 连续失败多少次才告警。tunnel 重连、边缘节点切换都有
	// 秒级到分钟级的窗口,单次失败不值得推送;3 次(约 20 分钟持续不可达)才是真故障。
	healthFailThreshold = 3
	// healthAlertInterval 持续故障时的重复告警间隔。入口故障基本不自愈(实测那次是
	// 代理拒绝 QUIC,断了 10.5 小时),只推一条容易被划掉就忘,每 6 小时提醒一次。
	healthAlertInterval = 6 * time.Hour
	// healthCheckTimeout 单次自检的超时。请求要绕公网走一圈,给足余量但必须有上界。
	healthCheckTimeout = 20 * time.Second
)

// EnableHealthCheck 配置公网入口自检目标(完整 URL);不调用则 StartHealthChecker 空操作。
func (s *Server) EnableHealthCheck(url string) {
	s.healthURL = url
}

// StartHealthChecker 定期自检公网入口能否端到端抵达本进程,连续不可达时 Bark 告警。
//
// 为什么需要主动探测:记账链路是「快捷指令 → Cloudflare edge → tunnel → 本进程」,
// 前两段挂掉时服务端这边完全无感——表现只是"没有请求进来",与"今天没记账"无法区分,
// 而本机的进程、端口、账本一切正常。实测发生过 tunnel 静默断开 10.5 小时,期间每笔
// 记账都石沉大海,直到手工翻日志才发现。被动等日志是发现不了这类故障的。
func (s *Server) StartHealthChecker() {
	if s.healthURL == "" {
		return
	}
	go func() {
		time.Sleep(healthStartupDelay)
		t := time.NewTicker(healthCheckInterval)
		defer t.Stop()
		for {
			s.runHealthCheck()
			<-t.C
		}
	}()
}

// runHealthCheck 执行一次自检并按状态机决定是否告警。拆出来是为了能被测试直接驱动。
func (s *Server) runHealthCheck() {
	if err := s.probePublicEndpoint(); err != nil {
		s.healthFails++
		log.Printf("health: 公网入口自检失败(连续 %d 次): %v", s.healthFails, err)
		if s.healthFails < healthFailThreshold {
			return
		}
		// 已告警过则按间隔节流,首次达到阈值立即推
		if s.healthDown && time.Since(s.lastHealthAlert) < healthAlertInterval {
			return
		}
		s.healthDown = true
		s.lastHealthAlert = time.Now()
		s.notify("Neve 入口不可达", fmt.Sprintf(
			"公网记账入口连续 %d 次自检失败,此刻用快捷指令记账会静默丢失。\n%v",
			s.healthFails, err))
		return
	}

	// 恢复通知只在状态翻转时推一次,否则每次成功都推等于每 10 分钟一条
	if s.healthDown {
		log.Printf("health: 公网入口已恢复")
		s.notify("Neve 入口已恢复", "公网记账入口重新可达,可以补记这期间的账单了")
		s.healthDown = false
	}
	s.healthFails = 0
}

// probePublicEndpoint 从公网侧请求自己的 /api/inbox,期望 401。
//
// 为什么拿 401 当健康信号,而不新加一个 /api/health 端点:tunnel 的 ingress 白名单
// 只放行 ^/api/inbox$(analytics/debts 等无鉴权端点绝不暴露公网),加端点就得放宽
// 白名单、扩大暴露面。而不带令牌的请求由 handleInbox 亲自返回 401,它在读请求体之前
// 就返回,零副作用不占用限流额度;更关键的是 401 只可能由本进程产生——edge 侧故障
// 给的是 5xx/1033,拿到 401 就证明整条链路确实打通到了应用层。
func (s *Server) probePublicEndpoint() error {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.healthURL, nil)
	if err != nil {
		return err
	}
	// 便于在 access log 里把自检的 401 与真实的令牌错误区分开
	req.Header.Set("User-Agent", "neve-healthcheck")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		// 404:inbox 未启用或 ingress 路径配错;5xx/530:tunnel 与 edge 断开
		return fmt.Errorf("期望 HTTP 401,实际 %d", resp.StatusCode)
	}
	return nil
}
