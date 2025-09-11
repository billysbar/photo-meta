package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// NoGPSError represents an error when GPS data is not found
type NoGPSError struct {
	File string
	Err  error
}

func (e *NoGPSError) Error() string {
	return fmt.Sprintf("no GPS data in %s: %v", e.File, e.Err)
}

// isNoGPSError checks if the error is due to missing GPS data
func isNoGPSError(err error) bool {
	_, ok := err.(*NoGPSError)
	return ok
}

func main() {
	if len(os.Args) < 2 {
		showUsage()
		return
	}

	command := os.Args[1]
	
	switch command {
	case "process":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./photo-metadata-editor process /source/path /destination/path [--workers N] [--dry-run] [--dry-run1] [--progress]")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		destPath := os.Args[3]
		
		// Check for incorrectly formatted dry-run arguments
		for i := 4; i < len(os.Args); i++ {
			arg := strings.ToLower(os.Args[i])
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" && arg != "--dry-run1" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run' or '--dry-run1' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		workers := 4 // Default worker count
		dryRun := false
		dryRun1 := false
		showProgress := true // Default to showing progress
		for i := 4; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--workers":
				if i+1 < len(os.Args) {
					if _, err := fmt.Sscanf(os.Args[i+1], "%d", &workers); err != nil {
						log.Fatalf("Invalid worker count: %s", os.Args[i+1])
					}
					i++ // Skip the next argument since it's the worker count
				}
			case "--dry-run":
				dryRun = true
			case "--dry-run1":
				dryRun1 = true
				dryRun = true // dry-run1 implies dry-run
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			}
		}
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Create destination path if it doesn't exist
		if err := os.MkdirAll(destPath, 0755); err != nil {
			log.Fatalf("Failed to create destination path: %v", err)
		}
		
		// Process photos concurrently
		if err := processPhotosConcurrently(sourcePath, destPath, workers, dryRun, dryRun1, showProgress); err != nil {
			log.Fatal(err)
		}
		
		// Clean up empty directories after processing
		cleanupEmptyDirectoriesIfNeeded("process", sourcePath, dryRun, -1) // -1 means always run cleanup
		
	case "fallback":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./photo-metadata-editor fallback /source/path /destination/path [--workers N] [--dry-run] [--dry-run1] [--progress]")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		destPath := os.Args[3]
		
		// Check for incorrectly formatted dry-run arguments
		for i := 4; i < len(os.Args); i++ {
			arg := strings.ToLower(os.Args[i])
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" && arg != "--dry-run1" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run' or '--dry-run1' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		workers := 4 // Default worker count
		dryRun := false
		dryRun1 := false
		showProgress := true // Default to showing progress
		for i := 4; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--workers":
				if i+1 < len(os.Args) {
					if _, err := fmt.Sscanf(os.Args[i+1], "%d", &workers); err != nil {
						log.Fatalf("Invalid worker count: %s", os.Args[i+1])
					}
					i++ // Skip the next argument since it's the worker count
				}
			case "--dry-run":
				dryRun = true
			case "--dry-run1":
				dryRun1 = true
				dryRun = true // dry-run1 implies dry-run
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			}
		}
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Create destination path if it doesn't exist
		if err := os.MkdirAll(destPath, 0755); err != nil {
			log.Fatalf("Failed to create destination path: %v", err)
		}
		
		// Process fallback organization
		if err := processFallbackOrganization(sourcePath, destPath, dryRun, dryRun1, showProgress); err != nil {
			log.Fatal(err)
		}
		
		// Clean up empty directories after fallback processing
		cleanupEmptyDirectoriesIfNeeded("fallback", sourcePath, dryRun, -1)
		
	case "datetime":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./photo-metadata-editor datetime /source/path /destination/path [--workers N] [--dry-run] [--dry-run1] [--progress]")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		destPath := os.Args[3]
		
		// Check for incorrectly formatted dry-run arguments
		for i := 4; i < len(os.Args); i++ {
			arg := strings.ToLower(os.Args[i])
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" && arg != "--dry-run1" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run' or '--dry-run1' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		workers := 4 // Default worker count
		dryRun := false
		dryRun1 := false
		showProgress := true // Default to showing progress
		for i := 4; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--workers":
				if i+1 < len(os.Args) {
					if _, err := fmt.Sscanf(os.Args[i+1], "%d", &workers); err != nil {
						log.Fatalf("Invalid worker count: %s", os.Args[i+1])
					}
					i++ // Skip the next argument since it's the worker count
				}
			case "--dry-run":
				dryRun = true
			case "--dry-run1":
				dryRun1 = true
				dryRun = true // dry-run1 implies dry-run
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			}
		}
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Check if destination path exists
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			log.Fatalf("Destination path does not exist: %s", destPath)
		}
		
		// Process datetime matching
		if err := processDateTimeMatching(sourcePath, destPath, dryRun, dryRun1, showProgress); err != nil {
			log.Fatal(err)
		}
		
		// Clean up empty directories after datetime processing
		cleanupEmptyDirectoriesIfNeeded("datetime", sourcePath, dryRun, -1)
		
	case "clean":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ./photo-metadata-editor clean /target/path [--dry-run] [--dry-run1] [--verbose] [--workers N] [--progress]")
			os.Exit(1)
		}
		
		targetPath := os.Args[2]
		
		// Check if target path exists
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			log.Fatalf("Target path does not exist: %s", targetPath)
		}
		
		// Check for incorrectly formatted dry-run arguments
		for i := 3; i < len(os.Args); i++ {
			arg := strings.ToLower(os.Args[i])
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" && arg != "--dry-run1" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run' or '--dry-run1' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		dryRun := false
		dryRun1 := false
		verbose := false
		workers := 4 // Default worker count
		showProgress := true // Default to showing progress (unless verbose)
		for i := 3; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--dry-run":
				dryRun = true
			case "--dry-run1":
				dryRun1 = true
				dryRun = true // dry-run1 implies dry-run
			case "--verbose":
				verbose = true
				showProgress = false // Disable progress in verbose mode
			case "--workers":
				if i+1 < len(os.Args) {
					if _, err := fmt.Sscanf(os.Args[i+1], "%d", &workers); err != nil {
						log.Fatalf("Invalid worker count: %s", os.Args[i+1])
					}
					i++ // Skip the next argument since it's the worker count
				}
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			}
		}
		
		// Process clean (duplicate removal)
		if err := processClean(targetPath, dryRun, dryRun1, verbose, workers, showProgress); err != nil {
			log.Fatal(err)
		}
		
		// Clean up empty directories after cleaning
		cleanupEmptyDirectoriesIfNeeded("clean", targetPath, dryRun, -1)
		
	case "merge":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./photo-metadata-editor merge /source/path /target/path [--workers N] [--dry-run] [--dry-run1] [--progress]")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		targetPath := os.Args[3]
		
		// Check for incorrectly formatted dry-run arguments
		for i := 4; i < len(os.Args); i++ {
			arg := strings.ToLower(os.Args[i])
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" && arg != "--dry-run1" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run' or '--dry-run1' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		workers := 4 // Default worker count
		dryRun := false
		dryRun1 := false
		showProgress := true // Default to showing progress
		for i := 4; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--workers":
				if i+1 < len(os.Args) {
					if _, err := fmt.Sscanf(os.Args[i+1], "%d", &workers); err != nil {
						log.Fatalf("Invalid worker count: %s", os.Args[i+1])
					}
					i++ // Skip the next argument since it's the worker count
				}
			case "--dry-run":
				dryRun = true
			case "--dry-run1":
				dryRun1 = true
				dryRun = true // dry-run1 implies dry-run
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			}
		}
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Check if target path exists
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			log.Fatalf("Target path does not exist: %s", targetPath)
		}
		
		// Process merge
		if err := processMerge(sourcePath, targetPath, workers, dryRun, dryRun1, showProgress); err != nil {
			log.Fatal(err)
		}
		
	case "summary":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ./photo-metadata-editor summary /source/path")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Process summary
		if err := processSummary(sourcePath); err != nil {
			log.Fatal(err)
		}
		
	case "report":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./photo-metadata-editor report <type> /source/path [--save] [--progress] [--verbose]")
			fmt.Println("Types: summary, duplicates, stats")
			os.Exit(1)
		}
		
		reportTypeStr := os.Args[2]
		sourcePath := os.Args[3]
		
		// Parse report type
		var reportType ReportType
		switch reportTypeStr {
		case "summary":
			reportType = ReportTypeSummary
		case "duplicates":
			reportType = ReportTypeDuplicates
		case "stats":
			reportType = ReportTypeStats
		default:
			fmt.Printf("Invalid report type: %s\n", reportTypeStr)
			fmt.Println("Valid types: summary, duplicates, stats")
			os.Exit(1)
		}
		
		// Parse additional flags
		var saveFile bool
		var showProgress = true
		var verbose bool
		
		for i := 4; i < len(os.Args); i++ {
			arg := os.Args[i]
			switch arg {
			case "--save":
				saveFile = true
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			case "--verbose":
				verbose = true
			}
		}
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Configure report
		config := ReportConfig{
			GenerateFile:  saveFile,
			ShowProgress:  showProgress,
			VerboseOutput: verbose,
			DateFormat:    "2006-01-02",
		}
		
		// Generate report
		if err := processReport(sourcePath, reportType, config); err != nil {
			log.Fatal(err)
		}
		
	default:
		showUsage()
		os.Exit(1)
	}
}

