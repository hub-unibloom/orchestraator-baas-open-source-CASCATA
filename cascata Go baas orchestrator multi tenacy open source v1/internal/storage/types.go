package storage

import "path/filepath"

// ProviderType defines the supported storage backends.
type ProviderType string

const (
	Local      ProviderType = "local"
	S3         ProviderType = "s3"
	Cloudinary ProviderType = "cloudinary"
	GDrive     ProviderType = "gdrive"
)

// Sector represents the classification of files for governance.
type Sector string

const (
	Visual   Sector = "visual"   // Images
	Motion   Sector = "motion"   // Videos
	Docs     Sector = "docs"     // PDF, Word, etc.
	Archives Sector = "archives" // Zip, Tar
	Exec     Sector = "exec"     // Binary/Scripts (Blocked by default)
	Other    Sector = "other"
)

// MagicNumbers maps file signatures to their expected extensions.
var MagicNumbers = map[string]string{
	"\xff\xd8\xff":         "jpg",
	"\x89PNG\r\n\x1a\n":    "png",
	"GIF87a":               "gif",
	"GIF89a":               "gif",
	"%PDF-":                "pdf",
	"PK\x03\x04":          "zip", // Also Office Open XML (docx, xlsx)
	"\x1f\x8b\x08":         "gz",
	"fLaC":                 "flac",
	"ID3":                  "mp3",
	"\x00\x00\x00\x18ftyp": "mp4",
}

// GetSectorForExt classifies an extension into a sector.
func GetSectorForExt(ext string) Sector {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg":
		return Visual
	case ".mp4", ".mov", ".avi", ".webm", ".mkv":
		return Motion
	case ".pdf", ".doc", ".docx", ".txt", ".csv", ".xlsx":
		return Docs
	case ".zip", ".tar", ".gz", ".7z", ".rar":
		return Archives
	case ".exe", ".sh", ".bat", ".bin", ".py", ".js":
		return Exec
	default:
		return Other
	}
}

// GetSafePath prevents path traversal attacks.
func GetSafePath(path string) string {
	// Clean the path and remove any relative components
	clean := filepath.Clean("/" + path)
	return filepath.ToSlash(clean[1:])
}
