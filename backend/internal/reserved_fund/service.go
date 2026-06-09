package reserved_fund

import (
	"context"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Helper to parse dates from Excel
func parseExcelDate(val string) (time.Time, error) {
	val = strings.TrimSpace(val)
	// Remove any non-breaking spaces or extra internal whitespace (e.g. "29/09 /2025")
	val = strings.Join(strings.Fields(val), "")
	if val == "" {
		return time.Time{}, fmt.Errorf("empty date value")
	}

	// Try text formats: DD-MM-YYYY, DD/MM/YYYY, D/MM/YYYY, and single-digit month/day variants
	for _, layout := range []string{"02-01-2006", "02/01/2006", "2/01/2006", "2-1-2006", "2/1/2006"} {
		if t, err := time.Parse(layout, val); err == nil {
			return t, nil
		}
	}

	// Try Excel serial number (stored as float string by excelize when no format applied)
	if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
		t, err := excelize.ExcelDateToTime(floatVal, false)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date format: %q (expected DD/MM/YYYY, DD-MM-YYYY, or Excel serial)", val)
}

// Helper to parse currency amounts
func parseAmount(val string) (float64, error) {
	val = strings.TrimSpace(val)
	val = strings.ReplaceAll(val, ",", "") // Remove thousands separators
	if val == "" {
		return 0, fmt.Errorf("empty amount value")
	}
	amt, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount numeric value: %s", val)
	}
	// Round to 2 decimal places
	return math.Round(amt*100) / 100, nil
}

func amountToCents(amt float64) int64 {
	return int64(math.Round(amt * 100))
}

func amountsEqual(a, b float64) bool {
	return amountToCents(a) == amountToCents(b)
}

func sameBusinessDate(a, b time.Time) bool {
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}

func sameTrimmedText(a, b string) bool {
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}

func sameOptionalText(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return strings.TrimSpace(*a) == strings.TrimSpace(*b)
}

func matchedProtectedFieldsChanged(existing *GroupReservedFund, incoming *GroupReservedFund) bool {
	return !sameBusinessDate(existing.RefundDate, incoming.RefundDate) ||
		!sameTrimmedText(existing.FBName, incoming.FBName) ||
		!sameBusinessDate(existing.ReservedFundsDate, incoming.ReservedFundsDate) ||
		!sameTrimmedText(existing.BankOrderDetail, incoming.BankOrderDetail) ||
		!sameOptionalText(existing.DeductedOrderNo, incoming.DeductedOrderNo) ||
		!amountsEqual(existing.Amt, incoming.Amt) ||
		!sameTrimmedText(existing.AsstName, incoming.AsstName)
}

var expected470Headers = []string{
	"DATE",
	"BANK",
	"FB NAME",
	"购物金预留的      DATE",
	"购物金预留的订单编号",
	"被抵扣的订单编号",
	"购物金AMT",
	"ASST NAME",
	"用谁家购物金 GRP / SISTER/ COCO / RB / RWS",
	"谁家的单 GRP / RWS",
	"AR PAYMENT NO",
	"CHECKED",
	"JV NO",
}

func validate470HeaderRow(headerRow []string) error {
	if len(headerRow) < len(expected470Headers) {
		return fmt.Errorf("470 header row must contain columns A-M; got %d columns", len(headerRow))
	}

	for idx, expected := range expected470Headers {
		actual := strings.TrimSpace(headerRow[idx])
		if actual != expected {
			cellName, _ := excelize.CoordinatesToCellName(idx+1, 1)
			return fmt.Errorf("invalid 470 header at %s: expected %q, got %q", cellName, expected, actual)
		}
	}
	return nil
}

type HeaderMapping struct {
	RefundDateCol      int
	FBNameCol          int
	ReservedDateCol    int
	BankOrderDetailCol int
	DeductedOrderCol   int
	AmtCol             int
	AsstNameCol        int
	BankCol            int // col B "BANK"
	GroupCol           int // col I "用谁家购物金 GRP / RWS" (optional, not stored)
	PaymentNoCol       int
}

