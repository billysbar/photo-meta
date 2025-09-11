package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DateLocationDB stores date to location mappings
type DateLocationDB struct {
	DateToLocation map[string]string // date -> location path (e.g., "2025-09-03" -> "2025/spain/palma")
}

// NewDateLocationDB creates a new date-location database
func NewDateLocationDB() *DateLocationDB {
	return &DateLocationDB{
		DateToLocation: make(map[string]string),
	}
}

// processDateTimeMatching handles the datetime command workflow
func processDateTimeMatching(sourcePath, destPath string, dryRun bool, dryRun1 bool, showProgress bool) error {
	fmt.Printf("🕒 DateTime Matching Mode\n")
	fmt.Printf("🔍 Source: %s\n", sourcePath)
	fmt.Printf("📁 Destination: %s\n", destPath)

	if dryRun {
		if dryRun1 {
			fmt.Println("🔍 DRY RUN1 MODE - Sample only 1 file per type per subdirectory")
		} else {
			fmt.Println("🔍 DRY RUN MODE - No files will be moved")
		}
	}
	fmt.Println()

	// Step 1: Check for GPS data in source path
	fmt.Println("🔍 Step 1: Checking for GPS data in source path...")
	hasGPS, gpsFile, err := checkForGPSInSource(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to check for GPS data: %v", err)
	}

	if hasGPS {
		fmt.Printf("❌ Found GPS data in source file: %s\n", gpsFile)
		fmt.Println("⚠️  Please run 'process' command first to organize GPS-enabled photos.")
		fmt.Println("   This will create the location database needed for datetime matching.")
		return nil
	}
	fmt.Println("✅ No GPS data found in source - proceeding with datetime matching")
	fmt.Println()

	// Step 2: Build date-location database from destination
	fmt.Println("📚 Step 2: Building date-location database from destination...")
	db, err := buildDateLocationDB(destPath)
	if err != nil {
		return fmt.Errorf("failed to build date-location database: %v", err)
	}

	fmt.Printf("✅ Built database with %d date-location mappings\n", len(db.DateToLocation))
	if len(db.DateToLocation) == 0 {
		fmt.Println("⚠️  No processed photos found in destination. Run 'process' command first.")
		return nil
	}

	// Show database contents for debugging
	fmt.Println("\n📋 Date-Location Database:")
	for date, location := range db.DateToLocation {
		fmt.Printf("  %s -> %s\n", date, location)
	}
	fmt.Println()

	// Step 3: Process files in source path
	fmt.Println("🔄 Step 3: Processing files in source path...")
	return processFilesWithDateTimeMatching(sourcePath, destPath, db, dryRun, dryRun1, showProgress)
}

// checkForGPSInSource scans source path for any files with GPS data
func checkForGPSInSource(sourcePath string) (bool, string, error) {
	var hasGPS bool
	var gpsFile string

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

		// Quick check for GPS data
		_, _, err = extractGPSCoordinates(path)
		if err == nil {
			hasGPS = true
			gpsFile = path
			return fmt.Errorf("found GPS") // Early termination
		}

		return nil
	})

	if err != nil && err.Error() == "found GPS" {
		return hasGPS, gpsFile, nil
	}

	return hasGPS, gpsFile, err
}

// buildDateLocationDB scans destination and builds date->location mapping
func buildDateLocationDB(destPath string) (*DateLocationDB, error) {
	db := NewDateLocationDB()

	err := filepath.Walk(destPath, func(path string, info os.FileInfo, err error) error {
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

		// Extract date and location from file path and name
		date, location, err := extractDateLocationFromPath(path, destPath)
		if err != nil {
			fmt.Printf("⚠️  Could not parse %s: %v\n", filepath.Base(path), err)
			return nil // Continue processing other files
		}

		// Store in database with special UK replacement logic
		if existingLocation, exists := db.DateToLocation[date]; exists {
			if existingLocation != location {
				// Special case: if existing location contains 'united-kingdom' and new location doesn't, replace it
				if strings.Contains(existingLocation, "united-kingdom") && !strings.Contains(location, "united-kingdom") {
					db.DateToLocation[date] = location
					fmt.Printf("🔄 Date %s: replacing UK location %s with %s\n", date, existingLocation, location)
				} else {
					fmt.Printf("🔄 Date %s: keeping %s, ignoring %s\n", date, existingLocation, location)
				}
			}
		} else {
			db.DateToLocation[date] = location
			fmt.Printf("📅 Added: %s -> %s\n", date, location)
		}

		return nil
	})

	return db, err
}

