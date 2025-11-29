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
func (s *AttachmentService) Store(ctx context.Context, srcPath string, name string, description string) (string, error) {
	// 1. Open Source & Calculate Hash
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, srcFile); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}
	srcHash := hex.EncodeToString(hasher.Sum(nil))

	srcFile.Seek(0, 0) // Reset for copy

	// 2. Determine Target Filename
	ext := strings.ToLower(filepath.Ext(srcPath))
	baseName := domain.GenerateSlug(name)
	targetName := baseName + ext
	destPath := s.vault.GetAssetPath(targetName)

	// 3. Collision Resolution
	counter := 1
	for {
		if info, err := os.Stat(destPath); err == nil && !info.IsDir() {
			if matches, _ := s.checkHashMatch(destPath, srcHash); matches {
				// EXISTING MATCH: Content is identical, reuse existing file.
				// We update the metadata to the latest description provided.
				s.saveMetadata(ctx, filepath.Base(destPath), srcPath, description, srcHash)
				return filepath.Base(destPath), nil
			}
			// Name collision but different content -> Rename (graph-1.png)
			targetName = fmt.Sprintf("%s-%d%s", baseName, counter, ext)
			destPath = s.vault.GetAssetPath(targetName)
			counter++
			continue
		}
		break
	}

	// 4. Copy File
	dst, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	io.Copy(dst, srcFile)

	// 5. Save Metadata
	if err := s.saveMetadata(ctx, targetName, srcPath, description, srcHash); err != nil {
		fmt.Printf("Warning: failed to save asset metadata: %v\n", err)
	}

	return targetName, nil
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

func (s *AttachmentService) checkHashMatch(path string, expectedHash string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	hasher := sha256.New()
	io.Copy(hasher, file)
	return hex.EncodeToString(hasher.Sum(nil)) == expectedHash, nil
}
