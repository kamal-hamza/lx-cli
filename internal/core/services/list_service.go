package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/ports"
)

// ListService handles listing and filtering notes
type ListService struct {
	noteRepo ports.Repository
}

// NewListService creates a new list service
func NewListService(noteRepo ports.Repository) *ListService {
	return &ListService{
		noteRepo: noteRepo,
	}
}

// ListRequest represents a request to list notes
type ListRequest struct {
	TagFilter string // Filter by specific tag (optional)
	SortBy    string // "date", "title" (default: date)
	Reverse   bool   // Reverse sort order
}

// ListResponse represents the response from listing notes
type ListResponse struct {
	Notes []domain.NoteHeader
	Total int
}

// Execute lists notes with optional filtering and sorting
func (s *ListService) Execute(ctx context.Context, req ListRequest) (*ListResponse, error) {
	// Get all note headers
	headers, err := s.noteRepo.ListHeaders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}

	// Apply tag filter if specified
	if req.TagFilter != "" {
		headers = s.filterByTag(headers, req.TagFilter)
	}

	// Sort the results
	s.sortHeaders(headers, req.SortBy, req.Reverse)

	return &ListResponse{
		Notes: headers,
		Total: len(headers),
	}, nil
}

// filterByTag filters notes by tag
func (s *ListService) filterByTag(headers []domain.NoteHeader, tag string) []domain.NoteHeader {
	var filtered []domain.NoteHeader

	tag = strings.ToLower(strings.TrimSpace(tag))

	for _, header := range headers {
		if header.HasTag(tag) {
			filtered = append(filtered, header)
		}
	}

	return filtered
}

// sortHeaders sorts notes by the specified field
func (s *ListService) sortHeaders(headers []domain.NoteHeader, sortBy string, reverse bool) {
	switch sortBy {
	case "title":
		sort.Slice(headers, func(i, j int) bool {
			if reverse {
				return strings.ToLower(headers[i].Title) > strings.ToLower(headers[j].Title)
			}
			return strings.ToLower(headers[i].Title) < strings.ToLower(headers[j].Title)
		})
	case "date":
		fallthrough
	default:
		sort.Slice(headers, func(i, j int) bool {
			if reverse {
				return headers[i].Date > headers[j].Date
			}
			return headers[i].Date < headers[j].Date
		})
	}
}

// SearchRequest represents a request to search notes
type SearchRequest struct {
	Query string
}

// SearchResponse represents the response from searching notes
type SearchResponse struct {
	Notes []domain.NoteHeader
	Total int
}

// Search performs a fuzzy search on notes
func (s *ListService) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	// Get all note headers
	headers, err := s.noteRepo.ListHeaders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}

	// If no query, return all
	if strings.TrimSpace(req.Query) == "" {
		return &SearchResponse{
			Notes: headers,
			Total: len(headers),
		}, nil
	}

	// Filter by query
	matches := s.fuzzySearch(headers, req.Query)

	return &SearchResponse{
		Notes: matches,
		Total: len(matches),
	}, nil
}

// fuzzySearch performs a simple fuzzy search on titles, slugs, and tags
func (s *ListService) fuzzySearch(headers []domain.NoteHeader, query string) []domain.NoteHeader {
	var matches []domain.NoteHeader
	query = strings.ToLower(strings.TrimSpace(query))

	for _, header := range headers {
		// Check title match
		if strings.Contains(strings.ToLower(header.Title), query) {
			matches = append(matches, header)
			continue
		}

		// Check slug match
		if strings.Contains(strings.ToLower(header.Slug), query) {
			matches = append(matches, header)
			continue
		}

		// Check tag match
		for _, tag := range header.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				matches = append(matches, header)
				break
			}
		}
	}

	return matches
}
