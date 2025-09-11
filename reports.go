package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ReportType defines the type of report to generate
type ReportType string

const (
	ReportTypeSummary    ReportType = "summary"
	ReportTypeDuplicates ReportType = "duplicates" 
	ReportTypeStats      ReportType = "stats"
)

// SummaryScanner tracks directory analysis for comprehensive reporting
type SummaryScanner struct {
	ProcessedFiles     map[string]map[string]map[string]int // year -> country -> city -> count
	UnprocessedFiles   map[string]int                       // directory -> count
	TotalProcessed     int
	TotalUnprocessed   int
	MovedNonImageFiles []string // list of files that would be moved to VIDEO-FILES
	VideoFiles         map[string]int // extension -> count
	TotalVideoFiles    int
	FilesScanned       int // progress counter
	TotalImageFiles    int
	ScanStartTime      time.Time
	ProcessedPatterns  []*regexp.Regexp // patterns for identifying processed files
}

// ReportDuplicateFile extends DuplicateFile with additional report-specific fields
type ReportDuplicateFile struct {
	DuplicateFile
	IsKeep  bool // whether this file should be kept
	Quality int  // structure quality score
}

// ReportDuplicateGroup extends DuplicateGroup with additional report-specific fields
type ReportDuplicateGroup struct {
	DuplicateGroup
	WastedSpace int64
	ReportFiles []ReportDuplicateFile // extended file info
}

// DuplicateScanner tracks duplicate file analysis
type DuplicateScanner struct {
	Groups          []ReportDuplicateGroup
	TotalGroups     int
	TotalWastedSpace int64
	FilesScanned    int
	ScanStartTime   time.Time
}

// ReportConfig controls report generation behavior
type ReportConfig struct {
	OutputFile      string
	GenerateFile    bool
	ShowProgress    bool
	VerboseOutput   bool
	DateFormat      string
	LocationFormat  string
}

// NewSummaryScanner creates a new directory summary scanner
func NewSummaryScanner() *SummaryScanner {
	scanner := &SummaryScanner{
		ProcessedFiles:     make(map[string]map[string]map[string]int),
		UnprocessedFiles:   make(map[string]int),
		MovedNonImageFiles: make([]string, 0),
		VideoFiles:         make(map[string]int),
		ScanStartTime:      time.Now(),
	}
	
	// Initialize processed file patterns
	scanner.ProcessedPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-.*\.(jpg|jpeg|heic|png|tiff|tif)$`),
		regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-.*\.(jpg|jpeg|heic|png|tiff|tif)$`),
	}
	
	return scanner
}

// NewDuplicateScanner creates a new duplicate file scanner
func NewDuplicateScanner() *DuplicateScanner {
	return &DuplicateScanner{
		Groups:        make([]ReportDuplicateGroup, 0),
		ScanStartTime: time.Now(),
	}
}

// processReport generates comprehensive reports based on the report type
func processReport(sourcePath string, reportType ReportType, config ReportConfig) error {
	fmt.Printf("📋 Report Generation - %s\n", strings.Title(string(reportType)))
	fmt.Printf("🔍 Analyzing: %s\n", sourcePath)
	fmt.Printf("⏰ Started: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	switch reportType {
	case ReportTypeSummary:
		return generateSummaryReport(sourcePath, config)
	case ReportTypeDuplicates:
		return generateDuplicatesReport(sourcePath, config)
	case ReportTypeStats:
		return generateStatsReport(sourcePath, config)
	default:
		return fmt.Errorf("unknown report type: %s", reportType)
	}
}

