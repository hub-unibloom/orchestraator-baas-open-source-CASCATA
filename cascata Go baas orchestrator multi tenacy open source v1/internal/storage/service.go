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
)

	"cascata/internal/database"
	"cascata/internal/utils"
)

// Service handles multi-tenant storage orchestration and security.
type Service struct {
	indexer    *Indexer
	repo       *database.Repository
	storageRoot string
}

func NewService(indexer *Indexer, repo *database.Repository, storageRoot string) *Service {
	return &Service{
		indexer:    indexer,
		repo:       repo,
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

// CheckQuota validates if project has enough space for the new upload.
func (s *Service) CheckQuota(ctx context.Context, projectSlug string, newFileSize int64) error {
	usage, err := s.getPhysicalDiskUsage(projectSlug)
	if err != nil {
		slog.Warn("storage: disk usage check failed, falling back to db meta", "error", err)
	}

	// TODO: Fetch project quota limit from metadata DB.
	var limit int64 = 1 * 1024 * 1024 * 1024 // Default 1GB
	
	if usage+newFileSize > limit {
		return fmt.Errorf("Quota exceeded: project has %d bytes, limit is %d", usage, limit)
	}
	return nil
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

	// Index in DB.
	return s.indexer.IndexObject(ctx, projectSlug, bucket, relPath, IndexItem{
		Size:     size,
		MimeType: mimeType,
		IsFolder: false,
		Provider: "local",
	})
}
