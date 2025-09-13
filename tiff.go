package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Global location database for TIFF processing
// This is a simple solution to pass the database to concurrent workers
var globalTiffLocationDB *LocationDB

// processTiffTimestampFix fixes midnight timestamps using EXIF ModifyDate
func processTiffTimestampFix(targetPath string, workers int, dryRun bool, dryRunSampleSize int, showProgress bool) error {
	fmt.Printf("🕐 TIFF Timestamp Fix\n")
	fmt.Printf("🔍 Target: %s\n", targetPath)

	if dryRun {
		if dryRunSampleSize > 0 {
			fmt.Printf("🔍 DRY RUN MODE - Sample only %d file(s)\n", dryRunSampleSize)
		} else {
			fmt.Printf("🔍 DRY RUN MODE - No files will be modified\n")
		}
	}
	fmt.Printf("⚙️ Workers: %d\n", workers)
	fmt.Println()

	// Initialize location database for GPS-less location detection
	locationDB, err := NewLocationDB()
	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to initialize location database: %v\n", err)
		fmt.Println("   Location detection from image descriptions will be limited")
		globalTiffLocationDB = nil
	} else {
		defer locationDB.Close()
		globalTiffLocationDB = locationDB
		// Show existing mappings
		if mappings, err := locationDB.ListAllMappings(); err == nil && len(mappings) > 0 {
			fmt.Printf("📚 Found %d existing location mappings in database\n", len(mappings))
		}
	}

	// Collect all media files that need timestamp fixing
	var jobs []WorkJob

	if dryRunSampleSize > 0 {
		jobs, err = collectTiffSampleFiles(targetPath, dryRun, dryRunSampleSize)
	} else {
		err = filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
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

			// Add to jobs list
			jobs = append(jobs, WorkJob{
				PhotoPath: path,
				DestPath:  targetPath,
				JobType:   "tiff",
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

	// First, check if any files need location prompting (GPS-less with descriptions)
	// If so, we need to process them sequentially to handle user input
	// In dry-run mode, we still want to show what would be detected
	needsUserInput := false
	if globalTiffLocationDB != nil {
		for _, job := range jobs {
			if requiresLocationPrompting(job.PhotoPath) {
				needsUserInput = true
				break
			}
		}
	}

	if needsUserInput {
		fmt.Printf("🤔 Some files may require location input - processing sequentially\n")
		return processTiffJobsSequentially(jobs, showProgress)
	} else {
		// Process jobs concurrently when no user input is needed
		return processJobsConcurrentlyWithProgress(jobs, workers, showProgress)
	}
}

// collectTiffSampleFiles collects a sample of files for dry-run mode
func collectTiffSampleFiles(targetPath string, dryRun bool, sampleSize int) ([]WorkJob, error) {
	var jobs []WorkJob
	count := 0

	err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if count >= sampleSize {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if it's a media file
		if !isMediaFile(path) {
			return nil
		}

		jobs = append(jobs, WorkJob{
			PhotoPath: path,
			DestPath:  targetPath,
			JobType:   "tiff",
			DryRun:    dryRun,
		})
		count++

		return nil
	})

	return jobs, err
}

// processTiffFile fixes midnight timestamps in a single file
func processTiffFile(filePath string, dryRun bool) error {
	// Determine file type for display
	var fileType string
	var fileIcon string
	if isVideoFile(filePath) {
		fileType = "video"
		fileIcon = "🎥"
	} else {
		fileType = "photo"
		fileIcon = "📷"
	}

	if dryRun {
		fmt.Printf("%s [DRY RUN] Processing %s: %s\n", fileIcon, fileType, filepath.Base(filePath))
	} else {
		fmt.Printf("%s Processing %s: %s\n", fileIcon, fileType, filepath.Base(filePath))
	}

	// Extract all relevant datetime fields from EXIF
	dateInfo, err := extractAllDatetimeInfo(filePath)
	if err != nil {
		fmt.Printf("❌ Failed to extract datetime info from %s: %v\n", filepath.Base(filePath), err)
		return nil // Continue processing other files
	}

	// Check if any original timestamp is set to midnight (00:00:00) or if filename needs updating
	needsProcessing, correctTime := needsTimestampFix(dateInfo)
	if !needsProcessing {
		fmt.Printf("❌ %s has no valid timestamp data\n", filepath.Base(filePath))
		return nil
	}

	// Check if EXIF timestamps need fixing (ModifyDate differs from original timestamps)
	needsExifFix := dateInfo.ModifyDate != dateInfo.DateTimeOriginal ||
					dateInfo.ModifyDate != dateInfo.CreateDate ||
					dateInfo.ModifyDate != dateInfo.DateTime

	// Generate the expected filename
	newFilename := generateNewFilename(filePath, correctTime)
	currentFilename := filepath.Base(filePath)
	needsFilenameUpdate := newFilename != currentFilename

	// Skip if no changes needed
	if !needsExifFix && !needsFilenameUpdate {
		fmt.Printf("✅ %s already correct\n", filepath.Base(filePath))
		return nil
	}

	if needsExifFix {
		fmt.Printf("🔧 EXIF timestamps differ from ModifyDate\n")
		fmt.Printf("   ModifyDate: %s (using as correct time)\n", correctTime.Format("2006:01:02 15:04:05"))
		fmt.Printf("   DateTimeOriginal: %s\n", dateInfo.DateTimeOriginal)
	} else {
		fmt.Printf("📝 Using timestamp: %s\n", correctTime.Format("2006:01:02 15:04:05"))
	}

	if dryRun {
		// In dry run, show what would be changed
		if needsExifFix {
			fmt.Printf("📸 [DRY RUN] Would update EXIF timestamp\n")
		}
		if needsFilenameUpdate {
			fmt.Printf("📝 [DRY RUN] Would rename to: %s\n", newFilename)
		}
		return nil
	}

	// Update EXIF timestamps only if they need fixing
	if needsExifFix {
		if err := updateExifTimestamp(filePath, correctTime); err != nil {
			fmt.Printf("❌ Failed to update EXIF timestamp for %s: %v\n", filepath.Base(filePath), err)
			return nil // Continue processing
		}
	}

	// Update filename if needed
	currentFilePath := filePath
	if needsFilenameUpdate {
		newPath := filepath.Join(filepath.Dir(filePath), newFilename)
		if err := os.Rename(filePath, newPath); err != nil {
			fmt.Printf("❌ Failed to rename %s to %s: %v\n", filepath.Base(filePath), newFilename, err)
			return nil // Continue processing
		}
		fmt.Printf("📝 Renamed to: %s\n", newFilename)
		currentFilePath = newPath // Update the path for location detection
	}

	// Handle location detection for GPS-less files
	if globalTiffLocationDB != nil {
		if err := handleLocationDetectionAndFilenameUpdate(currentFilePath, correctTime, globalTiffLocationDB, dryRun); err != nil {
			fmt.Printf("❌ Location detection failed for %s: %v\n", filepath.Base(currentFilePath), err)
			// Continue processing - don't fail the whole operation
		}
	}

	fmt.Printf("✅ %s timestamp fixed\n", filepath.Base(filePath))
	return nil
}

// handleLocationDetectionAndFilenameUpdate handles location detection and filename updates
func handleLocationDetectionAndFilenameUpdate(filePath string, timestamp time.Time, locationDB *LocationDB, dryRun bool) error {
	// Extract location info from EXIF
	locationInfo, err := extractLocationInfo(filePath)
	if err != nil {
		return fmt.Errorf("failed to extract location info: %v", err)
	}

	// Skip if file has GPS data
	if locationInfo.HasGPS {
		return nil
	}

	// Skip if no image description
	if locationInfo.Description == "" {
		return nil
	}

	// Detect locations from description
	detectedLocations := detectLocationFromDescription(locationInfo.Description)
	fmt.Printf("🗺️  Found image description: '%s'\n", locationInfo.Description)

	if len(detectedLocations) == 0 {
		fmt.Printf("❌ No locations detected in description\n")
		return nil // No locations detected
	}

	fmt.Printf("📍 Detected potential locations: %s\n", strings.Join(detectedLocations, ", "))

	if dryRun {
		fmt.Printf("🤔 [DRY RUN] Would prompt for location confirmation\n")
		return nil
	}

	// Prompt user for confirmation
	country, city, shouldSkip, err := promptUserForLocationFromDescription(filePath, locationInfo.Description, detectedLocations, locationDB)
	if err != nil {
		return fmt.Errorf("user prompt failed: %v", err)
	}

	if shouldSkip {
		fmt.Printf("⏭️ Skipping location update for %s\n", filepath.Base(filePath))
		return nil
	}

	// Save the mapping to database
	if err := locationDB.SaveLocationMapping(city, country, true); err != nil {
		fmt.Printf("⚠️ Warning: Failed to save location mapping: %v\n", err)
	} else {
		fmt.Printf("💾 Saved location mapping: %s -> %s\n", city, country)
	}

	// Generate new filename with location
	newFilename := generateLocationFilename(filePath, timestamp, country, city)
	currentFilename := filepath.Base(filePath)

	if newFilename != currentFilename {
		newPath := filepath.Join(filepath.Dir(filePath), newFilename)
		if err := os.Rename(filePath, newPath); err != nil {
			return fmt.Errorf("failed to rename file: %v", err)
		}
		fmt.Printf("📝 Renamed to include location: %s\n", newFilename)
	}

	return nil
}

// generateLocationFilename generates filename with location data
func generateLocationFilename(filePath string, timestamp time.Time, country, city string) string {
	currentFilename := filepath.Base(filePath)
	ext := filepath.Ext(currentFilename)

	// Generate time portion
	var timePortion string
	if timestamp.Hour() == 0 && timestamp.Minute() == 0 && timestamp.Second() == 0 {
		timePortion = timestamp.Format("2006-01-02")
	} else {
		timePortion = timestamp.Format("2006-01-02-1504")
	}

	// Create new filename: DATE[-TIME]-CITY-LOCATION.ext
	nameWithoutExt := fmt.Sprintf("%s-%s", timePortion, city)

	// Add location if it's different from city
	if strings.ToLower(country) != strings.ToLower(city) {
		// Check if filename already contains location info
		if !strings.Contains(strings.ToLower(currentFilename), strings.ToLower(city)) {
			nameWithoutExt = fmt.Sprintf("%s-%s", nameWithoutExt, strings.ToLower(country))
		}
	}

	return nameWithoutExt + ext
}

// DateTimeInfo holds all relevant datetime fields from EXIF
type DateTimeInfo struct {
	DateTimeOriginal string
	CreateDate      string
	DateTime        string
	ModifyDate      string
}

// TiffLocationInfo holds location data from EXIF or filename for TIFF processing
type TiffLocationInfo struct {
	Country     string
	City        string
	Description string // From ImageDescription field
	HasGPS      bool
}

// extractAllDatetimeInfo extracts all datetime fields from EXIF
func extractAllDatetimeInfo(filePath string) (*DateTimeInfo, error) {
	cmd := exec.Command("exiftool",
		"-DateTimeOriginal",
		"-CreateDate",
		"-DateTime",
		"-ModifyDate",
		"-T", // Tab-separated output
		filePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exiftool failed: %v", err)
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return nil, fmt.Errorf("no datetime metadata found")
	}

	// Parse tab-separated output
	parts := strings.Split(outputStr, "\t")
	if len(parts) < 4 {
		return nil, fmt.Errorf("insufficient datetime fields in EXIF")
	}

	info := &DateTimeInfo{
		DateTimeOriginal: strings.TrimSpace(parts[0]),
		CreateDate:       strings.TrimSpace(parts[1]),
		DateTime:         strings.TrimSpace(parts[2]),
		ModifyDate:       strings.TrimSpace(parts[3]),
	}

	return info, nil
}

// needsTimestampFix checks if the file needs timestamp fixing or filename updating
func needsTimestampFix(info *DateTimeInfo) (bool, time.Time) {
	// Always prefer ModifyDate as the correct timestamp source
	// This represents when the file was actually processed/modified
	correctTimeStr := info.ModifyDate

	// Fallback to other timestamps if ModifyDate is not available
	if correctTimeStr == "" || correctTimeStr == "-" {
		correctTimeStr = info.DateTimeOriginal
		if correctTimeStr == "" || correctTimeStr == "-" {
			correctTimeStr = info.CreateDate
		}
		if correctTimeStr == "" || correctTimeStr == "-" {
			correctTimeStr = info.DateTime
		}
	}

	// Parse the correct time
	if correctTimeStr == "" || correctTimeStr == "-" {
		return false, time.Time{}
	}

	// Try different date formats
	dateFormats := []string{
		"2006:01:02 15:04:05",
		"2006-01-02 15:04:05",
		"2006:01:02",
		"2006-01-02",
	}

	for _, format := range dateFormats {
		if correctTime, err := time.Parse(format, correctTimeStr); err == nil {
			return true, correctTime
		}
	}

	return false, time.Time{}
}

// updateExifTimestamp updates the EXIF timestamp fields
func updateExifTimestamp(filePath string, correctTime time.Time) error {
	timeStr := correctTime.Format("2006:01:02 15:04:05")

	cmd := exec.Command("exiftool",
		"-overwrite_original",
		fmt.Sprintf("-DateTimeOriginal=%s", timeStr),
		fmt.Sprintf("-CreateDate=%s", timeStr),
		fmt.Sprintf("-DateTime=%s", timeStr),
		filePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exiftool update failed: %v, output: %s", err, output)
	}

	return nil
}

// generateNewFilename generates a new filename with the corrected date and time
func generateNewFilename(filePath string, correctTime time.Time) string {
	currentFilename := filepath.Base(filePath)
	ext := filepath.Ext(currentFilename)

	// Check for existing time patterns (YYYY-MM-DD-HHMM or YYYY-MM-DD-HH-MM-SS)
	timePattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-\d{2}\d{2}`)
	dateOnlyPattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})`)

	// Generate new date portion - include time if it's not midnight
	var newDatePortion string
	if correctTime.Hour() == 0 && correctTime.Minute() == 0 && correctTime.Second() == 0 {
		// Time is midnight, use date only
		newDatePortion = correctTime.Format("2006-01-02")
	} else {
		// Time is not midnight, include time in filename as HHMM format
		newDatePortion = correctTime.Format("2006-01-02-1504")
	}

	// If filename already has time pattern, replace it
	if timePattern.MatchString(currentFilename) {
		return timePattern.ReplaceAllString(currentFilename, newDatePortion)
	}

	// If filename has date-only pattern, replace it
	if dateOnlyPattern.MatchString(currentFilename) {
		return dateOnlyPattern.ReplaceAllString(currentFilename, newDatePortion)
	}

	// If no date pattern found, prepend the new date (and time if applicable)
	nameWithoutExt := strings.TrimSuffix(currentFilename, ext)
	return fmt.Sprintf("%s-%s%s", newDatePortion, nameWithoutExt, ext)
}

