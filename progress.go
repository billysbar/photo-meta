package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ProgressState represents the state of an operation that can be resumed
type ProgressState struct {
	Operation      string            `json:"operation"`
	SourcePath     string            `json:"source_path"`
	DestPath       string            `json:"dest_path"`
	StartTime      time.Time         `json:"start_time"`
	LastSaveTime   time.Time         `json:"last_save_time"`
	ProcessedFiles []string          `json:"processed_files"`
	FailedFiles    map[string]string `json:"failed_files"` // filepath -> error message
	TotalFiles     int               `json:"total_files"`
	Workers        int               `json:"workers"`
	DryRun         bool              `json:"dry_run"`
	DryRunSample   int               `json:"dry_run_sample"`
	ShowProgress   bool              `json:"show_progress"`
	GenerateInfo   bool              `json:"generate_info"`
	CurrentPhase   string            `json:"current_phase"` // "scanning", "processing", "cleanup", "complete"
}

// ProgressManager handles saving and loading progress state
type ProgressManager struct {
	stateFile string
	state     *ProgressState
}

// NewProgressManager creates a new progress manager
func NewProgressManager(operation, sourcePath, destPath string) *ProgressManager {
	// Create state file name based on operation and paths
	stateFileName := fmt.Sprintf(".photo-meta-progress-%s-%d.json", 
		operation, 
		time.Now().Unix())
	
	// Place state file in destination directory if possible, otherwise in temp
	var stateDir string
	if destPath != "" && destPath != sourcePath {
		stateDir = destPath
	} else {
		stateDir = os.TempDir()
	}
	
	stateFile := filepath.Join(stateDir, stateFileName)
	
	return &ProgressManager{
		stateFile: stateFile,
		state: &ProgressState{
			Operation:      operation,
			SourcePath:     sourcePath,
			DestPath:       destPath,
			StartTime:      time.Now(),
			LastSaveTime:   time.Now(),
			ProcessedFiles: make([]string, 0),
			FailedFiles:    make(map[string]string),
			CurrentPhase:   "scanning",
		},
	}
}

// LoadProgressManager attempts to load an existing progress state
func LoadProgressManager(stateFile string) (*ProgressManager, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read progress file: %v", err)
	}
	
	var state ProgressState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse progress file: %v", err)
	}
	
	return &ProgressManager{
		stateFile: stateFile,
		state:     &state,
	}, nil
}

// FindExistingProgress looks for existing progress files for resumption
func FindExistingProgress(operation, sourcePath, destPath string) ([]string, error) {
	var searchDirs []string
	
	// Search in destination directory first
	if destPath != "" {
		searchDirs = append(searchDirs, destPath)
	}
	
	// Then search in source directory
	if sourcePath != destPath {
		searchDirs = append(searchDirs, sourcePath)
	}
	
	// Finally search in temp directory
	searchDirs = append(searchDirs, os.TempDir())
	
	var progressFiles []string
	
	for _, dir := range searchDirs {
		pattern := filepath.Join(dir, fmt.Sprintf(".photo-meta-progress-%s-*.json", operation))
		matches, err := filepath.Glob(pattern)
		if err == nil {
			progressFiles = append(progressFiles, matches...)
		}
	}
	
	return progressFiles, nil
}

// SaveState saves the current progress state to disk
func (pm *ProgressManager) SaveState() error {
	pm.state.LastSaveTime = time.Now()
	
	data, err := json.MarshalIndent(pm.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal progress state: %v", err)
	}
	
	// Write to temporary file first, then rename for atomic operation
	tempFile := pm.stateFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write progress file: %v", err)
	}
	
	if err := os.Rename(tempFile, pm.stateFile); err != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to update progress file: %v", err)
	}
	
	return nil
}

// UpdateProgress updates the progress state with new information
func (pm *ProgressManager) UpdateProgress(phase string, totalFiles int, workers int, dryRun bool, dryRunSample int, showProgress bool, generateInfo bool) {
	pm.state.CurrentPhase = phase
	pm.state.TotalFiles = totalFiles
	pm.state.Workers = workers
	pm.state.DryRun = dryRun
	pm.state.DryRunSample = dryRunSample
	pm.state.ShowProgress = showProgress
	pm.state.GenerateInfo = generateInfo
}

