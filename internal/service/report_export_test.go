package service

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestEncodeCSV(t *testing.T) {
	data := encodeCSV([]string{"药品", "数量"}, [][]string{{"阿莫西林", "10"}, {"含,逗号\"引号", "5"}})
	if !bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("CSV 缺少 UTF-8 BOM（Excel 会乱码）")
	}
	body := string(data[3:])
	if !strings.Contains(body, "药品,数量\n") {
		t.Fatalf("表头错误：%q", body)
	}
	// 含逗号/引号的字段必须被引号包裹且引号转义
	if !strings.Contains(body, "\"含,逗号\"\"引号\",5\n") {
		t.Fatalf("特殊字符转义错误：%q", body)
	}
}

func TestYuan(t *testing.T) {
	cases := map[int64]string{0: "0.00", 1: "0.01", 100: "1.00", 1234: "12.34", -50: "-0.50"}
	for in, want := range cases {
		if got := yuan(in); got != want {
			t.Fatalf("yuan(%d) = %s, want %s", in, got, want)
		}
	}
}

func TestExportCSVUnknownName(t *testing.T) {
	// 未知报表名应在触碰数据库前返回 ErrBadRequest（nil DB 也安全）
	_, _, err := NewReportService(nil).ExportCSV(context.Background(), "no-such-report", 0, nil, nil, time.Now())
	if err == nil {
		t.Fatal("未知报表名应报错")
	}
}

func TestParseRetentionDays(t *testing.T) {
	if n, err := parseRetentionDays("180"); err != nil || n != 180 {
		t.Fatalf("parseRetentionDays(180) = %d,%v", n, err)
	}
	if n, err := parseRetentionDays("0"); err != nil || n != 0 {
		t.Fatalf("parseRetentionDays(0) = %d,%v（0=不归档须合法）", n, err)
	}
	if _, err := parseRetentionDays("abc"); err == nil {
		t.Fatal("非数字应报错")
	}
	if _, err := parseRetentionDays(""); err == nil {
		t.Fatal("空串应报错")
	}
}
