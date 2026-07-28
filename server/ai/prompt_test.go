package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeMainBean(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.bean")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入 fixture 失败: %v", err)
	}
	return path
}

func TestExtractAccounts(t *testing.T) {
	path := writeMainBean(t, `;; main.bean
option "operating_currency" "CNY"

2020-01-01 open Assets:Cash:WeChat                     CNY ; 微信零钱
2020-01-01 open Expenses:Food:Delivery                 CNY ; 外卖
; 2020-01-01 open Assets:Fake CNY 这是注释不该被提取
include "inbox.bean"
2025-01-01 * "商户" "不是 open 行"
`)

	got, err := ExtractAccounts(path)
	if err != nil {
		t.Fatalf("ExtractAccounts 出错: %v", err)
	}
	want := "2020-01-01 open Assets:Cash:WeChat                     CNY ; 微信零钱\n" +
		"2020-01-01 open Expenses:Food:Delivery                 CNY ; 外卖"
	if got != want {
		t.Errorf("提取结果不符\n得到:\n%s\n期望:\n%s", got, want)
	}
}

// 账户 open 在哪个文件是用户的自由;漏掉子文件里的账户会让这些账户既进不了提示词、
// 也进不了 validateCandidate 的临时账本,AI 照着账本写反而必然 UNKNOWN_ACCOUNT
func TestExtractAccountsFollowsInclude(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("main.bean", `option "operating_currency" "CNY"
2020-01-01 open Assets:Cash:WeChat CNY ; 微信零钱
include "accounts/cards.bean"
include "missing.bean"
include "accounts/cards.bean"
`)
	write("accounts/cards.bean", `2021-01-01 open Liabilities:CreditCard:CMB CNY ; 招行信用卡
  2021-01-01 open Assets:Fake CNY ; 缩进的不是 open 指令,是 posting
include "loans.bean"
include "../main.bean"
`)
	write("accounts/loans.bean", `2022-01-01 open Liabilities:Loan:Mortgage CNY ; 房贷`)

	got, err := ExtractAccounts(filepath.Join(dir, "main.bean"))
	if err != nil {
		t.Fatalf("ExtractAccounts 出错: %v", err)
	}
	want := "2020-01-01 open Assets:Cash:WeChat CNY ; 微信零钱\n" +
		"2021-01-01 open Liabilities:CreditCard:CMB CNY ; 招行信用卡\n" +
		"2022-01-01 open Liabilities:Loan:Mortgage CNY ; 房贷"
	if got != want {
		t.Errorf("提取结果不符\n得到:\n%s\n期望:\n%s", got, want)
	}
}

func TestExtractAccountsMissingMainBean(t *testing.T) {
	if _, err := ExtractAccounts(filepath.Join(t.TempDir(), "nope.bean")); err == nil {
		t.Error("main.bean 读不到时应返回错误")
	}
}

func TestExtractAccountsEmpty(t *testing.T) {
	path := writeMainBean(t, `option "title" "empty"`)
	if _, err := ExtractAccounts(path); err == nil {
		t.Error("无 open 指令时应返回错误")
	}
}

func TestBuildPrompt(t *testing.T) {
	accounts := "2020-01-01 open Assets:Cash:WeChat CNY ; 微信零钱"
	now := time.Date(2026, 7, 18, 10, 0, 0, 0, time.Local)

	p := BuildPrompt(accounts, now)

	if strings.Contains(p, "{{DATE}}") || strings.Contains(p, "{{ACCOUNTS}}") {
		t.Error("占位符未被替换")
	}
	if !strings.Contains(p, "2026-07-18") {
		t.Error("提示词未注入当前日期")
	}
	if !strings.Contains(p, accounts) {
		t.Error("提示词未注入账户列表(含行尾注释)")
	}
	if !strings.Contains(p, "Beancount") {
		t.Error("提示词模板正文缺失")
	}
}
