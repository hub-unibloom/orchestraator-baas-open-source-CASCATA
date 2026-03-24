package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"cascata/internal/database"
	"cascata/internal/utils"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	indexer     *Indexer
	repo        *database.Repository
	dfly        *redis.Client // Phase 8.4 Quota High-Speed Control
	storageRoot string
}

func NewService(storageRoot string, dfly *redis.Client, repo *database.Repository, indexer *Indexer) *Service {
	return &Service{
		indexer:     indexer,
		repo:        repo,
		dfly:        dfly,
		storageRoot: storageRoot,
	}
}

// ValidateMagicBytes reads the start of a stream and checks if it matches any signature.
// Returns an error if signature mismatch occurs.
func (s *Service) ValidateMagicBytes(r io.Reader, expectedExt string) (io.Reader, error) {
	// Read first 8 bytes (enough for PK/PDF/PNG)
	header := make([]byte, 8)
	n, err := r.Read(header)
	if err != nil && err != io.EOF {
		return nil, err
	}

	// Signatures match against the head.
	head := string(header[:n])
	for signature, ext := range MagicNumbers {
		if strings.HasPrefix(head, signature) {
			// Found a signature. Does it match expectedExt?
			if !strings.EqualFold(ext, strings.TrimPrefix(expectedExt, ".")) {
				// Mismatch! e.g. .jpg file that starts with "PK\x03\x04" (Zip)
				return nil, fmt.Errorf("Security Alert: File signature mismatch. Expected %s, found %s", expectedExt, ext)
			}
			break
		}
	}

	// To avoid losing the read bytes, return a multi-reader.
	return io.MultiReader(strings.NewReader(head), r), nil
}

// CheckQuota validates if project has enough space for the new upload using Dragonfly reservation.
func (s *Service) CheckQuota(ctx context.Context, projectSlug string, newFileSize int64) error {
	usage, err := s.getPhysicalDiskUsage(projectSlug)
	if err != nil {
		slog.Warn("storage: disk usage check failed, falling back to db meta", "error", err)
	}

	// 1. Fetch quota limit from Dragonfly or DB (Phase 8.4)
	limitKey := fmt.Sprintf("cascata:limit:storage:%s", projectSlug)
	limitStr, err := s.dfly.Get(ctx, limitKey).Result()
	
	var limit int64 = 1 * 1024 * 1024 * 1024 // Default 1GB
	if err == nil {
		if val, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
			limit = val
		}
	}

	// 2. Perform Atomic Reservation in Dragonfly (Phantom Reservation)
	currentReservedKey := fmt.Sprintf("cascata:reserved:storage:%s", projectSlug)
	reserved, _ := s.dfly.Get(ctx, currentReservedKey).Int64()
	
	if usage+reserved+newFileSize > limit {
		return fmt.Errorf("Quota exceeded: project has %d bytes (reserved %d), limit is %d", usage, reserved, limit)
	}

	// Reserve for 10 minutes (Phase 8 timeout)
	return s.dfly.IncrBy(ctx, currentReservedKey, newFileSize).Err()
}

func (s *Service) releaseQuota(ctx context.Context, projectSlug string, fileSize int64) {
	currentReservedKey := fmt.Sprintf("cascata:reserved:storage:%s", projectSlug)
	_ = s.dfly.DecrBy(ctx, currentReservedKey, fileSize).Err()
}

func (s *Service) getPhysicalDiskUsage(projectSlug string) (int64, error) {
	target := filepath.Join(s.storageRoot, projectSlug)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		return 0, nil
	}

	// du -sb returns bytes.
	cmd := exec.Command("du", "-sb", target)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	parts := strings.Fields(string(out))
	if len(parts) == 0 {
		return 0, fmt.Errorf("du empty output")
	}

	return strconv.ParseInt(parts[0], 10, 64)
}

// IsSSRFSafe checks if a remote destination is not a local/private network.
func (s *Service) IsSSRFSafe(rawURL string) error {
	return utils.IsSSRFSafe(rawURL)
}

// UploadLocal saves the file stream to the physical storage root.
func (s *Service) UploadLocal(ctx context.Context, projectSlug, bucket, relPath string, r io.Reader, size int64, mimeType string) error {
	safePath := GetSafePath(path.Join(projectSlug, bucket, relPath))
	fullPath := filepath.Join(s.storageRoot, safePath)

	// Ensure parent dir exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	// Validate magic bytes against extension
	ext := filepath.Ext(relPath)
	sector := GetSectorForExt(ext)
	if sector == Exec {
		return fmt.Errorf("Security: Executable files are blocked by policy")
	}

	r, err := s.ValidateMagicBytes(r, ext)
	if err != nil {
		return err
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return err
	}

	// 4. Finalize Metadata Indexing (Phase 8 Sinergy)
	meta := IndexItem{
		Size:     size,
		MimeType: mimeType,
		IsFolder: false,
		Provider: "LOCAL",
	}

	if err := s.indexer.IndexObject(ctx, projectSlug, bucket, relPath, meta); err != nil {
		slog.Error("storage: failed to index object", "path", safePath, "err", err)
	}

	// 5. Release Quota Reservation
	s.releaseQuota(ctx, projectSlug, size)

	slog.Info("storage: upload successful", "slug", projectSlug, "path", safePath)
	return nil
}

// DeleteLocal removes a physical file and its system metadata.
func (s *Service) DeleteLocal(ctx context.Context, projectSlug, bucket, relPath string) error {
	safePath := GetSafePath(path.Join(projectSlug, bucket, relPath))
	fullPath := filepath.Join(s.storageRoot, safePath)

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("storage.DeleteLocal: %w", err)
	}

	if err := s.indexer.UnindexObject(ctx, projectSlug, bucket, relPath); err != nil {
		slog.Warn("storage: unindex failed during deletion", "path", relPath, "err", err)
	}

	return nil
}

// GetReader locates a physical file and returns an io.ReadCloser + the calculated full path.
func (s *Service) GetReader(ctx context.Context, projectSlug, bucket, relPath string) (io.ReadCloser, string, error) {
	safePath := GetSafePath(path.Join(projectSlug, bucket, relPath))
	fullPath := filepath.Join(s.storageRoot, safePath)

	f, err := os.Open(fullPath)
	if err != nil {
		return nil, "", err
	}

	return f, fullPath, nil
}
