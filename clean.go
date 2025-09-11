package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DuplicateFile represents information about a potentially duplicate file
type DuplicateFile struct {
	Path     string
	Hash     string
	Size     int64
	ModTime  time.Time
	Filename string
}

// DuplicateGroup represents a group of duplicate files with the same hash
type DuplicateGroup struct {
	Hash  string
	Size  int64
	Files []DuplicateFile
}

// DuplicateAction represents what to do with duplicates
type DuplicateAction int

const (
	DuplicateKeepAll DuplicateAction = iota
	DuplicateKeepNewest
	DuplicateKeepOldest
	DuplicateKeepFirst
	DuplicateKeepBestStructure
)

// processClean handles the clean command workflow
func processClean(targetPath string, dryRun bool, verbose bool, workers int) error {
	fmt.Printf("🧹 Clean Mode - Duplicate Detection and Removal\n")
	fmt.Printf("📁 Target: %s\n", targetPath)
	if dryRun {
		fmt.Println("🔍 DRY RUN MODE - No files will be deleted")
	}
	fmt.Println()

	// Find duplicate files
	duplicateGroups, err := findDuplicateFiles(targetPath, verbose)
	if err != nil {
		return fmt.Errorf("failed to find duplicates: %v", err)
	}

	if len(duplicateGroups) == 0 {
		fmt.Println("✅ No duplicate files found!")
		return nil
	}

	// Report duplicates
	reportDuplicates(duplicateGroups, verbose)

	// Remove duplicates using intelligent structure-based selection
	return removeDuplicateFiles(duplicateGroups, DuplicateKeepBestStructure, dryRun, verbose)
}

// calculateFileHash computes SHA-256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file %s: %v\n", filePath, err)
		}
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// findDuplicateFiles scans for duplicate files in the given directory
func findDuplicateFiles(baseDir string, verbose bool) ([]DuplicateGroup, error) {
	fmt.Printf("🔍 Scanning for duplicate files in %s...\n", baseDir)
	
	// Map of hash -> list of files
	hashMap := make(map[string][]DuplicateFile)
	fileCount := 0
	
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Only process image files (use our existing function)
		if !isPhotoFile(path) {
			return nil
		}
		
		// Skip macOS metadata files
		if strings.HasPrefix(info.Name(), "._") {
			return nil
		}
		
		fileCount++
		if verbose {
			fmt.Printf("Hashing: %s\n", filepath.Base(path))
		} else if fileCount%50 == 0 {
			fmt.Printf("Scanned %d files...\n", fileCount)
		}
		
		hash, err := calculateFileHash(path)
		if err != nil {
			fmt.Printf("Warning: Could not hash %s: %v\n", path, err)
			return nil
		}
		
		duplicateFile := DuplicateFile{
			Path:     path,
			Hash:     hash,
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			Filename: info.Name(),
		}
		
		hashMap[hash] = append(hashMap[hash], duplicateFile)
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	fmt.Printf("📊 Scanned %d image files.\n", fileCount)
	
	// Find groups with more than one file (duplicates)
	var duplicateGroups []DuplicateGroup
	duplicateCount := 0
	
	for hash, files := range hashMap {
		if len(files) > 1 {
			// Sort files by modification time (newest first)
			sort.Slice(files, func(i, j int) bool {
				return files[i].ModTime.After(files[j].ModTime)
			})
			
			group := DuplicateGroup{
				Hash:  hash,
				Size:  files[0].Size,
				Files: files,
			}
			duplicateGroups = append(duplicateGroups, group)
			duplicateCount += len(files)
		}
	}
	
	// Sort groups by file size (largest first)
	sort.Slice(duplicateGroups, func(i, j int) bool {
		return duplicateGroups[i].Size > duplicateGroups[j].Size
	})
	
	if len(duplicateGroups) > 0 {
		fmt.Printf("⚠️  Found %d duplicate groups containing %d files.\n", 
			len(duplicateGroups), duplicateCount)
	} else {
		fmt.Println("✅ No duplicate files found.")
	}
	
	return duplicateGroups, nil
}

