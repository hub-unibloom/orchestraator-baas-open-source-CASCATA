package api

import (
	"encoding/json"
	"net/http"
	"path"

	"cascata/internal/auth"
	"cascata/internal/domain"
	"cascata/internal/storage"
)

// StorageHandler manages the direct and proxy upload life cycle.
type StorageHandler struct {
	svc *storage.Service
}

func NewStorageHandler(svc *storage.Service) *StorageHandler {
	return &StorageHandler{svc: svc}
}

// Upload handles the multipart/form-data upload via proxy.
func (h *StorageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx, ok := domain.FromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 1. Parse multipart form (Max 100MB buffer)
	const _100MB = 100 * 1024 * 1024
	if err := r.ParseMultipartForm(_100MB); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. Resolve Bucket Name from request/metadata (Dashboard fallback: 'default')
	bucket := r.URL.Query().Get("bucket")
	if bucket == "" { bucket = "default" }

	// 3. Security Check: Quota reserva
	if err := h.svc.CheckQuota(ctx, authCtx.ProjectSlug, header.Size); err != nil {
		http.Error(w, err.Error(), http.StatusInsufficientStorage)
		return
	}

	// 4. Perform Upload via Proxy
	relPath := r.URL.Query().Get("path") // e.g. avatars/
	fullRelPath := path.Join(relPath, header.Filename)

	err = h.svc.UploadLocal(ctx, authCtx.ProjectSlug, bucket, fullRelPath, file, header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"full_path": fullRelPath,
		"size":      header.Size,
	})
}
