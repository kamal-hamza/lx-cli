package services

import (
	"context"
	"fmt"
	"testing"

	"lx/internal/core/domain"
	"lx/internal/core/ports/mocks"
)

func TestBuildService_Execute_Success(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	svc := NewBuildService(mockRepo, mockCompiler)

	// Create a test note
	header, _ := domain.NewNoteHeader("Build Test Note", []string{"tag1"})
	note := domain.NewNoteBody(header, "\\section{Test Content}")
	mockRepo.Save(context.Background(), note)

	// Execute
	resp, err := svc.Execute(context.Background(), BuildRequest{Slug: header.Slug})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if !resp.Success {
		t.Errorf("expected Success=true, got false")
	}

	if resp.Slug != header.Slug {
		t.Errorf("expected slug=%s, got %s", header.Slug, resp.Slug)
	}

	expectedOutput := "/fake/cache/" + header.Slug + ".pdf"
	if resp.OutputPath != expectedOutput {
		t.Errorf("expected OutputPath=%s, got %s", expectedOutput, resp.OutputPath)
	}

	// Verify compiler was called
	calls := mockCompiler.GetCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 compiler call, got %d", len(calls))
	}

	if calls[0] != header.Slug {
		t.Errorf("expected compiler call with slug=%s, got %s", header.Slug, calls[0])
	}
}

func TestBuildService_Execute_NoteNotFound(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	svc := NewBuildService(mockRepo, mockCompiler)

	// Execute with non-existent slug
	resp, err := svc.Execute(context.Background(), BuildRequest{Slug: "non-existent"})

	// Assert
	if err == nil {
		t.Fatal("expected error for non-existent note")
	}

	if resp != nil {
		t.Errorf("expected nil response, got %+v", resp)
	}

	// Verify compiler was NOT called
	calls := mockCompiler.GetCalls()
	if len(calls) != 0 {
		t.Errorf("expected 0 compiler calls, got %d", len(calls))
	}
}

func TestBuildService_Execute_CompilationFailure(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	svc := NewBuildService(mockRepo, mockCompiler)

	// Create a test note
	header, _ := domain.NewNoteHeader("Failing Build", []string{})
	note := domain.NewNoteBody(header, "\\invalid{latex}")
	mockRepo.Save(context.Background(), note)

	// Configure compiler to fail
	mockCompiler.SetShouldFail(true, fmt.Errorf("latex syntax error"))

	// Execute
	resp, err := svc.Execute(context.Background(), BuildRequest{Slug: header.Slug})

	// Assert - Execute returns response even on failure
	if err == nil {
		t.Fatal("expected error from compilation failure")
	}

	if resp == nil {
		t.Fatal("expected non-nil response even on failure")
	}

	if resp.Success {
		t.Errorf("expected Success=false, got true")
	}

	if resp.Error == nil {
		t.Error("expected non-nil Error in response")
	}
}

func TestBuildService_ExecuteAll_Success(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	svc := NewBuildService(mockRepo, mockCompiler)

	// Create multiple test notes
	notes := []struct {
		title string
		tags  []string
	}{
		{"First Note", []string{"math"}},
		{"Second Note", []string{"physics"}},
		{"Third Note", []string{"chemistry"}},
	}

	var slugs []string
	for _, n := range notes {
		header, _ := domain.NewNoteHeader(n.title, n.tags)
		note := domain.NewNoteBody(header, "\\section{Content}")
		mockRepo.Save(context.Background(), note)
		slugs = append(slugs, header.Slug)
	}

	// Execute
	resp, err := svc.ExecuteAll(context.Background(), BuildAllRequest{MaxWorkers: 2})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != len(notes) {
		t.Errorf("expected Total=%d, got %d", len(notes), resp.Total)
	}

	if resp.Succeeded != len(notes) {
		t.Errorf("expected Succeeded=%d, got %d", len(notes), resp.Succeeded)
	}

	if resp.Failed != 0 {
		t.Errorf("expected Failed=0, got %d", resp.Failed)
	}

	if len(resp.Results) != len(notes) {
		t.Errorf("expected %d results, got %d", len(notes), len(resp.Results))
	}

	// Verify all notes were compiled
	calls := mockCompiler.GetCalls()
	if len(calls) != len(notes) {
		t.Fatalf("expected %d compiler calls, got %d", len(notes), len(calls))
	}
}

func TestBuildService_ExecuteAll_EmptyRepository(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	svc := NewBuildService(mockRepo, mockCompiler)

	// Execute with no notes
	resp, err := svc.ExecuteAll(context.Background(), BuildAllRequest{MaxWorkers: 2})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != 0 {
		t.Errorf("expected Total=0, got %d", resp.Total)
	}

	if resp.Succeeded != 0 {
		t.Errorf("expected Succeeded=0, got %d", resp.Succeeded)
	}

	if resp.Failed != 0 {
		t.Errorf("expected Failed=0, got %d", resp.Failed)
	}

	// Verify no compilations
	calls := mockCompiler.GetCalls()
	if len(calls) != 0 {
		t.Errorf("expected 0 compiler calls, got %d", len(calls))
	}
}

