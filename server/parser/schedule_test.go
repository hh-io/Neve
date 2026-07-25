package parser

import (
	"testing"
)

func monthOf(t *testing.T, months []ScheduleMonth, key string) ScheduleMonth {
	t.Helper()
	for _, m := range months {
		if m.Month == key {
			return m
		}
	}
	t.Fatalf("计划表缺少月份 %s", key)
	return ScheduleMonth{}
}

// assertScheduleInvariants 校验每月合计 = 明细求和、累计 = 前缀和。
// 每个用例都跑一遍,口径改动时能立刻暴露。
func assertScheduleInvariants(t *testing.T, months []ScheduleMonth) {
	t.Helper()
	var running Amount
	for _, m := range months {
		var sum Amount
		for _, e := range m.Entries {
			sum += e.Amount
		}
		if m.Total != sum {
			t.Errorf("%s 合计 = %d, 明细求和 = %d", m.Month, m.Total, sum)
		}
		running += m.Total
		if m.Cumulative != running {
			t.Errorf("%s 累计 = %d, 期望前缀和 = %d", m.Month, m.Cumulative, running)
		}
		if m.Entries == nil {
			t.Errorf("%s 的 Entries 为 nil,应为空切片", m.Month)
		}
	}
}

// 三期分期,每期 33.33 元凑不满 100 元总额:尾差必须落在最后一期
func TestComputeScheduleTailDiffOnLastPeriod(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				Name: "招行信用卡", BillingDay: 9, DueDay: 25,
				Installments: []RevolvingInstallment{{
					Name: "耳机", TotalAmount: 10000, Months: 3,
					MonthlyAmount: 3333, FirstBillMonth: "2026-09",
				}},
			},
		},
	}

	months := ComputeSchedule(cfg, atDate("2026-07-26"), 12)
	assertScheduleInvariants(t, months)

	if len(months) != 12 {
		t.Fatalf("月份数 = %d, 期望 12", len(months))
	}
	if months[0].Month != "2026-07" || months[11].Month != "2027-06" {
		t.Fatalf("窗口 = %s..%s, 期望 2026-07..2027-06", months[0].Month, months[11].Month)
	}

	// 首期账单月之前无条目
	for _, key := range []string{"2026-07", "2026-08"} {
		if m := monthOf(t, months, key); m.Total != 0 || len(m.Entries) != 0 {
			t.Errorf("%s 应无出账,实际 total=%d entries=%d", key, m.Total, len(m.Entries))
		}
	}
	// 前两期整额,末期由总额差值收口
	for _, tc := range []struct {
		month string
		want  Amount
	}{
		{"2026-09", 3333},
		{"2026-10", 3333},
		{"2026-11", 3334},
	} {
		m := monthOf(t, months, tc.month)
		if m.Total != tc.want {
			t.Errorf("%s 合计 = %d, 期望 %d", tc.month, m.Total, tc.want)
		}
	}
	// 分期结束后不再出现
	if m := monthOf(t, months, "2026-12"); m.Total != 0 {
		t.Errorf("2026-12 应无出账,实际 %d", m.Total)
	}
	// 三期之和恰为总额
	if got := monthOf(t, months, "2027-06").Cumulative; got != 10000 {
		t.Errorf("累计 = %d, 期望等于分期总额 10000", got)
	}

	entry := monthOf(t, months, "2026-09").Entries[0]
	if entry.Source != "revolving" || entry.Account != ccAccount || entry.Name != "耳机" {
		t.Errorf("条目元信息不对: %+v", entry)
	}
	if entry.AccountName != "招行信用卡" {
		t.Errorf("AccountName = %q, 期望配置名 招行信用卡", entry.AccountName)
	}
	// 9 号账单 → 当月 25 号还款
	if entry.DueDate != "2026-09-25" {
		t.Errorf("还款日 = %s, 期望 2026-09-25", entry.DueDate)
	}
}