// generateSummaryReport creates a comprehensive directory summary report
func generateSummaryReport(sourcePath string, config ReportConfig) error {
	scanner := NewSummaryScanner()
	
	// Scan directory
	err := scanner.scanDirectory(sourcePath, config.ShowProgress)
	if err != nil {
		return fmt.Errorf("failed to scan directory: %v", err)
	}
	
	// Generate report content
	report := scanner.generateReport(sourcePath)
	
	// Display to console
	fmt.Print(report)
	
	// Save to file if requested
	if config.GenerateFile {
		filename := generateReportFilename(sourcePath, "summary")
		filepath := filepath.Join(sourcePath, filename)
		
		err := saveReportToFile(filepath, report)
		if err != nil {
			return fmt.Errorf("failed to save report: %v", err)
		}
		
		fmt.Printf("\n📄 Report saved to: %s\n", filename)
	}
	
	return nil
}

// scanDirectory performs recursive directory analysis
func (s *SummaryScanner) scanDirectory(sourcePath string, showProgress bool) error {
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		s.FilesScanned++
		
		// Show progress every 100 files
		if showProgress && s.FilesScanned%100 == 0 {
			fmt.Printf("\r🔍 Scanning... %d files analyzed", s.FilesScanned)
		}
		
		filename := filepath.Base(path)
		relPath, _ := filepath.Rel(sourcePath, path)
		dirPath := filepath.Dir(relPath)
		
		// Classify file
		if isPhotoFile(path) {
			s.TotalImageFiles++
			
			// Check if processed
			if s.isProcessedFile(filename, path, sourcePath) {
				s.TotalProcessed++
				s.trackProcessedFile(path, sourcePath)
			} else {
				s.TotalUnprocessed++
				s.UnprocessedFiles[dirPath]++
			}
		} else if isVideoFile(path) {
			// Video files would be moved to VIDEO-FILES
			s.TotalVideoFiles++
			ext := strings.ToLower(filepath.Ext(path))
			s.VideoFiles[ext]++
			s.MovedNonImageFiles = append(s.MovedNonImageFiles, relPath)
		} else {
			// Other files that would be moved
			s.MovedNonImageFiles = append(s.MovedNonImageFiles, relPath)
		}
		
		return nil
	})
}

// isProcessedFile determines if a file appears to be processed based on naming patterns and location
func (s *SummaryScanner) isProcessedFile(filename, fullPath, sourcePath string) bool {
	// Check filename patterns
	for _, pattern := range s.ProcessedPatterns {
		if pattern.MatchString(strings.ToLower(filename)) {
			// Check if it's in a structured directory (YYYY/Country/City)
			relPath, _ := filepath.Rel(sourcePath, fullPath)
			pathParts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
			
			// Look for year pattern in path
			for _, part := range pathParts {
				if matched, _ := regexp.MatchString(`^\d{4}$`, part); matched {
					year := strings.ToLower(part)
					// Basic year validation
					if year >= "1990" && year <= "2030" {
						return true
					}
				}
			}
		}
	}
	return false
}

// trackProcessedFile records processed file location data
func (s *SummaryScanner) trackProcessedFile(fullPath, sourcePath string) {
	relPath, _ := filepath.Rel(sourcePath, fullPath)
	pathParts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
	
	var year, country, city string
	
	// Extract year, country, city from path
	for i, part := range pathParts {
		if matched, _ := regexp.MatchString(`^\d{4}$`, part); matched {
			year = part
			if i+1 < len(pathParts) {
				country = pathParts[i+1]
			}
			if i+2 < len(pathParts) {
				city = pathParts[i+2]
			}
			break
		}
	}
	
	if year == "" {
		year = "unknown"
	}
	if country == "" {
		country = "unknown"
	}
	if city == "" {
		city = "unknown"
	}
	
	// Initialize nested maps
	if s.ProcessedFiles[year] == nil {
		s.ProcessedFiles[year] = make(map[string]map[string]int)
	}
	if s.ProcessedFiles[year][country] == nil {
		s.ProcessedFiles[year][country] = make(map[string]int)
	}
	
	s.ProcessedFiles[year][country][city]++
}

