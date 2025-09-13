package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// cleanupEmptyDirectories removes empty directories from the specified path
// It performs multiple passes to ensure nested empty directories are removed
func cleanupEmptyDirectories(basePath string, dryRun bool) error {
	if dryRun {
		fmt.Printf("🧹 [DRY RUN] Would clean up empty directories in: %s\n", basePath)
		return previewEmptyDirectoryCleanup(basePath)
	}

	fmt.Printf("🧹 Cleaning up empty directories in: %s\n", basePath)

	removedCount := 0
	maxPasses := 10 // Prevent infinite loops

	for pass := 0; pass < maxPasses; pass++ {
		removed, err := removeEmptyDirectoriesPass(basePath)
		if err != nil {
			return fmt.Errorf("error during cleanup pass %d: %v", pass+1, err)
		}

		removedCount += removed

		// If no directories were removed in this pass, we're done
		if removed == 0 {
			break
		}

		fmt.Printf("🧹 Pass %d: Removed %d empty directories\n", pass+1, removed)
	}

	if removedCount > 0 {
		fmt.Printf("✅ Cleanup complete: Removed %d empty directories total\n", removedCount)
	} else {
		fmt.Printf("✅ Cleanup complete: No empty directories found\n")
	}

	return nil
}

// previewEmptyDirectoryCleanup shows what empty directories would be removed
func previewEmptyDirectoryCleanup(basePath string) error {
	var emptyDirs []string

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip files, only process directories
		if !info.IsDir() {
			return nil
		}

		// Skip the base directory itself
		if path == basePath {
			return nil
		}

		// Check if directory is empty
		isEmpty, err := isDirectoryEmpty(path)
		if err != nil {
			return err
		}

		if isEmpty {
			emptyDirs = append(emptyDirs, path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(emptyDirs) > 0 {
		fmt.Printf("🧹 [DRY RUN] Found %d empty directories that would be removed:\n", len(emptyDirs))
		for _, dir := range emptyDirs {
			relPath, _ := filepath.Rel(basePath, dir)
			fmt.Printf("  - %s\n", relPath)
		}
	} else {
		fmt.Printf("🧹 [DRY RUN] No empty directories found\n")
	}

	return nil
}

// removeEmptyDirectoriesPass performs one pass of empty directory removal
// Returns the number of directories removed and any error
func removeEmptyDirectoriesPass(basePath string) (int, error) {
	var dirsToRemove []string

	// First, collect all empty directories
	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip files, only process directories
		if !info.IsDir() {
			return nil
		}

		// Skip the base directory itself
		if path == basePath {
			return nil
		}

		// Check if directory is empty
		isEmpty, err := isDirectoryEmpty(path)
		if err != nil {
			return err
		}

		if isEmpty {
			dirsToRemove = append(dirsToRemove, path)
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	// Remove directories (starting from deepest to avoid conflicts)
	// Sort by path length descending to remove deepest directories first
	for i := 0; i < len(dirsToRemove); i++ {
		for j := i + 1; j < len(dirsToRemove); j++ {
			if len(dirsToRemove[i]) < len(dirsToRemove[j]) {
				dirsToRemove[i], dirsToRemove[j] = dirsToRemove[j], dirsToRemove[i]
			}
		}
	}

	removedCount := 0
	for _, dir := range dirsToRemove {
		// First, remove any non-media files from directories we consider "empty"
		err := removeNonMediaFiles(dir)
		if err != nil {
			fmt.Printf("⚠️  Could not clean non-media files from %s: %v\n", dir, err)
			continue
		}

		// Now try to remove the directory
		err = os.Remove(dir)
		if err != nil {
			// Log the error but continue with other directories
			//fmt.Printf("⚠️  Could not remove empty directory %s: %v\n", dir, err)
		} else {
			removedCount++
			relPath, _ := filepath.Rel(basePath, dir)
			fmt.Printf("🗑️  Removed empty directory: %s\n", relPath)
		}
	}

	return removedCount, nil
}

// removeNonMediaFiles removes non-media files from a directory that we consider "empty"
// This allows us to remove directories that contain only non-media files
func removeNonMediaFiles(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Skip subdirectories
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())

		// Remove all non-media files (including hidden files)
		if !isMediaFile(filePath) {
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to remove non-media file %s: %v", entry.Name(), err)
			}
		}
	}

	return nil
}

// isDirectoryEmpty checks if a directory contains no image or video files
// A directory is considered "empty" if it contains no media files, regardless of other files
func isDirectoryEmpty(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}

	// Check if there are any media files (photos or videos)
	for _, entry := range entries {
		// Skip directories - we only care about files
		if entry.IsDir() {
			continue
		}

		// Skip hidden files like .DS_Store - we'll remove these when cleaning
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Check if this file is a media file (photo or video)
		filePath := filepath.Join(dirPath, entry.Name())
		if isMediaFile(filePath) {
			// Found a media file, directory is not empty
			return false, nil
		}
	}

	// Directory is considered "empty" if it contains no media files
	// This means directories with only non-media files (documents, etc.) will be removed
	// Note: This follows the requirement that empty = "no image or video file in"
	return true, nil
}

// cleanupEmptyDirectoriesIfNeeded performs cleanup based on the operation type and dry-run status
func cleanupEmptyDirectoriesIfNeeded(operationType, basePath string, dryRun bool, processedCount int) {
	// Always run cleanup for destructive operations (processedCount -1 means force cleanup)
	if processedCount == 0 && processedCount != -1 {
		return
	}

	destructiveOps := map[string]bool{
		"process":  true,
		"datetime": true,
		"fallback": true,
		"clean":    true,
	}

	if !destructiveOps[operationType] {
		return
	}

	fmt.Printf("\n🧹 Starting empty directory cleanup after %s operation...\n", operationType)

	if err := cleanupEmptyDirectories(basePath, dryRun); err != nil {
		fmt.Printf("⚠️  Warning: Could not clean up empty directories: %v\n", err)
	}

	// Also try to remove the base directory itself if it becomes empty after processing
	// (only for certain operations where the source directory might become empty)
	if operationType == "process" || operationType == "datetime" || operationType == "fallback" {
		if !dryRun {
			if isEmpty, err := isDirectoryEmpty(basePath); err == nil && isEmpty {
				fmt.Printf("🗑️  Source directory is empty after processing, removing: %s\n", basePath)
				if err := os.Remove(basePath); err != nil {
					fmt.Printf("⚠️  Could not remove empty source directory: %v\n", err)
				}
			}
		} else {
			if isEmpty, err := isDirectoryEmpty(basePath); err == nil && isEmpty {
				fmt.Printf("🗑️  [DRY RUN] Source directory would be removed as it's empty: %s\n", basePath)
			}
		}
	}
}