func findHeaderRowAndMapColumns(rows [][]string) (*HeaderMapping, int, error) {
	for rIdx := 0; rIdx < len(rows) && rIdx < 5; rIdx++ {
		row := rows[rIdx]
		mapping := &HeaderMapping{
			RefundDateCol:      -1,
			FBNameCol:          -1,
			ReservedDateCol:    -1,
			BankOrderDetailCol: -1,
			DeductedOrderCol:   -1,
			AmtCol:             -1,
			AsstNameCol:        -1,
			BankCol:            -1,
			GroupCol:           -1,
			PaymentNoCol:       -1,
		}

		for cIdx, cell := range row {
			// Normalise: trim whitespace AND collapse embedded newlines so multi-line
			// cell headers (e.g. col I and col J) match correctly.
			norm := strings.ReplaceAll(strings.TrimSpace(cell), "\n", " ")
			norm = strings.Join(strings.Fields(norm), " ") // collapse runs of whitespace
			val := strings.ToUpper(norm)

			if (val == "DATE" || val == "REFUND DATE") && !strings.Contains(norm, "预留") {
				mapping.RefundDateCol = cIdx
			}
			// col B: literal "BANK" — the bank name for this transaction
			if val == "BANK" {
				mapping.BankCol = cIdx
			}
			if val == "FB NAME" {
				mapping.FBNameCol = cIdx
			}
			if strings.Contains(norm, "预留") && strings.Contains(val, "DATE") {
				mapping.ReservedDateCol = cIdx
			}
			if strings.Contains(norm, "预留") && strings.Contains(norm, "订单") {
				mapping.BankOrderDetailCol = cIdx
			}
			if strings.Contains(norm, "被抵扣") {
				mapping.DeductedOrderCol = cIdx
			}
			if strings.Contains(norm, "购物金") && strings.Contains(val, "AMT") {
				mapping.AmtCol = cIdx
			}
			if val == "ASST NAME" {
				mapping.AsstNameCol = cIdx
			}
			// col J: "谁家的单 GRP / RWS" — which group's order (optional, not persisted)
			if strings.Contains(norm, "用谁家购物金") {
				mapping.GroupCol = cIdx
			}
			if strings.Contains(val, "AR PAYMENT") {
				mapping.PaymentNoCol = cIdx
			}
		}

		// Check if we found the core identifiers that confirm this is the header row
		if mapping.RefundDateCol != -1 && mapping.FBNameCol != -1 && mapping.PaymentNoCol != -1 {
			// Validate that all required columns are found
			if mapping.ReservedDateCol == -1 {
				return nil, rIdx, fmt.Errorf("Missing required column header containing: '购物金预留的 DATE'")
			}
			if mapping.BankOrderDetailCol == -1 {
				return nil, rIdx, fmt.Errorf("Missing required column header containing: '购物金预留的订单编号'")
			}
			if mapping.AmtCol == -1 {
				return nil, rIdx, fmt.Errorf("Missing required column header containing: '购物金AMT'")
			}
			if mapping.AsstNameCol == -1 {
				return nil, rIdx, fmt.Errorf("Missing required column header containing: 'ASST NAME'")
			}
			if mapping.BankCol == -1 {
				return nil, rIdx, fmt.Errorf("Missing required column header containing: 'BANK' (col B)")
			}

			return mapping, rIdx, nil
		}
	}
	return nil, -1, fmt.Errorf("Missing required column header containing: 'DATE'")
}