// generateReport creates the formatted summary report
func (s *SummaryScanner) generateReport(sourcePath string) string {
	var report strings.Builder
	
	// Header
	report.WriteString("Photo Metadata Editor - Directory Summary\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("Directory: %s\n", sourcePath))
	report.WriteString("============================================================\n\n")
	
	// Main summary
	report.WriteString("📊 DIRECTORY SUMMARY\n")
	report.WriteString(fmt.Sprintf("🗂️  Total image files found: %d\n", s.TotalImageFiles))
	report.WriteString(fmt.Sprintf("✅ Processed files: %d\n", s.TotalProcessed))
	report.WriteString(fmt.Sprintf("⏳ Unprocessed files: %d\n", s.TotalUnprocessed))
	
	if s.TotalImageFiles > 0 {
		completionPercent := float64(s.TotalProcessed) / float64(s.TotalImageFiles) * 100
		report.WriteString(fmt.Sprintf("📈 Processing completion: %.1f%%\n", completionPercent))
	}
	
	report.WriteString("\n")
	
	// Processed files breakdown
	if s.TotalProcessed > 0 {
		report.WriteString("📍 PROCESSED FILES BY LOCATION:\n")
		
		// Sort years
		var years []string
		for year := range s.ProcessedFiles {
			years = append(years, year)
		}
		sort.Strings(years)
		
		for _, year := range years {
			countries := s.ProcessedFiles[year]
			
			// Sort countries
			var countryNames []string
			for country := range countries {
				countryNames = append(countryNames, country)
			}
			sort.Strings(countryNames)
			
			for _, country := range countryNames {
				cities := countries[country]
				
				// Sort cities
				var cityNames []string
				for city := range cities {
					cityNames = append(cityNames, city)
				}
				sort.Strings(cityNames)
				
				for _, city := range cityNames {
					count := cities[city]
					report.WriteString(fmt.Sprintf("FILES IN %s/%s/%s (%d files)\n", year, country, city, count))
				}
			}
		}
		report.WriteString("\n")
	}
	
	// Unprocessed files breakdown
	if s.TotalUnprocessed > 0 {
		report.WriteString("⏳ UNPROCESSED FILES BY DIRECTORY:\n")
		
		// Sort directories
		var dirs []string
		for dir := range s.UnprocessedFiles {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)
		
		for _, dir := range dirs {
			count := s.UnprocessedFiles[dir]
			if count > 0 {
				report.WriteString(fmt.Sprintf("UNPROCESSED IN %s/ (%d files)\n", dir, count))
			}
		}
		report.WriteString("\n")
	}
	
	// Non-image files that would be moved
	if len(s.MovedNonImageFiles) > 0 {
		report.WriteString(fmt.Sprintf("📁 MOVED NON-IMAGE FILES (%d files):\n", len(s.MovedNonImageFiles)))
		for _, file := range s.MovedNonImageFiles {
			report.WriteString(fmt.Sprintf("   📄 %s -> VIDEO-FILES/%s\n", file, filepath.Base(file)))
		}
		report.WriteString("\n")
	}
	
	// VIDEO-FILES directory summary
	if s.TotalVideoFiles > 0 {
		report.WriteString(fmt.Sprintf("📹 VIDEO-FILES DIRECTORY SUMMARY (%d files):\n", s.TotalVideoFiles))
		
		// Sort extensions
		var exts []string
		for ext := range s.VideoFiles {
			exts = append(exts, ext)
		}
		sort.Strings(exts)
		
		for _, ext := range exts {
			count := s.VideoFiles[ext]
			report.WriteString(fmt.Sprintf("   %s: %d files\n", ext, count))
		}
		report.WriteString("\n")
	}
	
	// Timing information
	duration := time.Since(s.ScanStartTime)
	report.WriteString(fmt.Sprintf("⏱️  Scan completed in: %v\n", duration.Round(time.Millisecond)))
	report.WriteString(fmt.Sprintf("📊 Files scanned: %d\n", s.FilesScanned))
	
	return report.String()
}

// generateReportFilename creates a timestamped filename for reports
func generateReportFilename(sourcePath, reportType string) string {
	dirName := filepath.Base(sourcePath)
	// Clean directory name for filename
	dirName = strings.ReplaceAll(dirName, " ", "-")
	dirName = strings.ReplaceAll(dirName, "/", "-")
	
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	return fmt.Sprintf("%s_%s_%s.txt", reportType, dirName, timestamp)
}

// saveReportToFile writes the report content to a file
func saveReportToFile(filepath, content string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = file.WriteString(content)
	return err
}


// generateDuplicatesReport creates a comprehensive duplicate files report
func generateDuplicatesReport(sourcePath string, config ReportConfig) error {
	scanner := NewDuplicateScanner()
	
	// Scan for duplicates
	err := scanner.scanForDuplicates(sourcePath, config.ShowProgress)
	if err != nil {
		return fmt.Errorf("failed to scan for duplicates: %v", err)
	}
	
	// Generate report content
	report := scanner.generateReport(sourcePath)
	
	// Display to console
	fmt.Print(report)
	
	// Save to file if requested
	if config.GenerateFile {
		filename := generateReportFilename(sourcePath, "duplicates")
		filepath := filepath.Join(sourcePath, filename)
		
		err := saveReportToFile(filepath, report)
		if err != nil {
			return fmt.Errorf("failed to save report: %v", err)
		}
		
		fmt.Printf("\n📄 Report saved to: %s\n", filename)
	}
	
	return nil
}

// scanForDuplicates performs file hashing and duplicate detection
func (d *DuplicateScanner) scanForDuplicates(sourcePath string, showProgress bool) error {
	fileHashes := make(map[string][]ReportDuplicateFile)
	
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Only process media files
		if !isMediaFile(path) {
			return nil
		}
		
		d.FilesScanned++
		
		// Show progress every 50 files (hashing is slower)
		if showProgress && d.FilesScanned%50 == 0 {
			fmt.Printf("\r🔍 Scanning... %d files hashed", d.FilesScanned)
		}
		
		// Calculate file hash
		hash, err := calculateFileHash(path)
		if err != nil {
			return nil // Continue on hash errors
		}
		
		// Use first 16 characters of hash for grouping
		hashKey := hash[:16]
		
		duplicateFile := ReportDuplicateFile{
			DuplicateFile: DuplicateFile{
				Path:     path,
				Hash:     hash,
				Size:     info.Size(),
				ModTime:  info.ModTime(),
				Filename: filepath.Base(path),
			},
			Quality: calculateFileQuality(path, sourcePath),
		}
		
		fileHashes[hashKey] = append(fileHashes[hashKey], duplicateFile)
		
		return nil
	})
	
	if err != nil {
		return err
	}
	
	if showProgress {
		fmt.Printf("\r🔍 Analyzing duplicates...\n")
	}
	
	// Process duplicate groups
	for hashKey, files := range fileHashes {
		if len(files) > 1 {
			// Sort files by quality (best first)
			sort.Slice(files, func(i, j int) bool {
				return files[i].Quality > files[j].Quality
			})
			
			// Mark the best file as keep
			files[0].IsKeep = true
			
			// Calculate wasted space (all files except the keeper)
			var wastedSpace int64
			for i := 1; i < len(files); i++ {
				wastedSpace += files[i].Size
			}
			
			// Create base DuplicateGroup
			baseDuplicateFiles := make([]DuplicateFile, len(files))
			for i, f := range files {
				baseDuplicateFiles[i] = f.DuplicateFile
			}
			
			group := ReportDuplicateGroup{
				DuplicateGroup: DuplicateGroup{
					Hash:  hashKey,
					Size:  files[0].Size,
					Files: baseDuplicateFiles,
				},
				WastedSpace: wastedSpace,
				ReportFiles: files,
			}
			
			d.Groups = append(d.Groups, group)
			d.TotalWastedSpace += wastedSpace
		}
	}
	
	d.TotalGroups = len(d.Groups)
	
	// Sort groups by wasted space (highest first)
	sort.Slice(d.Groups, func(i, j int) bool {
		return d.Groups[i].WastedSpace > d.Groups[j].WastedSpace
	})
	
	return nil
}

