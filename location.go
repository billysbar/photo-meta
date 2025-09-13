package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed multi-word-countries.txt
var countryListFS embed.FS

var (
	multiWordCountries []string
	countryListOnce    sync.Once

	// Global flag to pause progress reporting during user input
	progressPaused     bool
	progressPauseMutex sync.Mutex
)

// loadMultiWordCountries loads the list of multi-word countries from the embedded file
func loadMultiWordCountries() {
	countryListOnce.Do(func() {
		file, err := countryListFS.Open("multi-word-countries.txt")
		if err != nil {
			fmt.Printf("Warning: Could not load multi-word countries list: %v\n", err)
			return
		}
		defer file.Close()
		
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Skip comments and empty lines
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			multiWordCountries = append(multiWordCountries, line)
		}
		
		if err := scanner.Err(); err != nil {
			fmt.Printf("Warning: Error reading multi-word countries list: %v\n", err)
		}
	})
}

// parseLocation attempts to extract country and city from a location string
func parseLocation(location string) (country, city string, err error) {
	// Load the multi-word countries list
	loadMultiWordCountries()
	
	// Location format is typically "city-country" like "manchester-united-kingdom"
	location = strings.ToLower(location)
	
	// Try to match against known multi-word countries (longest first)
	for _, countryName := range multiWordCountries {
		if strings.HasSuffix(location, "-"+countryName) {
			// Found a match - extract city by removing the country suffix
			cityPart := strings.TrimSuffix(location, "-"+countryName)
			if cityPart != "" {
				return countryName, cityPart, nil
			}
			// If no city part, the whole location is just the country
			return countryName, countryName, nil
		}
		
		// Also check if the entire location is just the country name
		if location == countryName {
			return countryName, countryName, nil
		}
	}
	
	// If no multi-word country matched, fall back to simple parsing
	parts := strings.Split(location, "-")
	if len(parts) >= 2 {
		// Take the last part as country, everything before as city
		country = parts[len(parts)-1]
		city = strings.Join(parts[:len(parts)-1], "-")
		return country, city, nil
	}
	
	// If we can't parse it, return an error so we can prompt user
	return "", "", fmt.Errorf("unable to parse location: %s", location)
}

// promptForLocation prompts user for location information when parsing fails
func promptForLocation(location, mediaPath string, lat, lon float64) (country, city string, err error) {
	// Pause any progress reporting during user input
	pauseProgress()
	defer resumeProgress()

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n📸 File: %s\n", filepath.Base(mediaPath))
	if location != "" {
		fmt.Printf("Could not parse location: '%s'\n", location)
	}
	if lat != 0 || lon != 0 {
		fmt.Printf("GPS coordinates: %.6f, %.6f\n", lat, lon)
	}

	fmt.Print("Enter country for this location (or 'skip' to use defaults): ")
	countryInput, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	country = strings.TrimSpace(countryInput)
	if country == "" || strings.ToLower(country) == "skip" {
		// Return defaults for skip/empty
		return "unknown-country", "unknown-city", nil
	}

	fmt.Print("Enter city name: ")
	cityInput, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	city = strings.TrimSpace(cityInput)
	if city == "" {
		city = country // Default city to country if empty
	}

	// Clean up the names using existing utility functions
	country = anglicizeName(country)
	city = anglicizeName(city)

	return country, city, nil
}

// promptForLocationWithDatabase prompts user for location with database integration
func promptForLocationWithDatabase(location, mediaPath string, lat, lon float64, locationDB *LocationDB) (country, city string, shouldSkip bool, err error) {
	// Pause any progress reporting during user input
	pauseProgress()
	defer resumeProgress()

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n📸 File: %s\n", filepath.Base(mediaPath))
	if location != "" {
		fmt.Printf("Could not parse location: '%s'\n", location)
	}
	if lat != 0 || lon != 0 {
		fmt.Printf("GPS coordinates: %.6f, %.6f\n", lat, lon)
	}

	// Check if we have existing mappings to suggest
	if locationDB != nil && location != "" {
		// Try to find partial matches in the database
		if mappings, err := locationDB.ListAllMappings(); err == nil {
			var suggestions []string
			locationLower := strings.ToLower(location)
			for city, country := range mappings {
				if strings.Contains(locationLower, city) || strings.Contains(city, locationLower) {
					suggestions = append(suggestions, fmt.Sprintf("%s -> %s", city, country))
				}
			}
			if len(suggestions) > 0 {
				fmt.Println("Similar locations in database:")
				for _, suggestion := range suggestions {
					fmt.Printf("  - %s\n", suggestion)
				}
			}
		}
	}

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

	// Save to database if provided
	if locationDB != nil {
		if err := locationDB.SaveLocationMapping(city, country, true); err != nil {
			fmt.Printf("⚠️ Warning: Failed to save location mapping: %v\n", err)
		} else {
			fmt.Printf("💾 Saved location mapping: %s -> %s\n", city, country)
		}
	}

	return country, city, false, nil
}

// pauseProgress pauses progress reporting during user input
func pauseProgress() {
	progressPauseMutex.Lock()
	progressPaused = true
	progressPauseMutex.Unlock()
}

// resumeProgress resumes progress reporting after user input
func resumeProgress() {
	progressPauseMutex.Lock()
	progressPaused = false
	progressPauseMutex.Unlock()
}

// isProgressPaused checks if progress reporting is paused
func isProgressPaused() bool {
	progressPauseMutex.Lock()
	defer progressPauseMutex.Unlock()
	return progressPaused
}