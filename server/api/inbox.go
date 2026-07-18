package api

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"neve/ai"
	"neve/parser"

	"github.com/gin-gonic/gin"
)

// maxInboxBodyBytes 限制请求体大小;base64 约放大 4/3,对应 ~11MB 原图
const maxInboxBodyBytes = 15 << 20

// maxPendingInboxJobs 在途识别任务上限,防止快捷指令连点/重试打爆 AI 配额
const maxPendingInboxJobs = 3

// inboxProcessTimeout 覆盖"识别 + 校验回喂重试"整个异步流程
const inboxProcessTimeout = 5 * time.Minute

var inboxImageExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// EnableInbox 配置无感记账入口;不调用则 /api/inbox 一律 404(本地开发默认不暴露)
func (s *Server) EnableInbox(client ai.Client, token, barkURL string) {
	s.aiClient = client
	s.inboxToken = token
	s.barkURL = barkURL
}

type inboxRequest struct {
	Image string `json:"image"`          // 账单图片 base64(无 data: 前缀)
	Mime  string `json:"mime,omitempty"` // 默认 image/jpeg
	Text  string `json:"text,omitempty"` // 可选补充说明,原样喂给 AI
}

func (s *Server) handleInbox(c *gin.Context) {
	if s.inboxToken == "" || s.aiClient == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound})
		return
	}

	token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(s.inboxToken)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": NewAPIError("UNAUTHORIZED", "无效的访问令牌")})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxInboxBodyBytes)
	var req inboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	if req.Mime == "" {
		req.Mime = "image/jpeg"
	}
	if _, known := inboxImageExt[req.Mime]; !known {
		c.JSON(http.StatusBadRequest, gin.H{"error": NewAPIError("INVALID_REQUEST", "不支持的图片类型: "+req.Mime)})
		return
	}
	// iOS 快捷指令的 Base64 编码可能带换行,统一剔除空白后再校验
	req.Image = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, req.Image)
	if req.Image == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": NewAPIError("INVALID_REQUEST", "缺少 image 字段")})
		return
	}
	if _, err := base64.StdEncoding.DecodeString(req.Image); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": NewAPIError("INVALID_REQUEST", "image 不是合法的 base64")})
		return
	}

	if s.inboxPending.Load() >= maxPendingInboxJobs {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": ErrRateLimited})
		return
	}
	s.inboxPending.Add(1)
	// 立即 202 返回,识别在后台完成——快捷指令前台只需等这一次上传
	go func() {
		defer s.inboxPending.Add(-1)
		s.processInbox(req)
	}()

	c.JSON(http.StatusAccepted, gin.H{"message": "accepted"})
}

// processInbox 是异步主流程:拼提示词 → AI 识别 → parser 预校验(失败回喂修正一次)
// → 追加 inbox.bean → 刷新缓存 → Bark 推送结果
func (s *Server) processInbox(req inboxRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), inboxProcessTimeout)
	defer cancel()

	accounts, err := ai.ExtractAccounts(filepath.Join(s.dataDir, "main.bean"))
	if err != nil {
		s.inboxFailed(req, "", fmt.Sprintf("提取账户列表失败: %v", err))
		return
	}
	prompt := ai.BuildPrompt(accounts, time.Now())
	if req.Text != "" {
		prompt += "\n\n【用户补充说明】\n" + req.Text
	}

	raw, err := s.aiClient.Recognize(ctx, prompt, req.Image, req.Mime)
	if err != nil {
		s.inboxFailed(req, "", fmt.Sprintf("AI 调用失败: %v", err))
		return
	}
	txn := cleanAIOutput(raw)
	if txn == "" || strings.EqualFold(txn, "ERROR") {
		s.inboxFailed(req, raw, "AI 无法识别账单内容")
		return
	}

	if verr := validateCandidate(accounts, txn); verr != nil {
		// 校验失败把错误回喂 AI 修正一次:账户笔误/借贷不平这类问题大多可自愈
		retryPrompt := prompt + "\n\n【你上次的输出未通过校验,请修正后重新输出】\n上次输出:\n" +
			txn + "\n校验错误:\n" + verr.Error()
		raw2, err2 := s.aiClient.Recognize(ctx, retryPrompt, req.Image, req.Mime)
		if err2 != nil {
			s.inboxFailed(req, txn, fmt.Sprintf("校验失败(%v),重试调用也失败: %v", verr, err2))
			return
		}
		txn = cleanAIOutput(raw2)
		if verr2 := validateCandidate(accounts, txn); verr2 != nil {
			s.inboxFailed(req, txn, fmt.Sprintf("重试后仍未通过校验: %v", verr2))
			return
		}
	}

	if err := s.appendToInbox(txn); err != nil {
		s.inboxFailed(req, txn, fmt.Sprintf("写入 inbox.bean 失败: %v", err))
		return
	}
	if err := s.Refresh(); err != nil {
		log.Printf("inbox: 记账成功但刷新缓存失败: %v", err)
	}
	log.Printf("inbox: 已记账 (provider=%s):\n%s", s.aiClient.Provider(), txn)
	s.notify("Neve 记账成功", truncateRunes(txn, 300))
}

