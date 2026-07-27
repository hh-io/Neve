package parser

import (
	"testing"
	"time"
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

	months := ComputeSchedule(cfg, nil, atDate("2026-07-26"), 12)
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

	months := ComputeSchedule(cfg, nil, atDate("2026-11-20"), 1)
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

	months := ComputeSchedule(cfg, nil, atDate("2026-07-26"), 12)
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
	months := ComputeSchedule(cfg, nil, atDate("2026-07-26"), 3)
	assertScheduleInvariants(t, months)
	if m := monthOf(t, months, "2026-07"); m.Total != 0 {
		t.Errorf("2026-07 两笔都已过期,应无出账,实际 %d(%d 条)", m.Total, len(m.Entries))
	}
	// 8 月是分期末期(2026-08 账单为第 3 期)+ 车贷月供
	if m := monthOf(t, months, "2026-08"); m.Total != 301000 {
		t.Errorf("2026-08 合计 = %d, 期望 301000", m.Total)
	}

	// 同配置换到 19 号:两笔都还没到期,必须计入
	early := ComputeSchedule(cfg, nil, atDate("2026-07-19"), 3)
	assertScheduleInvariants(t, early)
	if m := monthOf(t, early, "2026-07"); m.Total != 301000 {
		t.Errorf("2026-07-19 时两笔均未到期,合计 = %d, 期望 301000", m.Total)
	}

	// from 带时分秒不能改变结果:还款日是零点,不按天比会把今天到期的整笔丢掉
	afternoon := ComputeSchedule(cfg, nil, time.Date(2026, 7, 20, 14, 30, 0, 0, time.Local), 1)
	assertScheduleInvariants(t, afternoon)
	if afternoon[0].Total != ComputeSchedule(cfg, nil, atDate("2026-07-20"), 1)[0].Total {
		t.Errorf("带时分秒的 from 合计 = %d, 应与零点一致", afternoon[0].Total)
	}

	// 还款日当天算未来:钱当天才流出,不能提前剔掉
	sameDay := ComputeSchedule(cfg, nil, atDate("2026-07-20"), 1)
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

	months := ComputeSchedule(cfg, nil, atDate("2026-01-15"), 3)
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

	months := ComputeSchedule(cfg, nil, atDate("2026-07-26"), 12)
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

	months := ComputeSchedule(cfg, nil, atDate("2026-07-26"), 12)
	assertScheduleInvariants(t, months)

	if m := monthOf(t, months, "2026-12"); len(m.Entries) != 0 {
		t.Errorf("生效前应无条目,实际 %d 条", len(m.Entries))
	}
	if m := monthOf(t, months, "2027-01"); m.Total != 300000 {
		t.Errorf("2027-01 月供 = %d, 期望 300000", m.Total)
	}
}

