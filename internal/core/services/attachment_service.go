package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/ports"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

type AttachmentService struct {
	vault     *vault.Vault
	assetRepo ports.AssetRepository
}

func NewAttachmentService(v *vault.Vault, repo ports.AssetRepository) *AttachmentService {
	return &AttachmentService{
		vault:     v,
		assetRepo: repo,
	}
}

// Store saves a file and its metadata
// Returns: filename, isDuplicate, error
func (s *AttachmentService) Store(ctx context.Context, srcPath string, name string, description string) (string, bool, error) {
	// 1. Open Source & Calculate Hash
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, srcFile); err != nil {
		return "", false, fmt.Errorf("failed to calculate hash: %w", err)
	}
	srcHash := hex.EncodeToString(hasher.Sum(nil))

	// 2. GLOBAL DEDUPLICATION CHECK
	// Check if this content already exists anywhere in the vault
	if existing, err := s.assetRepo.GetByHash(ctx, srcHash); err == nil {
		// Verify file actually exists on disk to be safe
		existingPath := s.vault.GetAssetPath(existing.Filename)
		if _, err := os.Stat(existingPath); err == nil {
			// Found valid duplicate! Return existing filename.
			return existing.Filename, true, nil
		}
		// If record exists but file is missing, we proceed to re-save it
	}

	srcFile.Seek(0, 0) // Reset for copy

	// 3. Determine Target Filename
	ext := strings.ToLower(filepath.Ext(srcPath))
	baseName := domain.GenerateSlug(name)
	targetName := baseName + ext
	destPath := s.vault.GetAssetPath(targetName)

	// 4. Collision Resolution (Name collision, different content)
	counter := 1
	for {
		if info, err := os.Stat(destPath); err == nil && !info.IsDir() {
			// Name taken. We already checked hashes globally, so if we are here,
			// the content MUST be different. Rename.
			targetName = fmt.Sprintf("%s-%d%s", baseName, counter, ext)
			destPath = s.vault.GetAssetPath(targetName)
			counter++
			continue
		}
		break
	}

	// 5. Copy File
	dst, err := os.Create(destPath)
	if err != nil {
		return "", false, err
	}
	defer dst.Close()
	io.Copy(dst, srcFile)

	// 6. Save Metadata
	if err := s.saveMetadata(ctx, targetName, srcPath, description, srcHash); err != nil {
		fmt.Printf("Warning: failed to save asset metadata: %v\n", err)
	}

	return targetName, false, nil
}

func (s *AttachmentService) saveMetadata(ctx context.Context, filename, originalPath, description, hash string) error {
	asset := domain.Asset{
		Filename:     filename,
		OriginalName: filepath.Base(originalPath),
		Description:  description,
		Hash:         hash,
		UploadedAt:   time.Now(),
	}
	return s.assetRepo.Save(ctx, asset)
}
