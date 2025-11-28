package services

import (
	"context"
	"fmt"
	"sync"

	"lx/internal/core/domain"
	"lx/internal/core/ports"
)

// BuildService handles LaTeX compilation operations
type BuildService struct {
	noteRepo ports.Repository
	compiler ports.Compiler
}

// NewBuildService creates a new build service
func NewBuildService(noteRepo ports.Repository, compiler ports.Compiler) *BuildService {
	return &BuildService{
		noteRepo: noteRepo,
		compiler: compiler,
	}
}

// BuildRequest represents a request to build a note
type BuildRequest struct {
	Slug string
}

// BuildResponse represents the response from building a note
type BuildResponse struct {
	Slug       string
	OutputPath string
	Success    bool
	Error      error
}

// BuildAllRequest represents a request to build all notes
type BuildAllRequest struct {
	MaxWorkers int // Number of concurrent workers
}

// BuildAllResponse represents the response from building all notes
type BuildAllResponse struct {
	Total     int
	Succeeded int
	Failed    int
	Results   []BuildResponse
}

// Execute builds a single note
func (s *BuildService) Execute(ctx context.Context, req BuildRequest) (*BuildResponse, error) {
	// Check if note exists
	if !s.noteRepo.Exists(ctx, req.Slug) {
		return nil, fmt.Errorf("note not found: %s", req.Slug)
	}

	// Compile the note
	err := s.compiler.Compile(ctx, req.Slug, []string{})
	if err != nil {
		return &BuildResponse{
			Slug:    req.Slug,
			Success: false,
			Error:   err,
		}, err
	}

	outputPath := s.compiler.GetOutputPath(req.Slug)

	return &BuildResponse{
		Slug:       req.Slug,
		OutputPath: outputPath,
		Success:    true,
		Error:      nil,
	}, nil
}

// ExecuteAll builds all notes concurrently
func (s *BuildService) ExecuteAll(ctx context.Context, req BuildAllRequest) (*BuildAllResponse, error) {
	// Get all notes
	headers, err := s.noteRepo.ListHeaders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}

	if len(headers) == 0 {
		return &BuildAllResponse{
			Total:     0,
			Succeeded: 0,
			Failed:    0,
			Results:   []BuildResponse{},
		}, nil
	}

	// Default to 4 workers if not specified
	maxWorkers := req.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 4
	}

	// Create worker pool
	results := s.buildConcurrently(ctx, headers, maxWorkers)

	// Aggregate results
	response := &BuildAllResponse{
		Total:   len(headers),
		Results: results,
	}

	for _, result := range results {
		if result.Success {
			response.Succeeded++
		} else {
			response.Failed++
		}
	}

	return response, nil
}

// buildConcurrently builds notes using a worker pool
func (s *BuildService) buildConcurrently(ctx context.Context, headers []domain.NoteHeader, maxWorkers int) []BuildResponse {
	// Create channels for work distribution
	jobs := make(chan domain.NoteHeader, len(headers))
	results := make(chan BuildResponse, len(headers))

	// Start workers
	var wg sync.WaitGroup
	for range maxWorkers {
		wg.Go(func() {
			s.worker(ctx, jobs, results)
		})
	}

	// Send jobs
	for _, header := range headers {
		jobs <- header
	}
	close(jobs)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var buildResults []BuildResponse
	for result := range results {
		buildResults = append(buildResults, result)
	}

	return buildResults
}

// worker is a worker goroutine that processes build jobs
func (s *BuildService) worker(ctx context.Context, jobs <-chan domain.NoteHeader, results chan<- BuildResponse) {
	for header := range jobs {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			results <- BuildResponse{
				Slug:    header.Slug,
				Success: false,
				Error:   ctx.Err(),
			}
			continue
		default:
		}

		// Compile the note
		err := s.compiler.Compile(ctx, header.Slug, []string{})

		result := BuildResponse{
			Slug: header.Slug,
		}

		if err != nil {
			result.Success = false
			result.Error = err
		} else {
			result.Success = true
			result.OutputPath = s.compiler.GetOutputPath(header.Slug)
		}

		results <- result
	}
}

// BuildProgress represents the progress of a build operation
type BuildProgress struct {
	Current int
	Total   int
	Slug    string
	Success bool
	Error   error
}

// ExecuteAllWithProgress builds all notes and reports progress
func (s *BuildService) ExecuteAllWithProgress(ctx context.Context, req BuildAllRequest, progressChan chan<- BuildProgress) (*BuildAllResponse, error) {
	defer close(progressChan)

	// Get all notes
	headers, err := s.noteRepo.ListHeaders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list notes: %w", err)
	}

	if len(headers) == 0 {
		return &BuildAllResponse{
			Total:     0,
			Succeeded: 0,
			Failed:    0,
			Results:   []BuildResponse{},
		}, nil
	}

	// Default to 4 workers if not specified
	maxWorkers := req.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 4
	}

	// Build with progress reporting
	results := s.buildWithProgress(ctx, headers, maxWorkers, progressChan)

	// Aggregate results
	response := &BuildAllResponse{
		Total:   len(headers),
		Results: results,
	}

	for _, result := range results {
		if result.Success {
			response.Succeeded++
		} else {
			response.Failed++
		}
	}

	return response, nil
}

// buildWithProgress builds notes with progress reporting
func (s *BuildService) buildWithProgress(ctx context.Context, headers []domain.NoteHeader, maxWorkers int, progressChan chan<- BuildProgress) []BuildResponse {
	jobs := make(chan domain.NoteHeader, len(headers))
	results := make(chan BuildResponse, len(headers))

	total := len(headers)
	current := 0

	// Start workers
	var wg sync.WaitGroup
	for range maxWorkers {
		wg.Go(func() {
			s.worker(ctx, jobs, results)
		})
	}

	// Send jobs
	for _, header := range headers {
		jobs <- header
	}
	close(jobs)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and report progress
	var buildResults []BuildResponse
	for result := range results {
		buildResults = append(buildResults, result)
		current++

		// Report progress
		progressChan <- BuildProgress{
			Current: current,
			Total:   total,
			Slug:    result.Slug,
			Success: result.Success,
			Error:   result.Error,
		}
	}

	return buildResults
}
