package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// processFallbackOrganization handles the fallback command workflow
func processFallbackOrganization(sourcePath, destPath string, dryRun bool, dryRunSampleSize int, showProgress bool) error {
	fmt.Printf("📅 Fallback Organization Mode\n")
	fmt.Printf("🔍 Source: %s\n", sourcePath)
	fmt.Printf("📁 Destination: %s\n", destPath)

	if dryRun {
		if dryRunSampleSize > 0 {
			fmt.Printf("🔍 DRY RUN MODE - Sample only %d file(s) per type per subdirectory\n", dryRunSampleSize)
		} else {
			fmt.Println("🔍 DRY RUN MODE - No files will be moved")
		}
	}
	fmt.Println()

	// Process files in source path
	fmt.Println("🔄 Processing files for fallback organization...")
	return processFilesWithFallbackOrganization(sourcePath, destPath, dryRun, dryRunSampleSize, showProgress)
}

// Global fallback location database
var fallbackLocationDB *LocationDB

// processFilesWithFallbackOrganization processes source files using fallback year/month organization
func processFilesWithFallbackOrganization(sourcePath, destPath string, dryRun bool, dryRunSampleSize int, showProgress bool) error {
	// Initialize location database for prompting support
	var err error
	fallbackLocationDB, err = NewLocationDB()
	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize location database: %v\n", err)
		fallbackLocationDB = nil
	} else {
		defer fallbackLocationDB.Close()
	}

	processedCount := 0
	videoCount := 0
	photoCount := 0
	unmatchedFiles := []string{}

	// Collect files to process
	var filesToProcess []string

	if dryRunSampleSize > 0 {
		// Sample files for sampling mode
		var err error
		filesToProcess, err = collectSampleFilesForFallback(sourcePath, dryRunSampleSize)
		if err != nil {
			return err
		}
		fmt.Printf("📋 Sampled %d files for fallback organization preview\n", len(filesToProcess))
	} else {
		// Collect all files
		err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
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
			filesToProcess = append(filesToProcess, path)
			return nil
		})
		if err != nil {
			return err
		}
	}

	// Process the collected files
	for _, path := range filesToProcess {
		// Extract date from filename
		date, err := extractDateFromFilename(filepath.Base(path))
		if err != nil {
			fmt.Printf("🔍 DEBUG: %s - %v\n", filepath.Base(path), err)
			unmatchedFiles = append(unmatchedFiles, path)
			continue
		}

		// Parse the date to extract year and month
		dateParts := strings.Split(date, "-")
		if len(dateParts) >= 3 {
			year := dateParts[0]
			monthNum := dateParts[1]
			monthName := getMonthName(monthNum)

			// Create fallback location as YYYY/MonthName
			location := fmt.Sprintf("%s/%s", year, monthName)
			
			fmt.Printf("📅 File: %s -> Date: %s -> Location: %s\n", filepath.Base(path), date, location)

			// Prompt for location information using standardized approach
			country, city, shouldSkip, err := promptForLocationWithDatabase("", path, 0, 0, fallbackLocationDB)
			if err != nil {
				return fmt.Errorf("failed to get location for %s: %v", filepath.Base(path), err)
			}
			if shouldSkip {
				fmt.Printf("⏭️  Skipping file: %s\n", filepath.Base(path))
				continue
			}

			// Move file to fallback location with prompted location info
			if err := moveFileToFallbackLocationWithLocation(path, destPath, location, date, country, city, dryRun); err != nil {
				return fmt.Errorf("failed to move %s: %v", filepath.Base(path), err)
			}

			processedCount++
			if isVideoFile(path) {
				videoCount++
			} else {
				photoCount++
			}
		} else {
			// If we can't parse the date properly, add to unmatched
			unmatchedFiles = append(unmatchedFiles, path)
			continue
		}
	}

	// Summary
	fmt.Printf("\n📊 Fallback Processing Summary:\n")
	fmt.Printf("✅ Total files processed: %d\n", processedCount)
	fmt.Printf("📷 Photos processed: %d\n", photoCount)
	fmt.Printf("🎥 Videos processed: %d (moved to VIDEO-FILES/)\n", videoCount)
	fmt.Printf("⚠️  Files unmatched: %d\n", len(unmatchedFiles))

	if len(unmatchedFiles) > 0 {
		fmt.Println("\n📋 Unmatched Files:")
		for _, file := range unmatchedFiles {
			fmt.Printf("  - %s\n", filepath.Base(file))
		}
	}

	return nil
}