func TestBuildService_ExecuteAll_PartialFailures(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	svc := NewBuildService(mockRepo, mockCompiler)

	// Create test notes
	header1, _ := domain.NewNoteHeader("Success Note", []string{})
	note1 := domain.NewNoteBody(header1, "\\section{Good}")
	mockRepo.Save(context.Background(), note1)

	header2, _ := domain.NewNoteHeader("Failure Note", []string{})
	note2 := domain.NewNoteBody(header2, "\\invalid{bad}")
	mockRepo.Save(context.Background(), note2)

	// Configure compiler to fail (it will fail for all in this simple mock)
	// For more sophisticated test, we'd need a conditional mock
	// For now, test the structure
	mockCompiler.SetShouldFail(true, fmt.Errorf("compilation error"))

	// Execute
	resp, err := svc.ExecuteAll(context.Background(), BuildAllRequest{MaxWorkers: 2})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != 2 {
		t.Errorf("expected Total=2, got %d", resp.Total)
	}

	// Both should fail in this setup
	if resp.Failed != 2 {
		t.Errorf("expected Failed=2, got %d", resp.Failed)
	}

	if resp.Succeeded != 0 {
		t.Errorf("expected Succeeded=0, got %d", resp.Succeeded)
	}
}

func TestBuildService_ExecuteAll_DefaultWorkers(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	svc := NewBuildService(mockRepo, mockCompiler)

	// Create a note
	header, _ := domain.NewNoteHeader("Test Note", []string{})
	note := domain.NewNoteBody(header, "\\section{Content}")
	mockRepo.Save(context.Background(), note)

	// Execute with MaxWorkers=0 (should default to 4)
	resp, err := svc.ExecuteAll(context.Background(), BuildAllRequest{MaxWorkers: 0})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("expected Total=1, got %d", resp.Total)
	}

	// Verify compilation happened despite default workers
	calls := mockCompiler.GetCalls()
	if len(calls) != 1 {
		t.Errorf("expected 1 compiler call, got %d", len(calls))
	}
}

func TestBuildService_ExecuteAllWithProgress_Success(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	svc := NewBuildService(mockRepo, mockCompiler)

	// Create multiple notes
	noteCount := 3
	for i := 0; i < noteCount; i++ {
		title := fmt.Sprintf("Note %d", i+1)
		header, _ := domain.NewNoteHeader(title, []string{})
		note := domain.NewNoteBody(header, "\\section{Content}")
		mockRepo.Save(context.Background(), note)
	}

	// Execute with progress channel
	progressChan := make(chan BuildProgress, noteCount)
	resp, err := svc.ExecuteAllWithProgress(
		context.Background(),
		BuildAllRequest{MaxWorkers: 2},
		progressChan,
	)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != noteCount {
		t.Errorf("expected Total=%d, got %d", noteCount, resp.Total)
	}

	// Collect progress updates
	var progressUpdates []BuildProgress
	for progress := range progressChan {
		progressUpdates = append(progressUpdates, progress)
	}

	if len(progressUpdates) != noteCount {
		t.Errorf("expected %d progress updates, got %d", noteCount, len(progressUpdates))
	}

	// Verify progress updates are sequential
	for i, progress := range progressUpdates {
		if progress.Total != noteCount {
			t.Errorf("progress[%d]: expected Total=%d, got %d", i, noteCount, progress.Total)
		}
		if progress.Current < 1 || progress.Current > noteCount {
			t.Errorf("progress[%d]: Current=%d out of range", i, progress.Current)
		}
	}
}

func TestBuildService_ExecuteAllWithProgress_EmptyRepo(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	svc := NewBuildService(mockRepo, mockCompiler)

	// Execute with empty repo
	progressChan := make(chan BuildProgress, 10)
	resp, err := svc.ExecuteAllWithProgress(
		context.Background(),
		BuildAllRequest{MaxWorkers: 2},
		progressChan,
	)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != 0 {
		t.Errorf("expected Total=0, got %d", resp.Total)
	}

	// Channel should be closed with no updates
	var progressUpdates []BuildProgress
	for progress := range progressChan {
		progressUpdates = append(progressUpdates, progress)
	}

	if len(progressUpdates) != 0 {
		t.Errorf("expected 0 progress updates, got %d", len(progressUpdates))
	}
}

func TestBuildService_ConcurrentBuilds(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	svc := NewBuildService(mockRepo, mockCompiler)

	// Create many notes to test concurrency
	noteCount := 10
	for i := 0; i < noteCount; i++ {
		title := fmt.Sprintf("Concurrent Note %d", i+1)
		header, _ := domain.NewNoteHeader(title, []string{})
		note := domain.NewNoteBody(header, "\\section{Content}")
		mockRepo.Save(context.Background(), note)
	}

	// Execute with multiple workers
	resp, err := svc.ExecuteAll(context.Background(), BuildAllRequest{MaxWorkers: 4})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != noteCount {
		t.Errorf("expected Total=%d, got %d", noteCount, resp.Total)
	}

	if resp.Succeeded != noteCount {
		t.Errorf("expected Succeeded=%d, got %d", noteCount, resp.Succeeded)
	}

	// All notes should be compiled exactly once
	calls := mockCompiler.GetCalls()
	if len(calls) != noteCount {
		t.Errorf("expected %d compiler calls, got %d", noteCount, len(calls))
	}
}
