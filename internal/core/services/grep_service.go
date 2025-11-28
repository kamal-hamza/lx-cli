package services

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// GrepService handles full-text search operations
type GrepService struct {
	vaultRoot string
}

// NewGrepService creates a new grep service
func NewGrepService(vaultRoot string) *GrepService {
	return &GrepService{
		vaultRoot: vaultRoot,
	}
}

// GrepMatch represents a single line match
type GrepMatch struct {
	Slug     string
	Filename string
	LineNum  int
	Content  string
}

// Execute scans all notes and returns matches
// If query is empty, returns all non-empty lines (for fuzzy finding)
func (s *GrepService) Execute(ctx context.Context, query string) ([]GrepMatch, error) {
	// 1. Identify search root
	notesPath := filepath.Join(s.vaultRoot, "notes")

	// 2. Collect files
	var files []string
	entries, err := os.ReadDir(notesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read notes directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tex") {
			files = append(files, filepath.Join(notesPath, entry.Name()))
		}
	}

	// 3. Worker Pool
	numWorkers := runtime.NumCPU()
	jobs := make(chan string, len(files))
	results := make(chan []GrepMatch, len(files))
	var wg sync.WaitGroup

	queryLower := strings.ToLower(query)
	searchAll := query == ""

	// Start Workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				matches := s.scanFile(path, queryLower, searchAll)
				if len(matches) > 0 {
					results <- matches
				}
			}
		}()
	}

	// Send jobs
	for _, f := range files {
		jobs <- f
	}
	close(jobs)

	// Close results when workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// 4. Collect results
	var allMatches []GrepMatch
	for fileMatches := range results {
		allMatches = append(allMatches, fileMatches...)
	}

	return allMatches, nil
}

func (s *GrepService) scanFile(path, query string, searchAll bool) []GrepMatch {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var matches []GrepMatch
	scanner := bufio.NewScanner(file)

	filename := filepath.Base(path)
	slug := strings.TrimSuffix(filename, ".tex")
	if strings.Contains(slug, "-") {
		parts := strings.SplitN(slug, "-", 2)
		if len(parts) == 2 {
			slug = parts[1]
		}
	}

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		text := scanner.Text()

		// Skip empty lines if searching all
		if strings.TrimSpace(text) == "" {
			continue
		}

		if searchAll || strings.Contains(strings.ToLower(text), query) {
			matches = append(matches, GrepMatch{
				Slug:     slug,
				Filename: filename,
				LineNum:  lineNum,
				Content:  text,
			})
		}
	}

	return matches
}