// 计划表与当期状态必须同源:同一个账单月两边算出的金额要一致
func TestComputeScheduleMatchesCurrentPeriod(t *testing.T) {
	items := []RevolvingInstallment{{
		Name: "耳机", TotalAmount: 10000, Months: 3,
		MonthlyAmount: 3333, FirstBillMonth: "2026-09",
	}}
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {BillingDay: 9, DueDay: 25, Installments: items},
		},
	}

	months := ComputeSchedule(cfg, atDate("2026-11-20"), 1)
	assertScheduleInvariants(t, months)

	// 11 月 20 日的当期账单日是 11 月 9 日,恰为末期
	_, _, thisPeriod := revolvingInstallmentStatuses(items, atDate("2026-11-09"))
	if months[0].Total != thisPeriod {
		t.Errorf("计划表 %s 合计 = %d, 当期状态本期出账 = %d", months[0].Month, months[0].Total, thisPeriod)
	}
}

// 还款日跨月(账单 25 / 还款 10):条目必须落在钱实际流出的那个月,而非账单月
func TestComputeScheduleBucketsByDueMonth(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				BillingDay: 25, DueDay: 10,
				Installments: []RevolvingInstallment{{
					Name: "手机", TotalAmount: 12000, Months: 12,
					MonthlyAmount: 1000, FirstBillMonth: "2026-06",
				}},
			},
		},
	}

	months := ComputeSchedule(cfg, atDate("2026-07-26"), 12)
	assertScheduleInvariants(t, months)

	// 首桶的钱来自 6 月账单(due 2026-07-10)——今天 26 号已过,被过期过滤剔除
	if m := monthOf(t, months, "2026-07"); m.Total != 0 {
		t.Errorf("2026-07 的 due 已过,应无出账,实际 %d", m.Total)
	}
	// 其余每桶的 dueDate 必须与桶月同月,且金额是一期
	for _, key := range []string{"2026-08", "2026-12", "2027-05"} {
		m := monthOf(t, months, key)
		if m.Total != 1000 || len(m.Entries) != 1 {
			t.Fatalf("%s 合计 = %d(%d 条), 期望一期 1000", key, m.Total, len(m.Entries))
		}
		if got := m.Entries[0].DueDate[:7]; got != key {
			t.Errorf("%s 桶里的还款日 = %s, 应与桶月同月", key, m.Entries[0].DueDate)
		}
	}
	// 末期(2027-05 账单)出账落在窗口最后一桶,尾差归它
	last := monthOf(t, months, "2027-06")
	if last.Total != 1000 || last.Entries[0].DueDate != "2027-06-10" {
		t.Errorf("2027-06 = %d(due %s), 期望末期 1000 / 2027-06-10", last.Total, last.Entries[0].DueDate)
	}
	// 首期已过期被剔除,窗口内是第 2..12 期共 11 期
	if last.Cumulative != 11000 {
		t.Errorf("累计 = %d, 期望 11000", last.Cumulative)
	}
}

// 还款日已过的期不属于「未来计划」,口径与 InstallmentStatus.Paid / Overdue 对齐
func TestComputeScheduleSkipsPastDue(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				BillingDay: 9, DueDay: 25,
				Installments: []RevolvingInstallment{{
					Name: "耳机", TotalAmount: 3000, Months: 3,
					MonthlyAmount: 1000, FirstBillMonth: "2026-06",
				}},
			},
		},
		Installments: []InstallmentConfig{{
			ID: "car", Name: "车贷", Account: "Liabilities:Loan:Car", DueDay: 20,
			Schedule: []InstallmentPhase{{EffectiveFrom: "2020-01-01", Amount: 300000}},
		}},
	}

	// 26 号:本月两笔的还款日(25 / 20)都已过去
	months := ComputeSchedule(cfg, atDate("2026-07-26"), 3)
	assertScheduleInvariants(t, months)
	if m := monthOf(t, months, "2026-07"); m.Total != 0 {
		t.Errorf("2026-07 两笔都已过期,应无出账,实际 %d(%d 条)", m.Total, len(m.Entries))
	}
	// 8 月是分期末期(2026-08 账单为第 3 期)+ 车贷月供
	if m := monthOf(t, months, "2026-08"); m.Total != 301000 {
		t.Errorf("2026-08 合计 = %d, 期望 301000", m.Total)
	}

	// 同配置换到 19 号:两笔都还没到期,必须计入
	early := ComputeSchedule(cfg, atDate("2026-07-19"), 3)
	assertScheduleInvariants(t, early)
	if m := monthOf(t, early, "2026-07"); m.Total != 301000 {
		t.Errorf("2026-07-19 时两笔均未到期,合计 = %d, 期望 301000", m.Total)
	}

	// 还款日当天算未来:钱当天才流出,不能提前剔掉
	sameDay := ComputeSchedule(cfg, atDate("2026-07-20"), 1)
	assertScheduleInvariants(t, sameDay)
	var hasCar bool
	for _, e := range sameDay[0].Entries {
		if e.Source == "installment" && e.DueDate == "2026-07-20" {
			hasCar = true
		}
	}
	if !hasCar {
		t.Errorf("还款日当天的车贷应计入,实际条目 %+v", sameDay[0].Entries)
	}
}

