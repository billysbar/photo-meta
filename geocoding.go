package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// NominatimResponse represents the response from OpenStreetMap's Nominatim API
type NominatimResponse struct {
	DisplayName string `json:"display_name"`
	Address     struct {
		City    string `json:"city"`
		Town    string `json:"town"`
		Village string `json:"village"`
		County  string `json:"county"`
		State   string `json:"state"`
		Country string `json:"country"`
	} `json:"address"`
}

// getLocationFromCoordinates converts GPS coordinates to location using reverse geocoding
func getLocationFromCoordinates(lat, lon float64) (string, error) {
	// Check offline mapping first (for common locations)
	if offlineLocation := tryOfflineMapping(lat, lon); offlineLocation != "" {
		return offlineLocation, nil
	}
	
	// Use OpenStreetMap Nominatim API for reverse geocoding
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?lat=%f&lon=%f&format=json&addressdetails=1&accept-language=en", lat, lon)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("geocoding request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("geocoding API returned status: %d", resp.StatusCode)
	}
	
	var result NominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse geocoding response: %v", err)
	}
	
	return formatLocationName(result), nil
}

// formatLocationName extracts city and country from geocoding result
func formatLocationName(result NominatimResponse) string {
	// Extract city name in order of preference
	cityName := ""
	if result.Address.City != "" {
		cityName = result.Address.City
	} else if result.Address.Town != "" {
		cityName = result.Address.Town
	} else if result.Address.Village != "" {
		cityName = result.Address.Village
	} else if result.Address.County != "" {
		cityName = result.Address.County
	} else if result.Address.State != "" {
		cityName = result.Address.State
	}
	
	countryName := result.Address.Country
	
	if cityName == "" && countryName == "" {
		return "unknown-location"
	}
	
	// Anglicize and clean up names first
	cityName = anglicizeName(cityName)
	cityName = strings.ReplaceAll(cityName, " ", "-")
	cityName = strings.ToLower(cityName)

	countryName = anglicizeName(countryName)
	countryName = strings.ReplaceAll(countryName, " ", "-")
	countryName = strings.ToLower(countryName)

	// Try to map common city names to their standard forms
	mappedCity := mapNonLatinCityName(cityName, countryName)
	if mappedCity != cityName && mappedCity != countryName {
		cityName = mappedCity
	}
	
	if cityName == "" {
		return countryName
	}
	
	if countryName == "" {
		return cityName
	}
	
	// Return format: city-country
	return fmt.Sprintf("%s-%s", cityName, countryName)
}