// extractLocationInfo extracts location data from EXIF and filename
func extractLocationInfo(filePath string) (*TiffLocationInfo, error) {
	location := &TiffLocationInfo{}

	// Check for GPS data first
	_, _, err := extractGPSCoordinates(filePath)
	if err == nil {
		location.HasGPS = true
		return location, nil
	}

	// No GPS data, check image description
	cmd := exec.Command("exiftool",
		"-ImageDescription",
		"-UserComment",
		"-XPComment",
		"-T", // Tab-separated output
		filePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exiftool failed: %v", err)
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return location, nil // No description data
	}

	// Parse tab-separated output
	parts := strings.Split(outputStr, "\t")
	if len(parts) > 0 {
		description := strings.TrimSpace(parts[0])
		if description != "" && description != "-" {
			location.Description = description
		}
	}

	return location, nil
}

// detectLocationFromDescription analyzes image description for location keywords
func detectLocationFromDescription(description string) (detectedLocations []string) {
	// Load the multi-word countries list
	loadMultiWordCountries()

	// Convert to lowercase for matching
	desc := strings.ToLower(description)
	words := strings.Fields(desc)

	// Check for known countries in description
	for _, countryName := range multiWordCountries {
		countryLower := strings.ToLower(countryName)
		if strings.Contains(desc, countryLower) {
			detectedLocations = append(detectedLocations, countryName)
		}
	}

	// Check for major cities and places (case-insensitive)
	// This is a simple heuristic - could be expanded
	commonLocationWords := []string{"dubai", "paris", "london", "tokyo", "berlin", "rome", "madrid", "amsterdam",
		"new york", "los angeles", "chicago", "houston", "philadelphia", "san antonio", "san diego", "dallas",
		"bangkok", "singapore", "hong kong", "mumbai", "delhi", "bangalore", "sydney", "melbourne", "barcelona"}

	for _, word := range words {
		wordLower := strings.ToLower(word)
		for _, location := range commonLocationWords {
			if wordLower == location {
				detectedLocations = append(detectedLocations, word)
			}
		}
	}

	// Also check for multi-word locations in the description
	for _, location := range commonLocationWords {
		if strings.Contains(desc, location) {
			// Capitalize first letter for consistency
			parts := strings.Fields(location)
			capitalizedParts := make([]string, len(parts))
			for i, part := range parts {
				capitalizedParts[i] = strings.Title(part)
			}
			detectedLocations = append(detectedLocations, strings.Join(capitalizedParts, " "))
		}
	}

	return detectedLocations
}