// extractDateLocationFromPath extracts date and location from processed file path
func extractDateLocationFromPath(filePath, basePath string) (string, string, error) {
	// Get relative path from base
	relPath, err := filepath.Rel(basePath, filePath)
	if err != nil {
		return "", "", err
	}

	// Extract filename for date parsing
	filename := filepath.Base(filePath)

	// Parse date from filename (expect YYYY-MM-DD-location.ext format)
	dateRegex := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})`)
	matches := dateRegex.FindStringSubmatch(filename)
	if len(matches) < 2 {
		return "", "", fmt.Errorf("no date found in filename: %s", filename)
	}

	date := matches[1]

	// Extract location from directory path
	dirPath := filepath.Dir(relPath)
	location := dirPath

	// Strip VIDEO-FILES/ prefix if present to get the actual location
	if strings.HasPrefix(location, "VIDEO-FILES/") {
		location = strings.TrimPrefix(location, "VIDEO-FILES/")
	}

	return date, location, nil
}

// processFilesWithDateTimeMatching processes source files using datetime matching
func processFilesWithDateTimeMatching(sourcePath, destPath string, db *DateLocationDB, dryRun bool, dryRun1 bool, showProgress bool) error {
	processedCount := 0
	videoCount := 0
	photoCount := 0
	unmatchedFiles := []string{}

	// Collect files to process
	var filesToProcess []string

	if dryRun1 {
		// Sample files for dry-run1 mode
		var err error
		filesToProcess, err = collectSampleFilesForDatetime(sourcePath)
		if err != nil {
			return err
		}
		fmt.Printf("📋 Sampled %d files for datetime matching preview\n", len(filesToProcess))
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

		// Look up location for this date
		location, exists := db.DateToLocation[date]
		if !exists {
			fmt.Printf("🔍 DEBUG: %s - date %s not found in database\n", filepath.Base(path), date)

			// Try temporal proximity matching
			nearbyLocation, nearbyDate, found := findNearbyDateMatch(db, date, path)
			if found {
				fmt.Printf("📅 Found nearby match for %s:\n", filepath.Base(path))
				fmt.Printf("   File date: %s\n", date)
				fmt.Printf("   Nearby date: %s -> %s\n", nearbyDate, nearbyLocation)

				// Automatically use the nearby location and add to database
				location = nearbyLocation
				db.DateToLocation[date] = location
				fmt.Printf("✅ Added %s -> %s to location database\n", date, location)
			} else {
				unmatchedFiles = append(unmatchedFiles, path)
				continue
			}
		}
		
		// Move file to matched location
		if err := moveFileToLocation(path, destPath, location, date, dryRun); err != nil {
			return fmt.Errorf("failed to move %s: %v", filepath.Base(path), err)
		}

		processedCount++
		if isVideoFile(path) {
			videoCount++
		} else {
			photoCount++
		}
	}

	// Summary
	fmt.Printf("\n📊 DateTime Processing Summary:\n")
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

// extractDateFromFilename extracts date from filename patterns
func extractDateFromFilename(filename string) (string, error) {
	// Pattern 1: YYYYMMDDHHMMSS format (e.g., 20250903175904.PNG)
	pattern1 := regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})\d{6}`)
	if matches := pattern1.FindStringSubmatch(filename); len(matches) >= 4 {
		year, month, day := matches[1], matches[2], matches[3]
		return fmt.Sprintf("%s-%s-%s", year, month, day), nil
	}

	// Pattern 2: YYYY-MM-DD format already in filename
	pattern2 := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	if matches := pattern2.FindStringSubmatch(filename); len(matches) >= 2 {
		return matches[1], nil
	}

	// Pattern 3: YYYYMMDD format
	pattern3 := regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})`)
	if matches := pattern3.FindStringSubmatch(filename); len(matches) >= 4 {
		year, month, day := matches[1], matches[2], matches[3]
		return fmt.Sprintf("%s-%s-%s", year, month, day), nil
	}

	// Pattern 4: DD-MM-YYYY format (e.g., 10-10-2018-DSC_0996.JPG)
	pattern4 := regexp.MustCompile(`^(\d{2})-(\d{2})-(\d{4})`)
	if matches := pattern4.FindStringSubmatch(filename); len(matches) >= 4 {
		day, month, year := matches[1], matches[2], matches[3]
		return fmt.Sprintf("%s-%s-%s", year, month, day), nil
	}

	return "", fmt.Errorf("no date pattern found in filename")
}


// moveFileToLocation moves file to the specified location path, with special handling for video files
func moveFileToLocation(sourcePath, destBasePath, location, date string, dryRun bool) error {
	// Parse location to get components for filename
	locationParts := strings.Split(location, string(filepath.Separator))
	var city string
	if len(locationParts) > 0 {
		city = locationParts[len(locationParts)-1] // Last component is city
	} else {
		city = "unknown"
	}

	// Generate new filename
	ext := filepath.Ext(sourcePath)
	newFilename := fmt.Sprintf("%s-%s%s", date, city, ext)

	var destDir string
	var fileType string

	// Check if this is a video file
	if isVideoFile(sourcePath) {
		// For video files, place in VIDEO-FILES/YYYY/COUNTRY/CITY structure
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

// promptForConfirmation prompts user for y/n confirmation
func promptForConfirmation(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(prompt)
		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}

		response = strings.ToLower(strings.TrimSpace(response))
		switch response {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please enter 'y' or 'n'")
		}
	}
}

// collectSampleFilesForDatetime collects sample files for datetime dry-run1 mode
func collectSampleFilesForDatetime(sourcePath string) ([]string, error) {
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

	// Sample files: 1 photo and 1 video per subdirectory (if available)
	var sampleFiles []string

	for _, files := range dirFiles {
		// Sample 1 photo per subdirectory
		if len(files["photos"]) > 0 {
			sampleFiles = append(sampleFiles, files["photos"][0])
		}

		// Sample 1 video per subdirectory
		if len(files["videos"]) > 0 {
			sampleFiles = append(sampleFiles, files["videos"][0])
		}
	}

	return sampleFiles, nil
}

// findNearbyDateMatch looks for dates within a reasonable proximity (±3 days) to infer location
func findNearbyDateMatch(db *DateLocationDB, targetDate, filePath string) (location, nearbyDate string, found bool) {
	// Parse the target date
	targetTime, err := time.Parse("2006-01-02", targetDate)
	if err != nil {
		return "", "", false
	}

	// Define search range (±3 days)
	const searchDays = 3
	var closestMatch string
	var closestLocation string
	var closestDistance time.Duration

	// Search through all dates in the database
	for dbDate, dbLocation := range db.DateToLocation {
		dbTime, err := time.Parse("2006-01-02", dbDate)
		if err != nil {
			continue
		}

		// Calculate time difference
		diff := targetTime.Sub(dbTime)
		if diff < 0 {
			diff = -diff
		}

		// Check if within search range
		if diff <= searchDays*24*time.Hour {
			// Check if this is closer than our current best match
			if closestMatch == "" || diff < closestDistance {
				closestMatch = dbDate
				closestLocation = dbLocation
				closestDistance = diff
			}
		}
	}

	if closestMatch != "" {
		return closestLocation, closestMatch, true
	}

	return "", "", false
}
