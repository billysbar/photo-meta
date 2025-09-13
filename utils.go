package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// anglicizeName converts non-Latin location names to anglicized versions
func anglicizeName(name string) string {
	// Common name mappings for cities and countries
	nameMappings := map[string]string{
		"kraków":     "krakow",
		"krakow":     "krakow",
		"warszawa":   "warsaw",
		"gdańsk":     "gdansk",
		"wrocław":    "wroclaw",
		"poznań":     "poznan",
		"łódź":       "lodz",
		"münchen":    "munich",
		"köln":       "cologne",
		"zürich":     "zurich",
		"genève":     "geneva",
		"moskva":     "moscow",
		"moskau":     "moscow",
		"praha":      "prague",
		"wien":       "vienna",
		"firenze":    "florence",
		"venezia":    "venice",
		"roma":       "rome",
		"napoli":     "naples",
		"lisboa":     "lisbon",
		"sevilla":    "seville",
		"parís":      "paris",
		"marseille":  "marseilles",
		"athína":     "athens",
		"bucureşti":  "bucharest",
		"beograd":    "belgrade",
		"cambodia":   "cambodia",
		"thailand":   "thailand",
		"vietnam":    "vietnam",
		"singapore":  "singapore",
		"malaysia":   "malaysia",
		"indonesia":  "indonesia",
		"japan":      "japan",
		"nippon":     "japan",
		"nihon":      "japan",
		"deutschland": "germany",
		"france":     "france",
		"españa":     "spain",
		"italia":     "italy",
		"österreich": "austria",
		"states":     "united-states",
		"republic":   "czech-republic",
		"czechia":    "czech-republic",
		
		// Additional Asian locations
		"ha-ni":       "hanoi",          
		"ha-noi":      "hanoi",          
		"hanoi":       "hanoi",
		"ho-chi-minh": "ho-chi-minh-city", 
		"da-nang":     "da-nang",
		"hue":         "hue",
		"can-tho":     "can-tho",
		"hai-phong":   "hai-phong",
		"chiang-mai":  "chiang-mai",
		"phuket":      "phuket",
		"pattaya":     "pattaya",
		"phnom-penh":  "phnom-penh",
		"siem-reap":   "siem-reap",
		"battambang":  "battambang",
		"beijing":     "beijing",
		"shanghai":    "shanghai",
		"guangzhou":   "guangzhou", 
		"shenzhen":    "shenzhen",
		"chengdu":     "chengdu",
		"tokyo":       "tokyo",
		"osaka":       "osaka", 
		"kyoto":       "kyoto",
		"hiroshima":   "hiroshima",
		"nagoya":      "nagoya",
	}

	nameLower := strings.ToLower(name)

	// Check if we have a direct mapping
	if mapped, exists := nameMappings[nameLower]; exists {
		return mapped
	}

	// Remove accents and diacritics
	return removeAccents(name)
}

// removeAccents removes accented characters and converts them to ASCII equivalents
func removeAccents(name string) string {
	result := make([]rune, 0, len(name))
	for _, r := range name {
		switch r {
		// Latin accented characters
		case 'á', 'à', 'ä', 'â', 'ã', 'å', 'ā', 'ă', 'ą':
			result = append(result, 'a')
		case 'Á', 'À', 'Ä', 'Â', 'Ã', 'Å', 'Ā', 'Ă', 'Ą':
			result = append(result, 'A')
		case 'é', 'è', 'ë', 'ê', 'ē', 'ĕ', 'ě', 'ę':
			result = append(result, 'e')
		case 'É', 'È', 'Ë', 'Ê', 'Ē', 'Ĕ', 'Ě', 'Ę':
			result = append(result, 'E')
		case 'í', 'ì', 'ï', 'î', 'ī', 'ĭ', 'į':
			result = append(result, 'i')
		case 'Í', 'Ì', 'Ï', 'Î', 'Ī', 'Ĭ', 'Į':
			result = append(result, 'I')
		case 'ó', 'ò', 'ö', 'ô', 'õ', 'ō', 'ŏ', 'ő':
			result = append(result, 'o')
		case 'Ó', 'Ò', 'Ö', 'Ô', 'Õ', 'Ō', 'Ŏ', 'Ő':
			result = append(result, 'O')
		case 'ú', 'ù', 'ü', 'û', 'ū', 'ŭ', 'ů', 'ű', 'ų':
			result = append(result, 'u')
		case 'Ú', 'Ù', 'Ü', 'Û', 'Ū', 'Ŭ', 'Ů', 'Ű', 'Ų':
			result = append(result, 'U')
		case 'ý', 'ÿ', 'ȳ':
			result = append(result, 'y')
		case 'Ý', 'Ÿ', 'Ȳ':
			result = append(result, 'Y')
		case 'ñ', 'ń', 'ň', 'ņ':
			result = append(result, 'n')
		case 'Ñ', 'Ń', 'Ň', 'Ņ':
			result = append(result, 'N')
		case 'ç', 'ć', 'č', 'ĉ', 'ċ':
			result = append(result, 'c')
		case 'Ç', 'Ć', 'Č', 'Ĉ', 'Ċ':
			result = append(result, 'C')
		case 'ß':
			result = append(result, 's', 's')
		case 'ł', 'ľ', 'ļ', 'ĺ':
			result = append(result, 'l')
		case 'Ł', 'Ľ', 'Ļ', 'Ĺ':
			result = append(result, 'L')
		case 'ś', 'š', 'ş', 'ŝ':
			result = append(result, 's')
		case 'Ś', 'Š', 'Ş', 'Ŝ':
			result = append(result, 'S')
		case 'ź', 'ž', 'ż', 'ẑ':
			result = append(result, 'z')
		case 'Ź', 'Ž', 'Ż', 'Ẑ':
			result = append(result, 'Z')
		case 'ř', 'ŕ', 'ŗ':
			result = append(result, 'r')
		case 'Ř', 'Ŕ', 'Ŗ':
			result = append(result, 'R')
		case 'ť', 'ţ', 'ṫ':
			result = append(result, 't')
		case 'Ť', 'Ţ', 'Ṫ':
			result = append(result, 'T')
		case 'ď', 'đ':
			result = append(result, 'd')
		case 'Ď', 'Đ':
			result = append(result, 'D')
		case 'ğ', 'ģ':
			result = append(result, 'g')
		case 'Ğ', 'Ģ':
			result = append(result, 'G')
		case 'ķ':
			result = append(result, 'k')
		case 'Ķ':
			result = append(result, 'K')
		default:
			// Keep ASCII characters and basic punctuation, remove others
			if r <= 127 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' || r == '.') {
				result = append(result, r)
			}
		}
	}

	return string(result)
}

