package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/adk/artifact"
	"google.golang.org/genai"
)

// FileSystemArtifactService implements artifact.Service using the local file system.
type FileSystemArtifactService struct {
	RootDir string
}

// NewFileSystemArtifactService creates a new service rooted at the given directory.
func NewFileSystemArtifactService(rootDir string) *FileSystemArtifactService {
	return &FileSystemArtifactService{RootDir: rootDir}
}

// List returns a list of files in the root directory.
func (s *FileSystemArtifactService) List(ctx context.Context, req *artifact.ListRequest) (*artifact.ListResponse, error) {
	var fileNames []string

	entries, err := os.ReadDir(s.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		// Ignore hidden files and directories for simplicity
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !entry.IsDir() {
			fileNames = append(fileNames, entry.Name())
		}
	}

	return &artifact.ListResponse{FileNames: fileNames}, nil
}

// Load reads a file from disk and returns it as an artifact.
func (s *FileSystemArtifactService) Load(ctx context.Context, req *artifact.LoadRequest) (*artifact.LoadResponse, error) {
	// Security check: simple path traversal prevention
	if strings.Contains(req.FileName, "..") || strings.HasPrefix(req.FileName, "/") {
		return nil, fmt.Errorf("invalid filename: %s", req.FileName)
	}

	fullPath := filepath.Join(s.RootDir, req.FileName)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", req.FileName, err)
	}

	fmt.Printf("[FSArtifactService] Successfully read '%s'. Size: %d bytes\n", req.FileName, len(content))

	// Optimization: If text/plain, return as FunctionResponse part
	// This ensures it passes through ADK/GenAI layers as a PROPER tool response for OpenAI Adapter
	return &artifact.LoadResponse{
		Part: &genai.Part{
			FunctionResponse: &genai.FunctionResponse{
				Name: "load_artifacts", // Match the tool name for ID generation
				Response: map[string]any{
					"content": string(content),
				},
			},
		},
	}, nil
}

// Versions returns available versions for an artifact.
// For FileSystem, we only support specific version "1" if file exists.
func (s *FileSystemArtifactService) Versions(ctx context.Context, req *artifact.VersionsRequest) (*artifact.VersionsResponse, error) {
	// Check if file exists
	fullPath := filepath.Join(s.RootDir, req.FileName)
	if _, err := os.Stat(fullPath); err == nil {
		return &artifact.VersionsResponse{Versions: []int64{1}}, nil
	}
	return &artifact.VersionsResponse{Versions: []int64{}}, nil
}

// Save is not supported (Read-Only).
func (s *FileSystemArtifactService) Save(ctx context.Context, req *artifact.SaveRequest) (*artifact.SaveResponse, error) {
	return nil, fmt.Errorf("save not supported by FileSystemArtifactService")
}

// Delete is not supported (Read-Only).
func (s *FileSystemArtifactService) Delete(ctx context.Context, req *artifact.DeleteRequest) error {
	return fmt.Errorf("delete not supported by FileSystemArtifactService")
}

// DeleteAll is not supported (Read-Only).
func (s *FileSystemArtifactService) DeleteAll(ctx context.Context, req *artifact.DeleteRequest) error {
	return fmt.Errorf("delete_all not supported by FileSystemArtifactService")
}