// Import470Excel processes all sheets containing "470" in the Excel file and saves rows to the database.
func (s *Service) Import470Excel(ctx context.Context, batch *FileBatch) {
	filePath := *batch.OriginalFilePath
	log.Printf("Starting 470 Import for batch %d, file %s", batch.ID, filePath)

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		s.failBatch(ctx, batch, fmt.Sprintf("Failed to open Excel file: %v", err))
		return
	}
	defer f.Close()

	// Find all sheets that match "470" in name
	sheets := f.GetSheetList()
	var sheetsToImport []string
	for _, name := range sheets {
		if strings.Contains(strings.ToLower(name), "470") {
			sheetsToImport = append(sheetsToImport, name)
		}
	}

	if len(sheetsToImport) == 0 {
		s.failBatch(ctx, batch, "No sheets containing '470' found in Excel file")
		return
	}

	// Initialize error report workbook
	errFile := excelize.NewFile()

	totalRows := 0
	successRows := 0
	errorRows := 0

	for _, sheetName := range sheetsToImport {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			s.writeSheetLevelError(errFile, sheetName, fmt.Sprintf("Failed to read sheet rows: %v", err))
			errorRows++
			continue
		}

		if len(rows) < 1 {
			s.writeSheetLevelError(errFile, sheetName, "Excel sheet is empty")
			errorRows++
			continue
		}

		// 1. Validate Header Row and Map Columns
		mapping, headerRowIdx, err := findHeaderRowAndMapColumns(rows)
		if err != nil {
			s.writeSheetLevelError(errFile, sheetName, fmt.Sprintf("Header validation failed: %v", err))
			errorRows++
			continue
		}

		// Initialize sheet in errFile lazily when the first row error occurs on this sheet
		var errSheetInitialized bool
		errRowIdx := 2

		initErrSheet := func() {
			if errSheetInitialized {
				return
			}
			errFile.NewSheet(sheetName)
			errHeaders := []string{"Row No", "DATE", "FB NAME", "Reserved Funds DATE", "Bank Order Detail", "Deducted Order No", "Amount", "Asst Name", "Bank", "AR Payment No", "Error Message"}
			for colIdx, h := range errHeaders {
				cellName, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
				errFile.SetCellValue(sheetName, cellName, h)
			}
			errSheetInitialized = true
		}

		// First pass: count total non-blank rows in this sheet
		sheetTotalRows := 0
		for idx, row := range rows {
			rowNo := idx + 1
			if rowNo <= headerRowIdx+1 { // Data starts after the header row
				continue
			}

			// Skip blank rows
			isBlank := true
			for _, cell := range row {
				if strings.TrimSpace(cell) != "" {
					isBlank = false
					break
				}
			}
			if isBlank {
				continue
			}
			sheetTotalRows++
		}
		totalRows += sheetTotalRows

		// Second pass: Parse and insert/update
		for idx, row := range rows {
			rowNo := idx + 1
			if rowNo <= headerRowIdx+1 {
				continue
			}

			// Skip blank rows
			isBlank := true
			for _, cell := range row {
				if strings.TrimSpace(cell) != "" {
					isBlank = false
					break
				}
			}
			if isBlank {
				continue
			}

			// Check if row has enough columns for all mapped headers
			maxCol := mapping.RefundDateCol
			for _, col := range []int{mapping.FBNameCol, mapping.ReservedDateCol, mapping.BankOrderDetailCol, mapping.DeductedOrderCol, mapping.AmtCol, mapping.AsstNameCol, mapping.BankCol, mapping.PaymentNoCol} {
				if col > maxCol {
					maxCol = col
				}
			}
			if len(row) <= maxCol {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, "Row is partially filled or missing columns")
				errRowIdx++
				errorRows++
				continue
			}

			// Read cells
			rawRefundDate := row[mapping.RefundDateCol]
			rawFbName := row[mapping.FBNameCol]
			rawReservedDate := row[mapping.ReservedDateCol]
			rawBankOrderDetail := row[mapping.BankOrderDetailCol]

			var rawDeductedOrder string
			if mapping.DeductedOrderCol != -1 && mapping.DeductedOrderCol < len(row) {
				rawDeductedOrder = row[mapping.DeductedOrderCol]
			}

			rawAmt := row[mapping.AmtCol]
			rawAsstName := row[mapping.AsstNameCol]
			// col B (BANK) may be absent or blank; bounds-check before reading
			var rawBank string
			if mapping.BankCol >= 0 && mapping.BankCol < len(row) {
				rawBank = row[mapping.BankCol]
			}
			rawPaymentNo := row[mapping.PaymentNoCol]
			paymentNo := strings.TrimSpace(rawPaymentNo)

			// Field validation
			refundDate, err := parseExcelDate(rawRefundDate)
			if err != nil {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, fmt.Sprintf("Refund Date: %v", err))
				errRowIdx++
				errorRows++
				continue
			}

			fbName := strings.TrimSpace(rawFbName)
			if fbName == "" {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, "FB Name cannot be empty")
				errRowIdx++
				errorRows++
				continue
			}

			reservedDate, err := parseExcelDate(rawReservedDate)
			if err != nil {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, fmt.Sprintf("Reserved Date: %v", err))
				errRowIdx++
				errorRows++
				continue
			}

			bankOrderDetail := strings.TrimSpace(rawBankOrderDetail)
			if bankOrderDetail == "" {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, "Bank Order Detail cannot be empty")
				errRowIdx++
				errorRows++
				continue
			}

			amt, err := parseAmount(rawAmt)
			if err != nil {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, fmt.Sprintf("Amount: %v", err))
				errRowIdx++
				errorRows++
				continue
			}

			asstName := strings.TrimSpace(rawAsstName)
			if asstName == "" {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, "Assistant Name cannot be empty")
				errRowIdx++
				errorRows++
				continue
			}

			bank := strings.TrimSpace(rawBank)
			// Col B (BANK) is often blank; fall back to col I 用谁家购物金 (GRP/SISTER/COCO/RB/RWS)
			if bank == "" && mapping.GroupCol >= 0 && mapping.GroupCol < len(row) {
				bank = strings.TrimSpace(row[mapping.GroupCol])
			}
			if bank == "" {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, "Bank cannot be empty (col B BANK and col I 用谁家购物金 are both blank)")
				errRowIdx++
				errorRows++
				continue
			}

			if paymentNo == "" {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, "AR Payment No cannot be empty")
				errRowIdx++
				errorRows++
				continue
			}

			// Clean deducted_order_no
			var deductedOrderNo *string
			cleanDeducted := strings.TrimSpace(rawDeductedOrder)
			if cleanDeducted != "" && cleanDeducted != "-" {
				deductedOrderNo = &cleanDeducted
			}

			// DB Operations
			existing, err := s.repo.GetReservedFundByARPaymentNo(ctx, paymentNo)
			if err != nil {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, fmt.Sprintf("DB Query Error: %v", err))
				errRowIdx++
				errorRows++
				continue
			}

			rf := &GroupReservedFund{
				RefundDate:          refundDate,
				FBName:              fbName,
				Bank:                bank,
				ReservedFundsDate:   reservedDate,
				BankOrderDetail:     bankOrderDetail,
				ARPaymentNo:         paymentNo,
				DeductedOrderNo:     deductedOrderNo,
				Amt:                 amt,
				AsstName:            asstName,
				SourceImportBatchID: &batch.ID,
			}

			if existing != nil {
				// Row already exists - check if matched state locks changes
				if existing.MatchedAt != nil {
					// Block matches to fields affecting matching or output
					if matchedProtectedFieldsChanged(existing, rf) {
						initErrSheet()
						s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, "Row already matched; matching fields cannot be changed")
						errRowIdx++
						errorRows++
						continue
					}

					// Only update allowed fields (bank, source_import_batch_id)
					rf.ID = existing.ID
					rf.MatchedExportBatchID = existing.MatchedExportBatchID
					rf.MatchedAt = existing.MatchedAt
					rf.CreatedAt = existing.CreatedAt
				} else {
					// Row is not matched, full update is allowed
					rf.ID = existing.ID
					rf.CreatedAt = existing.CreatedAt
				}
			}

			// Perform upsert inside transaction
			tx, err := s.repo.BeginTx(ctx)
			if err != nil {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, fmt.Sprintf("Transaction start failed: %v", err))
				errRowIdx++
				errorRows++
				continue
			}

			err = s.repo.UpsertReservedFund(ctx, tx, rf)
			if err != nil {
				tx.Rollback(ctx)
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, fmt.Sprintf("Upsert failed: %v", err))
				errRowIdx++
				errorRows++
				continue
			}

			err = tx.Commit(ctx)
			if err != nil {
				initErrSheet()
				s.writeErrorRow(errFile, sheetName, errRowIdx, rowNo, mapping, row, fmt.Sprintf("Transaction commit failed: %v", err))
				errRowIdx++
				errorRows++
				continue
			}

			successRows++
		}
	}

	// Update batch statuses and write error report if necessary
	batch.TotalRows = totalRows
	batch.SuccessRows = successRows
	batch.ErrorRows = errorRows

	if errorRows > 0 {
		if len(errFile.GetSheetList()) > 1 {
			errFile.DeleteSheet("Sheet1")
		}
		errPath := filepath.Join(filepath.Dir(filePath), fmt.Sprintf("error_report_%d.xlsx", batch.ID))
		if err := errFile.SaveAs(errPath); err != nil {
			log.Printf("Failed to save error report for batch %d: %v", batch.ID, err)
		} else {
			batch.ErrorReportPath = &errPath
		}
		batch.Status = "completed_with_errors"
	} else {
		batch.Status = "completed"
	}

	err = s.repo.UpdateBatch(ctx, batch)
	if err != nil {
		log.Printf("Failed to update final batch status for batch %d: %v", batch.ID, err)
	}
	log.Printf("Completed 470 Import for batch %d. Total: %d, Success: %d, Errors: %d", batch.ID, totalRows, successRows, errorRows)
}