// moveFileToFallbackLocation moves file to the fallback year/month location
func moveFileToFallbackLocation(sourcePath, destBasePath, location, date string, dryRun bool) error {
	// Parse date string to time.Time for generateFilenameWithTime
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("failed to parse date %s: %v", date, err)
	}

	// Generate new filename preserving existing hour+minute if present
	newFilename := generateFilenameWithTime(sourcePath, parsedDate, "")

	var destDir string
	var fileType string

	// Check if this is a video file
	if isVideoFile(sourcePath) {
		// For video files, place in VIDEO-FILES/YYYY/Month structure
		fileType = "video"
		destDir = filepath.Join(destBasePath, "VIDEO-FILES", location)
		if dryRun {
			fmt.Printf("🎥 [DRY RUN] Processing video file: %s\n", filepath.Base(sourcePath))
		} else {
			fmt.Printf("🎥 Processing video file: %s\n", filepath.Base(sourcePath))
		}
	} else {
		// For photo files, use the regular location structure
		fileType = "photo"
		destDir = filepath.Join(destBasePath, location)
		if dryRun {
			fmt.Printf("📷 [DRY RUN] Processing photo file: %s\n", filepath.Base(sourcePath))
		} else {
			fmt.Printf("📷 Processing photo file: %s\n", filepath.Base(sourcePath))
		}
	}

	destPath := filepath.Join(destDir, newFilename)

	// Handle duplicates simulation
	ext := filepath.Ext(sourcePath)
	finalPath := destPath
	counter := 1
	for {
		if _, err := os.Stat(finalPath); os.IsNotExist(err) {
			break
		}

		base := strings.TrimSuffix(newFilename, ext)
		duplicateFilename := fmt.Sprintf("%s-%d%s", base, counter, ext)
		finalPath = filepath.Join(destDir, duplicateFilename)
		counter++

		if counter > 1000 {
			return fmt.Errorf("too many duplicate filenames")
		}
	}

	if dryRun {
		// Dry run mode - just show what would happen
		if fileType == "video" {
			fmt.Printf("✅ [DRY RUN] Video would be moved to: %s\n", finalPath)
		} else {
			fmt.Printf("✅ [DRY RUN] Photo would be moved to: %s\n", finalPath)
		}
		return nil
	}

	// Create directory structure if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", destDir, err)
	}

	// Move the file
	if err := os.Rename(sourcePath, finalPath); err != nil {
		return err
	}

	if fileType == "video" {
		fmt.Printf("✅ Video moved to: %s\n", finalPath)
	} else {
		fmt.Printf("✅ Photo moved to: %s\n", finalPath)
	}
	return nil
}

