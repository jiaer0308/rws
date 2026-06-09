package reserved_fund

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func setBatchFileAvailability(batch *FileBatch) {
	batch.ResultFileAvailable = batch.ResultFilePath != nil && strings.TrimSpace(*batch.ResultFilePath) != ""
	batch.ErrorReportAvailable = batch.ErrorReportPath != nil && strings.TrimSpace(*batch.ErrorReportPath) != ""
}

// CreateBatch inserts a new file batch
func (r *Repository) CreateBatch(ctx context.Context, batch *FileBatch) error {
	query := `
		INSERT INTO file_batches (
			batch_type, status, file_name, original_file_path, result_file_path, 
			error_report_path, total_rows, success_rows, error_rows, 
			created_by_user_id, created_by_name, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		batch.BatchType, batch.Status, batch.FileName, batch.OriginalFilePath, batch.ResultFilePath,
		batch.ErrorReportPath, batch.TotalRows, batch.SuccessRows, batch.ErrorRows,
		batch.CreatedByUserID, batch.CreatedByName,
	).Scan(&batch.ID, &batch.CreatedAt, &batch.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create file batch: %w", err)
	}
	setBatchFileAvailability(batch)
	return nil
}

// UpdateBatch updates an existing file batch
func (r *Repository) UpdateBatch(ctx context.Context, batch *FileBatch) error {
	query := `
		UPDATE file_batches
		SET status = $1, result_file_path = $2, error_report_path = $3, 
		    total_rows = $4, success_rows = $5, error_rows = $6, updated_at = NOW()
		WHERE id = $7
	`
	_, err := r.pool.Exec(ctx, query,
		batch.Status, batch.ResultFilePath, batch.ErrorReportPath,
		batch.TotalRows, batch.SuccessRows, batch.ErrorRows, batch.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update file batch: %w", err)
	}
	return nil
}

// GetBatch retrieves a file batch by ID
func (r *Repository) GetBatch(ctx context.Context, id int) (*FileBatch, error) {
	query := `
		SELECT id, batch_type, status, file_name, original_file_path, result_file_path, 
		       error_report_path, total_rows, success_rows, error_rows, 
		       created_by_user_id, created_by_name, created_at, updated_at
		FROM file_batches
		WHERE id = $1
	`
	var batch FileBatch
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&batch.ID, &batch.BatchType, &batch.Status, &batch.FileName, &batch.OriginalFilePath, &batch.ResultFilePath,
		&batch.ErrorReportPath, &batch.TotalRows, &batch.SuccessRows, &batch.ErrorRows,
		&batch.CreatedByUserID, &batch.CreatedByName, &batch.CreatedAt, &batch.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("batch %d not found", id)
		}
		return nil, fmt.Errorf("failed to query batch: %w", err)
	}
	setBatchFileAvailability(&batch)
	return &batch, nil
}

// ListBatches lists all file batches, ordered by newest first
func (r *Repository) ListBatches(ctx context.Context) ([]FileBatch, error) {
	query := `
		SELECT id, batch_type, status, file_name, original_file_path, result_file_path, 
		       error_report_path, total_rows, success_rows, error_rows, 
		       created_by_user_id, created_by_name, created_at, updated_at
		FROM file_batches
		ORDER BY created_at DESC
		LIMIT 100
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query batches: %w", err)
	}
	defer rows.Close()

	var batches []FileBatch
	for rows.Next() {
		var batch FileBatch
		err = rows.Scan(
			&batch.ID, &batch.BatchType, &batch.Status, &batch.FileName, &batch.OriginalFilePath, &batch.ResultFilePath,
			&batch.ErrorReportPath, &batch.TotalRows, &batch.SuccessRows, &batch.ErrorRows,
			&batch.CreatedByUserID, &batch.CreatedByName, &batch.CreatedAt, &batch.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan batch: %w", err)
		}
		setBatchFileAvailability(&batch)
		batches = append(batches, batch)
	}
	return batches, nil
}

// BeginTx starts a transaction
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.pool.Begin(ctx)
}

// GetReservedFundByARPaymentNo retrieves a reserved fund record by its unique payment number
func (r *Repository) GetReservedFundByARPaymentNo(ctx context.Context, arPaymentNo string) (*GroupReservedFund, error) {
	query := `
		SELECT id, refund_date, fb_name, bank, reserved_funds_date, bank_order_detail, 
		       ar_payment_no, deducted_order_no, amt, asst_name, 
		       source_import_batch_id, matched_export_batch_id, matched_at, created_at, updated_at
		FROM group_reserved_funds
		WHERE ar_payment_no = $1
	`
	var rf GroupReservedFund
	err := r.pool.QueryRow(ctx, query, arPaymentNo).Scan(
		&rf.ID, &rf.RefundDate, &rf.FBName, &rf.Bank, &rf.ReservedFundsDate, &rf.BankOrderDetail,
		&rf.ARPaymentNo, &rf.DeductedOrderNo, &rf.Amt, &rf.AsstName,
		&rf.SourceImportBatchID, &rf.MatchedExportBatchID, &rf.MatchedAt, &rf.CreatedAt, &rf.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query reserved fund: %w", err)
	}
	return &rf, nil
}

