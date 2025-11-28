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

	"lx/pkg/vault"
)

type AttachmentService struct {
	vault *vault.Vault
}

func NewAttachmentService(v *vault.Vault) *AttachmentService {
	return &AttachmentService{vault: v}
}

// Store saves a file to the vault's assets directory using content-addressable naming
func (s *AttachmentService) Store(ctx context.Context, srcPath string) (string, error) {
	// 1. Open Source File
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// 2. Calculate Hash (SHA-256)
	hasher := sha256.New()
	if _, err := io.Copy(hasher, src); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}
	hash := hex.EncodeToString(hasher.Sum(nil))

	// 3. Determine Extension and Destination
	ext := strings.ToLower(filepath.Ext(srcPath))
	// Use first 12 chars of hash for filename (plenty for uniqueness in personal vault)
	filename := hash[:12] + ext
	destPath := s.vault.GetAssetPath(filename)

	// 4. Check Deduplication
	if _, err := os.Stat(destPath); err == nil {
		// File already exists with same content (hash match)
		return filename, nil
	}

	// 5. Copy File (Since we read it for hashing, we need to reset or reopen)
	// Reopening is safer/simpler than seeking
	src.Close()
	src, err = os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create asset file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return filename, nil
}
