package repository

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

type FileAssetRepository struct {
	vault        *vault.Vault
	manifestPath string
	mu           sync.RWMutex
	cache        map[string]domain.Asset
}

func NewFileAssetRepository(v *vault.Vault) *FileAssetRepository {
	return &FileAssetRepository{
		vault:        v,
		manifestPath: filepath.Join(v.AssetsPath, ".manifest.json"),
		cache:        make(map[string]domain.Asset),
	}
}

// Load reads the manifest from disk
func (r *FileAssetRepository) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &r.cache)
}

// Save persists an asset to the manifest
func (r *FileAssetRepository) Save(ctx context.Context, asset domain.Asset) error {
	// Ensure cache is loaded
	if len(r.cache) == 0 {
		r.Load()
	}

	r.mu.Lock()
	r.cache[asset.Filename] = asset
	r.mu.Unlock()

	return r.flush()
}

// flush writes cache to disk
func (r *FileAssetRepository) flush() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.MarshalIndent(r.cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.manifestPath, data, 0644)
}

func (r *FileAssetRepository) Get(ctx context.Context, filename string) (*domain.Asset, error) {
	if len(r.cache) == 0 {
		r.Load()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	asset, ok := r.cache[filename]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &asset, nil
}

func (r *FileAssetRepository) Search(ctx context.Context, query string) ([]domain.Asset, error) {
	if len(r.cache) == 0 {
		r.Load()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	var matches []domain.Asset

	for _, asset := range r.cache {
		if strings.Contains(strings.ToLower(asset.Filename), query) ||
			strings.Contains(strings.ToLower(asset.OriginalName), query) ||
			strings.Contains(strings.ToLower(asset.Description), query) {
			matches = append(matches, asset)
		}
	}

	return matches, nil
}