// validateCandidate 复用 parser 做落盘前的质量闸门:在临时目录拼一个最小账本
// (真实账户 open 指令 + 候选交易)试解析,任何 issue 都视为失败,坏数据进不了 iCloud。
func validateCandidate(accountLines, candidate string) error {
	tmp, err := os.MkdirTemp("", "neve-inbox-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmp)

	content := accountLines + "\n\n" + candidate + "\n"
	if err := os.WriteFile(filepath.Join(tmp, "main.bean"), []byte(content), 0o644); err != nil {
		return fmt.Errorf("写入临时账本失败: %w", err)
	}
	ledger, err := parser.NewParser(tmp).Parse()
	if err != nil {
		return err
	}
	if len(ledger.Issues) > 0 {
		msgs := make([]string, 0, len(ledger.Issues))
		for _, issue := range ledger.Issues {
			msgs = append(msgs, fmt.Sprintf("[%s] %s", issue.Code, issue.Message))
		}
		return fmt.Errorf("%s", strings.Join(msgs, "; "))
	}
	if len(ledger.Transactions) == 0 {
		return fmt.Errorf("输出中未解析出任何交易")
	}
	return nil
}

func (s *Server) appendToInbox(txn string) error {
	s.inboxMu.Lock()
	defer s.inboxMu.Unlock()

	f, err := os.OpenFile(filepath.Join(s.dataDir, "inbox.bean"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString("\n" + txn + "\n"); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// inboxFailed 统一失败路径:不落盘账本,原始输出与图片留档 data/failed/ 供手工补记,并推送告警
func (s *Server) inboxFailed(req inboxRequest, aiOutput, reason string) {
	log.Printf("inbox: 记账失败: %s", reason)

	dir := filepath.Join(s.dataDir, "failed", time.Now().Format("20060102-150405.000"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("inbox: 创建留档目录失败: %v", err)
	} else {
		detail := fmt.Sprintf("原因: %s\n\n用户补充: %s\n\nAI 输出:\n%s\n", reason, req.Text, aiOutput)
		if err := os.WriteFile(filepath.Join(dir, "detail.txt"), []byte(detail), 0o644); err != nil {
			log.Printf("inbox: 写入留档 detail.txt 失败: %v", err)
		}
		if img, err := base64.StdEncoding.DecodeString(req.Image); err == nil {
			if err := os.WriteFile(filepath.Join(dir, "bill"+inboxImageExt[req.Mime]), img, 0o644); err != nil {
				log.Printf("inbox: 写入留档图片失败: %v", err)
			}
		}
	}

	body := reason
	if out := strings.TrimSpace(aiOutput); out != "" {
		body += "\n" + truncateRunes(out, 300)
	}
	s.notify("Neve 记账失败", body)
}

// notify 通过 Bark 推送到 iPhone;未配置或失败只记日志,绝不影响记账主流程
func (s *Server) notify(title, body string) {
	if s.barkURL == "" {
		return
	}
	payload, err := json.Marshal(map[string]string{"title": title, "body": body, "group": "neve"})
	if err != nil {
		log.Printf("inbox: 构造推送内容失败: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.barkURL, bytes.NewReader(payload))
	if err != nil {
		log.Printf("inbox: 构造推送请求失败: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("inbox: 推送失败: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("inbox: 推送返回 HTTP %d", resp.StatusCode)
	}
}

// cleanAIOutput 去掉模型偶发包裹的 Markdown 代码块与首尾空白
func cleanAIOutput(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return strings.TrimSpace(s)
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
