package services

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"lx/internal/core/domain"
	"lx/internal/core/ports"
)

type GraphService struct {
	noteRepo  ports.Repository
	vaultRoot string
	cachePath string
}

func NewGraphService(noteRepo ports.Repository, vaultRoot string) *GraphService {
	return &GraphService{
		noteRepo:  noteRepo,
		vaultRoot: vaultRoot,
		cachePath: filepath.Join(vaultRoot, "cache", "graph_cache.json"),
	}
}

// Data Structures
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Links []GraphLink `json:"links"`
}

type GraphNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Group int    `json:"group"`
	Value int    `json:"val"`
}

type GraphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Value  int    `json:"value"`
}

// GetGraph retrieves the graph, using cache if available
func (s *GraphService) GetGraph(ctx context.Context, forceRefresh bool) (GraphData, error) {
	// 1. Try Cache
	if !forceRefresh {
		if info, err := os.Stat(s.cachePath); err == nil {
			// Cache valid for 24 hours
			if time.Since(info.ModTime()) < 24*time.Hour {
				data, err := s.loadFromCache()
				if err == nil {
					return data, nil
				}
			}
		}
	}

	// 2. Generate Fresh
	data, err := s.Generate(ctx)
	if err != nil {
		return GraphData{}, err
	}

	// 3. Save Cache (Async)
	go s.saveToCache(data)

	return data, nil
}

// Generate builds the graph using O(N) Tokenization
func (s *GraphService) Generate(ctx context.Context) (GraphData, error) {
	// 1. Get Headers & Build Lookup Map
	headers, err := s.noteRepo.ListHeaders(ctx)
	if err != nil {
		return GraphData{}, err
	}

	nodes := make([]GraphNode, 0, len(headers))
	slugMap := make(map[string]bool)

	for _, h := range headers {
		slugMap[h.Slug] = true
		nodes = append(nodes, GraphNode{
			ID:    h.Slug,
			Title: h.Title,
			Group: 1,
			Value: 1,
		})
	}

	// 2. Scan Files Concurrently
	var links []GraphLink
	var mu sync.Mutex // Protects links slice
	var wg sync.WaitGroup

	// Semaphore to limit concurrency to CPU count (prevents I/O choking)
	semaphore := make(chan struct{}, runtime.NumCPU())

	for _, h := range headers {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire token

		go func(src domain.NoteHeader) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release token

			path := filepath.Join(s.vaultRoot, "notes", src.Filename)
			file, err := os.Open(path)
			if err != nil {
				return
			}
			defer file.Close()

			// --- THE OPTIMIZATION IS HERE ---
			// We scan the file once, tokenizing words, and checking the map.
			foundSlugs := s.findMentions(file, slugMap)

			if len(foundSlugs) > 0 {
				mu.Lock()
				for target := range foundSlugs {
					if target != src.Slug { // Ignore self-references
						links = append(links, GraphLink{
							Source: src.Slug,
							Target: target,
							Value:  1,
						})
					}
				}
				mu.Unlock()
			}
		}(h)
	}

	wg.Wait()

	return GraphData{
		Nodes: nodes,
		Links: links,
	}, nil
}

// findMentions tokenizes the stream and looks up slugs in O(1)
func (s *GraphService) findMentions(r io.Reader, validSlugs map[string]bool) map[string]bool {
	found := make(map[string]bool)
	scanner := bufio.NewScanner(r)

	// Custom Split Function:
	// Isolate "slug-like" tokens (alphanumeric + hyphens)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		start := 0
		// Skip non-slug characters
		for start < len(data) && !isSlugChar(rune(data[start])) {
			start++
		}
		if start >= len(data) {
			return start, nil, nil
		}

		// Find end of token
		end := start
		for end < len(data) && isSlugChar(rune(data[end])) {
			end++
		}

		if end < len(data) || atEOF {
			return end, data[start:end], nil
		}
		return 0, nil, nil
	})

	for scanner.Scan() {
		word := strings.ToLower(scanner.Text())
		// O(1) Lookup
		if validSlugs[word] {
			found[word] = true
		}
	}

	return found
}

// isSlugChar defines valid slug characters
func isSlugChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-'
}

func (s *GraphService) loadFromCache() (GraphData, error) {
	file, err := os.Open(s.cachePath)
	if err != nil {
		return GraphData{}, err
	}
	defer file.Close()
	var data GraphData
	err = json.NewDecoder(file).Decode(&data)
	return data, err
}

func (s *GraphService) saveToCache(data GraphData) {
	file, err := os.Create(s.cachePath)
	if err != nil {
		return
	}
	defer file.Close()
	json.NewEncoder(file).Encode(data)
}
