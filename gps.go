package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// extractGPSCoordinates extracts GPS latitude and longitude from photo metadata
func extractGPSCoordinates(photoPath string) (lat float64, lon float64, err error) {
	// Use exiftool to extract GPS coordinates
	args := []string{
		"-GPSLatitude",
		"-GPSLongitude", 
		"-n", // Output coordinates in decimal degrees
		"-T", // Tab-separated output
		photoPath,
	}
	
	cmd := exec.Command("exiftool", args...)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return 0, 0, fmt.Errorf("exiftool failed: %v", err)
	}
	
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" || outputStr == "-\t-" {
		return 0, 0, fmt.Errorf("no GPS data found")
	}
	
	// Parse tab-separated output: lat \t lon
	parts := strings.Split(outputStr, "\t")
	if len(parts) != 2 || parts[0] == "-" || parts[1] == "-" {
		return 0, 0, fmt.Errorf("invalid GPS data format")
	}
	
	latStr := strings.TrimSpace(parts[0])
	lonStr := strings.TrimSpace(parts[1])
	
	if latStr == "" || lonStr == "" {
		return 0, 0, fmt.Errorf("empty GPS coordinates")
	}
	
	// Convert to float
	lat, err = strconv.ParseFloat(latStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid latitude: %v", err)
	}
	
	lon, err = strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid longitude: %v", err)
	}
	
	// Validate coordinate ranges
	if lat < -90.0 || lat > 90.0 {
		return 0, 0, fmt.Errorf("latitude %f out of valid range [-90, 90]", lat)
	}
	
	if lon < -180.0 || lon > 180.0 {
		return 0, 0, fmt.Errorf("longitude %f out of valid range [-180, 180]", lon)
	}
	
	return lat, lon, nil
}