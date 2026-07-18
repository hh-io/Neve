package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client 是账单识别的视觉模型抽象,inbox 流程只依赖这一个方法,测试用 fake 注入
type Client interface {
	// Recognize 用 prompt + 账单图片调用视觉模型,返回模型的文本输出
	Recognize(ctx context.Context, prompt, imageB64, mimeType string) (string, error)
	Provider() string
}

// NewClientFromEnv 按环境变量构造客户端:
// NEVE_AI_PROVIDER=claude|gemini, NEVE_AI_API_KEY, NEVE_AI_MODEL(claude 有默认值)。
// 两家均走原生 HTTP 而非官方 SDK,维持"后端唯一依赖 gin"的单二进制约定。
func NewClientFromEnv() (Client, error) {
	provider := os.Getenv("NEVE_AI_PROVIDER")
	key := os.Getenv("NEVE_AI_API_KEY")
	model := os.Getenv("NEVE_AI_MODEL")
	if provider == "" || key == "" {
		return nil, fmt.Errorf("需要设置 NEVE_AI_PROVIDER (claude|gemini) 与 NEVE_AI_API_KEY")
	}
	hc := &http.Client{Timeout: 120 * time.Second}
	switch provider {
	case "claude":
		if model == "" {
			model = "claude-opus-4-8"
		}
		return &claudeClient{apiKey: key, model: model, hc: hc}, nil
	case "gemini":
		if model == "" {
			return nil, fmt.Errorf("NEVE_AI_PROVIDER=gemini 时必须设置 NEVE_AI_MODEL(填快捷指令中现用的模型名)")
		}
		return &geminiClient{apiKey: key, model: model, hc: hc}, nil
	default:
		return nil, fmt.Errorf("不支持的 NEVE_AI_PROVIDER: %q", provider)
	}
}

// ---- Claude (Anthropic Messages API) ----

type claudeClient struct {
	apiKey string
	model  string
	hc     *http.Client
}

func (c *claudeClient) Provider() string { return "claude" }

func (c *claudeClient) Recognize(ctx context.Context, prompt, imageB64, mimeType string) (string, error) {
	payload := map[string]any{
		"model":      c.model,
		"max_tokens": 2048,
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": mimeType,
							"data":       imageB64,
						},
					},
					map[string]any{"type": "text", "text": prompt},
				},
			},
		},
	}
	headers := map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": "2023-06-01",
	}
	body, err := postJSONWithRetry(ctx, c.hc, "https://api.anthropic.com/v1/messages", headers, payload)
	if err != nil {
		return "", fmt.Errorf("claude: %w", err)
	}

	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("claude: 解析响应失败: %w", err)
	}
	var out strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			out.WriteString(block.Text)
		}
	}
	if out.Len() == 0 {
		return "", fmt.Errorf("claude: 未返回文本 (stop_reason=%s)", resp.StopReason)
	}
	return out.String(), nil
}

// ---- Gemini (Google generateContent API) ----

type geminiClient struct {
	apiKey string
	model  string
	hc     *http.Client
}

func (c *geminiClient) Provider() string { return "gemini" }

func (c *geminiClient) Recognize(ctx context.Context, prompt, imageB64, mimeType string) (string, error) {
	payload := map[string]any{
		"contents": []any{
			map[string]any{
				"parts": []any{
					map[string]any{
						"inline_data": map[string]any{
							"mime_type": mimeType,
							"data":      imageB64,
						},
					},
					map[string]any{"text": prompt},
				},
			},
		},
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", c.model)
	// key 走请求头而非 URL 查询参数,避免泄漏进访问日志
	headers := map[string]string{"x-goog-api-key": c.apiKey}
	body, err := postJSONWithRetry(ctx, c.hc, url, headers, payload)
	if err != nil {
		return "", fmt.Errorf("gemini: %w", err)
	}

	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("gemini: 解析响应失败: %w", err)
	}
	var out strings.Builder
	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			out.WriteString(part.Text)
		}
	}
	if out.Len() == 0 {
		reason := "无候选"
		if len(resp.Candidates) > 0 {
			reason = resp.Candidates[0].FinishReason
		}
		return "", fmt.Errorf("gemini: 未返回文本 (finishReason=%s)", reason)
	}
	return out.String(), nil
}

// ---- 共用 HTTP 层 ----

// postJSONWithRetry 对网络错误/429/5xx 退避重试(最多 3 次尝试),其余 4xx 直接失败
func postJSONWithRetry(ctx context.Context, hc *http.Client, url string, headers map[string]string, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("构造请求体失败: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt*2) * time.Second):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := hc.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode == http.StatusOK {
			return body, nil
		}
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			continue
		}
		return nil, lastErr
	}
	return nil, lastErr
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