// processSummary analyzes the source directory and shows what remains unprocessed
func processSummary(sourcePath string) error {
	fmt.Printf("📋 Source Directory Summary\n")
	fmt.Printf("🔍 Analyzing: %s\n\n", sourcePath)

	// Counters for different file types and categories
	var totalFiles int
	var totalPhotos int
	var totalVideos int
	var gpsPhotos int
	var gpsVideos int
	var nonGPSPhotos int
	var nonGPSVideos int
	var unsupportedFiles int
	
	// File extension breakdown
	photoExtensions := make(map[string]int)
	videoExtensions := make(map[string]int)
	unsupportedExtensions := make(map[string]int)
	
	// Date extraction stats
	var filesWithDates int
	var filesWithoutDates int
	dateRanges := make(map[string]int) // YYYY-MM format
	
	// Directory structure analysis
	subdirs := make(map[string]int) // relative path -> file count

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		totalFiles++
		
		// Get file extension
		ext := strings.ToLower(filepath.Ext(path))
		
		// Get subdirectory
		relPath, err := filepath.Rel(sourcePath, path)
		if err == nil {
			subDir := filepath.Dir(relPath)
			if subDir == "." {
				subDir = "root"
			}
			subdirs[subDir]++
		}

		// Classify file type
		if isPhotoFile(path) {
			totalPhotos++
			photoExtensions[ext]++
			
			// Check for GPS data
			_, _, err := extractGPSCoordinates(path)
			if err == nil {
				gpsPhotos++
			} else {
				nonGPSPhotos++
			}
			
			// Try to extract date
			filename := filepath.Base(path)
			if _, err := extractDateFromFilename(filename); err == nil {
				filesWithDates++
				// Extract year-month for date range analysis
				if date, err := extractDateFromFilename(filename); err == nil && len(date) >= 7 {
					yearMonth := date[:7] // YYYY-MM
					dateRanges[yearMonth]++
				}
			} else {
				filesWithoutDates++
			}
			
		} else if isVideoFile(path) {
			totalVideos++
			videoExtensions[ext]++
			
			// Check for GPS data
			_, _, err := extractGPSCoordinates(path)
			if err == nil {
				gpsVideos++
			} else {
				nonGPSVideos++
			}
			
			// Try to extract date
			filename := filepath.Base(path)
			if _, err := extractDateFromFilename(filename); err == nil {
				filesWithDates++
				// Extract year-month for date range analysis
				if date, err := extractDateFromFilename(filename); err == nil && len(date) >= 7 {
					yearMonth := date[:7] // YYYY-MM
					dateRanges[yearMonth]++
				}
			} else {
				filesWithoutDates++
			}
			
		} else {
			unsupportedFiles++
			unsupportedExtensions[ext]++
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to analyze directory: %v", err)
	}

	// Display summary
	fmt.Printf("📊 File Type Summary:\n")
	fmt.Printf("  📷 Photos: %d\n", totalPhotos)
	fmt.Printf("  🎥 Videos: %d\n", totalVideos)
	fmt.Printf("  ❌ Unsupported: %d\n", unsupportedFiles)
	fmt.Printf("  📁 Total files: %d\n\n", totalFiles)

	fmt.Printf("🗺️  GPS Processing Status:\n")
	fmt.Printf("  📷 Photos with GPS: %d (can be processed with 'process')\n", gpsPhotos)
	fmt.Printf("  📷 Photos without GPS: %d (need 'datetime' matching)\n", nonGPSPhotos)
	fmt.Printf("  🎥 Videos with GPS: %d (can be processed with 'process')\n", gpsVideos)
	fmt.Printf("  🎥 Videos without GPS: %d (need 'datetime' matching)\n", nonGPSVideos)
	fmt.Printf("\n")

	fmt.Printf("📅 Date Extraction Status:\n")
	fmt.Printf("  ✅ Files with extractable dates: %d\n", filesWithDates)
	fmt.Printf("  ❌ Files without extractable dates: %d\n", filesWithoutDates)
	fmt.Printf("\n")

	if len(dateRanges) > 0 {
		fmt.Printf("📆 Date Ranges Found:\n")
		for yearMonth, count := range dateRanges {
			fmt.Printf("  %s: %d files\n", yearMonth, count)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("📂 Directory Structure:\n")
	for subDir, count := range subdirs {
		if subDir == "root" {
			fmt.Printf("  📁 (root): %d files\n", count)
		} else {
			fmt.Printf("  📁 %s: %d files\n", subDir, count)
		}
	}
	fmt.Printf("\n")

	if totalPhotos > 0 {
		fmt.Printf("📷 Photo Extensions:\n")
		for ext, count := range photoExtensions {
			fmt.Printf("  %s: %d files\n", ext, count)
		}
		fmt.Printf("\n")
	}

	if totalVideos > 0 {
		fmt.Printf("🎥 Video Extensions:\n")
		for ext, count := range videoExtensions {
			fmt.Printf("  %s: %d files\n", ext, count)
		}
		fmt.Printf("\n")
	}

	if unsupportedFiles > 0 {
		fmt.Printf("❌ Unsupported Extensions:\n")
		for ext, count := range unsupportedExtensions {
			if ext == "" {
				fmt.Printf("  (no extension): %d files\n", count)
			} else {
				fmt.Printf("  %s: %d files\n", ext, count)
			}
		}
		fmt.Printf("\n")
	}

	// Recommendations
	fmt.Printf("💡 Recommendations:\n")
	if gpsPhotos > 0 || gpsVideos > 0 {
		fmt.Printf("  1. Run 'process' command first for %d files with GPS data\n", gpsPhotos+gpsVideos)
	}
	if nonGPSPhotos > 0 || nonGPSVideos > 0 {
		fmt.Printf("  2. Run 'datetime' command for %d files without GPS data\n", nonGPSPhotos+nonGPSVideos)
	}
	if filesWithoutDates > 0 {
		fmt.Printf("  3. %d files have no extractable dates and may need manual organization\n", filesWithoutDates)
	}
	if unsupportedFiles > 0 {
		fmt.Printf("  4. %d unsupported files will be ignored during processing\n", unsupportedFiles)
	}

	return nil
}

func showUsage() {
	fmt.Println("📸 Photo Metadata Editor - High Performance Concurrent Version")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  ./photo-metadata-editor process /source/path /destination/path [--workers N] [--dry-run] [--dry-run1] [--progress]")
	fmt.Println("  ./photo-metadata-editor datetime /source/path /destination/path [--workers N] [--dry-run] [--dry-run1] [--progress]")
	fmt.Println("  ./photo-metadata-editor fallback /source/path /destination/path [--workers N] [--dry-run] [--dry-run1] [--progress]")
	fmt.Println("  ./photo-metadata-editor clean /target/path [--dry-run] [--dry-run1] [--verbose] [--workers N] [--progress]")
	fmt.Println("  ./photo-metadata-editor merge /source/path /target/path [--workers N] [--dry-run] [--dry-run1] [--progress]")
	fmt.Println("  ./photo-metadata-editor summary /source/path")
	fmt.Println("  ./photo-metadata-editor report <type> /source/path [--save] [--progress] [--verbose]")
	fmt.Println()
	fmt.Println("Report Types:")
	fmt.Println("  summary      Comprehensive directory analysis with processing status")
	fmt.Println("  duplicates   Find and analyze duplicate files with quality scoring")  
	fmt.Println("  stats        General file statistics and extension breakdown")
	fmt.Println()
	fmt.Println("Performance Options:")
	fmt.Println("  --workers N    Number of concurrent workers (1-16, default: 4)")
	fmt.Println("               Higher values process more files simultaneously")
	fmt.Println("               Lower values reduce system load and memory usage")
	fmt.Println("  --progress     Show enhanced progress bar (default: true)")
	fmt.Println("  --no-progress  Disable progress bar display")
	fmt.Println()
	fmt.Println("Process Features:")
	fmt.Println("  - 🚀 Concurrent processing with configurable worker pools")
	fmt.Println("  - 🔒 Thread-safe file operations with intelligent locking")
	fmt.Println("  - 📊 Enhanced progress bars with visual indicators and ETA")
	fmt.Println("  - ⏹️  Graceful cancellation (Ctrl+C) with cleanup")
	fmt.Println("  - 🔍 --dry-run mode for safe preview without moving files")
	fmt.Println("  - 🔍 --dry-run1 mode for quick overview (1 file per type per directory)")
	fmt.Println("  - 📍 Extracts GPS location data from photos and videos")
	fmt.Println("  - 📁 Photos organized in YEAR/COUNTRY/CITY structure")
	fmt.Println("  - 🎥 Videos organized in VIDEO-FILES/YEAR/COUNTRY/CITY structure")
	fmt.Println("  - 🔄 Smart duplicate handling with counter suffixes")
	fmt.Println()
	fmt.Println("DateTime Features:")
	fmt.Println("  - 🔄 Concurrent date-based file matching for photos and videos")
	fmt.Println("  - 📊 Enhanced progress bars with visual feedback")
	fmt.Println("  - 🔍 --dry-run mode for safe preview without moving files")
	fmt.Println("  - 🔍 --dry-run1 mode for quick overview (1 file per type per directory)")
	fmt.Println("  - 🗃️  Uses processed photos as location database")
	fmt.Println("  - 🎥 Video files organized in VIDEO-FILES/YYYY/COUNTRY/CITY")
	fmt.Println("  - 📷 Photo files placed in regular YYYY/COUNTRY/CITY structure")
	fmt.Println("  - ⏱️  Temporal proximity matching (±3 days)")
	fmt.Println()
	fmt.Println("Fallback Features:")
	fmt.Println("  - 📅 Date-based organization for files without location data")
	fmt.Println("  - 📁 Organizes files into YYYY/Month directory structure")
	fmt.Println("  - 🔄 Concurrent processing with configurable worker pools")
	fmt.Println("  - 📊 Enhanced progress bars with visual feedback")
	fmt.Println("  - 🔍 --dry-run mode for safe preview without moving files")
	fmt.Println("  - 🔍 --dry-run1 mode for quick overview (1 file per type per directory)")
	fmt.Println("  - 📷 Simple YYYY-MM-DD.ext filename format")
	fmt.Println("  - 🎥 Videos organized in VIDEO-FILES/YYYY/Month structure")
	fmt.Println()
	fmt.Println("Clean Features:")
	fmt.Println("  - ⚡ High-speed duplicate detection using SHA-256")
	fmt.Println("  - 🧠 Intelligent file prioritization")
	fmt.Println("  - 🔒 Safe concurrent duplicate removal")
	fmt.Println("  - 📊 Enhanced progress bars (disabled in --verbose mode)")
	fmt.Println("  - 🔍 --dry-run mode for safe preview")
	fmt.Println("  - 🔍 --dry-run1 mode for quick summary (samples first 3 duplicate groups)")
	fmt.Println("  - 📝 --verbose mode for detailed logging")
	fmt.Println()
	fmt.Println("Merge Features:")
	fmt.Println("  - 🔀 Merge photos from source into target using YEAR/COUNTRY/CITY structure")
	fmt.Println("  - 🚀 Concurrent processing with configurable worker pools")
	fmt.Println("  - 📊 Enhanced progress bars with visual feedback")
	fmt.Println("  - 🔍 --dry-run mode for safe preview without copying files")
	fmt.Println("  - 🔍 --dry-run1 mode for quick overview (1 file per type per directory)")
	fmt.Println("  - 📍 GPS-based location detection or intelligent inference")
	fmt.Println("  - 🔄 Smart duplicate detection to avoid overwriting existing files")
	fmt.Println("  - 🎥 Videos organized in VIDEO-FILES/YEAR/COUNTRY/CITY structure")
	fmt.Println("  - 📷 Photos organized in YEAR/COUNTRY/CITY structure")
	fmt.Println("  - 💾 Copies files (preserves originals in source)")
	fmt.Println()
	fmt.Println("Performance Tips:")
	fmt.Println("  - Use --workers 8-16 for large photo collections")
	fmt.Println("  - Use --workers 1-4 for slower storage (USB drives)")
	fmt.Println("  - Press Ctrl+C for graceful cancellation")
	fmt.Println("  - Monitor system resources during processing")
	fmt.Println()
}

func processPhotos(sourcePath, destPath string) error {
	return processPhotosConcurrently(sourcePath, destPath, 1, false, false, true)
}

func processPhotosConcurrently(sourcePath, destPath string, workers int, dryRun bool, dryRun1 bool, showProgress bool) error {
	fmt.Printf("🔍 Scanning media files from: %s\n", sourcePath)
	fmt.Printf("📁 Destination: %s\n", destPath)
	
	if dryRun {
		if dryRun1 {
			fmt.Println("🔍 DRY RUN1 MODE - Sample only 1 file per type per directory")
		} else {
			fmt.Println("🔍 DRY RUN MODE - No files will be moved")
		}
		fmt.Println()
	}
	
	// Collect all media files
	var jobs []WorkJob
	var err error
	
	if dryRun1 {
		// For dry-run1, sample 1 photo and 1 video per directory
		jobs, err = collectSampleFiles(sourcePath, destPath, dryRun)
	} else {
		// Normal collection - all files
		err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			// Skip directories
			if info.IsDir() {
				return nil
			}
			
			// Check if it's a media file (photo or video)
			if !isMediaFile(path) {
				return nil
			}
			
			// Add to jobs list
			jobs = append(jobs, WorkJob{
				PhotoPath: path,
				DestPath:  destPath,
				JobType:   "process",
				DryRun:    dryRun,
			})
			
			return nil
		})
	}
	
	if err != nil {
		return err
	}
	
	if len(jobs) == 0 {
		fmt.Println("📭 No media files found to process")
		return nil
	}
	
	photoCount := 0
	videoCount := 0
	for _, job := range jobs {
		if isVideoFile(job.PhotoPath) {
			videoCount++
		} else {
			photoCount++
		}
	}
	
	fmt.Printf("📝 Found %d media files to process (%d photos, %d videos)\n", len(jobs), photoCount, videoCount)
	
	// Process jobs concurrently
	return processJobsConcurrentlyWithProgress(jobs, workers, showProgress)
}

func isPhotoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	
	// Common photo/image formats that can contain GPS metadata
	supportedFormats := map[string]bool{
		".jpg":   true,
		".jpeg":  true,
		".png":   true,
		".heic":  true,
		".heif":  true,
		".tiff":  true,
		".tif":   true,
		".dng":   true,  // RAW format
		".cr2":   true,  // Canon RAW
		".nef":   true,  // Nikon RAW
		".arw":   true,  // Sony RAW
		".orf":   true,  // Olympus RAW
		".rw2":   true,  // Panasonic RAW
		".raf":   true,  // Fuji RAW
		".srw":   true,  // Samsung RAW
		".pef":   true,  // Pentax RAW
		".3fr":   true,  // Hasselblad RAW
		".fff":   true,  // Imacon RAW
		".iiq":   true,  // Phase One RAW
		".k25":   true,  // Kodak RAW
		".kdc":   true,  // Kodak RAW
		".dcr":   true,  // Kodac RAW
		".mrw":   true,  // Minolta RAW
		".raw":   true,  // Generic RAW
	}
	
	return supportedFormats[ext]
}

func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	
	// Common video formats
	supportedFormats := map[string]bool{
		".mp4":   true,
		".mov":   true,
		".avi":   true,
		".mkv":   true,
		".wmv":   true,
		".flv":   true,
		".webm":  true,
		".m4v":   true,
		".3gp":   true,
		".3g2":   true,
		".mts":   true,
		".m2ts":  true,
		".ts":    true,
		".mxf":   true,
		".asf":   true,
		".rm":    true,
		".rmvb":  true,
		".vob":   true,
		".ogv":   true,
		".dv":    true,
		".qt":    true,
	}
	
	return supportedFormats[ext]
}

