package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/kamal-hamza/lx-cli/internal/core/domain"
	"github.com/kamal-hamza/lx-cli/internal/core/ports/mocks"
)

func TestBuildService_Execute_Success(t *testing.T) {
	// Setup
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	mockPreprocessor := mocks.NewMockPreprocessor()

	// FIX: Use the test constructor to inject mocks
	svc := NewBuildServiceWithPreprocessor(mockRepo, mockCompiler, mockPreprocessor, nil)

	// Create a test note
	header, _ := domain.NewNoteHeader("Build Test Note", []string{"tag1"}, "Test.md")
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

	// Verify Preprocessor was called
	prepCalls := mockPreprocessor.GetCalls()
	if len(prepCalls) != 1 {
		t.Fatalf("expected 1 preprocessor call, got %d", len(prepCalls))
	}
	if prepCalls[0] != header.Slug {
		t.Errorf("expected preprocessor call with slug=%s, got %s", header.Slug, prepCalls[0])
	}

	// Verify Compiler was called with the OUTPUT of the preprocessor
	compilerCalls := mockCompiler.GetCalls()
	if len(compilerCalls) != 1 {
		t.Fatalf("expected 1 compiler call, got %d", len(compilerCalls))
	}

	expectedInput := "/fake/cache/mock_preprocessed.tex"
	if compilerCalls[0] != expectedInput {
		t.Errorf("expected compiler input=%s, got %s", expectedInput, compilerCalls[0])
	}
}

func TestBuildService_Execute_NoteNotFound(t *testing.T) {
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	mockPreprocessor := mocks.NewMockPreprocessor()
	svc := NewBuildServiceWithPreprocessor(mockRepo, mockCompiler, mockPreprocessor, nil)

	resp, err := svc.Execute(context.Background(), BuildRequest{Slug: "non-existent"})

	if err == nil {
		t.Fatal("expected error for non-existent note")
	}

	if resp == nil {
		t.Fatal("expected response even on error")
	}

	if resp.Success {
		t.Error("expected Success to be false")
	}

	if len(mockPreprocessor.GetCalls()) != 0 {
		t.Error("expected 0 preprocessor calls")
	}
}

func TestBuildService_Execute_PreprocessingFailure(t *testing.T) {
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	mockPreprocessor := mocks.NewMockPreprocessor()
	svc := NewBuildServiceWithPreprocessor(mockRepo, mockCompiler, mockPreprocessor, nil)

	header, _ := domain.NewNoteHeader("Bad Note", []string{}, "Bad.md")
	note := domain.NewNoteBody(header, "content")
	mockRepo.Save(context.Background(), note)

	mockPreprocessor.SetShouldFail(true, fmt.Errorf("syntax error"))

	resp, err := svc.Execute(context.Background(), BuildRequest{Slug: header.Slug})

	if err == nil {
		t.Fatal("expected error from preprocessing failure")
	}

	if resp == nil {
		t.Fatal("expected response even on error")
	}

	if resp.Success {
		t.Error("expected Success to be false")
	}

	if len(mockCompiler.GetCalls()) != 0 {
		t.Error("compiler should not be called if preprocessing fails")
	}
}

func TestBuildService_Execute_CompilationFailure(t *testing.T) {
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	mockPreprocessor := mocks.NewMockPreprocessor()
	svc := NewBuildServiceWithPreprocessor(mockRepo, mockCompiler, mockPreprocessor, nil)

	header, _ := domain.NewNoteHeader("Failing Build", []string{}, "Fail.md")
	note := domain.NewNoteBody(header, "\\invalid{latex}")
	mockRepo.Save(context.Background(), note)

	mockCompiler.SetShouldFail(true, fmt.Errorf("latex syntax error"))

	resp, err := svc.Execute(context.Background(), BuildRequest{Slug: header.Slug})

	if err == nil {
		t.Fatal("expected error from compilation failure")
	}

	if resp == nil {
		t.Fatal("expected non-nil response even on failure")
	}

	if resp.Success {
		t.Errorf("expected Success=false, got true")
	}

	if len(mockPreprocessor.GetCalls()) != 1 {
		t.Error("preprocessor should have been called")
	}
}

func TestBuildService_ExecuteAll_Success(t *testing.T) {
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	mockPreprocessor := mocks.NewMockPreprocessor()
	svc := NewBuildServiceWithPreprocessor(mockRepo, mockCompiler, mockPreprocessor, nil)

	notes := []struct {
		title string
		tags  []string
	}{
		{"First Note", []string{"math"}},
		{"Second Note", []string{"physics"}},
		{"Third Note", []string{"chemistry"}},
	}

	for _, n := range notes {
		header, _ := domain.NewNoteHeader(n.title, n.tags, "Test.md")
		note := domain.NewNoteBody(header, "\\section{Content}")
		mockRepo.Save(context.Background(), note)
	}

	resp, err := svc.ExecuteAll(context.Background(), BuildAllRequest{MaxWorkers: 2})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != len(notes) {
		t.Errorf("expected Total=%d, got %d", len(notes), resp.Total)
	}

	if resp.Succeeded != len(notes) {
		t.Errorf("expected Succeeded=%d, got %d", len(notes), resp.Succeeded)
	}

	if len(mockPreprocessor.GetCalls()) != len(notes) {
		t.Errorf("expected %d preprocessor calls, got %d", len(notes), len(mockPreprocessor.GetCalls()))
	}
}

func TestBuildService_ExecuteAllWithProgress_Success(t *testing.T) {
	mockRepo := mocks.NewMockRepository()
	mockCompiler := mocks.NewMockCompiler()
	mockPreprocessor := mocks.NewMockPreprocessor()
	svc := NewBuildServiceWithPreprocessor(mockRepo, mockCompiler, mockPreprocessor, nil)

	noteCount := 3
	for i := 0; i < noteCount; i++ {
		title := fmt.Sprintf("Note %d", i+1)
		header, _ := domain.NewNoteHeader(title, []string{}, "Note.md")
		note := domain.NewNoteBody(header, "\\section{Content}")
		mockRepo.Save(context.Background(), note)
	}

	progressChan := make(chan BuildProgress, noteCount)
	resp, err := svc.ExecuteAllWithProgress(
		context.Background(),
		BuildAllRequest{MaxWorkers: 2},
		progressChan,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != noteCount {
		t.Errorf("expected Total=%d, got %d", noteCount, resp.Total)
	}

	var progressUpdates []BuildProgress
	for progress := range progressChan {
		progressUpdates = append(progressUpdates, progress)
	}

	if len(progressUpdates) != noteCount {
		t.Errorf("expected %d progress updates, got %d", noteCount, len(progressUpdates))
	}
}