// reportDuplicates displays information about duplicate files
func reportDuplicates(duplicateGroups []DuplicateGroup, verbose bool) {
	if len(duplicateGroups) == 0 {
		fmt.Println("No duplicates to report.")
		return
	}
	
	totalWastedSpace := int64(0)
	
	fmt.Println("\n📋 === Duplicate Files Report ===")
	
	for i, group := range duplicateGroups {
		fmt.Printf("\nGroup %d: %d files (%s each)\n", 
			i+1, len(group.Files), formatFileSize(group.Size))
		fmt.Printf("Hash: %s...\n", group.Hash[:16])
		
		// Calculate wasted space (all files except the first one)
		wastedSpace := group.Size * int64(len(group.Files)-1)
		totalWastedSpace += wastedSpace
		
		fmt.Printf("Wasted space: %s\n", formatFileSize(wastedSpace))
		
		keepIndex := getKeepIndex(group, DuplicateKeepBestStructure, verbose)
		
		for j, file := range group.Files {
			status := ""
			if j == keepIndex {
				status = " ✅ (KEEP)"
			} else {
				status = " ❌ (REMOVE)"
			}
			
			if verbose {
				structureScore := calculateStructureScore(file.Path)
				fmt.Printf("  %d. %s%s (structure score: %d)\n", j+1, file.Path, status, structureScore)
			} else {
				fmt.Printf("  %d. %s%s\n", j+1, file.Path, status)
			}
			fmt.Printf("     Modified: %s\n", file.ModTime.Format("2006-01-02 15:04:05"))
		}
	}
	
	fmt.Printf("\n📊 === Summary ===\n")
	fmt.Printf("Total duplicate groups: %d\n", len(duplicateGroups))
	fmt.Printf("Total wasted space: %s\n", formatFileSize(totalWastedSpace))
	fmt.Printf("Strategy: Intelligent structure-based selection\n")
}

// removeDuplicateFiles removes duplicate files according to the specified action
func removeDuplicateFiles(duplicateGroups []DuplicateGroup, action DuplicateAction, dryRun, verbose bool) error {
	if len(duplicateGroups) == 0 {
		fmt.Println("No duplicates to remove.")
		return nil
	}
	
	totalRemoved := 0
	totalSpace := int64(0)
	
	fmt.Printf("\n🗑️  Removing duplicate files (using %s strategy)...\n", 
		getDuplicateActionDescription(action))
	
	for _, group := range duplicateGroups {
		keepIndex := getKeepIndex(group, action, verbose)
		
		for i, file := range group.Files {
			if i == keepIndex {
				continue // Skip the file we want to keep
			}
			
			if dryRun {
				fmt.Printf("[DRY RUN] Would remove: %s\n", file.Path)
			} else {
				if verbose {
					fmt.Printf("Removing: %s\n", file.Path)
				}
				
				if err := os.Remove(file.Path); err != nil {
					fmt.Printf("Error removing %s: %v\n", file.Path, err)
					continue
				}
			}
			
			totalRemoved++
			totalSpace += file.Size
		}
	}
	
	fmt.Println()
	if dryRun {
		fmt.Printf("📊 [DRY RUN] Would remove %d duplicate files, saving %s\n", 
			totalRemoved, formatFileSize(totalSpace))
	} else {
		fmt.Printf("✅ Removed %d duplicate files, saved %s\n", 
			totalRemoved, formatFileSize(totalSpace))
	}
	
	return nil
}

// findBestStructuredFile finds the file with the best folder and filename structure
func findBestStructuredFile(files []DuplicateFile) int {
	bestIndex := -1
	bestScore := -1000 // Start with very low score to handle negative scores
	
	for i, file := range files {
		score := calculateStructureScore(file.Path)
		if score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}
	
	// Always return the best scoring file (even if score is negative)
	// This ensures we keep the "least bad" file when all have penalties
	return bestIndex
}

// calculateStructureScore assigns a score to a file based on its structure quality
func calculateStructureScore(filePath string) int {
	score := 0
	filename := filepath.Base(filePath)
	filenameNoExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	filenameLower := strings.ToLower(filename)
	dirPath := filepath.Dir(filePath)
	
	// PENALTY SCORES (negative points for undesirable files)
	
	// Penalty -5: Files with 'copy' in the name (should be removed first)
	if strings.Contains(filenameLower, "copy") {
		score -= 5
	}
	
	// Penalty -3: Files ending with hyphenated numbers (e.g., -1, -2, -10)
	// This catches duplicates like "photo-1.jpg", "document-2.pdf", etc.
	if matched, _ := filepath.Match("*-[0-9]*", filenameNoExt); matched {
		score -= 3
	}
	
	// Penalty -2: Files with " copy" or "(copy)" variations
	if strings.Contains(filenameLower, " copy") || strings.Contains(filenameLower, "(copy)") {
		score -= 2
	}
	
	// Penalty -2: Files with duplicate indicators like " (1)", " (2)", etc.
	if matched, _ := filepath.Match("* ([0-9])*", filenameNoExt); matched {
		score -= 2
	}
	
	// POSITIVE SCORES (for good file organization)
	
	// Score 3: Perfect processed filename format (YYYY-MM-DD-*)
	if matched, _ := filepath.Match("????-??-??-*", filename); matched {
		score += 3
	}
	
	// Score 2: Good date format in filename (YYYYMMDD*)
	if matched, _ := filepath.Match("????????*", filename); matched {
		// Check if it starts with what looks like a date
		if len(filename) >= 8 {
			possibleDate := filename[:8]
			if isNumeric(possibleDate) && isReasonableYear(possibleDate[:4]) {
				score += 2
			}
		}
	}
	
	// Score 2: File is in a proper year directory structure (*/YYYY/* or */YYYY/COUNTRY/CITY/*)
	pathParts := strings.Split(dirPath, string(filepath.Separator))
	for _, part := range pathParts {
		if len(part) == 4 && isNumeric(part) && isReasonableYear(part) {
			score += 2
			break
		}
	}
	
	// Score 1: File is in a descriptive directory (contains location info)
	if hasLocationInPath(dirPath) {
		score += 1
	}
	
	return score
}