// collectSampleFilesForFallback collects sample files for fallback dry-run1 mode
func collectSampleFilesForFallback(sourcePath string, sampleSize int) ([]string, error) {
	// Map to track files by subdirectory and type
	dirFiles := make(map[string]map[string][]string) // subdirectory -> {photos: [], videos: []}

	// Collect all files grouped by subdirectory and type
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

		// Get relative subdirectory path from source
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		subDir := filepath.Dir(relPath)

		// Use "." for files directly in source path (no subdirectory)
		if subDir == "." {
			subDir = "root"
		}

		// Initialize subdirectory map if needed
		if dirFiles[subDir] == nil {
			dirFiles[subDir] = map[string][]string{
				"photos": []string{},
				"videos": []string{},
			}
		}

		// Add to appropriate type list
		if isVideoFile(path) {
			dirFiles[subDir]["videos"] = append(dirFiles[subDir]["videos"], path)
		} else {
			dirFiles[subDir]["photos"] = append(dirFiles[subDir]["photos"], path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sample files: N photos and N videos per subdirectory (if available)
	var sampleFiles []string

	for _, files := range dirFiles {
		// Sample N photos per subdirectory
		if len(files["photos"]) > 0 {
			count := sampleSize
			if count > len(files["photos"]) {
				count = len(files["photos"])
			}
			for i := 0; i < count; i++ {
				sampleFiles = append(sampleFiles, files["photos"][i])
			}
		}

		// Sample N videos per subdirectory
		if len(files["videos"]) > 0 {
			count := sampleSize
			if count > len(files["videos"]) {
				count = len(files["videos"])
			}
			for i := 0; i < count; i++ {
				sampleFiles = append(sampleFiles, files["videos"][i])
			}
		}
	}

	return sampleFiles, nil
}

// getMonthName converts a month number (01-12) to full English month name
func getMonthName(monthNum string) string {
	months := map[string]string{
		"01": "January", "02": "February", "03": "March", "04": "April",
		"05": "May", "06": "June", "07": "July", "08": "August",
		"09": "September", "10": "October", "11": "November", "12": "December",
	}
	if name, exists := months[monthNum]; exists {
		return name
	}
	return "Unknown"
}

// promptForFallbackLocation prompts user for country and city information
func promptForFallbackLocation(filePath string) (country, city string, shouldSkip bool, err error) {
	// Pause any progress reporting during user input
	pauseProgress()
	defer resumeProgress()

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n📸 File: %s\n", filepath.Base(filePath))
	fmt.Print("Enter country for this location (or 'skip' to skip this file): ")
	countryInput, err := reader.ReadString('\n')
	if err != nil {
		return "", "", false, err
	}
	country = strings.TrimSpace(countryInput)
	if country == "" || strings.ToLower(country) == "skip" {
		return "", "", true, nil
	}

	fmt.Print("Enter city name: ")
	cityInput, err := reader.ReadString('\n')
	if err != nil {
		return "", "", false, err
	}
	city = strings.TrimSpace(cityInput)
	if city == "" {
		city = country // Default city to country if empty
	}

	// Clean up the names using existing utility functions
	country = anglicizeName(country)
	city = anglicizeName(city)

	return country, city, false, nil
}

// moveFileToFallbackLocationWithLocation moves file to fallback location using provided country/city
func moveFileToFallbackLocationWithLocation(sourcePath, destBasePath, location, date, country, city string, dryRun bool) error {
	// Parse date string to time.Time for generateFilenameWithTime
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("failed to parse date %s: %v", date, err)
	}

	// Generate new filename with location information preserving existing hour+minute if present
	newFilename := generateFilenameWithTime(sourcePath, parsedDate, city)

	var destDir string
	var fileType string

	// Check if this is a video file
	if isVideoFile(sourcePath) {
		// For video files, place in VIDEO-FILES/YYYY/Country/City structure
		fileType = "video"
		dateParts := strings.Split(date, "-")
		if len(dateParts) >= 1 {
			year := dateParts[0]
			destDir = filepath.Join(destBasePath, "VIDEO-FILES", year, country, city)
		} else {
			destDir = filepath.Join(destBasePath, "VIDEO-FILES", location)
		}
		if dryRun {
			fmt.Printf("🎥 [DRY RUN] Processing video file: %s\n", filepath.Base(sourcePath))
		} else {
			fmt.Printf("🎥 Processing video file: %s\n", filepath.Base(sourcePath))
		}
	} else {
		// For photo files, use the YYYY/Country/City structure
		fileType = "photo"
		dateParts := strings.Split(date, "-")
		if len(dateParts) >= 1 {
			year := dateParts[0]
			destDir = filepath.Join(destBasePath, year, country, city)
		} else {
			destDir = filepath.Join(destBasePath, location)
		}
		if dryRun {
			fmt.Printf("📷 [DRY RUN] Processing photo file: %s\n", filepath.Base(sourcePath))
		} else {
			fmt.Printf("📷 Processing photo file: %s\n", filepath.Base(sourcePath))
		}
	}

	destPath := filepath.Join(destDir, newFilename)

	// Handle duplicates
	finalPath := destPath
	ext := filepath.Ext(sourcePath)
	counter := 1
	for {
		if _, err := os.Stat(finalPath); os.IsNotExist(err) {
			break
		}

		base := strings.TrimSuffix(newFilename, ext)
		duplicateFilename := fmt.Sprintf("%s-%d%s", base, counter, ext)
		finalPath = filepath.Join(destDir, duplicateFilename)
		counter++

		if counter > 1000 {
			return fmt.Errorf("too many duplicate filenames")
		}
	}

	if dryRun {
		// Dry run mode - just show what would happen
		if fileType == "video" {
			fmt.Printf("✅ [DRY RUN] Video would be moved to: %s\n", finalPath)
		} else {
			fmt.Printf("✅ [DRY RUN] Photo would be moved to: %s\n", finalPath)
		}
		return nil
	}

	// Create directory structure if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", destDir, err)
	}

	// Move the file
	if err := os.Rename(sourcePath, finalPath); err != nil {
		return err
	}

	if fileType == "video" {
		fmt.Printf("✅ Video moved to: %s\n", finalPath)
	} else {
		fmt.Printf("✅ Photo moved to: %s\n", finalPath)
	}
	return nil
}