// 有终止期的固定分期只铺到末期月,不外推;无终止期的房贷铺满整个窗口
func TestComputeScheduleInstallmentEndMonth(t *testing.T) {
	cfg := &DebtsConfig{
		Installments: []InstallmentConfig{
			{
				ID: "ecmb", Name: "E招贷", Account: "Liabilities:Loan:ECMB", DueDay: 3,
				EndMonth: "2026-10",
				Schedule: []InstallmentPhase{{EffectiveFrom: "2026-08-03", Amount: 172139}},
			},
			{
				ID: "mortgage", Name: "房贷", Account: "Liabilities:Loan:Mortgage", DueDay: 5,
				Schedule: []InstallmentPhase{{EffectiveFrom: "2026-08-05", Amount: 465601}},
			},
		},
	}

	months := ComputeSchedule(cfg, nil, atDate("2026-07-26"), 6)
	assertScheduleInvariants(t, months)

	for _, tc := range []struct {
		month string
		want  Amount
	}{
		{"2026-08", 172139 + 465601},
		{"2026-10", 172139 + 465601}, // 末期月仍出账
		{"2026-11", 465601},          // 结清后只剩房贷
		{"2026-12", 465601},
	} {
		if got := monthOf(t, months, tc.month).Total; got != tc.want {
			t.Errorf("%s 合计 = %d, 期望 %d", tc.month, got, tc.want)
		}
	}
	// 累计只含 3 期 E招贷
	if got := monthOf(t, months, "2026-12").Cumulative; got != 172139*3+465601*5 {
		t.Errorf("累计 = %d, 期望 %d", got, 172139*3+465601*5)
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

	months := ComputeSchedule(cfg, nil, atDate("2026-07-01"), 1)
	assertScheduleInvariants(t, months)

	for _, e := range months[0].Entries {
		want := e.Account == "Liabilities:Loan:Mortgage"
		if e.LongTerm != want {
			t.Errorf("%s 的 LongTerm = %v, 期望 %v", e.Account, e.LongTerm, want)
		}
	}
}

func TestComputeScheduleEmptyConfig(t *testing.T) {
	months := ComputeSchedule(&DebtsConfig{}, nil, atDate("2026-07-26"), 12)
	assertScheduleInvariants(t, months)

	if len(months) != 12 {
		t.Fatalf("月份数 = %d, 期望 12", len(months))
	}
	for _, m := range months {
		if m.Total != 0 || m.Cumulative != 0 {
			t.Errorf("%s 应全为 0, 实际 total=%d cumulative=%d", m.Month, m.Total, m.Cumulative)
		}
	}

	if got := ComputeSchedule(&DebtsConfig{}, nil, atDate("2026-07-26"), 0); got == nil || len(got) != 0 {
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

	months := ComputeSchedule(cfg, nil, atDate("2026-07-01"), 1)
	assertScheduleInvariants(t, months)

	if len(months[0].Entries) != 1 || months[0].Total != 1000 {
		t.Errorf("应只保留合法分期,实际 entries=%d total=%d", len(months[0].Entries), months[0].Total)
	}
}

// ========== 已出账账单余额并入现金流 ==========

// 本期账单已含该期内嵌分期,两者不得同时出现在首桶,否则那笔分期算两次
func TestComputeScheduleStatementSupersedesBilledInstallment(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				Name: "招行信用卡", BillingDay: 9, DueDay: 25,
				Installments: []RevolvingInstallment{{
					Name: "耳机", TotalAmount: 30000, Months: 3,
					MonthlyAmount: 10000, FirstBillMonth: "2026-07",
				}},
			},
		},
	}
	// 7 月账单(含分期首期 100 元)未还,共 500 元,还款日 7-25
	statements := []RevolvingStatus{{
		Account: ccAccount, Name: "招行信用卡",
		StatementDate: "2026-07-09", DueDate: "2026-07-25",
		StatementDue: 50000, Remaining: 50000, InstallmentThisPeriod: 10000,
	}}

	months := ComputeSchedule(cfg, statements, atDate("2026-07-20"), 3)
	assertScheduleInvariants(t, months)

	jul := monthOf(t, months, "2026-07")
	if len(jul.Entries) != 1 {
		t.Fatalf("7 月应只有账单一条,实际 %d 条:%+v", len(jul.Entries), jul.Entries)
	}
	if jul.Entries[0].Source != "statement" || jul.Entries[0].Amount != 50000 {
		t.Errorf("7 月条目 = %+v, want statement/50000", jul.Entries[0])
	}
	if jul.Total != 50000 {
		t.Errorf("7 月合计 = %d, want 50000(分期已含在账单里,不得重复计入)", jul.Total)
	}
	// 8、9 月的分期尚未出账,照常按分期展开
	if aug := monthOf(t, months, "2026-08"); aug.Total != 10000 {
		t.Errorf("8 月合计 = %d, want 10000", aug.Total)
	}
}

// 账单已还清:该卡当期既无账单条目,也不该让被覆盖的那期分期"复活"
func TestComputeScheduleSettledStatementLeavesNoEntry(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				Name: "招行信用卡", BillingDay: 9, DueDay: 25,
				Installments: []RevolvingInstallment{{
					Name: "耳机", TotalAmount: 30000, Months: 3,
					MonthlyAmount: 10000, FirstBillMonth: "2026-07",
				}},
			},
		},
	}
	statements := []RevolvingStatus{{
		Account: ccAccount, Name: "招行信用卡",
		StatementDate: "2026-07-09", DueDate: "2026-07-25",
		StatementDue: 50000, PaidSince: 50000, Remaining: 0, InstallmentThisPeriod: 10000,
	}}

	months := ComputeSchedule(cfg, statements, atDate("2026-07-20"), 2)
	assertScheduleInvariants(t, months)

	if jul := monthOf(t, months, "2026-07"); len(jul.Entries) != 0 || jul.Total != 0 {
		t.Errorf("账单已还清,7 月应无出账,实际 %+v", jul.Entries)
	}
}

