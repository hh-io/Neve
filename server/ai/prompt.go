package ai

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// prompt.md 是账单识别提示词模板,占位符 {{DATE}}/{{ACCOUNTS}} 运行时注入。
// 账户上下文由服务端实时提供,提示词无需随 main.bean 手工同步。
//
//go:embed prompt.md
var promptTemplate string

var (
	openLineRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\s+open\s`)
	includeLineRe = regexp.MustCompile(`^include\s+"([^"]+)"`)
)

// ExtractAccounts 从账本原文提取 open 指令行,逐层展开 include。走原文而非 parser 的
// 结构化数据,是为了保留行尾的中文注释——这些注释正是 AI 选账户的关键语义。
//
// 必须跟着 include 走:账户 open 在哪个文件是用户的自由,而这份输出既是提示词里的
// 账户白名单、又是 validateCandidate 临时账本里**唯一**的 open 来源。漏掉子文件里的
// 账户,AI 既看不见它、照着写出来也过不了预校验(UNKNOWN_ACCOUNT),两次尝试全废后
// 整单转失败留档。
func ExtractAccounts(mainBeanPath string) (string, error) {
	lines, err := collectOpenLines(mainBeanPath, make(map[string]bool))
	if err != nil {
		return "", fmt.Errorf("读取账本失败: %w", err)
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("%s 中未找到任何 open 指令", mainBeanPath)
	}
	return strings.Join(lines, "\n"), nil
}

// collectOpenLines 收集一个文件的 open 行并递归其 include,按读取顺序返回。
// seen 兼做去重与循环检测:同一文件被多处 include 只收一次,循环引用不会无限递归。
func collectOpenLines(path string, seen map[string]bool) ([]string, error) {
	clean := filepath.Clean(path)
	if seen[clean] {
		return nil, nil
	}
	seen[clean] = true

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case openLineRe.MatchString(line):
			// 匹配未缩进的原始行:缩进的 open 在 parser 眼里是 posting,并不建账户
			lines = append(lines, strings.TrimRight(line, " \t\r"))
		case includeLineRe.MatchString(line):
			target := includeLineRe.FindStringSubmatch(line)[1]
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(path), target)
			}
			// 读不到的 include 跳过:账本本身也解析不出这些账户(parser 记
			// INCLUDE_MISSING 软错误),白名单不该比账本更严格
			if sub, err := collectOpenLines(target, seen); err == nil {
				lines = append(lines, sub...)
			}
		}
	}
	return lines, nil
}

// BuildPrompt 渲染提示词模板,注入当前日期(服务器本地时区,与记账归属口径一致)与账户列表
func BuildPrompt(accounts string, now time.Time) string {
	p := strings.ReplaceAll(promptTemplate, "{{DATE}}", now.Format("2006-01-02"))
	return strings.ReplaceAll(p, "{{ACCOUNTS}}", accounts)
}
