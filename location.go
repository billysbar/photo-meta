package main

import (
	"bufio"
	"fmt"
	"os"
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

// promptForLocation prompts the user for missing country/city information
func promptForLocation(location string) (country, city string, err error) {
	fmt.Printf("\n⚠️  Unable to determine country and city from location: %s\n", location)
	fmt.Println("Please provide the missing information:")
	
	reader := bufio.NewReader(os.Stdin)
	
	// Prompt for country
	for {
		fmt.Print("Country: ")
		countryInput, err := reader.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("failed to read country input: %v", err)
		}
		
		country = strings.TrimSpace(countryInput)
		if country != "" {
			country = anglicizeName(country)
		country = strings.ToLower(strings.ReplaceAll(country, " ", "-"))
			break
		}
		fmt.Println("⚠️  Country cannot be empty. Please try again.")
	}
	
	// Prompt for city
	for {
		fmt.Print("City: ")
		cityInput, err := reader.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("failed to read city input: %v", err)
		}
		
		city = strings.TrimSpace(cityInput)
		if city != "" {
			city = anglicizeName(city)
		city = strings.ToLower(strings.ReplaceAll(city, " ", "-"))
			break
		}
		fmt.Println("⚠️  City cannot be empty. Please try again.")
	}
	
	fmt.Printf("✅ Using: %s, %s\n\n", city, country)
	return country, city, nil
}