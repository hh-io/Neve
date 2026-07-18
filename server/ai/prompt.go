package ai

import (
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// prompt.md 是账单识别提示词模板,占位符 {{DATE}}/{{ACCOUNTS}} 运行时注入。
// 账户上下文由服务端实时提供,提示词无需随 main.bean 手工同步。
//
//go:embed prompt.md
var promptTemplate string

var openLineRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+open\s`)

// ExtractAccounts 从 main.bean 原文提取 open 指令行。走原文而非 parser 的结构化
// 数据,是为了保留行尾的中文注释——这些注释正是 AI 选账户的关键语义。
func ExtractAccounts(mainBeanPath string) (string, error) {
	data, err := os.ReadFile(mainBeanPath)
	if err != nil {
		return "", fmt.Errorf("读取账本失败: %w", err)
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if openLineRe.MatchString(strings.TrimSpace(line)) {
			lines = append(lines, strings.TrimRight(line, " \t"))
		}
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("%s 中未找到任何 open 指令", mainBeanPath)
	}
	return strings.Join(lines, "\n"), nil
}

// BuildPrompt 渲染提示词模板,注入当前日期(服务器本地时区,与记账归属口径一致)与账户列表
func BuildPrompt(accounts string, now time.Time) string {
	p := strings.ReplaceAll(promptTemplate, "{{DATE}}", now.Format("2006-01-02"))
	return strings.ReplaceAll(p, "{{ACCOUNTS}}", accounts)
}