// 账单日 25 / 还款日 10:账单余额要归到**还款日**所在月,与分期同一套归桶规则
func TestComputeScheduleStatementBucketsByDueMonth(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {Name: "招行信用卡", BillingDay: 25, DueDay: 10},
		},
	}
	statements := []RevolvingStatus{{
		Account: ccAccount, Name: "招行信用卡",
		StatementDate: "2026-07-25", DueDate: "2026-08-10",
		StatementDue: 30000, Remaining: 30000,
	}}

	months := ComputeSchedule(cfg, statements, atDate("2026-07-28"), 3)
	assertScheduleInvariants(t, months)

	if jul := monthOf(t, months, "2026-07"); jul.Total != 0 {
		t.Errorf("7 月不该有出账,实际 %d", jul.Total)
	}
	if aug := monthOf(t, months, "2026-08"); aug.Total != 30000 {
		t.Errorf("8 月合计 = %d, want 30000", aug.Total)
	}
}

// 逾期账单的 due 已过去,归 Summary.MonthRemaining 管,不进"未来"计划
func TestComputeScheduleSkipsOverdueStatement(t *testing.T) {
	cfg := revolvingCfg(9, 20)
	statements := []RevolvingStatus{{
		Account: ccAccount, Name: "招行信用卡",
		StatementDate: "2026-07-09", DueDate: "2026-07-20",
		StatementDue: 30000, Remaining: 30000, Overdue: true,
	}}

	months := ComputeSchedule(cfg, statements, atDate("2026-07-28"), 3)
	assertScheduleInvariants(t, months)

	for _, m := range months {
		if m.Total != 0 {
			t.Errorf("%s 不该有出账(逾期账单不进未来计划),实际 %d", m.Month, m.Total)
		}
	}
}

// 端到端:从账本算出的账单余额进入计划表,且与内嵌分期不重复计入
func TestComputeDebtsScheduleIncludesStatement(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				Name: "招行信用卡", BillingDay: 9, DueDay: 25,
				Installments: []RevolvingInstallment{{
					Name: "耳机", TotalAmount: 30000, Months: 3,
					MonthlyAmount: 10000, FirstBillMonth: "2026-07",
				}},
			},
		},
	}
	ledger := debtLedger(
		[]string{ccAccount, "Assets:Cash:Alipay"},
		// 分期消费记账时全额入负债账户
		mkTx("2026-06-20",
			po("Expenses:Shopping:Daily", 30000),
			po(ccAccount, -30000)),
		// 本期普通消费 400 元
		mkTx("2026-07-05",
			po("Expenses:Food:Coffee", 40000),
			po(ccAccount, -40000)),
	)

	report := ComputeDebts(ledger, cfg, atDate("2026-07-20"))
	rv := report.Revolving[0]
	// 快照 700 − 未出账分期 200 = 500
	if rv.Remaining != 50000 {
		t.Fatalf("Remaining = %d, want 50000", rv.Remaining)
	}

	jul := monthOf(t, report.Schedule, "2026-07")
	if jul.Total != rv.Remaining {
		t.Errorf("7 月合计 = %d, want %d(账单余额,分期不重复)", jul.Total, rv.Remaining)
	}
	if len(jul.Entries) != 1 || jul.Entries[0].Source != "statement" {
		t.Errorf("7 月条目 = %+v, want 单条 statement", jul.Entries)
	}
	// 剩余两期分期照常展开
	if aug := monthOf(t, report.Schedule, "2026-08"); aug.Total != 10000 {
		t.Errorf("8 月合计 = %d, want 10000", aug.Total)
	}
}

