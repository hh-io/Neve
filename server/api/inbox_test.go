package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

const testToken = "test-token"

// fakeAI 按次序返回预设输出,超出取最后一个;线程安全以配合 -race
type fakeAI struct {
	mu    sync.Mutex
	calls int
	outs  []string
	err   error
}

func (f *fakeAI) Provider() string { return "fake" }

func (f *fakeAI) Recognize(_ context.Context, _, _, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	i := f.calls - 1
	if i >= len(f.outs) {
		i = len(f.outs) - 1
	}
	return f.outs[i], nil
}

func (f *fakeAI) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

const fixtureMainBean = `option "operating_currency" "CNY"

2020-01-01 open Assets:Cash:WeChat                     CNY ; 微信零钱
2020-01-01 open Expenses:Food:Delivery                 CNY ; 外卖
2020-01-01 open Expenses:Unknown                       CNY ; 容错账户

include "inbox.bean"
`

func newInboxTestServer(t *testing.T, client *fakeAI) (*Server, *gin.Engine, string) {
	t.Helper()
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "main.bean"), []byte(fixtureMainBean), 0o644); err != nil {
		t.Fatalf("写入 main.bean 失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "inbox.bean"), []byte(""), 0o644); err != nil {
		t.Fatalf("写入 inbox.bean 失败: %v", err)
	}

	s := NewServer(dataDir)
	if client != nil {
		s.EnableInbox(client, testToken, "")
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	s.SetupRoutes(r)
	return s, r, dataDir
}

func postInbox(r *gin.Engine, token string, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/inbox", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func validInboxBody() map[string]string {
	return map[string]string{
		"image": base64.StdEncoding.EncodeToString([]byte("fake-image-bytes")),
	}
}

// waitFor 轮询等待异步 goroutine 完成副作用
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("等待超时: %s", what)
}

func readInbox(t *testing.T, dataDir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dataDir, "inbox.bean"))
	if err != nil {
		t.Fatalf("读取 inbox.bean 失败: %v", err)
	}
	return string(data)
}

func TestInboxDisabled(t *testing.T) {
	_, r, _ := newInboxTestServer(t, nil)
	if w := postInbox(r, testToken, validInboxBody()); w.Code != http.StatusNotFound {
		t.Errorf("未启用时应 404,得到 %d", w.Code)
	}
}

func TestInboxAuth(t *testing.T) {
	_, r, _ := newInboxTestServer(t, &fakeAI{outs: []string{"ERROR"}})
	if w := postInbox(r, "", validInboxBody()); w.Code != http.StatusUnauthorized {
		t.Errorf("缺少令牌应 401,得到 %d", w.Code)
	}
	if w := postInbox(r, "wrong-token", validInboxBody()); w.Code != http.StatusUnauthorized {
		t.Errorf("错误令牌应 401,得到 %d", w.Code)
	}
}

func TestInboxBadRequest(t *testing.T) {
	_, r, _ := newInboxTestServer(t, &fakeAI{outs: []string{"ERROR"}})
	if w := postInbox(r, testToken, map[string]string{"image": "!!!not-base64!!!"}); w.Code != http.StatusBadRequest {
		t.Errorf("非法 base64 应 400,得到 %d", w.Code)
	}
	if w := postInbox(r, testToken, map[string]string{"image": "aGk=", "mime": "image/heic"}); w.Code != http.StatusBadRequest {
		t.Errorf("不支持的图片类型应 400,得到 %d", w.Code)
	}
}

func TestInboxSuccess(t *testing.T) {
	txn := `2026-07-01 * "赛百味" "金枪鱼三明治套餐" #eleme
  Expenses:Food:Delivery      35.50 CNY
  Assets:Cash:WeChat         -35.50 CNY`
	fake := &fakeAI{outs: []string{txn}}
	_, r, dataDir := newInboxTestServer(t, fake)

	w := postInbox(r, testToken, validInboxBody())
	if w.Code != http.StatusAccepted {
		t.Fatalf("应 202,得到 %d: %s", w.Code, w.Body.String())
	}

	waitFor(t, "交易写入 inbox.bean", func() bool {
		return strings.Contains(readInbox(t, dataDir), "赛百味")
	})
	if fake.callCount() != 1 {
		t.Errorf("首次输出即合法,AI 应只调用 1 次,实际 %d 次", fake.callCount())
	}
}

func TestInboxValidationRetryFixes(t *testing.T) {
	bad := `2026-07-01 * "赛百味" "套餐" #eleme
  Expenses:NotExist           35.50 CNY
  Assets:Cash:WeChat         -35.50 CNY`
	good := `2026-07-01 * "赛百味" "套餐" #eleme
  Expenses:Food:Delivery      35.50 CNY
  Assets:Cash:WeChat         -35.50 CNY`
	fake := &fakeAI{outs: []string{bad, good}}
	_, r, dataDir := newInboxTestServer(t, fake)

	postInbox(r, testToken, validInboxBody())

	waitFor(t, "修正后的交易写入 inbox.bean", func() bool {
		return strings.Contains(readInbox(t, dataDir), "Expenses:Food:Delivery")
	})
	if strings.Contains(readInbox(t, dataDir), "Expenses:NotExist") {
		t.Error("未通过校验的输出不应落盘")
	}
	if fake.callCount() != 2 {
		t.Errorf("校验失败应回喂重试 1 次(共 2 次调用),实际 %d 次", fake.callCount())
	}
}

func TestInboxUnrecognized(t *testing.T) {
	fake := &fakeAI{outs: []string{"ERROR"}}
	_, r, dataDir := newInboxTestServer(t, fake)

	postInbox(r, testToken, map[string]string{
		"image": base64.StdEncoding.EncodeToString([]byte("not-a-bill")),
		"text":  "备注",
	})

	failedDir := filepath.Join(dataDir, "failed")
	waitFor(t, "失败留档目录生成", func() bool {
		entries, err := os.ReadDir(failedDir)
		return err == nil && len(entries) == 1
	})
	if got := readInbox(t, dataDir); strings.TrimSpace(got) != "" {
		t.Errorf("识别失败不应写入 inbox.bean,实际内容: %q", got)
	}
	// 留档应包含原因与图片
	entries, err := os.ReadDir(failedDir)
	if err != nil {
		t.Fatalf("读取留档目录失败: %v", err)
	}
	sub := filepath.Join(failedDir, entries[0].Name())
	if _, err := os.Stat(filepath.Join(sub, "detail.txt")); err != nil {
		t.Errorf("留档缺少 detail.txt: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sub, "bill.jpg")); err != nil {
		t.Errorf("留档缺少账单图片: %v", err)
	}
}

func TestCleanAIOutput(t *testing.T) {
	cases := map[string]string{
		"plain":                          "plain",
		"```beancount\ntxn body\n```":    "txn body",
		"```\ntxn body\n```":             "txn body",
		"  \n2026-01-01 * \"a\" \"b\"\n": `2026-01-01 * "a" "b"`,
	}
	for in, want := range cases {
		if got := cleanAIOutput(in); got != want {
			t.Errorf("cleanAIOutput(%q) = %q, 期望 %q", in, got, want)
		}
	}
}