// MatchReservedFundsExcel processes the 购物金 Excel, finds matches in the database, and writes columns E-J.
func (s *Service) MatchReservedFundsExcel(ctx context.Context, batch *FileBatch) {
	filePath := *batch.OriginalFilePath
	log.Printf("Starting 购物金 matching for batch %d, file %s", batch.ID, filePath)

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		s.failBatch(ctx, batch, fmt.Sprintf("Failed to open Excel file: %v", err))
		return
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		s.failBatch(ctx, batch, fmt.Sprintf("Failed to read sheet rows: %v", err))
		return
	}

	if len(rows) < 4 { // Data starts at row 4
		s.failBatch(ctx, batch, "Excel file is empty or missing data rows (data starts on row 4)")
		return
	}

	// Initialize matching errors sheet
	errFile := excelize.NewFile()
	errSheet := "Matching Errors"
	errFile.SetSheetName("Sheet1", errSheet)
	errHeaders := []string{"Row No", "FB Name", "Reserved Funds Date", "Bank Order Detail", "Amount", "Error Message", "Matched Count"}
	for colIdx, h := range errHeaders {
		cellName, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		errFile.SetCellValue(errSheet, cellName, h)
	}

	// Setup candidate diagnostic sheet
	diagSheet := "Diagnostic Candidates"
	errFile.NewSheet(diagSheet)
	diagHeaders := []string{"Input Row No", "Refund Date", "FB Name", "Reserved Funds Date", "Bank Order Detail", "AR Payment No", "Deducted Order No", "Amount", "Asst Name", "Matched At"}
	for colIdx, h := range diagHeaders {
		cellName, _ := excelize.CoordinatesToCellName(colIdx+1, 1)
		errFile.SetCellValue(diagSheet, cellName, h)
	}

	totalRows := 0
	successRows := 0
	errorRows := 0
	errRowIdx := 2
	diagRowIdx := 2

	// Struct to store parsed input row details
	type inputRow struct {
		rowNo           int
		fbName          string
		reservedDate    time.Time
		bankOrderDetail string
		amt             float64
		parseError      string
	}

	var parsedRows []inputRow
	// Track matches: db_id -> list of input row indexes matching it
	dbMatchCounts := make(map[int][]int)
	// Tentative matches: input row index -> db record
	tentativeMatches := make(map[int]*GroupReservedFund)
	// Failure messages: input row index -> error string
	matchFailures := make(map[int]string)
	// Matched counts for errors: input row index -> count of final candidate rows
	matchCountsForErrors := make(map[int]int)

	// Step 1: Parse all input rows
	for idx, row := range rows {
		rowNo := idx + 1
		if rowNo < 4 { // Header and layout rows are 1-3
			continue
		}

		// Skip blank rows
		isBlank := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				isBlank = false
				break
			}
		}
		if isBlank {
			continue
		}

		totalRows++

		if len(row) < 4 {
			parsedRows = append(parsedRows, inputRow{rowNo: rowNo, parseError: "Row missing required Columns A-D"})
			continue
		}

		rawFbName := row[0]
		rawReservedDate := row[1]
		rawBankOrderDetail := row[2]
		rawAmt := row[3]

		reservedDate, err := parseExcelDate(rawReservedDate)
		if err != nil {
			parsedRows = append(parsedRows, inputRow{rowNo: rowNo, parseError: fmt.Sprintf("Date error: %v", err)})
			continue
		}

		amt, err := parseAmount(rawAmt)
		if err != nil {
			parsedRows = append(parsedRows, inputRow{rowNo: rowNo, parseError: fmt.Sprintf("Amount error: %v", err)})
			continue
		}

		parsedRows = append(parsedRows, inputRow{
			rowNo:           rowNo,
			fbName:          strings.TrimSpace(rawFbName),
			reservedDate:    reservedDate,
			bankOrderDetail: strings.TrimSpace(rawBankOrderDetail),
			amt:             amt,
		})
	}

	// Step 2: Query and match against database candidates (First pass)
	for parsedIdx, ir := range parsedRows {
		if ir.parseError != "" {
			matchFailures[parsedIdx] = ir.parseError
			continue
		}

		// Fetch candidates matching FB Name (case-insensitive & TRIMmed)
		candidates, err := s.repo.GetCandidatesByFBName(ctx, ir.fbName)
		if err != nil {
			matchFailures[parsedIdx] = fmt.Sprintf("DB Query failed: %v", err)
			continue
		}

		if len(candidates) == 0 {
			matchFailures[parsedIdx] = "fb_name not found"
			continue
		}

		// Stage 2: Filter by bank_order_detail
		var stage2 []GroupReservedFund
		for _, c := range candidates {
			if strings.ToLower(strings.TrimSpace(c.BankOrderDetail)) == strings.ToLower(ir.bankOrderDetail) {
				stage2 = append(stage2, c)
			}
		}
		if len(stage2) == 0 {
			matchFailures[parsedIdx] = "bank detail mismatch"
			s.writeDiagnosticCandidates(errFile, diagSheet, &diagRowIdx, ir.rowNo, candidates)
			continue
		}

		// Stage 3: Filter by reserved_funds_date
		var stage3 []GroupReservedFund
		for _, c := range stage2 {
			// Date comparison ignoring time zones
			if c.ReservedFundsDate.Format("2006-01-02") == ir.reservedDate.Format("2006-01-02") {
				stage3 = append(stage3, c)
			}
		}
		if len(stage3) == 0 {
			matchFailures[parsedIdx] = "date mismatch"
			s.writeDiagnosticCandidates(errFile, diagSheet, &diagRowIdx, ir.rowNo, stage2)
			continue
		}

		// Stage 4: Filter by amt
		var stage4 []GroupReservedFund
		for _, c := range stage3 {
			if amountsEqual(c.Amt, ir.amt) {
				stage4 = append(stage4, c)
			}
		}
		if len(stage4) == 0 {
			matchFailures[parsedIdx] = "amount mismatch"
			s.writeDiagnosticCandidates(errFile, diagSheet, &diagRowIdx, ir.rowNo, stage3)
			continue
		}

		// Diagnostics matching count
		matchCountsForErrors[parsedIdx] = len(stage4)

		// Check matching status
		var unmatchedCandidates []GroupReservedFund
		for _, c := range stage4 {
			if c.MatchedAt == nil {
				unmatchedCandidates = append(unmatchedCandidates, c)
			}
		}

		if len(unmatchedCandidates) == 0 {
			// All matching candidates are already matched
			matchFailures[parsedIdx] = "already matched before"
			s.writeDiagnosticCandidates(errFile, diagSheet, &diagRowIdx, ir.rowNo, stage4)
		} else if len(unmatchedCandidates) > 1 {
			// Ambiguous matches (multiple unmatched records with same keys)
			matchFailures[parsedIdx] = "duplicate amount / double payment suspicion"
			s.writeDiagnosticCandidates(errFile, diagSheet, &diagRowIdx, ir.rowNo, unmatchedCandidates)
		} else {
			// Exactly one unmatched candidate remains - Success!
			matchedRecord := unmatchedCandidates[0]
			tentativeMatches[parsedIdx] = &matchedRecord
			dbMatchCounts[matchedRecord.ID] = append(dbMatchCounts[matchedRecord.ID], parsedIdx)
		}
	}

	// Step 3: Handle multi-row matching conflicts in this file
	// "If the same uploaded 购物金 file has two rows matching the same unused 470 row, mark both rows as errors and do not match either one."
	for dbID, parsedIdxList := range dbMatchCounts {
		if len(parsedIdxList) > 1 {
			log.Printf("Conflict: Multiple rows in upload file match the same database record ID %d. Rows: %v", dbID, parsedIdxList)
			for _, pIdx := range parsedIdxList {
				delete(tentativeMatches, pIdx)
				matchFailures[pIdx] = "same uploaded 购物金 file tries to use the same 470 row twice"
				matchCountsForErrors[pIdx] = len(parsedIdxList)
			}
		}
	}

	// Step 4: Perform DB Updates and Write Excel (inside transaction)
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.failBatch(ctx, batch, fmt.Sprintf("Failed to start database transaction: %v", err))
		return
	}
	defer tx.Rollback(ctx)

	for parsedIdx, ir := range parsedRows {
		if errStr, failed := matchFailures[parsedIdx]; failed {
			// Log match failure in error report
			s.writeMatchErrorRow(errFile, errSheet, errRowIdx, ir.rowNo, ir.fbName, ir.reservedDate, ir.bankOrderDetail, ir.amt, errStr, matchCountsForErrors[parsedIdx])
			errRowIdx++
			errorRows++
			continue
		}

		dbRecord, matched := tentativeMatches[parsedIdx]
		if !matched {
			continue // Should not happen
		}

		// Calculate Group Total Amount from DB (including already matched rows)
		groupTotal, err := s.repo.GetGroupTotal(ctx, dbRecord.FBName, dbRecord.BankOrderDetail, dbRecord.ReservedFundsDate)
		if err != nil {
			tx.Rollback(ctx)
			s.failBatch(ctx, batch, fmt.Sprintf("Failed to query group total amount for row %d: %v", ir.rowNo, err))
			return
		}

		// Update matched status in database
		err = s.repo.UpdateMatchedStatus(ctx, tx, dbRecord.ID, batch.ID)
		if err != nil {
			tx.Rollback(ctx)
			s.failBatch(ctx, batch, fmt.Sprintf("Failed to update database match state for row %d: %v", ir.rowNo, err))
			return
		}

		// Write matched fields into original file columns E-K
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", ir.rowNo), dbRecord.RefundDate.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", ir.rowNo), dbRecord.ARPaymentNo)

		deductedVal := "-"
		if dbRecord.DeductedOrderNo != nil {
			deductedVal = *dbRecord.DeductedOrderNo
		}
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", ir.rowNo), deductedVal)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", ir.rowNo), dbRecord.Amt)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", ir.rowNo), groupTotal)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", ir.rowNo), dbRecord.AsstName)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", ir.rowNo), dbRecord.Bank)

		successRows++
	}

	// Save modified Excel file copy
	resultPath := filepath.Join(filepath.Dir(filePath), fmt.Sprintf("result_%d.xlsx", batch.ID))
	err = f.SaveAs(resultPath)
	if err != nil {
		tx.Rollback(ctx)
		s.failBatch(ctx, batch, fmt.Sprintf("Failed to save result Excel copy: %v", err))
		return
	}

	batch.TotalRows = totalRows
	batch.SuccessRows = successRows
	batch.ErrorRows = errorRows
	batch.ResultFilePath = &resultPath

	if errorRows > 0 {
		errPath := filepath.Join(filepath.Dir(filePath), fmt.Sprintf("matching_errors_%d.xlsx", batch.ID))
		if err := errFile.SaveAs(errPath); err != nil {
			tx.Rollback(ctx)
			s.failBatch(ctx, batch, fmt.Sprintf("Failed to save matching error report: %v", err))
			return
		}
		batch.ErrorReportPath = &errPath
		batch.Status = "completed_with_errors"
	} else {
		batch.Status = "completed"
	}

	// Commit matching transactions ONLY after successful result and error file generation.
	err = tx.Commit(ctx)
	if err != nil {
		s.failBatch(ctx, batch, fmt.Sprintf("Failed to commit database matches: %v", err))
		return
	}

	err = s.repo.UpdateBatch(ctx, batch)
	if err != nil {
		log.Printf("Failed to update final batch status for batch %d: %v", batch.ID, err)
	}

	log.Printf("Completed 购物金 matching for batch %d. Total: %d, Success: %d, Errors: %d", batch.ID, totalRows, successRows, errorRows)
}