// 已消费未出账:落到**下个**账单日之后的还款日,且与未出账分期不重复
func TestComputeScheduleUnbilledSpending(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				Name: "招行信用卡", BillingDay: 9, DueDay: 25,
				Installments: []RevolvingInstallment{{
					Name: "耳机", TotalAmount: 30000, Months: 3,
					MonthlyAmount: 10000, FirstBillMonth: "2026-07",
				}},
			},
		},
	}
	// 账单 500 未还,未出账分期 200(8、9 月各一期),另有 150 已刷未出账
	statements := []RevolvingStatus{{
		Account: ccAccount, Name: "招行信用卡",
		StatementDate: "2026-07-09", DueDate: "2026-07-25",
		StatementDue: 50000, Remaining: 50000, CurrentBalance: 85000,
		Installments: []RevolvingInstallmentStatus{{Name: "耳机", UnbilledAmount: 20000}},
	}}

	months := ComputeSchedule(cfg, statements, atDate("2026-07-20"), 4)
	assertScheduleInvariants(t, months)

	// 85000 - 50000 - 20000 = 15000,8-09 出账 → 8-25 还款
	aug := monthOf(t, months, "2026-08")
	var unbilled *ScheduleEntry
	for i := range aug.Entries {
		if aug.Entries[i].Source == "unbilled" {
			unbilled = &aug.Entries[i]
		}
	}
	if unbilled == nil {
		t.Fatalf("8 月缺少 unbilled 条目:%+v", aug.Entries)
	}
	if unbilled.Amount != 15000 || unbilled.DueDate != "2026-08-25" {
		t.Errorf("unbilled = %v @ %s, want 15000 @ 2026-08-25", unbilled.Amount, unbilled.DueDate)
	}
	// 8 月还该有分期第二期,两者并存不冲突
	if aug.Total != 25000 {
		t.Errorf("8 月合计 = %d, want 25000(分期 10000 + 未出账 15000)", aug.Total)
	}

	// 三份之和恒等于当前欠款:本期 50000 + 分期 20000 + 未出账 15000 = 85000
	var sum Amount
	for _, m := range months {
		sum += m.Total
	}
	if sum != 85000 {
		t.Errorf("窗口内合计 = %d, want 85000(= CurrentBalance,总量守恒)", sum)
	}
}

// 账单日 31 的卡:下个账单月是短月时必须顺延到月末,不能进位到次月 1 号
func TestComputeScheduleUnbilledMonthEndClamp(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {Name: "招行信用卡", BillingDay: 31, DueDay: 20},
		},
	}
	statements := []RevolvingStatus{{
		Account: ccAccount, Name: "招行信用卡",
		StatementDate: "2026-01-31", DueDate: "2026-02-20",
		CurrentBalance: 30000, Remaining: 0,
	}}

	months := ComputeSchedule(cfg, statements, atDate("2026-02-05"), 3)
	assertScheduleInvariants(t, months)

	// 2 月账单日顺延到 2-28,之后第一个 20 号 = 3-20
	mar := monthOf(t, months, "2026-03")
	if len(mar.Entries) != 1 || mar.Entries[0].DueDate != "2026-03-20" {
		t.Errorf("2 月账单(31 日顺延至 2-28)应 3-20 还款,实际 %+v", mar.Entries)
	}
}

// 欠款已全部由账单和分期解释时不产生 unbilled 条目(负数更不能出现)
func TestComputeScheduleNoUnbilledWhenFullyExplained(t *testing.T) {
	cfg := revolvingCfg(9, 25)
	statements := []RevolvingStatus{{
		Account: ccAccount, Name: "招行信用卡",
		StatementDate: "2026-07-09", DueDate: "2026-07-25",
		// 多还了款,CurrentBalance 低于 Remaining
		StatementDue: 50000, Remaining: 50000, CurrentBalance: 40000,
	}}

	months := ComputeSchedule(cfg, statements, atDate("2026-07-20"), 3)
	assertScheduleInvariants(t, months)

	for _, m := range months {
		for _, e := range m.Entries {
			if e.Source == "unbilled" {
				t.Errorf("%s 不该有 unbilled 条目:%+v", m.Month, e)
			}
		}
	}
}

