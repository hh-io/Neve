package parser

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Amount 是以"分"为单位的定点金额。
// 全链路用整数运算,避免 float64 累加误差导致对账偏差。
type Amount int64

// String 以"元"为单位输出,如 "1234.56"、"-0.05"。
func (a Amount) String() string {
	v := int64(a)
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	return fmt.Sprintf("%s%d.%02d", sign, v/100, v%100)
}

// MarshalJSON 输出"元"为单位的 JSON 数字(如 1234.56)。
// 两位小数在 float64 精度内无损,前端按普通数字消费即可。
func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(a.String()), nil
}

// Yuan 返回以元为单位的 float64,仅用于百分比/均值等比值计算,不参与金额累加。
func (a Amount) Yuan() float64 {
	return float64(a) / 100
}

// parseAmount 解析 "1,234.56" 形式的金额为分。
// CNY 固定两位小数,超过两位或含非法字符时报错,不静默取零。
func parseAmount(s string) (Amount, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return 0, fmt.Errorf("金额为空")
	}
	neg := false
	if s[0] == '+' || s[0] == '-' {
		neg = s[0] == '-'
		s = s[1:]
	}
	intPart, fracPart, _ := strings.Cut(s, ".")
	if intPart == "" {
		return 0, fmt.Errorf("金额缺少整数部分: %q", s)
	}
	yuan, err := strconv.ParseUint(intPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("非法金额: %q", s)
	}
	var fen uint64
	switch len(fracPart) {
	case 0:
		fen = 0
	case 1:
		fen, err = strconv.ParseUint(fracPart, 10, 64)
		fen *= 10
	case 2:
		fen, err = strconv.ParseUint(fracPart, 10, 64)
	default:
		return 0, fmt.Errorf("金额小数超过两位: %q", s)
	}
	if err != nil {
		return 0, fmt.Errorf("非法金额: %q", s)
	}
	if yuan > math.MaxInt64/100-1 {
		return 0, fmt.Errorf("金额超出范围: %q", s)
	}
	total := int64(yuan)*100 + int64(fen)
	if neg {
		total = -total
	}
	return Amount(total), nil
}