// calculateFileQuality assigns a quality score based on file location and naming
func calculateFileQuality(path, sourcePath string) int {
	quality := 0
	
	relPath, _ := filepath.Rel(sourcePath, path)
	filename := filepath.Base(path)
	
	// Higher quality for processed files (good naming pattern)
	processedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-.*\.(jpg|jpeg|heic|png|tiff|tif)$`),
		regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-.*\.(jpg|jpeg|heic|png|tiff|tif)$`),
	}
	
	for _, pattern := range processedPatterns {
		if pattern.MatchString(strings.ToLower(filename)) {
			quality += 50
			break
		}
	}
	
	// Higher quality for structured directory (YYYY/Country/City)
	pathParts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
	for _, part := range pathParts {
		if matched, _ := regexp.MatchString(`^\d{4}$`, part); matched {
			quality += 30
			break
		}
	}
	
	// Lower quality for certain directories
	lowerPath := strings.ToLower(relPath)
	if strings.Contains(lowerPath, "temp") || strings.Contains(lowerPath, "tmp") {
		quality -= 20
	}
	if strings.Contains(lowerPath, "duplicate") || strings.Contains(lowerPath, "copy") {
		quality -= 30
	}
	
	// File extension preferences
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		quality += 10
	case ".heic":
		quality += 5
	case ".png":
		quality += 3
	}
	
	return quality
}

