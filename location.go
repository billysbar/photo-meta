package main

import (
	"fmt"
	"strings"
)

// parseLocation attempts to extract country and city from a location string
func parseLocation(location string) (country, city string, err error) {
	// Location format is typically "city-country" like "manchester-united-kingdom"
	parts := strings.Split(location, "-")
	
	if len(parts) >= 2 {
		// Handle multi-word countries like "united-kingdom"
		if len(parts) >= 3 && parts[len(parts)-2] == "united" && parts[len(parts)-1] == "kingdom" {
			// Special case for "united-kingdom"
			country = "united-kingdom"
			city = strings.Join(parts[:len(parts)-2], "-")
			return country, city, nil
		}
		
		// Handle other multi-word countries if needed
		// For now, take the last part as country, everything before as city
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