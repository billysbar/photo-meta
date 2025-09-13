package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LocationMapping represents a single city-country mapping
type LocationMapping struct {
	CityName      string    `json:"city_name"`
	Country       string    `json:"country"`
	UserConfirmed bool      `json:"user_confirmed"`
	CreatedAt     time.Time `json:"created_at"`
}

// LocationDB manages the location mapping database
type LocationDB struct {
	filePath string
	mappings map[string]LocationMapping // city_name -> LocationMapping
}

// NewLocationDB creates a new location database
func NewLocationDB() (*LocationDB, error) {
	// Create database file in the same directory as the binary
	dbPath := "photo-locations.json"
	
	ldb := &LocationDB{
		filePath: dbPath,
		mappings: make(map[string]LocationMapping),
	}
	
	// Try to load existing data
	if err := ldb.loadFromFile(); err != nil {
		// If file doesn't exist, that's okay - we'll create it on save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load location database: %v", err)
		}
	}
	
	return ldb, nil
}

// Close closes the location database
func (ldb *LocationDB) Close() error {
	// Save any pending changes to file
	return ldb.saveToFile()
}

// loadFromFile loads the location mappings from the JSON file
func (ldb *LocationDB) loadFromFile() error {
	data, err := os.ReadFile(ldb.filePath)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, &ldb.mappings)
}