// mapNonLatinCityName provides fallback mappings for major cities with non-Latin names
func mapNonLatinCityName(cityName, countryName string) string {
	// City mappings by country
	cityMappings := map[string]map[string]string{
		"vietnam": {
			"ha-ni":         "hanoi",          
			"ha-noi":        "hanoi",          
			"hanoi":         "hanoi",
			"ho-chi-minh":   "ho-chi-minh-city", 
			"da-nang":       "da-nang",
			"hue":           "hue",
			"can-tho":       "can-tho",
			"hai-phong":     "hai-phong",
		},
		"thailand": {
			"bangkok":       "bangkok",
			"chiang-mai":    "chiang-mai",
			"phuket":        "phuket",
			"pattaya":       "pattaya",
		},
		"cambodia": {
			"phnom-penh":    "phnom-penh",
			"siem-reap":     "siem-reap",
			"battambang":    "battambang",
		},
		"japan": {
			"tokyo":         "tokyo",
			"osaka":         "osaka", 
			"kyoto":         "kyoto",
			"hiroshima":     "hiroshima",
			"nagoya":        "nagoya",
		},
		"china": {
			"beijing":       "beijing",
			"shanghai":      "shanghai",
			"guangzhou":     "guangzhou", 
			"shenzhen":      "shenzhen",
			"chengdu":       "chengdu",
		},
		"poland": {
			"kraków":        "krakow",
			"krakow":        "krakow",
			"warszawa":      "warsaw",
			"gdańsk":        "gdansk",
			"wrocław":       "wroclaw",
			"poznań":        "poznan",
			"łódź":          "lodz",
		},
		"germany": {
			"münchen":       "munich",
			"köln":          "cologne",
			"deutschland":   "germany",
		},
		"spain": {
			"sevilla":       "seville",
			"españa":        "spain",
		},
		"italy": {
			"firenze":       "florence",
			"venezia":       "venice",
			"roma":          "rome",
			"napoli":        "naples",
		},
		"portugal": {
			"lisboa":        "lisbon",
		},
		"france": {
			"parís":         "paris",
			"marseille":     "marseilles",
		},
		"austria": {
			"wien":          "vienna",
			"österreich":    "austria",
		},
		"switzerland": {
			"zürich":        "zurich",
			"genève":        "geneva",
		},
		"czech-republic": {
			"praha":         "prague",
		},
		"russia": {
			"moskva":        "moscow",
			"moskau":        "moscow",
		},
		"greece": {
			"athína":        "athens",
		},
		"romania": {
			"bucureşti":     "bucharest",
		},
		"serbia": {
			"beograd":       "belgrade",
		},
	}

	// Check if we have mappings for this country
	if countryMappings, exists := cityMappings[countryName]; exists {
		// Try to find a matching city
		cityLower := strings.ToLower(cityName)
		for pattern, mappedName := range countryMappings {
			if strings.Contains(cityLower, pattern) {
				return mappedName
			}
		}
	}

	// If no mapping found, use the country name as city (previous behavior)
	return countryName
}

// parseMonth converts month name strings to month numbers
func parseMonth(monthStr string) (int, error) {
	monthStr = strings.ToLower(monthStr)

	months := map[string]int{
		"january": 1, "jan": 1,
		"february": 2, "feb": 2,
		"march": 3, "mar": 3,
		"april": 4, "apr": 4,
		"may": 5,
		"june": 6, "jun": 6,
		"july": 7, "jul": 7,
		"august": 8, "aug": 8,
		"september": 9, "sep": 9, "sept": 9,
		"october": 10, "oct": 10,
		"november": 11, "nov": 11,
		"december": 12, "dec": 12,
	}

	if month, exists := months[monthStr]; exists {
		return month, nil
	}

	return 0, fmt.Errorf("unknown month: %s", monthStr)
}

// generateFilenameWithTime creates a filename preserving existing hour+minute if present
func generateFilenameWithTime(originalPath string, date time.Time, city string) string {
	currentFilename := filepath.Base(originalPath)
	ext := filepath.Ext(originalPath)

	// Check if filename already has time component (YYYY-MM-DD-HHMM pattern)
	timePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(\d{4})`)
	matches := timePattern.FindStringSubmatch(currentFilename)

	var dateComponent string
	if len(matches) > 1 {
		// Preserve existing time component
		existingTime := matches[1]
		dateComponent = date.Format("2006-01-02") + "-" + existingTime
	} else {
		// No existing time component, use date only
		dateComponent = date.Format("2006-01-02")
	}

	return fmt.Sprintf("%s-%s%s", dateComponent, city, ext)
}