// tryOfflineMapping attempts to map coordinates to known locations without network calls
func tryOfflineMapping(lat, lon float64) string {
	// Define bounding boxes for major cities/regions with comprehensive coverage
	locations := []struct {
		name   string
		latMin float64
		latMax float64
		lonMin float64
		lonMax float64
	}{
		// Japan major cities
		{"tokyo-japan", 35.4, 35.9, 139.4, 139.9},
		{"osaka-japan", 34.5, 34.8, 135.3, 135.7},
		{"kyoto-japan", 34.9, 35.1, 135.6, 135.9},
		{"kanazawa-japan", 36.5, 36.7, 136.5, 136.8},
		{"hiroshima-japan", 34.3, 34.5, 132.3, 132.6},
		{"nagoya-japan", 35.0, 35.3, 136.7, 137.1},
		{"yokohama-japan", 35.3, 35.5, 139.5, 139.8},
		{"kobe-japan", 34.6, 34.8, 135.1, 135.3},
		{"fukuoka-japan", 33.5, 33.7, 130.3, 130.5},
		
		// Southeast Asia
		{"singapore-singapore", 1.2, 1.5, 103.6, 104.0},
		{"bangkok-thailand", 13.6, 13.9, 100.4, 100.7},
		{"chiang-mai-thailand", 18.7, 18.9, 98.9, 99.1},
		{"phuket-thailand", 7.8, 8.1, 98.3, 98.4},
		{"pattaya-thailand", 12.9, 13.0, 100.8, 100.9},
		{"siem-reap-cambodia", 13.2, 13.5, 103.7, 104.0},
		{"phnom-penh-cambodia", 11.4, 11.7, 104.8, 105.1},
		{"battambang-cambodia", 13.0, 13.2, 103.1, 103.3},
		{"hanoi-vietnam", 20.9, 21.1, 105.7, 106.0},
		{"ho-chi-minh-city-vietnam", 10.6, 10.9, 106.5, 106.9},
		{"da-nang-vietnam", 15.9, 16.1, 108.1, 108.3},
		{"hue-vietnam", 16.4, 16.5, 107.5, 107.6},
		{"can-tho-vietnam", 10.0, 10.1, 105.7, 105.8},
		{"hai-phong-vietnam", 20.8, 20.9, 106.6, 106.7},
		{"kuala-lumpur-malaysia", 3.0, 3.3, 101.6, 101.8},
		{"penang-malaysia", 5.3, 5.5, 100.2, 100.4},
		{"jakarta-indonesia", -6.3, -6.0, 106.7, 107.0},
		{"bali-indonesia", -8.8, -8.1, 115.0, 115.7},
		{"yogyakarta-indonesia", -7.9, -7.7, 110.3, 110.4},
		
		// Europe - Western
		{"paris-france", 48.7, 49.0, 2.1, 2.5},
		{"marseilles-france", 43.2, 43.4, 5.3, 5.4},
		{"nice-france", 43.6, 43.8, 7.2, 7.3},
		{"lyon-france", 45.7, 45.8, 4.8, 4.9},
		{"toulouse-france", 43.5, 43.7, 1.4, 1.5},
		{"bordeaux-france", 44.8, 44.9, -0.6, -0.5},
		
		{"london-united-kingdom", 51.3, 51.7, -0.3, 0.2},
		{"manchester-united-kingdom", 53.4, 53.5, -2.3, -2.2},
		{"liverpool-united-kingdom", 53.3, 53.5, -3.0, -2.9},
		{"birmingham-united-kingdom", 52.4, 52.5, -1.9, -1.8},
		{"glasgow-united-kingdom", 55.8, 55.9, -4.3, -4.2},
		{"edinburgh-united-kingdom", 55.9, 56.0, -3.2, -3.1},
		
		{"madrid-spain", 40.2, 40.6, -3.9, -3.5},
		{"barcelona-spain", 41.2, 41.5, 1.9, 2.3},
		{"seville-spain", 37.3, 37.4, -6.0, -5.9},
		{"valencia-spain", 39.4, 39.5, -0.4, -0.3},
		{"palma-spain", 39.3, 39.8, 2.4, 2.9},          // Palma, Majorca
		{"majorca-spain", 39.2, 39.9, 2.3, 3.5},        // Majorca island
		{"ibiza-spain", 38.9, 39.0, 1.4, 1.5},
		{"san-sebastian-spain", 43.3, 43.4, -2.0, -1.9},
		
		{"rome-italy", 41.7, 42.0, 12.3, 12.7},
		{"florence-italy", 43.7, 43.8, 11.2, 11.3},
		{"venice-italy", 45.4, 45.5, 12.3, 12.4},
		{"naples-italy", 40.8, 40.9, 14.2, 14.3},
		{"milan-italy", 45.4, 45.5, 9.1, 9.2},
		{"turin-italy", 45.0, 45.1, 7.6, 7.7},
		{"bologna-italy", 44.4, 44.5, 11.3, 11.4},
		
		{"lisbon-portugal", 38.6, 38.8, -9.2, -9.1},
		{"porto-portugal", 41.1, 41.2, -8.7, -8.6},
		
		// Europe - Central
		{"berlin-germany", 52.3, 52.7, 13.2, 13.6},
		{"munich-germany", 48.1, 48.2, 11.5, 11.6},
		{"hamburg-germany", 53.5, 53.6, 9.9, 10.0},
		{"cologne-germany", 50.9, 51.0, 6.9, 7.0},
		{"frankfurt-germany", 50.1, 50.2, 8.6, 8.7},
		{"stuttgart-germany", 48.7, 48.8, 9.1, 9.2},
		{"dusseldorf-germany", 51.2, 51.3, 6.7, 6.8},
		
		{"amsterdam-netherlands", 52.2, 52.5, 4.7, 5.1},
		{"rotterdam-netherlands", 51.9, 52.0, 4.4, 4.5},
		{"the-hague-netherlands", 52.0, 52.1, 4.3, 4.4},
		{"utrecht-netherlands", 52.0, 52.1, 5.1, 5.2},
		
		{"brussels-belgium", 50.8, 50.9, 4.3, 4.4},
		{"antwerp-belgium", 51.2, 51.3, 4.4, 4.5},
		{"ghent-belgium", 51.0, 51.1, 3.7, 3.8},
		
		{"zurich-switzerland", 47.3, 47.4, 8.5, 8.6},
		{"geneva-switzerland", 46.1, 46.3, 6.1, 6.2},
		{"basel-switzerland", 47.5, 47.6, 7.5, 7.6},
		{"bern-switzerland", 46.9, 47.0, 7.4, 7.5},
		
		{"vienna-austria", 48.1, 48.3, 16.2, 16.5},
		{"salzburg-austria", 47.7, 47.8, 13.0, 13.1},
		{"innsbruck-austria", 47.2, 47.3, 11.3, 11.4},
		
		// Europe - Eastern
		{"prague-czech-republic", 50.0, 50.1, 14.4, 14.5},
		{"brno-czech-republic", 49.1, 49.3, 16.5, 16.7},
		
		{"krakow-poland", 50.0, 50.1, 19.9, 20.0},
		{"warsaw-poland", 52.2, 52.3, 21.0, 21.1},
		{"gdansk-poland", 54.3, 54.4, 18.6, 18.7},
		{"wroclaw-poland", 51.1, 51.2, 17.0, 17.1},
		{"poznan-poland", 52.4, 52.5, 16.9, 17.0},
		{"lodz-poland", 51.7, 51.8, 19.4, 19.5},
		
		{"budapest-hungary", 47.4, 47.6, 19.0, 19.1},
		
		{"bucharest-romania", 44.4, 44.5, 26.0, 26.2},
		{"cluj-napoca-romania", 46.7, 46.8, 23.5, 23.6},
		
		{"sofia-bulgaria", 42.6, 42.8, 23.3, 23.4},
		
		{"belgrade-serbia", 44.7, 44.9, 20.4, 20.5},
		
		{"zagreb-croatia", 45.8, 45.9, 15.9, 16.0},
		{"split-croatia", 43.5, 43.6, 16.4, 16.5},
		{"dubrovnik-croatia", 42.6, 42.7, 18.1, 18.2},
		
		// Europe - Nordic
		{"stockholm-sweden", 59.2, 59.4, 17.9, 18.2},
		{"gothenburg-sweden", 57.7, 57.8, 11.9, 12.0},
		
		{"copenhagen-denmark", 55.6, 55.7, 12.5, 12.6},
		
		{"oslo-norway", 59.9, 60.0, 10.7, 10.8},
		{"bergen-norway", 60.3, 60.4, 5.3, 5.4},
		
		{"helsinki-finland", 60.1, 60.2, 24.9, 25.0},
		
		{"reykjavik-iceland", 64.1, 64.2, -22.0, -21.9},
		
		// Europe - Mediterranean
		{"athens-greece", 37.9, 38.0, 23.7, 23.8},
		{"thessaloniki-greece", 40.6, 40.7, 22.9, 23.0},
		{"santorini-greece", 36.3, 36.5, 25.4, 25.5},
		{"mykonos-greece", 37.4, 37.5, 25.3, 25.4},
		
		{"istanbul-turkey", 41.0, 41.1, 28.9, 29.0},
		{"ankara-turkey", 39.9, 40.0, 32.8, 32.9},
		{"antalya-turkey", 36.8, 37.0, 30.6, 30.8},
		{"cappadocia-turkey", 38.6, 38.8, 34.8, 35.0},
		
		{"valletta-malta", 35.8, 35.9, 14.5, 14.6},
		
		{"nicosia-cyprus", 35.1, 35.2, 33.3, 33.4},
		{"limassol-cyprus", 34.6, 34.7, 33.0, 33.1},
		
		// Russia
		{"moscow-russia", 55.7, 55.8, 37.6, 37.7},
		{"saint-petersburg-russia", 59.9, 60.0, 30.3, 30.4},
		
		// North America
		{"new-york-united-states", 40.5, 40.9, -74.1, -73.7},
		{"los-angeles-united-states", 33.9, 34.3, -118.5, -118.1},
		{"san-francisco-united-states", 37.6, 37.9, -122.6, -122.2},
		{"chicago-united-states", 41.7, 42.1, -87.9, -87.5},
		{"las-vegas-united-states", 36.1, 36.2, -115.2, -115.1},
		{"miami-united-states", 25.7, 25.8, -80.3, -80.2},
		{"seattle-united-states", 47.5, 47.7, -122.4, -122.2},
		{"boston-united-states", 42.3, 42.4, -71.1, -71.0},
		{"washington-united-states", 38.8, 39.0, -77.1, -76.9},
		{"philadelphia-united-states", 39.9, 40.0, -75.2, -75.1},
		{"phoenix-united-states", 33.4, 33.5, -112.1, -112.0},
		{"denver-united-states", 39.7, 39.8, -105.0, -104.9},
		{"atlanta-united-states", 33.7, 33.8, -84.4, -84.3},
		
		{"toronto-canada", 43.5, 43.9, -79.6, -79.2},
		{"vancouver-canada", 49.2, 49.3, -123.2, -123.1},
		{"montreal-canada", 45.4, 45.6, -73.7, -73.5},
		{"calgary-canada", 51.0, 51.1, -114.1, -114.0},
		{"ottawa-canada", 45.4, 45.5, -75.7, -75.6},
		
		{"mexico-city-mexico", 19.3, 19.5, -99.2, -99.0},
		{"cancun-mexico", 21.1, 21.2, -86.9, -86.8},
		{"guadalajara-mexico", 20.6, 20.7, -103.4, -103.3},
		
		// South America
		{"buenos-aires-argentina", -34.7, -34.5, -58.5, -58.3},
		{"rio-de-janeiro-brazil", -22.9, -22.8, -43.3, -43.1},
		{"sao-paulo-brazil", -23.6, -23.5, -46.7, -46.6},
		{"lima-peru", -12.1, -12.0, -77.1, -77.0},
		{"santiago-chile", -33.5, -33.4, -70.7, -70.6},
		{"bogota-colombia", 4.5, 4.7, -74.1, -74.0},
		{"caracas-venezuela", 10.4, 10.5, -66.9, -66.8},
		
		// Africa
		{"cairo-egypt", 30.0, 30.1, 31.2, 31.3},
		{"marrakech-morocco", 31.6, 31.7, -8.0, -7.9},
		{"casablanca-morocco", 33.5, 33.6, -7.6, -7.5},
		{"cape-town-south-africa", -33.9, -33.9, 18.4, 18.5},
		{"johannesburg-south-africa", -26.2, -26.1, 28.0, 28.1},
		{"nairobi-kenya", -1.3, -1.2, 36.8, 36.9},
		{"tunis-tunisia", 36.8, 36.9, 10.1, 10.2},
		{"algiers-algeria", 36.7, 36.8, 3.0, 3.1},
		
		// Oceania
		{"sydney-australia", -34.1, -33.7, 150.9, 151.3},
		{"melbourne-australia", -37.9, -37.7, 144.8, 145.2},
		{"brisbane-australia", -27.5, -27.4, 153.0, 153.1},
		{"perth-australia", -31.9, -31.9, 115.8, 115.9},
		{"adelaide-australia", -34.9, -34.9, 138.6, 138.7},
		{"canberra-australia", -35.3, -35.3, 149.1, 149.2},
		
		{"auckland-new-zealand", -36.9, -36.8, 174.7, 174.8},
		{"wellington-new-zealand", -41.3, -41.3, 174.7, 174.8},
		{"christchurch-new-zealand", -43.5, -43.5, 172.6, 172.7},
		
		// India & South Asia
		{"mumbai-india", 18.8, 19.3, 72.7, 73.1},
		{"delhi-india", 28.4, 28.9, 77.0, 77.4},
		{"bangalore-india", 12.9, 13.0, 77.5, 77.6},
		{"kolkata-india", 22.5, 22.6, 88.3, 88.4},
		{"chennai-india", 13.0, 13.1, 80.2, 80.3},
		{"hyderabad-india", 17.3, 17.4, 78.4, 78.5},
		{"pune-india", 18.5, 18.6, 73.8, 73.9},
		{"jaipur-india", 26.9, 27.0, 75.8, 75.9},
		{"goa-india", 15.2, 15.6, 73.8, 74.2},
		
		{"karachi-pakistan", 24.8, 24.9, 67.0, 67.1},
		{"lahore-pakistan", 31.5, 31.6, 74.3, 74.4},
		{"islamabad-pakistan", 33.6, 33.7, 73.0, 73.1},
		
		{"dhaka-bangladesh", 23.7, 23.8, 90.3, 90.4},
		
		{"colombo-sri-lanka", 6.9, 7.0, 79.8, 79.9},
		
		{"kathmandu-nepal", 27.7, 27.8, 85.3, 85.4},
		
		// China
		{"beijing-china", 39.7, 40.1, 116.2, 116.6},
		{"shanghai-china", 31.0, 31.4, 121.3, 121.7},
		{"guangzhou-china", 23.1, 23.2, 113.2, 113.3},
		{"shenzhen-china", 22.5, 22.6, 114.0, 114.1},
		{"chengdu-china", 30.6, 30.7, 104.0, 104.1},
		{"xi-an-china", 34.2, 34.3, 108.9, 109.0},
		{"hangzhou-china", 30.2, 30.3, 120.1, 120.2},
		{"nanjing-china", 32.0, 32.1, 118.7, 118.8},
		{"wuhan-china", 30.5, 30.6, 114.3, 114.4},
		{"tianjin-china", 39.1, 39.2, 117.2, 117.3},
		
		// Middle East
		{"dubai-united-arab-emirates", 25.2, 25.3, 55.2, 55.3},
		{"abu-dhabi-united-arab-emirates", 24.4, 24.5, 54.3, 54.4},
		{"doha-qatar", 25.2, 25.3, 51.5, 51.6},
		{"kuwait-city-kuwait", 29.3, 29.4, 47.9, 48.0},
		{"riyadh-saudi-arabia", 24.6, 24.7, 46.7, 46.8},
		{"jeddah-saudi-arabia", 21.4, 21.5, 39.2, 39.3},
		{"muscat-oman", 23.5, 23.6, 58.4, 58.5},
		{"manama-bahrain", 26.2, 26.3, 50.5, 50.6},
		{"beirut-lebanon", 33.8, 33.9, 35.4, 35.5},
		{"damascus-syria", 33.5, 33.6, 36.2, 36.3},
		{"amman-jordan", 31.9, 32.0, 35.9, 36.0},
		{"jerusalem-israel", 31.7, 31.8, 35.2, 35.3},
		{"tel-aviv-israel", 32.0, 32.1, 34.7, 34.8},
		{"tehran-iran", 35.6, 35.7, 51.3, 51.4},
		{"baghdad-iraq", 33.3, 33.4, 44.3, 44.4},
	}
	
	// Check if coordinates fall within any known region
	for _, loc := range locations {
		if lat >= loc.latMin && lat <= loc.latMax &&
			lon >= loc.lonMin && lon <= loc.lonMax {
			return loc.name
		}
	}
	
	return "" // No offline match found
}