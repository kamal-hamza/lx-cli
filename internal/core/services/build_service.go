package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/ports"
	"github.com/kamal-hamza/lx-cli/pkg/vault"
)

// BuildService handles LaTeX compilation operations
type BuildService struct {
	noteRepo     ports.Repository
	compiler     ports.Compiler
	preprocessor ports.Preprocessor // Interface type to allow mocking
	vault        *vault.Vault
}

// NewBuildService creates a new build service (Normal usage)
func NewBuildService(noteRepo ports.Repository, compiler ports.Compiler, v *vault.Vault) *BuildService {
	return &BuildService{
		noteRepo:     noteRepo,
		compiler:     compiler,
		preprocessor: NewPreprocessor(noteRepo, v), // Uses real preprocessor
		vault:        v,
	}
}

// NewBuildServiceWithPreprocessor creates a new build service with injected preprocessor (For testing)
func NewBuildServiceWithPreprocessor(noteRepo ports.Repository, compiler ports.Compiler, preprocessor ports.Preprocessor, v *vault.Vault) *BuildService {
	return &BuildService{
		noteRepo:     noteRepo,
		compiler:     compiler,
		preprocessor: preprocessor,
		vault:        v,
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

	// 1. Preprocess the note
	// This resolves links/paths and writes a compilable .tex file to the cache directory
	preprocessedPath, err := s.preprocessor.Process(req.Slug)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}

	// 2. Compile the preprocessed file
	err = s.compiler.Compile(ctx, preprocessedPath, []string{})
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
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(ctx, jobs, results)
		}()
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

		// 1. Preprocess
		preprocessedPath, err := s.preprocessor.Process(header.Slug)
		if err != nil {
			results <- BuildResponse{
				Slug:    header.Slug,
				Success: false,
				Error:   err,
			}
			continue
		}

		// 2. Compile
		err = s.compiler.Compile(ctx, preprocessedPath, []string{})

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
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(ctx, jobs, results)
		}()
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
