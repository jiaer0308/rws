package reserved_fund

import (
	"time"
)

// FileBatch represents the file_batches table schema.
type FileBatch struct {
	ID                   int       `json:"id"`
	BatchType            string    `json:"batchType"`
	Status               string    `json:"status"`
	FileName             string    `json:"fileName"`
	OriginalFilePath     *string   `json:"originalFilePath"`
	ResultFilePath       *string   `json:"resultFilePath"`
	ErrorReportPath      *string   `json:"errorReportPath"`
	ResultFileAvailable  bool      `json:"resultFileAvailable"`
	ErrorReportAvailable bool      `json:"errorReportAvailable"`
	TotalRows            int       `json:"totalRows"`
	SuccessRows          int       `json:"successRows"`
	ErrorRows            int       `json:"errorRows"`
	CreatedByUserID      *int      `json:"createdByUserId"`
	CreatedByName        *string   `json:"createdByName"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// GroupReservedFund represents the group_reserved_funds table schema.
type GroupReservedFund struct {
	ID                   int        `json:"id"`
	RefundDate           time.Time  `json:"refundDate"`
	FBName               string     `json:"fbName"`
	Bank                 string     `json:"bank"`
	ReservedFundsDate    time.Time  `json:"reservedFundsDate"`
	BankOrderDetail      string     `json:"bankOrderDetail"`
	ARPaymentNo          string     `json:"arPaymentNo"`
	DeductedOrderNo      *string    `json:"deductedOrderNo"`
	Amt                  float64    `json:"amt"`
	AsstName             string     `json:"asstName"`
	SourceImportBatchID  *int       `json:"sourceImportBatchId"`
	MatchedExportBatchID *int       `json:"matchedExportBatchId"`
	MatchedAt            *time.Time `json:"matchedAt"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}
