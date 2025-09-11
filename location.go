package main

import (
	"bufio"
	"embed"
	"fmt"
	"strings"
	"sync"
)

//go:embed multi-word-countries.txt
var countryListFS embed.FS

var (
	multiWordCountries []string
	countryListOnce    sync.Once
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

// promptForLocation defaults to unknown country/city when location cannot be parsed
func promptForLocation(location, mediaPath string, lat, lon float64) (country, city string, err error) {
	// Default country to 'unknown-country'
	country = "unknown-country"
	
	// Default city to 'unknown-city'
	city = "unknown-city"
	
	return country, city, nil
}