package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/ports"
)

type IndexerService struct {
	noteRepo  ports.Repository
	indexPath string
}

func NewIndexerService(noteRepo ports.Repository, indexPath string) *IndexerService {
	return &IndexerService{
		noteRepo:  noteRepo,
		indexPath: indexPath,
	}
}

type ReindexRequest struct{}

type ReindexResponse struct {
	TotalNotes       int
	TotalConnections int
	Duration         string
}

// Regex patterns
var linkPattern = regexp.MustCompile(`\\(?:input|include|ref|cref|cite)\{([^}]+)\}`)
var assetPattern = regexp.MustCompile(`\\includegraphics(?:\[.*?\])?\{([^}]+)\}`)

func (s *IndexerService) Execute(ctx context.Context, req ReindexRequest) (*ReindexResponse, error) {
	index := domain.NewIndex()

	headers, err := s.noteRepo.ListHeaders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}

	for _, header := range headers {
		note, err := s.noteRepo.Get(ctx, header.Slug)
		if err != nil {
			continue
		}

		outgoingLinks := s.extractLinks(note.Content, header.Slug)
		assets := s.extractAssets(note.Content)

		entry := domain.IndexEntry{
			Title:         header.Title,
			Date:          header.Date,
			Tags:          header.Tags,
			Filename:      header.Filename,
			OutgoingLinks: outgoingLinks,
			Backlinks:     []string{},
			Assets:        assets, // <--- Captured here
		}

		index.AddNote(header.Slug, entry)
	}

	s.calculateBacklinks(index)
	index.UpdateLastIndexed()

	if err := s.saveIndex(index); err != nil {
		return nil, fmt.Errorf("failed to save index: %w", err)
	}

	return &ReindexResponse{
		TotalNotes:       index.Count(),
		TotalConnections: index.CountConnections(),
	}, nil
}

func (s *IndexerService) extractLinks(content string, sourceSlug string) []string {
	matches := linkPattern.FindAllStringSubmatch(content, -1)
	linkMap := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			target := match[1]
			slug := s.normalizeLink(target)
			if slug != "" && slug != sourceSlug {
				linkMap[slug] = true
			}
		}
	}

	var links []string
	for link := range linkMap {
		links = append(links, link)
	}
	return links
}

// extractAssets scans for \includegraphics{filename}
func (s *IndexerService) extractAssets(content string) []string {
	matches := assetPattern.FindAllStringSubmatch(content, -1)
	assetMap := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			// Normalize path separators
			asset := filepath.ToSlash(match[1])
			assetMap[asset] = true
		}
	}

	var assets []string
	for asset := range assetMap {
		assets = append(assets, asset)
	}
	return assets
}

func (s *IndexerService) normalizeLink(link string) string {
	link = strings.ReplaceAll(link, "\\", "/")
	link = filepath.Base(link)
	link = strings.TrimSuffix(link, ".tex")

	if len(link) > 9 && link[8] == '-' {
		isDate := true
		for i := 0; i < 8; i++ {
			if link[i] < '0' || link[i] > '9' {
				isDate = false
				break
			}
		}
		if isDate {
			link = link[9:]
		}
	}
	return strings.TrimSpace(link)
}

func (s *IndexerService) calculateBacklinks(index *domain.Index) {
	for slug, entry := range index.Notes {
		entry.Backlinks = []string{}
		index.AddNote(slug, entry)
	}

	for sourceSlug, entry := range index.Notes {
		for _, targetSlug := range entry.OutgoingLinks {
			if target, exists := index.GetNote(targetSlug); exists {
				target.Backlinks = append(target.Backlinks, sourceSlug)
				index.AddNote(targetSlug, target)
			}
		}
	}
}

func (s *IndexerService) saveIndex(index *domain.Index) error {
	dir := filepath.Dir(s.indexPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	return os.WriteFile(s.indexPath, data, 0644)
}

func (s *IndexerService) LoadIndex() (*domain.Index, error) {
	data, err := os.ReadFile(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.NewIndex(), nil
		}
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var index domain.Index
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return &index, nil
}

func (s *IndexerService) IndexExists() bool {
	_, err := os.Stat(s.indexPath)
	return err == nil
}