// 滚动窗口只卡右边界(左边界由 ComputeSchedule 剔除过去的期负责),含末日当天
func TestSumDueWithin(t *testing.T) {
	cfg := &DebtsConfig{
		Installments: []InstallmentConfig{
			{ID: "a", Name: "房贷", Account: "Liabilities:Loan:Mortgage", DueDay: 5,
				Schedule: []InstallmentPhase{{EffectiveFrom: "2026-01-01", Amount: 100000}}},
		},
	}

	months := ComputeSchedule(cfg, nil, atDate("2026-07-28"), 12)
	// 8-05、9-05 在 45 天内,10-05 在窗口外
	if got := SumDueWithin(months, atDate("2026-07-28"), 45); got != 200000 {
		t.Errorf("45 天内 = %d, want 200000", got)
	}
	// 边界当天算在内:7-28 起 8 天正好到 8-05
	if got := SumDueWithin(months, atDate("2026-07-28"), 8); got != 100000 {
		t.Errorf("8 天内(边界当天)= %d, want 100000", got)
	}
	if got := SumDueWithin(months, atDate("2026-07-28"), 7); got != 0 {
		t.Errorf("7 天内 = %d, want 0", got)
	}
}

// Due30/Due90 与计划表同源:窗口足够长时应等于表内合计
func TestComputeDebtsRollingWindowsMatchSchedule(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {Name: "招行信用卡", BillingDay: 9, DueDay: 25},
		},
	}
	ledger := debtLedger(
		[]string{ccAccount, "Assets:Cash:Alipay"},
		mkTx("2026-07-05", po("Expenses:Food:Coffee", 30000), po(ccAccount, -30000)),
		mkTx("2026-07-12", po("Expenses:Shopping:Daily", 10000), po(ccAccount, -10000)),
	)

	report := ComputeDebts(ledger, cfg, atDate("2026-07-20"))
	// 账单 300 → 7-25 在 30 天内;未出账 100 → 8-25 已越过 8-19 的右边界
	if report.Summary.Due30 != 30000 {
		t.Errorf("Due30 = %d, want 30000", report.Summary.Due30)
	}
	if report.Summary.Due90 != 40000 {
		t.Errorf("Due90 = %d, want 40000", report.Summary.Due90)
	}
	// 与当期口径不是一回事:MonthDue 只有已出账那 300
	if report.Summary.MonthDue != 30000 {
		t.Errorf("MonthDue = %d, want 30000(当期口径不含未出账)", report.Summary.MonthDue)
	}
}

// 账单日 25 / 还款日 10:本期账单的 due 落在次月,跳过判定必须认**账单月**。
// 若改回按还款日比对,这里要求 debts.go 与 schedule.go 各自算出同一个日期才成立,
// 一旦账期语义微调就会双重计入该期分期(且不报错)。
func TestComputeScheduleSupersedesByBillingMonthNotDueDate(t *testing.T) {
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {
				Name: "招行信用卡", BillingDay: 25, DueDay: 10,
				Installments: []RevolvingInstallment{{
					Name: "耳机", TotalAmount: 30000, Months: 3,
					MonthlyAmount: 10000, FirstBillMonth: "2026-07",
				}},
			},
		},
	}
	// 7 月账单(账单日 7-25,含分期首期 100)未还,8-10 到期
	statements := []RevolvingStatus{{
		Account: ccAccount, Name: "招行信用卡",
		StatementDate: "2026-07-25", DueDate: "2026-08-10",
		StatementDue: 50000, Remaining: 50000, CurrentBalance: 70000,
		Installments: []RevolvingInstallmentStatus{{Name: "耳机", UnbilledAmount: 20000}},
	}}

	months := ComputeSchedule(cfg, statements, atDate("2026-07-28"), 4)
	assertScheduleInvariants(t, months)

	// 8 月只有账单本身,7 月那期分期不得再单列
	aug := monthOf(t, months, "2026-08")
	for _, e := range aug.Entries {
		if e.Source == "revolving" {
			t.Errorf("7 月账单期的分期不该单列:%+v", e)
		}
	}
	if aug.Total != 50000 {
		t.Errorf("8 月合计 = %d, want 50000(仅账单,分期已含在内)", aug.Total)
	}
	// 8 月账单月的分期(第二期)照常落到 9-10
	sep := monthOf(t, months, "2026-09")
	if sep.Total != 10000 || len(sep.Entries) != 1 || sep.Entries[0].DueDate != "2026-09-10" {
		t.Errorf("9 月应只有分期第二期 @ 09-10,实际 %+v", sep.Entries)
	}
}