// AddProcessedFile marks a file as successfully processed
func (pm *ProgressManager) AddProcessedFile(filepath string) {
	pm.state.ProcessedFiles = append(pm.state.ProcessedFiles, filepath)
}

// AddFailedFile marks a file as failed with error message
func (pm *ProgressManager) AddFailedFile(filepath, errorMsg string) {
	pm.state.FailedFiles[filepath] = errorMsg
}

// IsProcessed checks if a file has already been processed
func (pm *ProgressManager) IsProcessed(filepath string) bool {
	for _, processed := range pm.state.ProcessedFiles {
		if processed == filepath {
			return true
		}
	}
	return false
}

// GetProgress returns current progress statistics
func (pm *ProgressManager) GetProgress() (int, int, int) {
	processed := len(pm.state.ProcessedFiles)
	failed := len(pm.state.FailedFiles)
	total := pm.state.TotalFiles
	return processed, failed, total
}

// GetState returns the current progress state
func (pm *ProgressManager) GetState() *ProgressState {
	return pm.state
}

// CleanupStateFile removes the progress state file
func (pm *ProgressManager) CleanupStateFile() error {
	return os.Remove(pm.stateFile)
}

// PrintResumeSummary displays a summary of what will be resumed
func (pm *ProgressManager) PrintResumeSummary() {
	state := pm.state
	elapsed := time.Since(state.StartTime)
	
	fmt.Printf("📋 RESUMING OPERATION\n")
	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("Operation: %s\n", state.Operation)
	fmt.Printf("Source:    %s\n", state.SourcePath)
	if state.DestPath != "" {
		fmt.Printf("Target:    %s\n", state.DestPath)
	}
	fmt.Printf("Started:   %s\n", state.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Elapsed:   %v\n", elapsed.Round(time.Second))
	fmt.Printf("Phase:     %s\n", state.CurrentPhase)
	
	processed := len(state.ProcessedFiles)
	failed := len(state.FailedFiles)
	
	if state.TotalFiles > 0 {
		pct := float64(processed) / float64(state.TotalFiles) * 100
		fmt.Printf("Progress:  %d/%d files (%.1f%%)\n", processed, state.TotalFiles, pct)
	} else {
		fmt.Printf("Progress:  %d files processed\n", processed)
	}
	
	if failed > 0 {
		fmt.Printf("Failed:    %d files\n", failed)
	}
	
	remaining := state.TotalFiles - processed - failed
	if remaining > 0 {
		fmt.Printf("Remaining: %d files\n", remaining)
	}
	
	fmt.Printf("═══════════════════════════════════════════\n")
}

// shouldAutoSave determines if we should save progress (every 10 files or 30 seconds)
func (pm *ProgressManager) shouldAutoSave() bool {
	timeSinceLastSave := time.Since(pm.state.LastSaveTime)
	filesSinceLastSave := len(pm.state.ProcessedFiles) % 10
	
	return timeSinceLastSave >= 30*time.Second || filesSinceLastSave == 0
}

// AutoSave saves progress if conditions are met
func (pm *ProgressManager) AutoSave() error {
	if pm.shouldAutoSave() {
		return pm.SaveState()
	}
	return nil
}

// getOldProgressFiles finds and returns old progress files for cleanup
func getOldProgressFiles() ([]string, error) {
	tempDir := os.TempDir()
	pattern := filepath.Join(tempDir, ".photo-meta-progress-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	
	var oldFiles []string
	cutoff := time.Now().AddDate(0, 0, -7) // 7 days old
	
	for _, file := range matches {
		info, err := os.Stat(file)
		if err == nil && info.ModTime().Before(cutoff) {
			oldFiles = append(oldFiles, file)
		}
	}
	
	return oldFiles, nil
}

// CleanupOldProgressFiles removes progress files older than 7 days
func CleanupOldProgressFiles() error {
	oldFiles, err := getOldProgressFiles()
	if err != nil {
		return err
	}
	
	for _, file := range oldFiles {
		os.Remove(file) // Ignore errors for cleanup
	}
	
	return nil
}