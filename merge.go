package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// processMerge handles the merge command workflow
func processMerge(sourcePath, targetPath string, workers int, dryRun bool, dryRunSampleSize int, showProgress bool) error {
	fmt.Printf("🔀 Merge Mode - Combining Photos from Source into Target\n")
	fmt.Printf("📂 Source: %s\n", sourcePath)
	fmt.Printf("📁 Target: %s\n", targetPath)
	
	if dryRun {
		if dryRunSampleSize > 0 {
			fmt.Printf("🔍 DRY RUN MODE - Sample merge preview (%d file(s) per type per directory)\n", dryRunSampleSize)
		} else {
			fmt.Println("🔍 DRY RUN MODE - No files will be moved")
		}
	}
	fmt.Println()

	// Collect files to merge
	var jobs []WorkJob
	var err error
	
	if dryRunSampleSize > 0 {
		// For sampling mode, sample N photos and N videos per directory
		jobs, err = collectSampleFilesForMerge(sourcePath, targetPath, dryRun, dryRunSampleSize)
	} else {
		// Collect all files from source
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
				DestPath:  targetPath,
				JobType:   "merge",
				DryRun:    dryRun,
			})
			
			return nil
		})
	}
	
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		fmt.Println("📭 No media files found to merge")
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

	fmt.Printf("📝 Found %d media files to merge (%d photos, %d videos)\n", len(jobs), photoCount, videoCount)

	// Process jobs concurrently
	return processJobsConcurrentlyWithProgress(jobs, workers, showProgress)
}

// collectSampleFilesForMerge collects a sample of files for merge dry-run1 mode
func collectSampleFilesForMerge(sourcePath, targetPath string, dryRun bool, sampleSize int) ([]WorkJob, error) {
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
					DestPath:  targetPath,
					JobType:   "merge",
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
					DestPath:  targetPath,
					JobType:   "merge",
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

// processMergeFile processes a single file for merge operation
func processMergeFile(sourcePath, targetPath string, dryRun bool) error {
	// Determine file type for display
	var fileType string
	var fileIcon string
	if isVideoFile(sourcePath) {
		fileType = "video"
		fileIcon = "🎥"
	} else {
		fileType = "photo"
		fileIcon = "📷"
	}
	
	if dryRun {
		fmt.Printf("%s [DRY RUN] Merging %s: %s\n", fileIcon, fileType, filepath.Base(sourcePath))
	} else {
		fmt.Printf("%s Merging %s: %s\n", fileIcon, fileType, filepath.Base(sourcePath))
	}

	// Check if file already exists in target directory structure
	existingPath, exists, err := findExistingFileInTarget(sourcePath, targetPath)
	if err != nil {
		return fmt.Errorf("failed to check for existing file: %v", err)
	}

	if exists {
		if dryRun {
			fmt.Printf("⚠️  [DRY RUN] File already exists in target: %s\n", existingPath)
		} else {
			fmt.Printf("⚠️  File already exists in target: %s\n", existingPath)
		}
		return nil
	}

	// Extract GPS coordinates if available
	lat, lon, err := extractGPSCoordinates(sourcePath)
	if err != nil {
		// For merge, we'll try to infer location from the target directory structure
		// or fall back to a default location
		return processMergeFileWithoutGPS(sourcePath, targetPath, dryRun)
	}

	// Get location from coordinates
	location, err := getLocationFromCoordinates(lat, lon)
	if err != nil {
		return fmt.Errorf("failed to get location for %s: %v", filepath.Base(sourcePath), err)
	}

	fmt.Printf("📍 Location: %s (%.6f, %.6f)\n", location, lat, lon)

	// Extract date from media file
	date, err := extractPhotoDate(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to extract date from %s: %v", filepath.Base(sourcePath), err)
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
			country, city, err = promptForLocation(location, sourcePath, lat, lon)
			if err != nil {
				return fmt.Errorf("failed to get location information: %v", err)
			}
		}
	}

	// Generate target path using YEAR/COUNTRY/CITY structure
	return moveToTargetStructure(sourcePath, targetPath, date, country, city, dryRun)
}

