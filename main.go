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
			fmt.Println("Usage: ./photo-metadata-editor process /source/path /destination/path")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		destPath := os.Args[3]
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Create destination path if it doesn't exist
		if err := os.MkdirAll(destPath, 0755); err != nil {
			log.Fatalf("Failed to create destination path: %v", err)
		}
		
		// Process photos
		if err := processPhotos(sourcePath, destPath); err != nil {
			log.Fatal(err)
		}
		
	case "datetime":
		if len(os.Args) < 4 {
			fmt.Println("Usage: ./photo-metadata-editor datetime /source/path /destination/path")
			os.Exit(1)
		}
		
		sourcePath := os.Args[2]
		destPath := os.Args[3]
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			log.Fatalf("Source path does not exist: %s", sourcePath)
		}
		
		// Check if destination path exists
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			log.Fatalf("Destination path does not exist: %s", destPath)
		}
		
		// Process datetime matching
		if err := processDateTimeMatching(sourcePath, destPath); err != nil {
			log.Fatal(err)
		}
		
	case "clean":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ./photo-metadata-editor clean /target/path [--dry-run] [--verbose]")
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
				fmt.Println("Use '--dry-run' instead")
				os.Exit(1)
			}
		}
		
		// Parse optional flags
		dryRun := false
		verbose := false
		for i := 3; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "--dry-run":
				dryRun = true
			case "--verbose":
				verbose = true
			}
		}
		
		// Process clean (duplicate removal)
		if err := processClean(targetPath, dryRun, verbose); err != nil {
			log.Fatal(err)
		}
		
	default:
		showUsage()
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Println("📸 Photo Metadata Editor - Simplified Version")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  ./photo-metadata-editor process /source/path /destination/path")
	fmt.Println("  ./photo-metadata-editor datetime /source/path /destination/path")
	fmt.Println("  ./photo-metadata-editor clean /target/path [--dry-run] [--verbose]")
	fmt.Println()
	fmt.Println("Process Features:")
	fmt.Println("  - Extracts GPS location data from photos")
	fmt.Println("  - Renames files to YYYY-MM-DD-location format")
	fmt.Println("  - Organizes files into YEAR/COUNTRY/CITY directory structure under destination")
	fmt.Println("  - Merges files if destination structure already exists")
	fmt.Println()
	fmt.Println("DateTime Features:")
	fmt.Println("  - Matches files by date from filename to existing location structure")
	fmt.Println("  - Uses processed photos as location database")
	fmt.Println("  - Interactive mode with prompts for verification")
	fmt.Println()
	fmt.Println("Clean Features:")
	fmt.Println("  - Intelligent duplicate detection using SHA-256 hashing")
	fmt.Println("  - Structure-based selection (keeps best organized files)")
	fmt.Println("  - Supports --dry-run and --verbose modes")
	fmt.Println("  - Prioritizes processed files over unorganized ones")
	fmt.Println()
}

func processPhotos(sourcePath, destPath string) error {
	fmt.Printf("🔍 Processing photos from: %s\n", sourcePath)
	fmt.Printf("📁 Destination: %s\n", destPath)
	
	processedCount := 0
	skippedCounts := make(map[string]int) // Track by file extension
	
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check if it's a photo file
		if !isPhotoFile(path) {
			return nil
		}
		
		// Process the photo
		if err := processPhoto(path, destPath); err != nil {
			if isNoGPSError(err) {
				ext := filepath.Ext(path)
				skippedCounts[ext]++
				return nil // Continue processing other files
			}
			return err // Return other errors
		}
		
		processedCount++
		return nil
	})
	
	// Report summary
	fmt.Printf("\n📊 Processing Summary:\n")
	fmt.Printf("✅ Files processed with GPS data: %d\n", processedCount)
	
	totalSkipped := 0
	for _, count := range skippedCounts {
		totalSkipped += count
	}
	fmt.Printf("⚠️  Files skipped (no GPS data): %d\n", totalSkipped)
	
	if len(skippedCounts) > 0 {
		fmt.Printf("   Breakdown by type:\n")
		for ext, count := range skippedCounts {
			fmt.Printf("   - %s: %d files\n", ext, count)
		}
	}
	
	return err
}

func isPhotoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	
	// Common photo/image formats that can contain GPS metadata
	supportedFormats := map[string]bool{
		".jpg":   true,
		".jpeg":  true,
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
		".dcr":   true,  // Kodak RAW
		".mrw":   true,  // Minolta RAW
		".raw":   true,  // Generic RAW
	}
	
	return supportedFormats[ext]
}

func processPhoto(photoPath, destBasePath string) error {
	fmt.Printf("📷 Processing: %s\n", filepath.Base(photoPath))
	
	// Extract GPS coordinates
	lat, lon, err := extractGPSCoordinates(photoPath)
	if err != nil {
		return &NoGPSError{File: photoPath, Err: err}
	}
	
	// Get location from coordinates
	location, err := getLocationFromCoordinates(lat, lon)
	if err != nil {
		return fmt.Errorf("failed to get location for %s: %v", filepath.Base(photoPath), err)
	}
	
	fmt.Printf("📍 Location: %s (%.6f, %.6f)\n", location, lat, lon)
	
	// Extract date from photo
	date, err := extractPhotoDate(photoPath)
	if err != nil {
		return fmt.Errorf("failed to extract date from %s: %v", filepath.Base(photoPath), err)
	}
	
	// Parse location into country and city
	country, city, err := parseLocation(location)
	if err != nil {
		// Prompt for missing country/city information
		country, city, err = promptForLocation(location)
		if err != nil {
			return fmt.Errorf("failed to get location information: %v", err)
		}
	}
	
	// Generate new filename and directory structure using destination base path
	newFilename := fmt.Sprintf("%s-%s%s", 
		date.Format("2006-01-02"), 
		city, 
		filepath.Ext(photoPath))
	
	// Smart directory structure - check if destination already ends with the year
	year := date.Format("2006")
	var newDir string
	
	// Check if destination path already ends with the year
	destBase := filepath.Base(destBasePath)
	if destBase == year {
		// Destination already ends with year (e.g., "/tmp/2025"), so just add country/city
		newDir = filepath.Join(destBasePath, country, city)
	} else {
		// Destination doesn't end with year, so add full structure
		newDir = filepath.Join(destBasePath, year, country, city)
	}
	
	newPath := filepath.Join(newDir, newFilename)
	
	// Create directory structure if it doesn't exist
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
	if err := os.Rename(photoPath, finalPath); err != nil {
		return fmt.Errorf("failed to move file from %s to %s: %v", photoPath, finalPath, err)
	}
	
	fmt.Printf("✅ Moved to: %s\n", finalPath)
	return nil
}