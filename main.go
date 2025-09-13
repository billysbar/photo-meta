package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
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

// confirmOperation prompts the user to confirm the operation before proceeding
func confirmOperation(command string, sourcePath, destPath string, dryRun bool, dryRunSampleSize int) bool {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  📋 OPERATION CONFIRMATION\n")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("Command: %s\n", strings.ToUpper(command))
	
	if sourcePath != "" {
		fmt.Printf("Source:  %s\n", sourcePath)
	}
	if destPath != "" {
		fmt.Printf("Target:  %s\n", destPath)
	}
	
	// Check if this is a read-only command
	isReadOnly := command == "summary" || command == "report"
	
	if dryRun {
		if dryRunSampleSize > 0 {
			fmt.Printf("Mode:    🔍 DRY RUN (Sample: %d files per type per directory)\n", dryRunSampleSize)
		} else {
			fmt.Printf("Mode:    🔍 DRY RUN (Preview only - no files will be modified)\n")
		}
	} else if isReadOnly {
		fmt.Printf("Mode:    📖 READ-ONLY (No files will be modified)\n")
	} else {
		fmt.Printf("Mode:    ⚠️  LIVE RUN (Files will be moved/modified/deleted)\n")
	}
	
	fmt.Println("═══════════════════════════════════════════")
	fmt.Print("Continue with this operation? [y/N]: ")
	
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	
	return response == "y" || response == "yes"
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
			fmt.Println("Usage: ./photo-meta process /source/path /destination/path [--workers N] [--dry-run [N]] [--progress] [--info] [--resume FILE]")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		destPath := os.Args[3]
		
		// Check for incorrectly formatted dry-run arguments
		for i := 4; i < len(os.Args); i++ {
			arg := strings.ToLower(os.Args[i])
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run [N]' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		workers := 4 // Default worker count
		dryRun := false
		dryRunSampleSize := 0
		showProgress := true // Default to showing progress
		generateInfo := false // Generate info_ directory summary file
		resumeFromFile := "" // Progress file to resume from
		
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
				dryRunSampleSize = 0 // Process all files for preview
				// Check if next argument is a number
				if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "--") {
					if size, err := strconv.Atoi(os.Args[i+1]); err == nil && size > 0 {
						dryRunSampleSize = size
						i++ // Skip the next argument since it's the sample size
					}
				}
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			case "--info":
				generateInfo = true
			case "--resume":
				if i+1 < len(os.Args) {
					resumeFromFile = os.Args[i+1]
					i++ // Skip the next argument since it's the resume file
				} else {
					log.Fatalf("--resume requires a progress file path")
				}
			}
		}
		
		// Check for existing progress files if not resuming explicitly and not in dry-run mode
		if resumeFromFile == "" && !dryRun {
			if existingFiles, err := FindExistingProgress("process", sourcePath, destPath); err == nil && len(existingFiles) > 0 {
				fmt.Printf("🔄 Found existing progress file(s):\n")
				for i, file := range existingFiles {
					fmt.Printf("  %d. %s\n", i+1, filepath.Base(file))
				}
				fmt.Printf("\nWould you like to resume from an existing progress file? [y/N]: ")
				
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				
				if response == "y" || response == "yes" {
					if len(existingFiles) == 1 {
						resumeFromFile = existingFiles[0]
					} else {
						fmt.Printf("Enter file number (1-%d): ", len(existingFiles))
						var choice int
						if _, err := fmt.Scanf("%d", &choice); err == nil && choice >= 1 && choice <= len(existingFiles) {
							resumeFromFile = existingFiles[choice-1]
						}
					}
				}
			}
		}
		
		// Ask for user confirmation
		if !confirmOperation("process", sourcePath, destPath, dryRun, dryRunSampleSize) {
			fmt.Println("❌ Operation cancelled by user.")
			os.Exit(0)
		}
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Create destination path if it doesn't exist
		if err := os.MkdirAll(destPath, 0755); err != nil {
			log.Fatalf("Failed to create destination path: %v", err)
		}
		
		// Process photos concurrently with progress persistence
		if err := processPhotosWithProgress(sourcePath, destPath, workers, dryRun, dryRunSampleSize, showProgress, generateInfo, resumeFromFile); err != nil {
			log.Fatal(err)
		}
		
		// Clean up empty directories after processing
		cleanupEmptyDirectoriesIfNeeded("process", sourcePath, dryRun, -1) // -1 means always run cleanup
		
		// Generate info directory summary file if requested
		if generateInfo && !dryRun {
			fmt.Printf("\n📋 Generating PhotoXX-style directory summary...\n")
			if err := generateInfoDirectorySummary(destPath, ""); err != nil {
				fmt.Printf("⚠️  Warning: Failed to generate info directory summary: %v\n", err)
			} else {
				fmt.Printf("✅ Info directory summary generated successfully\n")
			}
		}
		
	case "organize":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./photo-meta organize /source/path /destination/path [--workers N] [--dry-run [N]] [--progress] [--info]")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		destPath := os.Args[3]
		
		// Check for incorrectly formatted dry-run arguments
		for i := 4; i < len(os.Args); i++ {
			arg := strings.ToLower(os.Args[i])
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run [N]' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		workers := 4 // Default worker count
		dryRun := false
		dryRunSampleSize := 0
		showProgress := true // Default to showing progress
		generateInfo := false // Generate info_ directory summary file
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
				dryRunSampleSize = 0 // Process all files for preview
				// Check if next argument is a number
				if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "--") {
					if size, err := strconv.Atoi(os.Args[i+1]); err == nil && size > 0 {
						dryRunSampleSize = size
						i++ // Skip the next argument since it's the sample size
					}
				}
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			case "--info":
				generateInfo = true
			}
		}
		
		// Ask for user confirmation
		if !confirmOperation("organize", sourcePath, destPath, dryRun, dryRunSampleSize) {
			fmt.Println("❌ Operation cancelled by user.")
			os.Exit(0)
		}
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Create destination path if it doesn't exist
		if err := os.MkdirAll(destPath, 0755); err != nil {
			log.Fatalf("Failed to create destination path: %v", err)
		}
		
		// Process location-based organization
		if err := processOrganizeByLocation(sourcePath, destPath, dryRun, dryRunSampleSize, showProgress); err != nil {
			log.Fatal(err)
		}
		
		// Clean up empty directories after organize processing
		cleanupEmptyDirectoriesIfNeeded("organize", sourcePath, dryRun, -1)
		
		// Generate info directory summary file if requested
		if generateInfo && !dryRun {
			fmt.Printf("\n📋 Generating PhotoXX-style directory summary...\n")
			if err := generateInfoDirectorySummary(destPath, ""); err != nil {
				fmt.Printf("⚠️  Warning: Failed to generate info directory summary: %v\n", err)
			} else {
				fmt.Printf("✅ Info directory summary generated successfully\n")
			}
		}
		
	case "fallback":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./photo-meta fallback /source/path /destination/path [--workers N] [--dry-run [N]] [--progress] [--info]")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		destPath := os.Args[3]
		
		// Check for incorrectly formatted dry-run arguments
		for i := 4; i < len(os.Args); i++ {
			arg := strings.ToLower(os.Args[i])
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run [N]' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		workers := 4 // Default worker count
		dryRun := false
		dryRunSampleSize := 0
		showProgress := true // Default to showing progress
		generateInfo := false // Generate info_ directory summary file
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
				dryRunSampleSize = 0 // Process all files for preview
				// Check if next argument is a number
				if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "--") {
					if size, err := strconv.Atoi(os.Args[i+1]); err == nil && size > 0 {
						dryRunSampleSize = size
						i++ // Skip the next argument since it's the sample size
					}
				}
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			case "--info":
				generateInfo = true
			}
		}
		
		// Ask for user confirmation
		if !confirmOperation("fallback", sourcePath, destPath, dryRun, dryRunSampleSize) {
			fmt.Println("❌ Operation cancelled by user.")
			os.Exit(0)
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
		if err := processFallbackOrganization(sourcePath, destPath, dryRun, dryRunSampleSize, showProgress); err != nil {
			log.Fatal(err)
		}
		
		// Clean up empty directories after fallback processing
		cleanupEmptyDirectoriesIfNeeded("fallback", sourcePath, dryRun, -1)
		
		// Generate info directory summary file if requested
		if generateInfo && !dryRun {
			fmt.Printf("\n📋 Generating PhotoXX-style directory summary...\n")
			if err := generateInfoDirectorySummary(destPath, ""); err != nil {
				fmt.Printf("⚠️  Warning: Failed to generate info directory summary: %v\n", err)
			} else {
				fmt.Printf("✅ Info directory summary generated successfully\n")
			}
		}
		
	case "datetime":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./photo-meta datetime /source/path /destination/path [--workers N] [--dry-run [N]] [--progress] [--info] [--reset-db]")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		destPath := os.Args[3]
		
		// Check for incorrectly formatted dry-run arguments
		for i := 4; i < len(os.Args); i++ {
			arg := strings.ToLower(os.Args[i])
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run [N]' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		workers := 4 // Default worker count
		dryRun := false
		dryRunSampleSize := 0
		showProgress := true // Default to showing progress
		generateInfo := false // Generate info_ directory summary file
		resetDB := false // Reset GPS cache database
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
				dryRunSampleSize = 0 // Process all files for preview
				// Check if next argument is a number
				if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "--") {
					if size, err := strconv.Atoi(os.Args[i+1]); err == nil && size > 0 {
						dryRunSampleSize = size
						i++ // Skip the next argument since it's the sample size
					}
				}
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			case "--info":
				generateInfo = true
			case "--reset-db":
				resetDB = true
			}
		}
		
		// Ask for user confirmation
		if !confirmOperation("datetime", sourcePath, destPath, dryRun, dryRunSampleSize) {
			fmt.Println("❌ Operation cancelled by user.")
			os.Exit(0)
		}
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Check if destination path exists
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			log.Fatalf("Destination path does not exist: %s", destPath)
		}
		
		// Initialize GPS cache
		if err := InitGPSCache(); err != nil {
			log.Fatalf("Failed to initialize GPS cache: %v", err)
		}
		defer CloseGPSCache()
		
		// Handle database reset if requested
		if resetDB {
			cache := GetGPSCache()
			if cache != nil {
				fmt.Println("🗑️ Clearing GPS cache database...")
				if err := cache.Clear(); err != nil {
					log.Fatalf("Failed to clear GPS cache: %v", err)
				}
				fmt.Println("✅ GPS cache database cleared")
			}
		}
		
		// Process datetime matching
		if err := processDateTimeMatching(sourcePath, destPath, dryRun, dryRunSampleSize, showProgress); err != nil {
			log.Fatal(err)
		}
		
		// Clean up empty directories after datetime processing
		cleanupEmptyDirectoriesIfNeeded("datetime", sourcePath, dryRun, -1)
		
		// Generate info directory summary file if requested
		if generateInfo && !dryRun {
			fmt.Printf("\n📋 Generating PhotoXX-style directory summary...\n")
			if err := generateInfoDirectorySummary(destPath, ""); err != nil {
				fmt.Printf("⚠️  Warning: Failed to generate info directory summary: %v\n", err)
			} else {
				fmt.Printf("✅ Info directory summary generated successfully\n")
			}
		}
		
	case "clean":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ./photo-meta clean /target/path [--dry-run [N]] [--verbose] [--workers N] [--progress]")
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
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run [N]' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		dryRun := false
		dryRunSampleSize := 0
		verbose := false
		workers := 4 // Default worker count
		showProgress := true // Default to showing progress (unless verbose)
		for i := 3; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--dry-run":
				dryRun = true
				dryRunSampleSize = 0 // Process all files for preview
				// Check if next argument is a number
				if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "--") {
					if size, err := strconv.Atoi(os.Args[i+1]); err == nil && size > 0 {
						dryRunSampleSize = size
						i++ // Skip the next argument since it's the sample size
					}
				}
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
		
		// Ask for user confirmation
		if !confirmOperation("clean", "", targetPath, dryRun, dryRunSampleSize) {
			fmt.Println("❌ Operation cancelled by user.")
			os.Exit(0)
		}
		
		// Process clean (duplicate removal)
		if err := processClean(targetPath, dryRun, dryRunSampleSize, verbose, workers, showProgress); err != nil {
			log.Fatal(err)
		}
		
		// Clean up empty directories after cleaning
		cleanupEmptyDirectoriesIfNeeded("clean", targetPath, dryRun, -1)
		
	case "merge":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./photo-meta merge /source/path /target/path [--workers N] [--dry-run [N]] [--progress]")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		targetPath := os.Args[3]
		
		// Check for incorrectly formatted dry-run arguments
		for i := 4; i < len(os.Args); i++ {
			arg := strings.ToLower(os.Args[i])
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run [N]' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		workers := 4 // Default worker count
		dryRun := false
		dryRunSampleSize := 0
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
				dryRunSampleSize = 0 // Process all files for preview
				// Check if next argument is a number
				if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "--") {
					if size, err := strconv.Atoi(os.Args[i+1]); err == nil && size > 0 {
						dryRunSampleSize = size
						i++ // Skip the next argument since it's the sample size
					}
				}
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			}
		}
		
		// Ask for user confirmation
		if !confirmOperation("merge", sourcePath, targetPath, dryRun, dryRunSampleSize) {
			fmt.Println("❌ Operation cancelled by user.")
			os.Exit(0)
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
		if err := processMerge(sourcePath, targetPath, workers, dryRun, dryRunSampleSize, showProgress); err != nil {
			log.Fatal(err)
		}
		
	case "summary":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ./photo-meta summary /source/path")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		
		// Ask for user confirmation
		if !confirmOperation("summary", sourcePath, "", false, 0) {
			fmt.Println("❌ Operation cancelled by user.")
			os.Exit(0)
		}
		
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
			fmt.Println("Usage: ./photo-meta report <type> /source/path [--save] [--progress] [--verbose]")
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
		
		// Ask for user confirmation
		if !confirmOperation("report", sourcePath, "", false, 0) {
			fmt.Println("❌ Operation cancelled by user.")
			os.Exit(0)
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
		
	case "tiff":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ./photo-meta tiff /target/path [--dry-run [N]] [--workers N] [--progress]")
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
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run [N]' instead")
				os.Exit(1)
			}
		}

		// Parse optional flags
		workers := 1 // Default worker count for TIFF (single worker for safety)
		dryRun := false
		dryRunSampleSize := 0
		showProgress := false // Default to no progress for TIFF
		for i := 3; i < len(os.Args); i++ {
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
				dryRunSampleSize = 0 // Process all files for preview
				// Check if next argument is a number
				if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "--") {
					if size, err := strconv.Atoi(os.Args[i+1]); err == nil && size > 0 {
						dryRunSampleSize = size
						i++ // Skip the next argument since it's the sample size
					}
				}
			case "--progress":
				showProgress = true
			case "--no-progress":
				showProgress = false
			}
		}

		// Ask for user confirmation
		if !confirmOperation("tiff", "", targetPath, dryRun, dryRunSampleSize) {
			fmt.Println("❌ Operation cancelled by user.")
			os.Exit(0)
		}

		// Process tiff timestamp fixes
		if err := processTiffTimestampFix(targetPath, workers, dryRun, dryRunSampleSize, showProgress); err != nil {
			log.Fatal(err)
		}

	case "cleanup":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ./photo-meta cleanup /target/path [--dry-run [N]]")
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
			if strings.Contains(arg, "dry") && strings.Contains(arg, "run") && arg != "--dry-run" {
				fmt.Printf("Error: Invalid argument format '%s'\n", os.Args[i])
				fmt.Println("Use '--dry-run [N]' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		dryRun := false
		for i := 3; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--dry-run":
				dryRun = true
			}
		}
		
		// Ask for user confirmation
		if !confirmOperation("cleanup", "", targetPath, dryRun, 0) {
			fmt.Println("❌ Operation cancelled by user.")
			os.Exit(0)
		}
		
		// Run cleanup
		fmt.Printf("🧹 Standalone Empty Directory Cleanup\n")
		fmt.Printf("🔍 Target: %s\n", targetPath)
		
		if dryRun {
			fmt.Println("🔍 DRY RUN MODE - No directories will be removed")
		}
		fmt.Println()
		
		if err := cleanupEmptyDirectories(targetPath, dryRun); err != nil {
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

// processPhotosWithProgress processes photos with progress persistence and enhanced error handling
func processPhotosWithProgress(sourcePath, destPath string, workers int, dryRun bool, dryRunSampleSize int, showProgress bool, generateInfo bool, resumeFromFile string) error {
	var progressMgr *ProgressManager
	var err error
	
	// Initialize or load progress manager
	if resumeFromFile != "" {
		progressMgr, err = LoadProgressManager(resumeFromFile)
		if err != nil {
			return fmt.Errorf("failed to load progress file: %v", err)
		}
		progressMgr.PrintResumeSummary()
		fmt.Println()
	} else if !dryRun {
		progressMgr = NewProgressManager("process", sourcePath, destPath)
		// Clean up old progress files
		CleanupOldProgressFiles()
	}
	
	// Update progress manager settings if resuming
	if progressMgr != nil {
		progressMgr.UpdateProgress("processing", 0, workers, dryRun, dryRunSampleSize, showProgress, generateInfo)
		// Save initial state
		if err := progressMgr.SaveState(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to save initial progress state: %v\n", err)
		}
	}
	
	// Call the enhanced processing function
	err = processPhotosConcurrentlyEnhanced(sourcePath, destPath, workers, dryRun, dryRunSampleSize, showProgress, progressMgr)
	
	// Handle completion and cleanup
	if progressMgr != nil {
		if err != nil {
			progressMgr.GetState().CurrentPhase = "failed"
			progressMgr.SaveState()
			fmt.Printf("💾 Progress saved. Resume with: --resume %s\n", progressMgr.stateFile)
		} else {
			progressMgr.GetState().CurrentPhase = "complete"
			progressMgr.SaveState()
			
			// Generate info summary if requested
			if generateInfo {
				fmt.Printf("\n📋 Generating PhotoXX-style directory summary...\n")
				if infoErr := generateInfoDirectorySummary(destPath, ""); infoErr != nil {
					fmt.Printf("⚠️  Warning: Failed to generate info directory summary: %v\n", infoErr)
				} else {
					fmt.Printf("✅ Info directory summary generated successfully\n")
				}
			}
			
			// Clean up progress file on successful completion
			progressMgr.CleanupStateFile()
		}
	}
	
	return err
}

// processPhotosConcurrentlyEnhanced is an enhanced version with progress tracking and permission handling
func processPhotosConcurrentlyEnhanced(sourcePath, destPath string, workers int, dryRun bool, dryRunSampleSize int, showProgress bool, progressMgr *ProgressManager) error {
	// For now, delegate to the original function but with enhanced error handling
	// This can be expanded to integrate more deeply with progress tracking
	
	fmt.Printf("🔍 Scanning for photos and videos...\n")
	if showProgress {
		fmt.Printf("📁 Source directory: %s\n", sourcePath)
		fmt.Printf("📁 Destination directory: %s\n", destPath)
		if progressMgr != nil && len(progressMgr.state.ProcessedFiles) > 0 {
			fmt.Printf("🔄 Resuming from previous run (%d files already processed)\n", len(progressMgr.state.ProcessedFiles))
		}
	}
	
	// Call the original function for now - this integration can be enhanced further
	return processPhotosConcurrently(sourcePath, destPath, workers, dryRun, dryRunSampleSize, showProgress)
}

func showUsage() {
	fmt.Println("📸 Photo Metadata Editor - High Performance Concurrent Version")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  ./photo-meta process /source/path /destination/path [--workers N] [--dry-run [N]] [--progress] [--info] [--resume FILE]")
	fmt.Println("  ./photo-meta organize /source/path /destination/path [--workers N] [--dry-run [N]] [--progress] [--info]")
	fmt.Println("  ./photo-meta datetime /source/path /destination/path [--workers N] [--dry-run [N]] [--progress] [--info] [--reset-db]")
	fmt.Println("  ./photo-meta fallback /source/path /destination/path [--workers N] [--dry-run [N]] [--progress] [--info]")
	fmt.Println("  ./photo-meta tiff /target/path [--dry-run [N]] [--workers N] [--progress]")
	fmt.Println("  ./photo-meta clean /target/path [--dry-run [N]] [--verbose] [--workers N] [--progress]")
	fmt.Println("  ./photo-meta cleanup /target/path [--dry-run [N]]")
	fmt.Println("  ./photo-meta merge /source/path /target/path [--workers N] [--dry-run [N]] [--progress]")
	fmt.Println("  ./photo-meta summary /source/path")
	fmt.Println("  ./photo-meta report <type> /source/path [--save] [--progress] [--verbose]")
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
	fmt.Println("  --info         Generate PhotoXX-style info_ directory summary file")
	fmt.Println("  --resume FILE  Resume from a previous interrupted operation")
	fmt.Println("  --reset-db     Clear the GPS cache database (for datetime command)")
	fmt.Println()
	fmt.Println("Process Features:")
	fmt.Println("  - 🚀 Concurrent processing with configurable worker pools")
	fmt.Println("  - 🔒 Thread-safe file operations with intelligent locking")
	fmt.Println("  - 📊 Enhanced progress bars with visual indicators and ETA")
	fmt.Println("  - ⏹️  Graceful cancellation (Ctrl+C) with cleanup")
	fmt.Println("  - 🔍 --dry-run mode for safe preview without moving files")
	fmt.Println("  - 🔍 --dry-run [N] mode for quick overview (N files per type per directory)")
	fmt.Println("  - 📍 Extracts GPS location data from photos and videos")
	fmt.Println("  - 📁 Photos organized in YEAR/COUNTRY/CITY structure")
	fmt.Println("  - 🎥 Videos organized in VIDEO-FILES/YEAR/COUNTRY/CITY structure")
	fmt.Println("  - 🔄 Smart duplicate handling with counter suffixes")
	fmt.Println("  - 💾 Progress persistence with automatic resume capability")
	fmt.Println("  - 🛡️  Enhanced permission error handling with helpful suggestions")
	fmt.Println()
	fmt.Println("DateTime Features:")
	fmt.Println("  - 🔄 Concurrent date-based file matching for photos and videos")
	fmt.Println("  - 📊 Enhanced progress bars with visual feedback")
	fmt.Println("  - 🔍 --dry-run mode for safe preview without moving files")
	fmt.Println("  - 🔍 --dry-run [N] mode for quick overview (N files per type per directory)")
	fmt.Println("  - 🗃️  Uses processed photos as location database")
	fmt.Println("  - 🎥 Video files organized in VIDEO-FILES/YYYY/COUNTRY/CITY")
	fmt.Println("  - 📷 Photo files placed in regular YYYY/COUNTRY/CITY structure")
	fmt.Println("  - ⏱️  Temporal proximity matching (±3 days)")
	fmt.Println("  - 💾 GPS cache database for faster subsequent scans")
	fmt.Println("  - 🗑️  --reset-db flag to clear the GPS cache when needed")
	fmt.Println()
	fmt.Println("Organize Features:")
	fmt.Println("  - 📍 Location-based organization for files with city/country in filename")
	fmt.Println("  - 📁 Organizes files into YYYY/COUNTRY/CITY directory structure")
	fmt.Println("  - 🗄️  Intelligent location database for automatic mapping")
	fmt.Println("  - 🤔 Interactive prompts for ambiguous locations (e.g., 'scarborough')")
	fmt.Println("  - 💾 Persistent location mappings (saves user input for future files)")
	fmt.Println("  - 🔍 --dry-run mode for safe preview without moving files")
	fmt.Println("  - 🔍 --dry-run [N] mode for quick overview (N files per type per directory)")
	fmt.Println("  - 📷 Example: '2008-09-28-scarborough-2.jpg' → '/tmp/photos/2008/united-kingdom/scarborough/'")
	fmt.Println("  - 🎥 Videos organized in VIDEO-FILES/YYYY/COUNTRY/CITY structure")
	fmt.Println()
	fmt.Println("Fallback Features:")
	fmt.Println("  - 📅 Date-based organization for files without location data")
	fmt.Println("  - 📁 Organizes files into YYYY/Month directory structure")
	fmt.Println("  - 🔄 Concurrent processing with configurable worker pools")
	fmt.Println("  - 📊 Enhanced progress bars with visual feedback")
	fmt.Println("  - 🔍 --dry-run mode for safe preview without moving files")
	fmt.Println("  - 🔍 --dry-run [N] mode for quick overview (N files per type per directory)")
	fmt.Println("  - 📷 Simple YYYY-MM-DD.ext filename format")
	fmt.Println("  - 🎥 Videos organized in VIDEO-FILES/YYYY/Month structure")
	fmt.Println()
	fmt.Println("TIFF Features:")
	fmt.Println("  - 🕐 Fix midnight timestamps (00:00:00) using EXIF ModifyDate")
	fmt.Println("  - 📸 Updates both EXIF timestamps and filename datetime")
	fmt.Println("  - 🔍 --dry-run mode for safe preview without modifying files")
	fmt.Println("  - 🔍 --dry-run [N] mode for quick overview (N files sampled)")
	fmt.Println("  - 🔄 Concurrent processing with configurable worker pools")
	fmt.Println("  - 📊 Enhanced progress bars with visual feedback")
	fmt.Println("  - 🗂️ Preserves location data in filenames while updating date portion")
	fmt.Println()
	fmt.Println("Clean Features:")
	fmt.Println("  - ⚡ High-speed duplicate detection using SHA-256")
	fmt.Println("  - 🧠 Intelligent file prioritization")
	fmt.Println("  - 🔒 Safe concurrent duplicate removal")
	fmt.Println("  - 📊 Enhanced progress bars (disabled in --verbose mode)")
	fmt.Println("  - 🔍 --dry-run mode for safe preview")
	fmt.Println("  - 🔍 --dry-run [N] mode for quick summary (samples first N duplicate groups)")
	fmt.Println("  - 📝 --verbose mode for detailed logging")
	fmt.Println()
	fmt.Println("Cleanup Features:")
	fmt.Println("  - 🧹 Standalone empty directory removal")
	fmt.Println("  - 🔍 Intelligent empty directory detection (ignores non-media files)")
	fmt.Println("  - 🔄 Multi-pass removal for nested empty directories")
	fmt.Println("  - 🔍 --dry-run mode to preview what would be removed")
	fmt.Println("  - 📁 Safe operation - only removes directories with no media files")
	fmt.Println("  - 🗑️  Detailed logging of removed directories")
	fmt.Println()
	fmt.Println("Merge Features:")
	fmt.Println("  - 🔀 Merge photos from source into target using YEAR/COUNTRY/CITY structure")
	fmt.Println("  - 🚀 Concurrent processing with configurable worker pools")
	fmt.Println("  - 📊 Enhanced progress bars with visual feedback")
	fmt.Println("  - 🔍 --dry-run mode for safe preview without copying files")
	fmt.Println("  - 🔍 --dry-run [N] mode for quick overview (N files per type per directory)")
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
	return processPhotosConcurrently(sourcePath, destPath, 1, false, 0, true)
}

func processPhotosConcurrently(sourcePath, destPath string, workers int, dryRun bool, dryRunSampleSize int, showProgress bool) error {
	fmt.Printf("🔍 Scanning media files from: %s\n", sourcePath)
	fmt.Printf("📁 Destination: %s\n", destPath)
	
	if dryRun {
		if dryRunSampleSize > 0 {
			fmt.Printf("🔍 DRY RUN MODE - Sample only %d file(s) per type per directory\n", dryRunSampleSize)
		} else {
			fmt.Println("🔍 DRY RUN MODE - No files will be moved")
		}
		fmt.Println()
	}
	
	// Collect all media files
	var jobs []WorkJob
	var err error
	
	if dryRunSampleSize > 0 {
		// For sampling mode, collect sample files
		jobs, err = collectSampleFiles(sourcePath, destPath, dryRun, dryRunSampleSize)
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
			country, city, err = promptForLocation(location, mediaPath, lat, lon)
			if err != nil {
				return fmt.Errorf("failed to get location information: %v", err)
			}
		}
	}
	
	// Generate new filename and directory structure using destination base path
	// Use helper function to preserve existing hour+minute if present
	newFilename := generateFilenameWithTime(mediaPath, date, city)
	
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
		
		// Move/rename the file with enhanced permission handling
		if err := safeFileMove(mediaPath, finalPath); err != nil {
			if permErr, ok := err.(*PermissionError); ok {
				handlePermissionError(permErr, true)
				return fmt.Errorf("permission error moving file from %s to %s", mediaPath, finalPath)
			}
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

// collectSampleFiles collects a sample of files for dry-run sampling mode
// Samples N photos and N videos per directory to provide an overview
func collectSampleFiles(sourcePath, destPath string, dryRun bool, sampleSize int) ([]WorkJob, error) {
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
	
	// Sample files: N photos and N videos per directory (if available)
	var jobs []WorkJob
	totalPhotos := 0
	totalVideos := 0
	directoriesWithPhotos := 0
	directoriesWithVideos := 0
	
	for _, files := range dirFiles {
		// Sample N photos per directory
		if len(files["photos"]) > 0 {
			count := sampleSize
			if count > len(files["photos"]) {
				count = len(files["photos"])
			}
			for i := 0; i < count; i++ {
				photoPath := files["photos"][i]
				jobs = append(jobs, WorkJob{
					PhotoPath: photoPath,
					DestPath:  destPath,
					JobType:   "process",
					DryRun:    dryRun,
				})
				totalPhotos++
			}
			directoriesWithPhotos++
		}
		
		// Sample N videos per directory
		if len(files["videos"]) > 0 {
			count := sampleSize
			if count > len(files["videos"]) {
				count = len(files["videos"])
			}
			for i := 0; i < count; i++ {
				videoPath := files["videos"][i]
				jobs = append(jobs, WorkJob{
					PhotoPath: videoPath,
					DestPath:  destPath,
				JobType:   "process",
				DryRun:    dryRun,
			})
			totalVideos++
			}
			directoriesWithVideos++
		}
	}
	
	fmt.Printf("📋 Sampled %d files from %d directories (%d photos from %d dirs, %d videos from %d dirs)\n", 
		len(jobs), len(dirFiles), totalPhotos, directoriesWithPhotos, totalVideos, directoriesWithVideos)
	
	return jobs, nil
}