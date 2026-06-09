package reserved_fund

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
	repo    *Repository
}

func NewHandler(service *Service, repo *Repository) *Handler {
	return &Handler{
		service: service,
		repo:    repo,
	}
}

// Helper to write JSON responses
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

// Helper to write JSON errors
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// RegisterRoutes sets up all routes for reserved funds under the provided router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/group-reserved-funds/import-470", h.Import470)
	r.Post("/api/reserved-fund-usage/export", h.ExportReservedFunds)
	r.Get("/api/batches", h.ListBatches)
	r.Get("/api/batches/{id}", h.GetBatch)
	r.Get("/api/batches/{id}/download/{fileType}", h.DownloadBatchFile)
	
	r.Get("/api/group-reserved-funds", h.ListReservedFunds)
	r.Get("/api/group-reserved-funds/stats", h.GetSummaryStats)
}

// Import470 handles uploading the 470 excel sheet
func (h *Handler) Import470(w http.ResponseWriter, r *http.Request) {
	// Limit file size to 15MB
	r.ParseMultipartForm(15 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse uploaded file: "+err.Error())
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".xlsx") {
		respondWithError(w, http.StatusBadRequest, "Invalid file format. Only .xlsx files are supported.")
		return
	}

	// Ensure uploads directory exists
	uploadsDir := "./uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create uploads directory")
		return
	}

	// Create a new FileBatch record
	batch := &FileBatch{
		BatchType: "import_470",
		Status:    "processing",
		FileName:  header.Filename,
	}

	err = h.repo.CreateBatch(r.Context(), batch)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create batch record: "+err.Error())
		return
	}

	// Save original file to disk named by batch ID
	savePath := filepath.Join(uploadsDir, fmt.Sprintf("original_470_%d.xlsx", batch.ID))
	out, err := os.Create(savePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save file: "+err.Error())
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to write file contents: "+err.Error())
		return
	}

	batch.OriginalFilePath = &savePath
	err = h.repo.UpdateBatch(r.Context(), batch)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update batch file path: "+err.Error())
		return
	}

	// Run processing asynchronously in goroutine
	go h.service.Import470Excel(context.Background(), batch)

	respondWithJSON(w, http.StatusAccepted, batch)
}

// ExportReservedFunds handles matching the 购物金 list and writing Columns E-J
func (h *Handler) ExportReservedFunds(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(15 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse uploaded file: "+err.Error())
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".xlsx") {
		respondWithError(w, http.StatusBadRequest, "Invalid file format. Only .xlsx files are supported.")
		return
	}

	uploadsDir := "./uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create uploads directory")
		return
	}

	batch := &FileBatch{
		BatchType: "reserved_fund_usage_export",
		Status:    "processing",
		FileName:  header.Filename,
	}

	err = h.repo.CreateBatch(r.Context(), batch)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create batch record: "+err.Error())
		return
	}

	savePath := filepath.Join(uploadsDir, fmt.Sprintf("original_export_%d.xlsx", batch.ID))
	out, err := os.Create(savePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save file: "+err.Error())
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to write file contents: "+err.Error())
		return
	}

	batch.OriginalFilePath = &savePath
	err = h.repo.UpdateBatch(r.Context(), batch)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update batch file path: "+err.Error())
		return
	}

	// Run matching asynchronously
	go h.service.MatchReservedFundsExcel(context.Background(), batch)

	respondWithJSON(w, http.StatusAccepted, batch)
}

// ListBatches lists recent batch operations
func (h *Handler) ListBatches(w http.ResponseWriter, r *http.Request) {
	batches, err := h.repo.ListBatches(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve batches: "+err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, batches)
}

// GetBatch retrieves a single batch details
func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	batch, err := h.repo.GetBatch(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, batch)
}

// DownloadBatchFile serves downloads for result and error reports
func (h *Handler) DownloadBatchFile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid batch ID")
		return
	}

	fileType := chi.URLParam(r, "fileType") // "original", "result", "error"

	batch, err := h.repo.GetBatch(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	var filePath *string
	var downloadName string

	switch fileType {
	case "original":
		filePath = batch.OriginalFilePath
		downloadName = "original_" + batch.FileName
	case "result":
		filePath = batch.ResultFilePath
		downloadName = "matched_" + batch.FileName
	case "error":
		filePath = batch.ErrorReportPath
		if batch.BatchType == "import_470" {
			downloadName = "import_errors_" + batch.FileName
		} else {
			downloadName = "matching_errors_" + batch.FileName
		}
	default:
		respondWithError(w, http.StatusBadRequest, "Invalid file download type")
		return
	}

	if filePath == nil || *filePath == "" {
		respondWithError(w, http.StatusNotFound, "Requested file is not available")
		return
	}

	// Verify file exists on disk
	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		respondWithError(w, http.StatusNotFound, "File does not exist on disk")
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", downloadName))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	http.ServeFile(w, r, *filePath)
}

// ListReservedFunds handles querying and pagination of the main reserved funds table
func (h *Handler) ListReservedFunds(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 15
	offset := 0

	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}
	if offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	list, total, err := h.repo.ListReservedFunds(r.Context(), search, limit, offset)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve records: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"records": list,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetSummaryStats retrieves metrics for the dashboard summary cards
func (h *Handler) GetSummaryStats(w http.ResponseWriter, r *http.Request) {
	issued, used, remaining, err := h.repo.GetSummaryStats(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve statistics: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"totalIssued":   issued,
		"totalUsed":     used,
		"remainingPool": remaining,
	})
}