// UpsertReservedFund inserts or updates a reserved fund inside a transaction
func (r *Repository) UpsertReservedFund(ctx context.Context, tx pgx.Tx, rf *GroupReservedFund) error {
	query := `
		INSERT INTO group_reserved_funds (
			refund_date, fb_name, bank, reserved_funds_date, bank_order_detail, 
			ar_payment_no, deducted_order_no, amt, asst_name, 
			source_import_batch_id, matched_export_batch_id, matched_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		ON CONFLICT (ar_payment_no) DO UPDATE
		SET refund_date = CASE WHEN group_reserved_funds.matched_at IS NULL THEN EXCLUDED.refund_date ELSE group_reserved_funds.refund_date END,
		    fb_name = CASE WHEN group_reserved_funds.matched_at IS NULL THEN EXCLUDED.fb_name ELSE group_reserved_funds.fb_name END,
		    bank = EXCLUDED.bank,
		    reserved_funds_date = CASE WHEN group_reserved_funds.matched_at IS NULL THEN EXCLUDED.reserved_funds_date ELSE group_reserved_funds.reserved_funds_date END,
		    bank_order_detail = CASE WHEN group_reserved_funds.matched_at IS NULL THEN EXCLUDED.bank_order_detail ELSE group_reserved_funds.bank_order_detail END,
		    deducted_order_no = CASE WHEN group_reserved_funds.matched_at IS NULL THEN EXCLUDED.deducted_order_no ELSE group_reserved_funds.deducted_order_no END,
		    amt = CASE WHEN group_reserved_funds.matched_at IS NULL THEN EXCLUDED.amt ELSE group_reserved_funds.amt END,
		    asst_name = CASE WHEN group_reserved_funds.matched_at IS NULL THEN EXCLUDED.asst_name ELSE group_reserved_funds.asst_name END,
		    source_import_batch_id = EXCLUDED.source_import_batch_id,
		    updated_at = CASE
		        WHEN group_reserved_funds.matched_at IS NULL AND (
		            group_reserved_funds.refund_date IS DISTINCT FROM EXCLUDED.refund_date OR
		            group_reserved_funds.fb_name IS DISTINCT FROM EXCLUDED.fb_name OR
		            group_reserved_funds.bank IS DISTINCT FROM EXCLUDED.bank OR
		            group_reserved_funds.reserved_funds_date IS DISTINCT FROM EXCLUDED.reserved_funds_date OR
		            group_reserved_funds.bank_order_detail IS DISTINCT FROM EXCLUDED.bank_order_detail OR
		            group_reserved_funds.deducted_order_no IS DISTINCT FROM EXCLUDED.deducted_order_no OR
		            group_reserved_funds.amt IS DISTINCT FROM EXCLUDED.amt OR
		            group_reserved_funds.asst_name IS DISTINCT FROM EXCLUDED.asst_name OR
		            group_reserved_funds.source_import_batch_id IS DISTINCT FROM EXCLUDED.source_import_batch_id
		        ) THEN NOW()
		        WHEN group_reserved_funds.matched_at IS NOT NULL AND (
		            group_reserved_funds.bank IS DISTINCT FROM EXCLUDED.bank OR
		            group_reserved_funds.source_import_batch_id IS DISTINCT FROM EXCLUDED.source_import_batch_id
		        ) THEN NOW()
		        ELSE group_reserved_funds.updated_at
		    END
		RETURNING id, created_at, updated_at
	`
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query,
			rf.RefundDate, rf.FBName, rf.Bank, rf.ReservedFundsDate, rf.BankOrderDetail,
			rf.ARPaymentNo, rf.DeductedOrderNo, rf.Amt, rf.AsstName,
			rf.SourceImportBatchID, rf.MatchedExportBatchID, rf.MatchedAt,
		).Scan(&rf.ID, &rf.CreatedAt, &rf.UpdatedAt)
	} else {
		err = r.pool.QueryRow(ctx, query,
			rf.RefundDate, rf.FBName, rf.Bank, rf.ReservedFundsDate, rf.BankOrderDetail,
			rf.ARPaymentNo, rf.DeductedOrderNo, rf.Amt, rf.AsstName,
			rf.SourceImportBatchID, rf.MatchedExportBatchID, rf.MatchedAt,
		).Scan(&rf.ID, &rf.CreatedAt, &rf.UpdatedAt)
	}

	if err != nil {
		return fmt.Errorf("failed to upsert reserved fund: %w", err)
	}
	return nil
}