// Helpers to write logs and failed items
func (s *Service) failBatch(ctx context.Context, batch *FileBatch, errStr string) {
	log.Printf("Batch %d failed: %s", batch.ID, errStr)
	batch.Status = "failed"
	// Save error to file name or structure if needed
	_ = s.repo.UpdateBatch(ctx, batch)
}

func (s *Service) writeErrorRow(f *excelize.File, sheet string, rowIdx int, rowNo int, mapping *HeaderMapping, originalRow []string, errMsg string) {
	getValSafe := func(colIdx int) string {
		if colIdx >= 0 && colIdx < len(originalRow) {
			return originalRow[colIdx]
		}
		return ""
	}

	f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), rowNo)
	f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), getValSafe(mapping.RefundDateCol))
	f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), getValSafe(mapping.FBNameCol))
	f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), getValSafe(mapping.ReservedDateCol))
	f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), getValSafe(mapping.BankOrderDetailCol))

	deductedVal := ""
	if mapping.DeductedOrderCol >= 0 {
		deductedVal = getValSafe(mapping.DeductedOrderCol)
	}
	f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), deductedVal)
	f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), getValSafe(mapping.AmtCol))
	f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), getValSafe(mapping.AsstNameCol))

	// Bank logic: BankCol fallback to GroupCol
	bankVal := getValSafe(mapping.BankCol)
	if strings.TrimSpace(bankVal) == "" && mapping.GroupCol >= 0 {
		bankVal = getValSafe(mapping.GroupCol)
	}
	f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIdx), bankVal)

	f.SetCellValue(sheet, fmt.Sprintf("J%d", rowIdx), getValSafe(mapping.PaymentNoCol))
	f.SetCellValue(sheet, fmt.Sprintf("K%d", rowIdx), errMsg)
}

