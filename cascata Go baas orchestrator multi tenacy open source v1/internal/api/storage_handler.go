package api

import (
	"log/slog"
	"net/http"
	"path"
	"strconv"

	"cascata/internal/domain"
	"cascata/internal/storage"
)

// StorageHandler manages the direct and proxy upload life cycle.
// It acts as the "Cargo Bay" of the Cascata ship.
type StorageHandler struct {
	svc      *storage.Service
	optimizer *storage.MediaOptimizer
}

func NewStorageHandler(svc *storage.Service, opt *storage.MediaOptimizer) *StorageHandler {
	return &StorageHandler{svc: svc, optimizer: opt}
}

// Upload handles the multipart/form-data upload via proxy.
func (h *StorageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx, ok := domain.FromContext(ctx)
	if !ok {
		SendError(w, r, http.StatusUnauthorized, ErrUnauthorized, "Authentication required for storage access")
		return
	}

	// 1. Parse multipart form (Max 100MB buffer)
	const _100MB = 100 * 1024 * 1024
	if err := r.ParseMultipartForm(_100MB); err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Invalid multipart form size or encoding")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "File field 'file' is missing in multipart payload")
		return
	}
	defer file.Close()

	// 2. Resolve Bucket Name from request/metadata (Dashboard fallback: 'default')
	bucket := r.URL.Query().Get("bucket")
	if bucket == "" { bucket = "default" }

	// 3. Security Check: Quota reserva
	if err := h.svc.CheckQuota(ctx, authCtx.ProjectSlug, header.Size); err != nil {
		SendError(w, r, http.StatusInsufficientStorage, ErrRateLimit, "QUOTA_EXCEEDED", err.Error())
		return
	}

	// 4. Perform Upload via Proxy
	relPath := r.URL.Query().Get("path") // e.g. avatars/
	fullRelPath := path.Join(relPath, header.Filename)

	err = h.svc.UploadLocal(ctx, authCtx.ProjectSlug, bucket, fullRelPath, file, header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		slog.Error("storage: upload failed", "slug", authCtx.ProjectSlug, "error", err)
		SendError(w, r, http.StatusInternalServerError, ErrInternalError, "UPLOAD_FAILURE")
		return
	}

	SendJSON(w, r, http.StatusOK, map[string]interface{}{
		"full_path": fullRelPath,
		"size":      header.Size,
		"bucket":    bucket,
	})
}

// Download serves a physical file or an optimized image (Phase 18).
func (h *StorageHandler) Download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx, _ := domain.FromContext(ctx)
	
	bucket := r.URL.Query().Get("bucket")
	if bucket == "" { bucket = "default" }
	
	fpath := r.URL.Query().Get("path")
	if fpath == "" {
		SendError(w, r, http.StatusBadRequest, ErrInvalidRequest, "Path is mandatory")
		return
	}

	// 1. Fetch File Reader
	reader, fullPath, err := h.svc.GetReader(ctx, authCtx.ProjectSlug, bucket, fpath)
	if err != nil {
		SendError(w, r, http.StatusNotFound, ErrInvalidRequest, "FILE_NOT_FOUND")
		return
	}
	defer reader.Close()

	// 2. Optimization check (Phase 18 CDN-at-the-Edge)
	wStr := r.URL.Query().Get("w")
	hStr := r.URL.Query().Get("h")
	fmtStr := r.URL.Query().Get("fmt")

	if wStr != "" || hStr != "" || fmtStr != "" {
		// Proceed with Optimization
		width, _ := strconv.Atoi(wStr)
		height, _ := strconv.Atoi(hStr)
		
		optimizedBytes, mime, err := h.optimizer.Transform(reader, width, height, fmtStr)
		if err != nil {
			slog.Error("storage: optimization failed", "path", fullPath, "error", err)
			// Fallback to original
			http.ServeFile(w, r, fullPath)
			return
		}

		w.Header().Set("Content-Type", mime)
		w.Header().Set("Content-Length", strconv.Itoa(len(optimizedBytes)))
		w.Header().Set("X-Cascata-Optimized", "true")
		w.Write(optimizedBytes)
		return
	}

	// Default: Serve Original
	http.ServeFile(w, r, fullPath)
}