// GetCandidatesByFBName returns all group_reserved_funds rows matching the fb_name (TRIM and case-insensitive)
func (r *Repository) GetCandidatesByFBName(ctx context.Context, fbName string) ([]GroupReservedFund, error) {
	query := `
		SELECT id, refund_date, fb_name, bank, reserved_funds_date, bank_order_detail, 
		       ar_payment_no, deducted_order_no, amt, asst_name, 
		       source_import_batch_id, matched_export_batch_id, matched_at, created_at, updated_at
		FROM group_reserved_funds
		WHERE LOWER(TRIM(fb_name)) = LOWER(TRIM($1))
	`
	rows, err := r.pool.Query(ctx, query, fbName)
	if err != nil {
		return nil, fmt.Errorf("failed to query candidates by fb_name: %w", err)
	}
	defer rows.Close()

	var list []GroupReservedFund
	for rows.Next() {
		var rf GroupReservedFund
		err = rows.Scan(
			&rf.ID, &rf.RefundDate, &rf.FBName, &rf.Bank, &rf.ReservedFundsDate, &rf.BankOrderDetail,
			&rf.ARPaymentNo, &rf.DeductedOrderNo, &rf.Amt, &rf.AsstName,
			&rf.SourceImportBatchID, &rf.MatchedExportBatchID, &rf.MatchedAt, &rf.CreatedAt, &rf.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan candidate: %w", err)
		}
		list = append(list, rf)
	}
	return list, nil
}

// GetGroupTotal returns the sum of amounts of all database rows in the same group: fb_name + bank_order_detail + reserved_funds_date
func (r *Repository) GetGroupTotal(ctx context.Context, fbName, bankOrderDetail string, reservedFundsDate time.Time) (float64, error) {
	query := `
		SELECT COALESCE(SUM(amt), 0)
		FROM group_reserved_funds
		WHERE LOWER(TRIM(fb_name)) = LOWER(TRIM($1))
		  AND LOWER(TRIM(bank_order_detail)) = LOWER(TRIM($2))
		  AND reserved_funds_date = $3
	`
	var total float64
	err := r.pool.QueryRow(ctx, query, fbName, bankOrderDetail, reservedFundsDate).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate group total: %w", err)
	}
	return total, nil
}

// UpdateMatchedStatus marks a row as matched under a batch inside a transaction
func (r *Repository) UpdateMatchedStatus(ctx context.Context, tx pgx.Tx, id int, matchedBatchID int) error {
	query := `
		UPDATE group_reserved_funds
		SET matched_export_batch_id = $1,
		    matched_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(ctx, query, matchedBatchID, id)
	} else {
		_, err = r.pool.Exec(ctx, query, matchedBatchID, id)
	}
	if err != nil {
		return fmt.Errorf("failed to update matched status: %w", err)
	}
	return nil
}

// ListReservedFunds lists reserved funds with pagination
func (r *Repository) ListReservedFunds(ctx context.Context, search string, limit, offset int) ([]GroupReservedFund, int, error) {
	var whereClause string
	var args []interface{}
	argCount := 1

	if search != "" {
		whereClause = "WHERE LOWER(fb_name) LIKE $" + fmt.Sprint(argCount) +
			" OR LOWER(bank_order_detail) LIKE $" + fmt.Sprint(argCount) +
			" OR LOWER(ar_payment_no) LIKE $" + fmt.Sprint(argCount)
		args = append(args, "%"+strings.ToLower(search)+"%")
		argCount++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM group_reserved_funds %s", whereClause)
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query total count: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, refund_date, fb_name, bank, reserved_funds_date, bank_order_detail, 
		       ar_payment_no, deducted_order_no, amt, asst_name, 
		       source_import_batch_id, matched_export_batch_id, matched_at, created_at, updated_at
		FROM group_reserved_funds
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount, argCount+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query list: %w", err)
	}
	defer rows.Close()

	var list []GroupReservedFund
	for rows.Next() {
		var rf GroupReservedFund
		err = rows.Scan(
			&rf.ID, &rf.RefundDate, &rf.FBName, &rf.Bank, &rf.ReservedFundsDate, &rf.BankOrderDetail,
			&rf.ARPaymentNo, &rf.DeductedOrderNo, &rf.Amt, &rf.AsstName,
			&rf.SourceImportBatchID, &rf.MatchedExportBatchID, &rf.MatchedAt, &rf.CreatedAt, &rf.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan item: %w", err)
		}
		list = append(list, rf)
	}
	return list, total, nil
}

// GetSummaryStats fetches summary statistics
func (r *Repository) GetSummaryStats(ctx context.Context) (totalIssued, totalUsed, remainingPool float64, err error) {
	query := `
		SELECT 
			COALESCE(SUM(amt), 0) as total_issued,
			COALESCE(SUM(CASE WHEN matched_at IS NOT NULL THEN amt ELSE 0 END), 0) as total_used
		FROM group_reserved_funds
	`
	err = r.pool.QueryRow(ctx, query).Scan(&totalIssued, &totalUsed)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to fetch stats: %w", err)
	}
	remainingPool = totalIssued - totalUsed
	return totalIssued, totalUsed, remainingPool, nil
}
