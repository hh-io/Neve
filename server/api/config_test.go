package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 损坏的配置文件绝不能降级成"空配置"回给前端:前端会显示成"从未配置过",
// 用户随手补一条再保存就把原文件整份覆盖,而配置只有这一份真源。

func newConfigTestServer(t *testing.T) (*Server, http.Handler, string) {
	t.Helper()
	s, r, dataDir := newInboxTestServer(t, nil)
	if err := s.Refresh(); err != nil {
		t.Fatalf("Refresh 失败: %v", err)
	}
	return s, r, dataDir
}

func doJSON(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func corruptFiles(t *testing.T, dataDir, base string) []string {
	t.Helper()
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatalf("读取 dataDir 失败: %v", err)
	}
	var found []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), base+".corrupt-") {
			found = append(found, e.Name())
		}
	}
	return found
}

func TestGetDebtsCorruptConfigFailsLoud(t *testing.T) {
	_, r, dataDir := newConfigTestServer(t)
	if err := os.WriteFile(filepath.Join(dataDir, "debts.json"), []byte(`{"revolving":`), 0o644); err != nil {
		t.Fatal(err)
	}

	w := doJSON(r, http.MethodGet, "/api/debts", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("状态码 = %d, want 500(降级成空配置会导致后续保存覆盖原文件)", w.Code)
	}
	if !strings.Contains(w.Body.String(), "DEBTS_CONFIG_CORRUPT") {
		t.Errorf("响应缺少错误码: %s", w.Body.String())
	}
}

func TestGetDebtsMissingConfigIsEmpty(t *testing.T) {
	_, r, _ := newConfigTestServer(t)

	w := doJSON(r, http.MethodGet, "/api/debts", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200(首次使用、文件不存在是正常状态)", w.Code)
	}
}

func TestSaveDebtsQuarantinesCorruptFile(t *testing.T) {
	_, r, dataDir := newConfigTestServer(t)
	original := `{"revolving": 坏掉的内容`
	if err := os.WriteFile(filepath.Join(dataDir, "debts.json"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	w := doJSON(r, http.MethodPost, "/api/debts", []byte(`{"longTermAccounts":["Liabilities:Loan:Mortgage"]}`))
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200: %s", w.Code, w.Body.String())
	}

	found := corruptFiles(t, dataDir, "debts.json")
	if len(found) != 1 {
		t.Fatalf("留档文件数 = %d, want 1(损坏内容被静默覆盖就再也拿不回来)", len(found))
	}
	data, err := os.ReadFile(filepath.Join(dataDir, found[0]))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Errorf("留档内容 = %q, want %q", data, original)
	}
}

func TestSaveDebtsKeepsValidFile(t *testing.T) {
	_, r, dataDir := newConfigTestServer(t)
	if err := os.WriteFile(filepath.Join(dataDir, "debts.json"), []byte(`{"installments":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if w := doJSON(r, http.MethodPost, "/api/debts", []byte(`{}`)); w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200: %s", w.Code, w.Body.String())
	}
	if found := corruptFiles(t, dataDir, "debts.json"); len(found) != 0 {
		t.Errorf("可解析的旧文件不该被留档: %v", found)
	}
}

func TestGetBudgetsCorruptFailsLoud(t *testing.T) {
	_, r, dataDir := newConfigTestServer(t)
	if err := os.WriteFile(filepath.Join(dataDir, "budgets.json"), []byte(`{"Food": }`), 0o644); err != nil {
		t.Fatal(err)
	}

	w := doJSON(r, http.MethodGet, "/api/budgets", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("状态码 = %d, want 500", w.Code)
	}
	if !strings.Contains(w.Body.String(), "BUDGETS_CORRUPT") {
		t.Errorf("响应缺少错误码: %s", w.Body.String())
	}
}

func TestGetBudgetsMissingIsEmpty(t *testing.T) {
	_, r, _ := newConfigTestServer(t)

	w := doJSON(r, http.MethodGet, "/api/budgets", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200", w.Code)
	}
	var got map[string]float64
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应不是合法 JSON 对象: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("预算 = %v, want 空", got)
	}
}

func TestSaveBudgetsQuarantinesCorruptFile(t *testing.T) {
	_, r, dataDir := newConfigTestServer(t)
	original := `{"Food": 坏掉的内容}`
	if err := os.WriteFile(filepath.Join(dataDir, "budgets.json"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if w := doJSON(r, http.MethodPost, "/api/budgets", []byte(`{"Food": 1500}`)); w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200: %s", w.Code, w.Body.String())
	}

	found := corruptFiles(t, dataDir, "budgets.json")
	if len(found) != 1 {
		t.Fatalf("留档文件数 = %d, want 1", len(found))
	}
	data, err := os.ReadFile(filepath.Join(dataDir, found[0]))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Errorf("留档内容 = %q, want %q", data, original)
	}
}

// 账本刷新不该被损坏的 debts.json 拖垮:分层字段退回全量口径即可
func TestRefreshSurvivesCorruptDebtsConfig(t *testing.T) {
	s, _, dataDir := newConfigTestServer(t)
	if err := os.WriteFile(filepath.Join(dataDir, "debts.json"), []byte(`不是 JSON`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := s.Refresh(); err != nil {
		t.Fatalf("Refresh 失败: %v", err)
	}
	if s.analytics == nil {
		t.Fatal("analytics 缓存为空")
	}
	if s.analytics.Summary.NetWorthExLongTerm != s.analytics.Summary.NetWorth {
		t.Errorf("分层未退回全量口径: ExLongTerm=%v NetWorth=%v",
			s.analytics.Summary.NetWorthExLongTerm, s.analytics.Summary.NetWorth)
	}
}