// promptUserForLocationFromDescription prompts user to confirm location from description
func promptUserForLocationFromDescription(filePath, description string, detectedLocations []string, locationDB *LocationDB) (country, city string, shouldSkip bool, err error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\nFile: %s\n", filepath.Base(filePath))
	fmt.Printf("Image Description: '%s'\n", description)

	if len(detectedLocations) > 0 {
		fmt.Printf("Detected potential locations: %s\n", strings.Join(detectedLocations, ", "))
	}

	// Check if we have an existing mapping for any detected location
	for _, detected := range detectedLocations {
		if mapping, exists := locationDB.GetLocationMapping(detected); exists {
			fmt.Printf("Found existing mapping: %s -> %s\n", detected, mapping.Country)
			fmt.Print("Use this mapping? (y/n/skip): ")

			response, err := reader.ReadString('\n')
			if err != nil {
				return "", "", false, err
			}
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "y" || response == "yes" {
				return mapping.Country, detected, false, nil
			} else if response == "skip" {
				return "", "", true, nil
			}
			// If "n", continue to manual input below
		}
	}

	// Manual input
	fmt.Print("Enter country for this location (or 'skip' to skip this file): ")
	countryInput, err := reader.ReadString('\n')
	if err != nil {
		return "", "", false, err
	}

	country = strings.TrimSpace(countryInput)
	if country == "" || country == "skip" {
		return "", "", true, nil
	}

	fmt.Print("Enter city name: ")
	cityInput, err := reader.ReadString('\n')
	if err != nil {
		return "", "", false, err
	}

	city = strings.TrimSpace(cityInput)
	if city == "" {
		city = "unknown-city"
	}

	return country, city, false, nil
}