// saveToFile saves the location mappings to the JSON file
func (ldb *LocationDB) saveToFile() error {
	data, err := json.MarshalIndent(ldb.mappings, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(ldb.filePath, data, 0644)
}

// GetCountryForCity looks up the country for a given city
func (ldb *LocationDB) GetCountryForCity(city string) (country string, found bool, err error) {
	city = strings.ToLower(strings.TrimSpace(city))
	
	mapping, exists := ldb.mappings[city]
	if !exists {
		return "", false, nil // Not found, but no error
	}
	
	return mapping.Country, true, nil
}

// SaveLocationMapping saves a city-country mapping to the database
func (ldb *LocationDB) SaveLocationMapping(city, country string, userConfirmed bool) error {
	city = strings.ToLower(strings.TrimSpace(city))
	country = strings.ToLower(strings.TrimSpace(country))
	
	if city == "" || country == "" {
		return fmt.Errorf("city and country cannot be empty")
	}
	
	// Create or update the mapping
	ldb.mappings[city] = LocationMapping{
		CityName:      city,
		Country:       country,
		UserConfirmed: userConfirmed,
		CreatedAt:     time.Now(),
	}
	
	// Save to file immediately
	return ldb.saveToFile()
}

// ListAllMappings returns all stored location mappings
func (ldb *LocationDB) ListAllMappings() (map[string]string, error) {
	mappings := make(map[string]string)

	for city, mapping := range ldb.mappings {
		mappings[city] = mapping.Country
	}

	return mappings, nil
}

// GetLocationMapping retrieves a location mapping by city name
func (ldb *LocationDB) GetLocationMapping(cityName string) (LocationMapping, bool) {
	cityName = strings.ToLower(strings.TrimSpace(cityName))
	mapping, exists := ldb.mappings[cityName]
	return mapping, exists
}

// processOrganizeByLocation handles organizing files that have location information in the filename
func processOrganizeByLocation(sourcePath, destPath string, dryRun bool, dryRunSampleSize int, showProgress bool) error {
	fmt.Printf("📍 Location-Based Organization Mode\n")
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

	// Initialize location database
	locationDB, err := NewLocationDB()
	if err != nil {
		return fmt.Errorf("failed to initialize location database: %v", err)
	}
	defer locationDB.Close()

	// Show existing mappings
	if mappings, err := locationDB.ListAllMappings(); err == nil && len(mappings) > 0 {
		fmt.Printf("📚 Found %d existing location mappings in database:\n", len(mappings))
		for city, country := range mappings {
			fmt.Printf("  %s → %s\n", city, country)
		}
		fmt.Println()
	}

	// Process files in source path that have location in filename
	fmt.Println("🔄 Processing files with location information in filename...")
	return processFilesWithLocationInFilename(sourcePath, destPath, locationDB, dryRun, dryRunSampleSize, showProgress)
}

// processFilesWithLocationInFilename processes files that have location info in their filename
func processFilesWithLocationInFilename(sourcePath, destPath string, locationDB *LocationDB, dryRun bool, dryRunSampleSize int, showProgress bool) error {
	processedCount := 0
	videoCount := 0
	photoCount := 0
	unmatchedFiles := []string{}

	// Collect files to process
	var filesToProcess []string

	if dryRunSampleSize > 0 {
		// Sample files for sampling mode
		var err error
		filesToProcess, err = collectSampleFilesForOrganize(sourcePath, dryRunSampleSize)
		if err != nil {
			return err
		}
		fmt.Printf("📋 Sampled %d files for location-based organization preview\n", len(filesToProcess))
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
		filename := filepath.Base(path)
		
		// Extract date from filename
		date, err := extractDateFromFilename(filename)
		if err != nil {
			fmt.Printf("🔍 DEBUG: %s - no date found: %v\n", filename, err)
			unmatchedFiles = append(unmatchedFiles, path)
			continue
		}

		// Parse the date to extract year and month
		dateParts := strings.Split(date, "-")
		if len(dateParts) < 3 {
			unmatchedFiles = append(unmatchedFiles, path)
			continue
		}
		
		year := dateParts[0]
		monthNum := dateParts[1]

		// Try to extract location from filename
		country, city, locationFound := extractLocationFromFilename(filename, year, monthNum)
		
		if !locationFound {
			fmt.Printf("🔍 DEBUG: %s - no location found in filename\n", filename)
			unmatchedFiles = append(unmatchedFiles, path)
			continue
		}

		// Check if we can determine the country for this city
		finalCountry, finalCity, needsPrompt := validateLocationWithDB(locationDB, country, city, filename)

		if needsPrompt && !dryRun {
			// Prompt immediately for this file
			fmt.Printf("\nFile: %s\n", filename)
			fmt.Printf("Detected location: %s\n", city)

			// Prompt for country and city
			promptedCountry, promptedCity, err := promptUserForLocation(city, filename)
			if err != nil {
				fmt.Printf("⚠️ Skipping %s due to user input error: %v\n", filename, err)
				unmatchedFiles = append(unmatchedFiles, path)
				continue
			}

			if promptedCountry == "skip" {
				fmt.Printf("⏭️ Skipping %s as requested\n", filename)
				unmatchedFiles = append(unmatchedFiles, path)
				continue
			}

			// Save the user-provided mapping to the database
			if err := locationDB.SaveLocationMapping(promptedCity, promptedCountry, true); err != nil {
				fmt.Printf("⚠️ Warning: Failed to save location mapping for %s->%s: %v\n", promptedCity, promptedCountry, err)
			} else {
				fmt.Printf("💾 Saved location mapping: %s -> %s\n", promptedCity, promptedCountry)
			}

			// Use the prompted values
			finalCountry = promptedCountry
			finalCity = promptedCity
		} else if needsPrompt && dryRun {
			// In dry run mode, just show what would be prompted
			fmt.Printf("🤔 [DRY RUN] Would prompt for location: %s (detected city: %s)\n", filename, city)
			unmatchedFiles = append(unmatchedFiles, path)
			continue
		}

		// Create location path: YYYY/COUNTRY/CITY
		location := fmt.Sprintf("%s/%s/%s", year, finalCountry, finalCity)
		fmt.Printf("📍 File: %s -> Date: %s -> Location: %s/%s\n", filename, date, finalCountry, finalCity)

		// Move file to location-based structure
		if err := moveFileToLocationStructure(path, destPath, location, date, finalCity, dryRun); err != nil {
			return fmt.Errorf("failed to move %s: %v", filename, err)
		}

		processedCount++
		if isVideoFile(path) {
			videoCount++
		} else {
			photoCount++
		}
	}


	// Summary
	fmt.Printf("\n📊 Location-Based Organization Summary:\n")
	fmt.Printf("✅ Total files processed: %d\n", processedCount)
	fmt.Printf("📷 Photos processed: %d\n", photoCount)
	fmt.Printf("🎥 Videos processed: %d (moved to VIDEO-FILES/)\n", videoCount)
	fmt.Printf("⚠️  Files unmatched: %d\n", len(unmatchedFiles))

	if len(unmatchedFiles) > 0 {
		fmt.Println("\n📋 Unmatched Files (no location in filename):")
		for _, file := range unmatchedFiles {
			fmt.Printf("  - %s\n", filepath.Base(file))
		}
		fmt.Println("\nℹ️ These files can be processed with the 'fallback' command for date-only organization.")
	}

	return nil
}

// extractLocationFromFilename tries to extract location information from filename
func extractLocationFromFilename(filename, year, month string) (country, city string, found bool) {
	// Remove the file extension and convert to lowercase
	name := strings.ToLower(strings.TrimSuffix(filename, filepath.Ext(filename)))
	
	// Remove date patterns to isolate location part
	// Pattern 1: Remove YYYY-MM-DD- prefix
	datePattern1 := fmt.Sprintf("%s-%s-\\d{2}-", year, month)
	if matched, _ := regexp.MatchString("^"+datePattern1, name); matched {
		re := regexp.MustCompile("^" + datePattern1)
		locationPart := re.ReplaceAllString(name, "")
		return parseLocationPart(locationPart)
	}
	
	// Pattern 2: Remove YYYYMMDD prefix and look for location
	datePattern2 := fmt.Sprintf("%s%s\\d{2}", year, month)
	if matched, _ := regexp.MatchString("^"+datePattern2, name); matched {
		re := regexp.MustCompile("^" + datePattern2 + "[^a-z]*")
		locationPart := re.ReplaceAllString(name, "")
		if locationPart != "" {
			return parseLocationPart(locationPart)
		}
	}
	
	// Pattern 3: Look for location names anywhere in filename (like "scarborough")
	// This is the example case: if filename contains a city name
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	
	for _, word := range words {
		if len(word) > 3 && isLikelyLocationName(word) {
			// Found a potential location, return it as city with unknown country
			return "unknown-country", word, true
		}
	}
	
	return "", "", false
}

// parseLocationPart parses a location part that may contain country-city format
func parseLocationPart(locationPart string) (country, city string, found bool) {
	if locationPart == "" {
		return "", "", false
	}

	// Split by hyphens to analyze parts
	parts := strings.Split(locationPart, "-")

	// Filter out numeric parts and short parts that are likely sequence numbers
	var locationWords []string
	for _, part := range parts {
		// Skip numeric parts (like "2", "3", etc. which are sequence numbers)
		if isNumericString(part) {
			continue
		}
		// Skip very short parts that are unlikely to be location names
		if len(part) <= 2 {
			continue
		}
		locationWords = append(locationWords, part)
	}

	// If we have location words, treat the first one as city
	if len(locationWords) > 0 {
		city = locationWords[0]
		// If we have more than one word, the last might be country
		if len(locationWords) >= 2 {
			country = locationWords[len(locationWords)-1]
			// But only if it looks like a known country pattern, otherwise assume it's part of city name
			if !isLikelyCountryName(country) {
				city = strings.Join(locationWords, "-")
				country = "unknown-country"
			}
		} else {
			country = "unknown-country"
		}
		return country, city, true
	}

	return "", "", false
}

// isNumericString checks if a string is purely numeric
func isNumericString(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

// isLikelyCountryName checks if a word looks like a country name
func isLikelyCountryName(word string) bool {
	// Very basic heuristics - in practice you might want a more comprehensive list
	commonCountries := map[string]bool{
		"usa": true, "uk": true, "france": true, "spain": true, "italy": true,
		"germany": true, "canada": true, "australia": true, "japan": true,
		"china": true, "india": true, "brazil": true, "mexico": true,
		"england": true, "scotland": true, "wales": true, "ireland": true,
	}
	return commonCountries[strings.ToLower(word)]
}

// isLikelyLocationName checks if a word looks like a location name
func isLikelyLocationName(word string) bool {
	// Simple heuristics for location names
	// - Length > 3 characters
	// - Contains only letters (no numbers)
	// - Not common photography terms
	
	if len(word) <= 3 {
		return false
	}
	
	// Check if it's all letters
	for _, r := range word {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	
	// Exclude common photography terms
	excludedWords := map[string]bool{
		"photo": true, "image": true, "picture": true, "shot": true,
		"camera": true, "lens": true, "flash": true, "digital": true,
		"copy": true, "edit": true, "final": true, "raw": true,
		"jpeg": true, "png": true, "heic": true,
	}
	
	return !excludedWords[word]
}

// validateLocationWithDB checks if the extracted location makes sense and whether we need to prompt user
// It uses the database to look up previously saved mappings
func validateLocationWithDB(locationDB *LocationDB, country, city, filename string) (finalCountry, finalCity string, needsPrompt bool) {
	// If country is "unknown-country", check database first
	if country == "unknown-country" {
		if dbCountry, found, err := locationDB.GetCountryForCity(city); err == nil && found {
			fmt.Printf("📚 Found %s in database: %s -> %s\n", city, city, dbCountry)
			return dbCountry, city, false
		}
		// Not in database, need to prompt
		return country, city, true
	}
	
	// If we have both country and city, validate them
	if country != "" && city != "" {
		// Save this mapping to database for future use (non-user-confirmed)
		if err := locationDB.SaveLocationMapping(city, country, false); err == nil {
			fmt.Printf("📚 Auto-saved location mapping: %s -> %s\n", city, country)
		}
		return country, city, false
	}
	
	// Something is missing, prompt user
	return country, city, true
}

// promptUserForLocation prompts the user to confirm/provide country and city
func promptUserForLocation(detectedCity, filename string) (country, city string, err error) {
	// Pause any progress reporting during user input
	pauseProgress()
	defer resumeProgress()

	reader := bufio.NewReader(os.Stdin)
	
	fmt.Printf("Cannot determine country for location '%s' in file: %s\n", detectedCity, filename)
	fmt.Print("Enter country for this location (or 'skip' to skip this file): ")
	
	countryInput, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	
	country = strings.TrimSpace(countryInput)
	if country == "" || country == "skip" {
		return "skip", "", nil
	}
	
	fmt.Print("Enter city name (press Enter to use detected city): ")
	cityInput, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	
	city = strings.TrimSpace(cityInput)
	if city == "" {
		city = detectedCity
	}
	
	// Clean up the names using standardized utility functions
	country = anglicizeName(country)
	city = anglicizeName(city)
	
	return country, city, nil
}

// moveFileToLocationStructure moves file to the location-based directory structure
func moveFileToLocationStructure(sourcePath, destBasePath, location, date, city string, dryRun bool) error {
	// Generate new filename using date-city format, preserving existing hour+minute if present
	dateTime, _ := time.Parse("2006-01-02", date) // Convert date string back to time for helper
	newFilename := generateFilenameWithTime(sourcePath, dateTime, city)

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

	// Handle duplicates
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

// collectSampleFilesForOrganize collects sample files for organize dry-run mode
func collectSampleFilesForOrganize(sourcePath string, sampleSize int) ([]string, error) {
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