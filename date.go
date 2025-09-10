package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// extractPhotoDate extracts the date from photo metadata
func extractPhotoDate(photoPath string) (time.Time, error) {
	// Use exiftool to extract date information
	args := []string{
		"-DateTimeOriginal",
		"-CreateDate", 
		"-DateTime",
		"-T", // Tab-separated output
		photoPath,
	}
	
	cmd := exec.Command("exiftool", args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return time.Time{}, fmt.Errorf("exiftool failed: %v", err)
	}
	
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return time.Time{}, fmt.Errorf("no date metadata found")
	}
	
	// Parse tab-separated output: DateTimeOriginal \t CreateDate \t DateTime
	parts := strings.Split(outputStr, "\t")
	
	// Try each date field in order of preference
	dateFormats := []string{
		"2006:01:02 15:04:05",
		"2006-01-02 15:04:05",
		"2006:01:02",
		"2006-01-02",
	}
	
	for _, part := range parts {
		dateStr := strings.TrimSpace(part)
		if dateStr == "" || dateStr == "-" {
			continue
		}
		
		// Try different date formats
		for _, format := range dateFormats {
			if date, err := time.Parse(format, dateStr); err == nil {
				return date, nil
			}
		}
	}
	
	return time.Time{}, fmt.Errorf("could not parse any date from metadata")
}