// requiresLocationPrompting checks if a file needs location prompting
func requiresLocationPrompting(filePath string) bool {
	// Extract location info from EXIF
	locationInfo, err := extractLocationInfo(filePath)
	if err != nil {
		return false
	}

	// Skip if file has GPS data
	if locationInfo.HasGPS {
		return false
	}

	// Skip if no image description
	if locationInfo.Description == "" {
		return false
	}

	// Detect locations from description
	detectedLocations := detectLocationFromDescription(locationInfo.Description)
	if len(detectedLocations) == 0 {
		return false // No locations detected
	}

	// Check if we already have mappings for all detected locations
	if globalTiffLocationDB != nil {
		for _, detected := range detectedLocations {
			if _, exists := globalTiffLocationDB.GetLocationMapping(detected); !exists {
				return true // Found a location without mapping - needs prompting
			}
		}
	}

	return false // All locations have existing mappings
}

// processTiffJobsSequentially processes TIFF jobs one by one to handle user input
func processTiffJobsSequentially(jobs []WorkJob, showProgress bool) error {
	total := len(jobs)

	if showProgress {
		fmt.Printf("🔄 Processing %d files sequentially...\n", total)
	}

	successCount := 0
	failCount := 0

	for i, job := range jobs {
		if showProgress {
			fmt.Printf("[%d/%d] ", i+1, total)
		}

		err := processTiffFile(job.PhotoPath, job.DryRun)
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
			failCount++
		} else {
			successCount++
		}
	}

	if showProgress {
		fmt.Printf("\n📊 Sequential Processing Summary:\n")
		fmt.Printf("✅ Successful: %d\n", successCount)
		fmt.Printf("❌ Failed: %d\n", failCount)
	}

	return nil
}