// generateReport creates the formatted duplicates report
func (d *DuplicateScanner) generateReport(sourcePath string) string {
	var report strings.Builder
	
	// Header
	report.WriteString("Photo Metadata Editor - Duplicate Files Report\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("Directory: %s\n", sourcePath))
	report.WriteString("============================================================\n\n")
	
	if d.TotalGroups == 0 {
		report.WriteString("🎉 No duplicate files found!\n\n")
	} else {
		report.WriteString(fmt.Sprintf("🔍 Found %d duplicate groups\n", d.TotalGroups))
		report.WriteString(fmt.Sprintf("💾 Total wasted space: %s\n\n", formatFileSize(d.TotalWastedSpace)))
		
		// Show each duplicate group
		for i, group := range d.Groups {
			report.WriteString(fmt.Sprintf("=== Group %d: %d files (%s each) ===\n", i+1, len(group.ReportFiles), formatFileSize(group.Size)))
			report.WriteString(fmt.Sprintf("Hash: %s...\n", group.Hash))
			report.WriteString(fmt.Sprintf("Wasted space: %s\n\n", formatFileSize(group.WastedSpace)))
			
			for j, file := range group.ReportFiles {
				status := "duplicate"
				if file.IsKeep {
					status = "KEEP"
				}
				
				report.WriteString(fmt.Sprintf("  %d. %s (%s)\n", j+1, file.Path, status))
				report.WriteString(fmt.Sprintf("     Modified: %s\n", file.ModTime.Format("2006-01-02 15:04:05")))
				if file.Quality > 0 {
					report.WriteString(fmt.Sprintf("     Quality score: %d\n", file.Quality))
				}
				report.WriteString("\n")
			}
		}
	}
	
	// Summary
	report.WriteString("=== Summary ===\n")
	report.WriteString(fmt.Sprintf("Total files scanned: %d\n", d.FilesScanned))
	report.WriteString(fmt.Sprintf("Total duplicate groups: %d\n", d.TotalGroups))
	report.WriteString(fmt.Sprintf("Total wasted space: %s\n", formatFileSize(d.TotalWastedSpace)))
	
	duration := time.Since(d.ScanStartTime)
	report.WriteString(fmt.Sprintf("Scan completed in: %v\n", duration.Round(time.Millisecond)))
	
	return report.String()
}

// generateStatsReport creates a general statistics report
func generateStatsReport(sourcePath string, config ReportConfig) error {
	fmt.Printf("📊 Statistics Report\n")
	fmt.Printf("🔍 Directory: %s\n", sourcePath)
	fmt.Printf("⏰ Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	// File type statistics
	var totalFiles, photoFiles, videoFiles, otherFiles int
	photoExtensions := make(map[string]int)
	videoExtensions := make(map[string]int)
	var totalSize int64
	
	startTime := time.Now()
	
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if info.IsDir() {
			return nil
		}
		
		totalFiles++
		totalSize += info.Size()
		
		ext := strings.ToLower(filepath.Ext(path))
		
		if isPhotoFile(path) {
			photoFiles++
			photoExtensions[ext]++
		} else if isVideoFile(path) {
			videoFiles++
			videoExtensions[ext]++
		} else {
			otherFiles++
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to scan directory: %v", err)
	}
	
	// Display statistics
	fmt.Printf("📁 File Statistics:\n")
	fmt.Printf("  📷 Photos: %d\n", photoFiles)
	fmt.Printf("  🎥 Videos: %d\n", videoFiles)
	fmt.Printf("  📄 Other files: %d\n", otherFiles)
	fmt.Printf("  📊 Total files: %d\n", totalFiles)
	fmt.Printf("  💾 Total size: %s\n\n", formatFileSize(totalSize))
	
	if len(photoExtensions) > 0 {
		fmt.Printf("📷 Photo Extensions:\n")
		// Sort by count (descending)
		type extCount struct {
			ext   string
			count int
		}
		var photoExts []extCount
		for ext, count := range photoExtensions {
			photoExts = append(photoExts, extCount{ext, count})
		}
		sort.Slice(photoExts, func(i, j int) bool {
			return photoExts[i].count > photoExts[j].count
		})
		for _, ec := range photoExts {
			fmt.Printf("  %s: %d files\n", ec.ext, ec.count)
		}
		fmt.Printf("\n")
	}
	
	if len(videoExtensions) > 0 {
		fmt.Printf("🎥 Video Extensions:\n")
		// Sort by count (descending)
		type extCount struct {
			ext   string
			count int
		}
		var videoExts []extCount
		for ext, count := range videoExtensions {
			videoExts = append(videoExts, extCount{ext, count})
		}
		sort.Slice(videoExts, func(i, j int) bool {
			return videoExts[i].count > videoExts[j].count
		})
		for _, ec := range videoExts {
			fmt.Printf("  %s: %d files\n", ec.ext, ec.count)
		}
		fmt.Printf("\n")
	}
	
	duration := time.Since(startTime)
	fmt.Printf("⏱️  Analysis completed in: %v\n", duration.Round(time.Millisecond))
	
	// Save to file if requested
	if config.GenerateFile {
		report := fmt.Sprintf(`Photo Metadata Editor - Statistics Report
Generated: %s
Directory: %s
============================================================

File Statistics:
  Photos: %d
  Videos: %d
  Other files: %d
  Total files: %d
  Total size: %s

Analysis completed in: %v
`, time.Now().Format("2006-01-02 15:04:05"), sourcePath, photoFiles, videoFiles, otherFiles, totalFiles, formatFileSize(totalSize), duration.Round(time.Millisecond))
		
		filename := generateReportFilename(sourcePath, "stats")
		filepath := filepath.Join(sourcePath, filename)
		
		err := saveReportToFile(filepath, report)
		if err != nil {
			return fmt.Errorf("failed to save report: %v", err)
		}
		
		fmt.Printf("📄 Report saved to: %s\n", filename)
	}
	
	return nil
}