func isMediaFile(path string) bool {
	return isPhotoFile(path) || isVideoFile(path)
}

func processPhoto(photoPath, destBasePath string) error {
	return processMediaFile(photoPath, destBasePath, false)
}

func processPhotoWithDryRun(photoPath, destBasePath string, dryRun bool) error {
	return processMediaFile(photoPath, destBasePath, dryRun)
}

func processMediaFile(mediaPath, destBasePath string, dryRun bool) error {
	// Use file locks to prevent race conditions (skip in dry run for performance)
	if dryRun {
		return processMediaFileInternal(mediaPath, destBasePath, dryRun)
	}
	
	return WithFilelocks(destBasePath, mediaPath, func() error {
		return processMediaFileInternal(mediaPath, destBasePath, dryRun)
	})
}

func processMediaFileInternal(mediaPath, destBasePath string, dryRun bool) error {
	// Determine file type for display
	var fileType string
	var fileIcon string
	if isVideoFile(mediaPath) {
		fileType = "video"
		fileIcon = "🎥"
	} else {
		fileType = "photo"
		fileIcon = "📷"
	}
	
	if dryRun {
		fmt.Printf("%s [DRY RUN] Processing %s: %s\n", fileIcon, fileType, filepath.Base(mediaPath))
	} else {
		fmt.Printf("%s Processing %s: %s\n", fileIcon, fileType, filepath.Base(mediaPath))
	}
	
	// Extract GPS coordinates
	lat, lon, err := extractGPSCoordinates(mediaPath)
	if err != nil {
		return &NoGPSError{File: mediaPath, Err: err}
	}
	
	// Get location from coordinates
	location, err := getLocationFromCoordinates(lat, lon)
	if err != nil {
		return fmt.Errorf("failed to get location for %s: %v", filepath.Base(mediaPath), err)
	}
	
	fmt.Printf("📍 Location: %s (%.6f, %.6f)\n", location, lat, lon)
	
	// Extract date from media file
	date, err := extractPhotoDate(mediaPath)
	if err != nil {
		return fmt.Errorf("failed to extract date from %s: %v", filepath.Base(mediaPath), err)
	}
	
	// Parse location into country and city
	country, city, err := parseLocation(location)
	if err != nil {
		// Skip prompting in dry run mode
		if dryRun {
			country = "unknown-country"
			city = "unknown-city"
		} else {
			// Prompt for missing country/city information
			country, city, err = promptForLocation(location)
			if err != nil {
				return fmt.Errorf("failed to get location information: %v", err)
			}
		}
	}
	
	// Generate new filename and directory structure using destination base path
	newFilename := fmt.Sprintf("%s-%s%s", 
		date.Format("2006-01-02"), 
		city, 
		filepath.Ext(mediaPath))
	
	// Smart directory structure - check if destination already ends with the year
	year := date.Format("2006")
	var newDir string
	
	// Check if destination path already ends with the year
	destBase := filepath.Base(destBasePath)
	
	// For video files, place in VIDEO-FILES subdirectory
	if isVideoFile(mediaPath) {
		if destBase == year {
			// Destination already ends with year (e.g., "/tmp/2025"), so add VIDEO-FILES/country/city
			newDir = filepath.Join(destBasePath, "VIDEO-FILES", country, city)
		} else {
			// Destination doesn't end with year, so add VIDEO-FILES/year/country/city
			newDir = filepath.Join(destBasePath, "VIDEO-FILES", year, country, city)
		}
	} else {
		// For photo files, use regular structure
		if destBase == year {
			// Destination already ends with year (e.g., "/tmp/2025"), so just add country/city
			newDir = filepath.Join(destBasePath, country, city)
		} else {
			// Destination doesn't end with year, so add full structure
			newDir = filepath.Join(destBasePath, year, country, city)
		}
	}
	
	newPath := filepath.Join(newDir, newFilename)
	
	// In dry run mode, just show what would happen
	if dryRun {
		// Create directory structure if it doesn't exist (for simulation)
		return WithBatchLocks([]string{newDir}, func() error {
			// Simulate duplicate handling
			finalPath := newPath
			counter := 1
			for {
				if _, err := os.Stat(finalPath); os.IsNotExist(err) {
					break // File doesn't exist, we can use this path
				}
				
				// File exists, create a new name with counter
				ext := filepath.Ext(newFilename)
				base := strings.TrimSuffix(newFilename, ext)
				duplicateFilename := fmt.Sprintf("%s-%d%s", base, counter, ext)
				finalPath = filepath.Join(newDir, duplicateFilename)
				counter++
				
				if counter > 1000 {
					return fmt.Errorf("too many duplicate filenames, stopping at counter %d", counter)
				}
			}
			
			if isVideoFile(mediaPath) {
				fmt.Printf("✅ [DRY RUN] Video would be moved to: %s\n", finalPath)
			} else {
				fmt.Printf("✅ [DRY RUN] Photo would be moved to: %s\n", finalPath)
			}
			return nil
		})
	}
	
	// Create directory structure if it doesn't exist (with additional lock)
	return WithBatchLocks([]string{newDir}, func() error {
		if err := os.MkdirAll(newDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", newDir, err)
		}
		
		// Handle duplicate filenames by adding a counter
		finalPath := newPath
		counter := 1
		for {
			if _, err := os.Stat(finalPath); os.IsNotExist(err) {
				break // File doesn't exist, we can use this path
			}
			
			// File exists, create a new name with counter
			ext := filepath.Ext(newFilename)
			base := strings.TrimSuffix(newFilename, ext)
			duplicateFilename := fmt.Sprintf("%s-%d%s", base, counter, ext)
			finalPath = filepath.Join(newDir, duplicateFilename)
			counter++
			
			if counter > 1000 {
				return fmt.Errorf("too many duplicate filenames, stopping at counter %d", counter)
			}
		}
		
		// Move/rename the file
		if err := os.Rename(mediaPath, finalPath); err != nil {
			return fmt.Errorf("failed to move file from %s to %s: %v", mediaPath, finalPath, err)
		}
		
		if isVideoFile(mediaPath) {
			fmt.Printf("✅ Video moved to: %s\n", finalPath)
		} else {
			fmt.Printf("✅ Photo moved to: %s\n", finalPath)
		}
		return nil
	})
}