// 账单日/还款日 31 号落在 2 月:必须顺延至月末,而非 time.Date 进位到 3 月
func TestComputeScheduleMonthEndClamp(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				BillingDay: 31, DueDay: 31,
				Installments: []RevolvingInstallment{{
					Name: "手机", TotalAmount: 24000, Months: 12,
					MonthlyAmount: 2000, FirstBillMonth: "2026-01",
				}},
			},
		},
		Installments: []InstallmentConfig{{
			ID: "mortgage", Name: "房贷", Account: "Liabilities:Loan:Mortgage", DueDay: 31,
			Schedule: []InstallmentPhase{{EffectiveFrom: "2020-01-01", Amount: 500000}},
		}},
	}

	months := ComputeSchedule(cfg, atDate("2026-01-15"), 3)
	assertScheduleInvariants(t, months)

	dueIn := func(key, source string) string {
		for _, e := range monthOf(t, months, key).Entries {
			if e.Source == source {
				return e.DueDate
			}
		}
		return ""
	}
	// 1 月 31 日出账,同日的还款日不算「账单日之后」,顺延到 2 月末(2026 非闰年)——
	// 钱 2 月流出,故落在 2 月桶而非 1 月桶
	if got := dueIn("2026-02", "revolving"); got != "2026-02-28" {
		t.Errorf("2026-02 桶的额度分期还款日 = %q, 期望 2026-02-28", got)
	}
	// 2 月 28 日出账 → 顺延到 3 月 31 日,落 3 月桶
	if got := dueIn("2026-03", "revolving"); got != "2026-03-31" {
		t.Errorf("2026-03 桶的额度分期还款日 = %q, 期望 2026-03-31", got)
	}
	// 固定分期还款日就在本月,31 号顺延至 2 月末
	if got := dueIn("2026-02", "installment"); got != "2026-02-28" {
		t.Errorf("固定分期还款日 = %q, 期望 2026-02-28", got)
	}
}

// 月供在窗口中途调整:自生效月的还款日起用新金额,旧月份不被改写
func TestComputeScheduleInstallmentPhases(t *testing.T) {
	cfg := &DebtsConfig{
		Installments: []InstallmentConfig{{
			ID: "mortgage", Name: "房贷", Account: "Liabilities:Loan:Mortgage", DueDay: 20,
			Schedule: []InstallmentPhase{
				{EffectiveFrom: "2020-01-01", Amount: 500000},
				{EffectiveFrom: "2026-10-01", Amount: 600000},
			},
		}},
	}

	months := ComputeSchedule(cfg, atDate("2026-07-26"), 12)
	assertScheduleInvariants(t, months)

	for _, tc := range []struct {
		month string
		want  Amount
	}{
		// 20 号还款、今天 26 号:本月那期已经出账,窗口从 8 月这期起算
		{"2026-07", 0},
		{"2026-08", 500000},
		{"2026-09", 500000},
		{"2026-10", 600000},
		{"2027-06", 600000},
	} {
		if got := monthOf(t, months, tc.month).Total; got != tc.want {
			t.Errorf("%s 月供 = %d, 期望 %d", tc.month, got, tc.want)
		}
	}
}