// isReasonableYear checks if a year string is in a reasonable range
func isReasonableYear(yearStr string) bool {
	if len(yearStr) != 4 {
		return false
	}
	year := yearStr
	return year >= "2000" && year <= "2030" // Reasonable photo year range
}

// hasLocationInPath checks if the directory path contains location-like information
func hasLocationInPath(dirPath string) bool {
	// Look for common location indicators in path components
	pathLower := strings.ToLower(dirPath)
	locationKeywords := []string{
		"amsterdam", "london", "paris", "berlin", "madrid", "rome", "vienna", 
		"barcelona", "prague", "athens", "lisbon", "budapest", "warsaw", "dublin",
		"glasgow", "edinburgh", "manchester", "birmingham", "liverpool", "bristol",
		"italy", "spain", "france", "germany", "austria", "netherlands", "belgium",
		"portugal", "greece", "poland", "czech", "hungary", "ireland", "scotland",
		"england", "wales", "switzerland", "norway", "sweden", "denmark", "finland",
		"palma", "majorca", "ibiza", "seville", "valencia", "florence", "venice",
		"naples", "milan", "turin", "bologna", "zurich", "geneva", "basel", "bern",
		"krakow", "gdansk", "wroclaw", "poznan", "lodz", "stockholm", "gothenburg",
		"copenhagen", "oslo", "bergen", "helsinki", "reykjavik", "dublin", "cork",
		"united-kingdom", "great-britain",
	}
	
	for _, keyword := range locationKeywords {
		if strings.Contains(pathLower, keyword) {
			return true
		}
	}
	
	// Also check for year-location pattern (e.g., "2024 Paris June")
	return strings.Contains(pathLower, " ") && (strings.Contains(pathLower, "jan") || 
		strings.Contains(pathLower, "feb") || strings.Contains(pathLower, "mar") ||
		strings.Contains(pathLower, "apr") || strings.Contains(pathLower, "may") ||
		strings.Contains(pathLower, "jun") || strings.Contains(pathLower, "jul") ||
		strings.Contains(pathLower, "aug") || strings.Contains(pathLower, "sep") ||
		strings.Contains(pathLower, "oct") || strings.Contains(pathLower, "nov") ||
		strings.Contains(pathLower, "dec"))
}

// getKeepIndex returns the index of the file to keep based on the action
func getKeepIndex(group DuplicateGroup, action DuplicateAction, verbose bool) int {
	// First, try to find a file with good structure (correct folder and filename format)
	if action == DuplicateKeepBestStructure {
		bestStructureIndex := findBestStructuredFile(group.Files)
		if bestStructureIndex != -1 {
			if verbose {
				fmt.Printf("   📂 Preferring file with good structure: %s\n", group.Files[bestStructureIndex].Path)
			}
			return bestStructureIndex
		}
	}
	
	// Fall back to other strategies if no file has good structure
	switch action {
	case DuplicateKeepNewest, DuplicateKeepBestStructure:
		// Files are already sorted by modification time (newest first)
		return 0
	case DuplicateKeepOldest:
		return len(group.Files) - 1
	case DuplicateKeepFirst:
		fallthrough
	default:
		return 0
	}
}

// getDuplicateActionDescription returns a human-readable description of the action
func getDuplicateActionDescription(action DuplicateAction) string {
	switch action {
	case DuplicateKeepBestStructure:
		return "intelligent structure-based selection"
	case DuplicateKeepNewest:
		return "newest file"
	case DuplicateKeepOldest:
		return "oldest file"
	case DuplicateKeepFirst:
		return "first file"
	default:
		return "intelligent structure-based selection"
	}
}

// formatFileSize formats file size in human-readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}