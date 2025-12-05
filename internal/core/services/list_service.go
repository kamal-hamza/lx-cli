package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

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

	// Sort
	headers = s.sortHeaders(headers, req.SortBy, req.Reverse)

	return &ListResponse{
		Notes: headers,
		Total: len(headers),
	}, nil
}

func (s *ListService) filterByTag(headers []domain.NoteHeader, tag string) []domain.NoteHeader {
	var filtered []domain.NoteHeader
	for _, header := range headers {
		for _, t := range header.Tags {
			if strings.EqualFold(t, tag) {
				filtered = append(filtered, header)
				break
			}
		}
	}
	return filtered
}

func (s *ListService) sortHeaders(headers []domain.NoteHeader, sortBy string, reverse bool) []domain.NoteHeader {
	sort.Slice(headers, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "title":
			less = strings.ToLower(headers[i].Title) < strings.ToLower(headers[j].Title)
		default: // "date"
			less = headers[i].Date < headers[j].Date
		}
		if reverse {
			return !less
		}
		return less
	})
	return headers
}

// SearchRequest represents a search query
type SearchRequest struct {
	Query string
}

// SearchResponse represents search results
type SearchResponse struct {
	Notes []domain.NoteHeader
	Total int
}

// Search performs fuzzy search on notes
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

	// Filter by query with fuzzy matching
	matches := s.fuzzySearch(headers, req.Query)

	return &SearchResponse{
		Notes: matches,
		Total: len(matches),
	}, nil
}

// fuzzyMatch represents a scored match
type fuzzyMatch struct {
	header domain.NoteHeader
	score  int
}

// fuzzySearch performs fuzzy search on titles, slugs, and tags with scoring
func (s *ListService) fuzzySearch(headers []domain.NoteHeader, query string) []domain.NoteHeader {
	query = strings.TrimSpace(query)
	if query == "" {
		return headers
	}

	var matches []fuzzyMatch

	for _, header := range headers {
		// Try matching against title (highest priority)
		if score := fuzzyMatchScore(header.Title, query); score > 0 {
			matches = append(matches, fuzzyMatch{header: header, score: score + 1000}) // Bonus for title match
			continue
		}

		// Try matching against slug
		if score := fuzzyMatchScore(header.Slug, query); score > 0 {
			matches = append(matches, fuzzyMatch{header: header, score: score + 500}) // Medium bonus for slug
			continue
		}

		// Try matching against tags
		for _, tag := range header.Tags {
			if score := fuzzyMatchScore(tag, query); score > 0 {
				matches = append(matches, fuzzyMatch{header: header, score: score + 200}) // Small bonus for tag
				break
			}
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	// Extract headers
	result := make([]domain.NoteHeader, len(matches))
	for i, m := range matches {
		result[i] = m.header
	}

	return result
}

// fuzzyMatchScore calculates a score for fuzzy matching query against text
// Returns 0 if no match, higher scores for better matches
func fuzzyMatchScore(text, query string) int {
	if text == "" || query == "" {
		return 0
	}

	textLower := strings.ToLower(text)
	queryLower := strings.ToLower(query)

	// Exact match gets highest score
	if text == query {
		return 10000
	}

	// Case-insensitive exact match
	if textLower == queryLower {
		return 9000
	}

	// Substring match (contains)
	if strings.Contains(textLower, queryLower) {
		score := 5000
		// Bonus for match at start
		if strings.HasPrefix(textLower, queryLower) {
			score += 2000
		}
		return score
	}

	// Fuzzy character-by-character matching
	score := 0
	textRunes := []rune(textLower)
	queryRunes := []rune(queryLower)

	queryIdx := 0
	consecutiveMatches := 0
	lastMatchIdx := -1

	for textIdx := 0; textIdx < len(textRunes) && queryIdx < len(queryRunes); textIdx++ {
		if textRunes[textIdx] == queryRunes[queryIdx] {
			// Base score for each matched character
			score += 100

			// Bonus for consecutive matches
			if textIdx == lastMatchIdx+1 {
				consecutiveMatches++
				score += consecutiveMatches * 50 // Increasing bonus for consecutive chars
			} else {
				consecutiveMatches = 0
			}

			// Bonus for matching at word boundary
			if textIdx == 0 || unicode.IsSpace(textRunes[textIdx-1]) || textRunes[textIdx-1] == '-' || textRunes[textIdx-1] == '_' {
				score += 200
			}

			// Bonus for matching at start of string
			if textIdx == 0 {
				score += 300
			}

			lastMatchIdx = textIdx
			queryIdx++
		}
	}

	// All query characters must be matched
	if queryIdx != len(queryRunes) {
		return 0
	}

	// Penalty for gaps between matches
	if lastMatchIdx >= 0 {
		matchSpan := lastMatchIdx + 1
		penalty := (matchSpan - len(queryRunes)) * 10
		score -= penalty
	}

	return score
}