// processMergeFileWithoutGPS handles files without GPS data by trying to infer location from target structure
func processMergeFileWithoutGPS(sourcePath, targetPath string, dryRun bool) error {
	fmt.Printf("⚠️  No GPS data found, attempting to infer from target directory structure\n")

	// Extract date from media file
	date, err := extractPhotoDate(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to extract date from %s: %v", filepath.Base(sourcePath), err)
	}

	// Try to find existing photos from the same date in target structure
	year := date.Format("2006")
	inferredLocation, err := inferLocationFromTarget(targetPath, year, date.Format("2006-01-02"))
	if err != nil || inferredLocation == nil {
		// Fall back to "unknown" location
		if dryRun {
			fmt.Printf("📍 Using fallback location: unknown-unknown\n")
		} else {
			fmt.Printf("📍 Using fallback location: unknown-unknown\n")
		}
		return moveToTargetStructure(sourcePath, targetPath, date, "unknown-country", "unknown-city", dryRun)
	}

	fmt.Printf("📍 Inferred location from target: %s/%s\n", inferredLocation.Country, inferredLocation.City)
	return moveToTargetStructure(sourcePath, targetPath, date, inferredLocation.Country, inferredLocation.City, dryRun)
}

// LocationInfo holds country and city information
type LocationInfo struct {
	Country string
	City    string
}

// inferLocationFromTarget tries to infer location from existing files in target structure
func inferLocationFromTarget(targetPath, year, dateStr string) (*LocationInfo, error) {
	yearPath := filepath.Join(targetPath, year)
	
	// Check if year directory exists
	if _, err := os.Stat(yearPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("year directory does not exist: %s", year)
	}

	// Look for existing files from the same date
	var foundLocation *LocationInfo
	
	err := filepath.Walk(yearPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check if filename starts with the target date
		if strings.HasPrefix(info.Name(), dateStr) {
			// Extract country and city from path
			relPath, err := filepath.Rel(targetPath, path)
			if err != nil {
				return nil
			}
			
			pathParts := strings.Split(relPath, string(filepath.Separator))
			if len(pathParts) >= 3 {
				// Format should be YEAR/COUNTRY/CITY/filename
				country := pathParts[1]
				city := pathParts[2]
				foundLocation = &LocationInfo{
					Country: country,
					City:    city,
				}
				return fmt.Errorf("found") // Stop walking
			}
		}
		
		return nil
	})
	
	if err != nil && err.Error() == "found" {
		return foundLocation, nil
	}
	
	return foundLocation, nil
}

// findExistingFileInTarget checks if a file already exists in the target directory structure
func findExistingFileInTarget(sourcePath, targetPath string) (string, bool, error) {
	sourceFilename := filepath.Base(sourcePath)
	
	// Search through target directory structure
	var foundPath string
	var found bool
	
	err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check if filename matches
		if info.Name() == sourceFilename {
			// Additional check: compare file sizes to ensure it's actually the same file
			sourceInfo, err := os.Stat(sourcePath)
			if err != nil {
				return nil
			}
			
			if info.Size() == sourceInfo.Size() {
				foundPath = path
				found = true
				return fmt.Errorf("found") // Stop walking
			}
		}
		
		return nil
	})
	
	if err != nil && err.Error() == "found" {
		return foundPath, found, nil
	}
	
	return foundPath, found, nil
}

// moveToTargetStructure moves/copies file to target with YEAR/COUNTRY/CITY structure
func moveToTargetStructure(sourcePath, targetPath string, date time.Time, country, city string, dryRun bool) error {
	// Generate new filename and directory structure
	year := date.Format("2006")
	newFilename := fmt.Sprintf("%s-%s%s", 
		date.Format("2006-01-02"), 
		city, 
		filepath.Ext(sourcePath))
	
	var newDir string
	
	// For video files, place in VIDEO-FILES subdirectory
	if isVideoFile(sourcePath) {
		newDir = filepath.Join(targetPath, "VIDEO-FILES", year, country, city)
	} else {
		// For photo files, use regular structure
		newDir = filepath.Join(targetPath, year, country, city)
	}
	
	newPath := filepath.Join(newDir, newFilename)
	
	// In dry run mode, just show what would happen
	if dryRun {
		// Handle duplicates simulation
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
		
		if isVideoFile(sourcePath) {
			fmt.Printf("✅ [DRY RUN] Video would be merged to: %s\n", finalPath)
		} else {
			fmt.Printf("✅ [DRY RUN] Photo would be merged to: %s\n", finalPath)
		}
		return nil
	}
	
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
	
	// Copy the file (preserve original in source)
	if err := copyFile(sourcePath, finalPath); err != nil {
		return fmt.Errorf("failed to copy file from %s to %s: %v", sourcePath, finalPath, err)
	}
	
	if isVideoFile(sourcePath) {
		fmt.Printf("✅ Video merged to: %s\n", finalPath)
	} else {
		fmt.Printf("✅ Photo merged to: %s\n", finalPath)
	}
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}
	
	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	return os.Chmod(dst, sourceInfo.Mode())
}