func (s *Service) writeMatchErrorRow(f *excelize.File, sheet string, rowIdx int, rowNo int, fbName string, reservedDate time.Time, bankDetail string, amt float64, errMsg string, matchCount int) {
	f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), rowNo)
	f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), fbName)
	f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), reservedDate.Format("2006-01-02"))
	f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), bankDetail)
	f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), amt)
	f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), errMsg)
	f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), matchCount)
}

func (s *Service) writeDiagnosticCandidates(f *excelize.File, sheet string, rowIdx *int, inputRowNo int, candidates []GroupReservedFund) {
	for _, c := range candidates {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", *rowIdx), inputRowNo)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", *rowIdx), c.RefundDate.Format("2006-01-02"))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", *rowIdx), c.FBName)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", *rowIdx), c.ReservedFundsDate.Format("2006-01-02"))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", *rowIdx), c.BankOrderDetail)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", *rowIdx), c.ARPaymentNo)

		deductedVal := "-"
		if c.DeductedOrderNo != nil {
			deductedVal = *c.DeductedOrderNo
		}
		f.SetCellValue(sheet, fmt.Sprintf("G%d", *rowIdx), deductedVal)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", *rowIdx), c.Amt)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", *rowIdx), c.AsstName)

		matchedAtVal := "-"
		if c.MatchedAt != nil {
			matchedAtVal = c.MatchedAt.Format("2006-01-02 15:04:05")
		}
		f.SetCellValue(sheet, fmt.Sprintf("J%d", *rowIdx), matchedAtVal)

		*rowIdx++
	}
}

func (s *Service) writeSheetLevelError(f *excelize.File, sheet string, errMsg string) {
	f.NewSheet(sheet)
	f.SetCellValue(sheet, "A1", "Sheet Error")
	f.SetCellValue(sheet, "B1", errMsg)
}