// schedule 全在窗口之后 = 尚未生效,该月不出条目而非出 0 元条目
func TestComputeScheduleInstallmentNotYetEffective(t *testing.T) {
	cfg := &DebtsConfig{
		Installments: []InstallmentConfig{{
			ID: "car", Name: "车贷", Account: "Liabilities:Loan:Car", DueDay: 5,
			Schedule: []InstallmentPhase{{EffectiveFrom: "2027-01-01", Amount: 300000}},
		}},
	}

	months := ComputeSchedule(cfg, atDate("2026-07-26"), 12)
	assertScheduleInvariants(t, months)

	if m := monthOf(t, months, "2026-12"); len(m.Entries) != 0 {
		t.Errorf("生效前应无条目,实际 %d 条", len(m.Entries))
	}
	if m := monthOf(t, months, "2027-01"); m.Total != 300000 {
		t.Errorf("2027-01 月供 = %d, 期望 300000", m.Total)
	}
}

func TestComputeScheduleLongTermFlag(t *testing.T) {
	cfg := &DebtsConfig{
		LongTermAccounts: []string{"Liabilities:Loan:Mortgage"},
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				BillingDay: 9, DueDay: 25,
				Installments: []RevolvingInstallment{{
					Name: "耳机", TotalAmount: 3000, Months: 3,
					MonthlyAmount: 1000, FirstBillMonth: "2026-07",
				}},
			},
		},
		Installments: []InstallmentConfig{{
			ID: "mortgage", Name: "房贷", Account: "Liabilities:Loan:Mortgage", DueDay: 20,
			Schedule: []InstallmentPhase{{EffectiveFrom: "2020-01-01", Amount: 500000}},
		}},
	}

	months := ComputeSchedule(cfg, atDate("2026-07-01"), 1)
	assertScheduleInvariants(t, months)

	for _, e := range months[0].Entries {
		want := e.Account == "Liabilities:Loan:Mortgage"
		if e.LongTerm != want {
			t.Errorf("%s 的 LongTerm = %v, 期望 %v", e.Account, e.LongTerm, want)
		}
	}
}

func TestComputeScheduleEmptyConfig(t *testing.T) {
	months := ComputeSchedule(&DebtsConfig{}, atDate("2026-07-26"), 12)
	assertScheduleInvariants(t, months)

	if len(months) != 12 {
		t.Fatalf("月份数 = %d, 期望 12", len(months))
	}
	for _, m := range months {
		if m.Total != 0 || m.Cumulative != 0 {
			t.Errorf("%s 应全为 0, 实际 total=%d cumulative=%d", m.Month, m.Total, m.Cumulative)
		}
	}

	if got := ComputeSchedule(&DebtsConfig{}, atDate("2026-07-26"), 0); got == nil || len(got) != 0 {
		t.Errorf("months<=0 应返回空切片而非 nil,实际 %v", got)
	}
}

// 首期月非法(手改文件绕过 Validate)只跳过该条,不让整份计划失败
func TestComputeScheduleSkipsInvalidFirstBillMonth(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				BillingDay: 9, DueDay: 25,
				Installments: []RevolvingInstallment{
					{Name: "坏数据", TotalAmount: 3000, Months: 3, MonthlyAmount: 1000, FirstBillMonth: "2026/07"},
					{Name: "耳机", TotalAmount: 3000, Months: 3, MonthlyAmount: 1000, FirstBillMonth: "2026-07"},
				},
			},
		},
	}

	months := ComputeSchedule(cfg, atDate("2026-07-01"), 1)
	assertScheduleInvariants(t, months)

	if len(months[0].Entries) != 1 || months[0].Total != 1000 {
		t.Errorf("应只保留合法分期,实际 entries=%d total=%d", len(months[0].Entries), months[0].Total)
	}
}
