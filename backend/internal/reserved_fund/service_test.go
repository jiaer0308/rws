package reserved_fund

import (
	"testing"
	"time"
)

func TestValidate470HeaderRowRequiresExactALHeaders(t *testing.T) {
	headers := append([]string(nil), expected470Headers...)
	if err := validate470HeaderRow(headers); err != nil {
		t.Fatalf("expected valid headers, got error: %v", err)
	}

	headers[1] = "AR PAYMENT NO"
	if err := validate470HeaderRow(headers); err == nil {
		t.Fatal("expected header validation to reject wrong column order")
	}
}

func TestParseExcelDateAcceptsOnlySpecTextFormat(t *testing.T) {
	parsed, err := parseExcelDate("08-06-2026")
	if err != nil {
		t.Fatalf("expected DD-MM-YYYY date to parse: %v", err)
	}
	if parsed.Format("2006-01-02") != "2026-06-08" {
		t.Fatalf("expected 2026-06-08, got %s", parsed.Format("2006-01-02"))
	}

	if _, err := parseExcelDate("2026-06-08"); err == nil {
		t.Fatal("expected YYYY-MM-DD text date to be rejected")
	}
}

func TestAmountsEqualComparesTwoDecimalAmounts(t *testing.T) {
	if !amountsEqual(1.1+2.2, 3.3) {
		t.Fatal("expected cent-normalized amounts to match")
	}
}

func TestMatchedProtectedFieldsChanged(t *testing.T) {
	deducted := "ORDER-1"
	existing := &GroupReservedFund{
		RefundDate:        time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
		FBName:            "Alice",
		Bank:              "GRP",
		ReservedFundsDate: time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC),
		BankOrderDetail:   "BANK-DETAIL",
		DeductedOrderNo:   &deducted,
		Amt:               11439,
		AsstName:          "Assistant",
	}

	incoming := *existing
	incoming.Bank = "RWS"
	if matchedProtectedFieldsChanged(existing, &incoming) {
		t.Fatal("expected bank-only change to be allowed for matched rows")
	}

	incoming = *existing
	incoming.FBName = "ALICE"
	if !matchedProtectedFieldsChanged(existing, &incoming) {
		t.Fatal("expected protected text case change to be blocked")
	}
}


// TestParseExcelDateFormats covers the actual date strings present in 470 xlsx files.
func TestParseExcelDateFormats(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"04/02/2026", "2026-02-04"},
		{"4/02/2026", "2026-02-04"},   // single-digit day
		{"29/09 /2025", "2025-09-29"}, // internal space typo in real file
		{"23-1-2026", "2026-01-23"},   // single-digit month with dash
		{"23-01-2026", "2026-01-23"},  // double-digit month with dash
		{"3-1-2026", "2026-01-03"},    // single-digit day and month with dash
		{"3/1/2026", "2026-01-03"},    // single-digit day and month with slash
	}
	for _, tc := range cases {
		parsed, err := parseExcelDate(tc.input)
		if err != nil {
			t.Errorf("parseExcelDate(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got := parsed.Format("2006-01-02"); got != tc.expected {
			t.Errorf("parseExcelDate(%q): want %s, got %s", tc.input, tc.expected, got)
		}
	}
}


// TestFindHeaderRowWithEmbeddedNewlines ensures the header finder handles col I/J
// headers that contain embedded newline characters as seen in the real xlsx.
func TestFindHeaderRowWithEmbeddedNewlines(t *testing.T) {
	// Row 0: summary row (should be skipped)
	// Row 1: real header row with embedded \n in col I and J
	rows := [][]string{
		{"", "", "", "", "", "#REF!", "26,073.79", "#REF!"},
		{
			"DATE",
			"BANK",
			"FB NAME",
			"购物金预留的      DATE",
			"购物金预留的订单编号",
			"被抵扣的订单编号",
			"购物金AMT",
			"ASST NAME",
			"用谁家购物金\nGRP / SISTER/ COCO / RB / RWS",
			"谁家的单\nGRP / RWS",
			"AR PAYMENT NO",
			"CHECKED",
			"JV NO",
		},
		{"04/02/2026", "", "Test User", "24/01/2026", "ORDER-123", "-", "1.00", "LR", "RWS", "REFUND - XN", "PVPBB-2602-001"},
	}

	mapping, headerIdx, err := findHeaderRowAndMapColumns(rows)
	if err != nil {
		t.Fatalf("findHeaderRowAndMapColumns: unexpected error: %v", err)
	}
	if headerIdx != 1 {
		t.Errorf("expected header at row index 1, got %d", headerIdx)
	}
	if mapping.RefundDateCol != 0 {
		t.Errorf("RefundDateCol: want 0, got %d", mapping.RefundDateCol)
	}
	if mapping.BankCol != 1 {
		t.Errorf("BankCol (BANK col B): want 1, got %d", mapping.BankCol)
	}
	if mapping.FBNameCol != 2 {
		t.Errorf("FBNameCol: want 2, got %d", mapping.FBNameCol)
	}
	if mapping.ReservedDateCol != 3 {
		t.Errorf("ReservedDateCol: want 3, got %d", mapping.ReservedDateCol)
	}
	if mapping.BankOrderDetailCol != 4 {
		t.Errorf("BankOrderDetailCol: want 4, got %d", mapping.BankOrderDetailCol)
	}
	if mapping.AmtCol != 6 {
		t.Errorf("AmtCol: want 6, got %d", mapping.AmtCol)
	}
	if mapping.AsstNameCol != 7 {
		t.Errorf("AsstNameCol: want 7, got %d", mapping.AsstNameCol)
	}
	if mapping.GroupCol != 8 {
		t.Errorf("GroupCol: want 8, got %d", mapping.GroupCol)
	}
	if mapping.PaymentNoCol != 10 {
		t.Errorf("PaymentNoCol: want 10, got %d", mapping.PaymentNoCol)
	}
}
