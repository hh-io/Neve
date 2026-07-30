package api

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// statusServer 起一个状态码可在测试中途切换的服务器,用来驱动自检的故障/恢复。
func statusServer(code *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(int(code.Load()))
	}))
}

func TestHealthProbeAcceptsOnly401(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		healthy bool
	}{
		{"401 是唯一的健康信号", http.StatusUnauthorized, true},
		{"404 说明 inbox 未启用或 ingress 路径配错", http.StatusNotFound, false},
		{"530 是 tunnel 与 edge 断开", 530, false},
		{"502 是 edge 侧故障", http.StatusBadGateway, false},
		{"200 无令牌时不可能出现,出现即异常", http.StatusOK, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var code atomic.Int32
			code.Store(int32(tt.status))
			srv := statusServer(&code)
			defer srv.Close()

			s := NewServer(t.TempDir())
			s.EnableHealthCheck(srv.URL)

			err := s.probePublicEndpoint()
			if tt.healthy && err != nil {
				t.Fatalf("HTTP %d 应判定为健康,却报错: %v", tt.status, err)
			}
			if !tt.healthy && err == nil {
				t.Fatalf("HTTP %d 应判定为故障,却判定为健康", tt.status)
			}
		})
	}
}

// TestHealthProbeUnreachable 覆盖连不上的情形:tunnel 挂掉时表现为连接错误而非状态码,
// 这才是实测那次故障的真实形态。
func TestHealthProbeUnreachable(t *testing.T) {
	var code atomic.Int32
	code.Store(http.StatusUnauthorized)
	srv := statusServer(&code)
	url := srv.URL
	srv.Close() // 关掉后端口不再监听,制造连接失败

	s := NewServer(t.TempDir())
	s.EnableHealthCheck(url)

	if err := s.probePublicEndpoint(); err == nil {
		t.Fatal("入口不可达时应报错,却判定为健康")
	}
}

// TestHealthAlertStateMachine 锁定告警状态机:阈值前静默、达阈值告警、持续故障节流、
// 恢复清零。barkURL 为空使 notify 成为空操作,测试只断言状态字段。
func TestHealthAlertStateMachine(t *testing.T) {
	var code atomic.Int32
	code.Store(http.StatusNotFound) // 故障态起步
	srv := statusServer(&code)
	defer srv.Close()

	s := NewServer(t.TempDir())
	s.EnableHealthCheck(srv.URL)

	// 阈值之前只累计不告警——tunnel 重连有秒级到分钟级窗口,单次失败不该惊动人
	for i := 1; i < healthFailThreshold; i++ {
		s.runHealthCheck()
		if s.healthDown {
			t.Fatalf("第 %d 次失败就告警了,阈值应是 %d", i, healthFailThreshold)
		}
	}

	s.runHealthCheck()
	if !s.healthDown {
		t.Fatalf("连续失败 %d 次应告警", healthFailThreshold)
	}
	firstAlert := s.lastHealthAlert
	if firstAlert.IsZero() {
		t.Fatal("告警后应记录时间戳供节流")
	}

	// 持续故障按 healthAlertInterval 节流,否则每 10 分钟一条会把通知刷爆
	s.runHealthCheck()
	if !s.lastHealthAlert.Equal(firstAlert) {
		t.Fatal("持续故障应节流,不该每次自检都推送")
	}

	// 节流窗口过后重新提醒:故障不自愈,只推一条容易被划掉就忘
	s.lastHealthAlert = time.Now().Add(-healthAlertInterval - time.Minute)
	s.runHealthCheck()
	if !s.lastHealthAlert.After(firstAlert) {
		t.Fatal("超过 healthAlertInterval 后应重新推送")
	}

	// 恢复:清故障态并清零计数,下次故障要重新走满阈值
	code.Store(http.StatusUnauthorized)
	s.runHealthCheck()
	if s.healthDown {
		t.Fatal("恢复后应清除故障态")
	}
	if s.healthFails != 0 {
		t.Fatalf("恢复后连续失败计数应清零,实际 %d", s.healthFails)
	}

	// 再次故障需重新累计到阈值,不能沿用上一轮的计数直接告警
	code.Store(http.StatusNotFound)
	s.runHealthCheck()
	if s.healthDown {
		t.Fatal("恢复后再故障应重新走满阈值,不该第一次失败就告警")
	}
}

// TestHealthCheckerDisabledWithoutURL 未配置 tunnel 域名时自检必须完全静默,
// 否则本地开发会持续报错。
func TestHealthCheckerDisabledWithoutURL(t *testing.T) {
	s := NewServer(t.TempDir())
	s.StartHealthChecker() // 不应 panic,也不应起 goroutine 去打空 URL
	if s.healthURL != "" {
		t.Fatal("未调用 EnableHealthCheck 时不应有自检目标")
	}
}

// TestInboxUnauthorizedIsHealthSignal 锁定自检赖以判活的契约:启用 inbox 后,不带令牌的
// POST 必须返回 401。若哪天鉴权改成别的状态码(如 403),自检会把正常的入口全判成故障。
func TestInboxUnauthorizedIsHealthSignal(t *testing.T) {
	_, r, _ := newInboxTestServer(t, &fakeAI{outs: []string{""}})

	w := postInbox(r, "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("自检以 401 判活,handleInbox 无令牌时却返回 %d", w.Code)
	}
}