// collectSampleFiles collects a sample of files for dry-run1 mode
// Samples 1 photo and 1 video per directory to provide an overview
func collectSampleFiles(sourcePath, destPath string, dryRun bool) ([]WorkJob, error) {
	// Map to track files by directory and type
	dirFiles := make(map[string]map[string][]string) // directory -> {photos: [], videos: []}
	
	// Collect all files grouped by directory and type
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check if it's a media file
		if !isMediaFile(path) {
			return nil
		}
		
		// Get directory path
		dir := filepath.Dir(path)
		
		// Initialize directory map if needed
		if dirFiles[dir] == nil {
			dirFiles[dir] = map[string][]string{
				"photos": []string{},
				"videos": []string{},
			}
		}
		
		// Add to appropriate type list
		if isVideoFile(path) {
			dirFiles[dir]["videos"] = append(dirFiles[dir]["videos"], path)
		} else {
			dirFiles[dir]["photos"] = append(dirFiles[dir]["photos"], path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Sample files: 1 photo and 1 video per directory (if available)
	var jobs []WorkJob
	totalPhotos := 0
	totalVideos := 0
	directoriesWithPhotos := 0
	directoriesWithVideos := 0
	
	for _, files := range dirFiles {
		// Sample 1 photo per directory
		if len(files["photos"]) > 0 {
			photoPath := files["photos"][0] // Take first photo
			jobs = append(jobs, WorkJob{
				PhotoPath: photoPath,
				DestPath:  destPath,
				JobType:   "process",
				DryRun:    dryRun,
			})
			totalPhotos++
			directoriesWithPhotos++
		}
		
		// Sample 1 video per directory
		if len(files["videos"]) > 0 {
			videoPath := files["videos"][0] // Take first video
			jobs = append(jobs, WorkJob{
				PhotoPath: videoPath,
				DestPath:  destPath,
				JobType:   "process",
				DryRun:    dryRun,
			})
			totalVideos++
			directoriesWithVideos++
		}
	}
	
	fmt.Printf("📋 Sampled %d files from %d directories (%d photos from %d dirs, %d videos from %d dirs)\n", 
		len(jobs), len(dirFiles), totalPhotos, directoriesWithPhotos, totalVideos, directoriesWithVideos)
	
	return jobs, nil
}