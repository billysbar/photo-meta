# Photo Metadata Editor - Simplified Version

A streamlined photo organization tool that processes photos with GPS data and organizes them into a structured directory format.

## Features

- **GPS Data Processing**: Extracts GPS coordinates from photo metadata
- **Smart Location Mapping**: Converts coordinates to city/country names using offline mapping and reverse geocoding
- **Automatic File Organization**: Creates `YEAR/COUNTRY/CITY` directory structure
- **File Renaming**: Renames files to `YYYY-MM-DD-location` format
- **Merge Support**: Merges files into existing directory structures
- **Comprehensive Coverage**: Supports 200+ major cities worldwide with offline mapping
- **Multi-format Support**: Works with JPG, JPEG, HEIC files

## Requirements

- **Go 1.21+** - For building the application
- **exiftool** - For extracting photo metadata
  
  Install exiftool:
  ```bash
  # macOS
  brew install exiftool
  
  # Ubuntu/Debian
  sudo apt-get install exiftool
  
  # Other systems: https://exiftool.org/install.html
  ```

## Installation

1. Clone or download the source code
2. Build the application:
   ```bash
   go build
   ```

## Usage

### Basic Command

```bash
./photo-meta process /source/path /destination/path
```

### Parameters

- **`/source/path`** - Directory containing photos to process
- **`/destination/path`** - Base directory where organized photos will be stored

### Examples

#### Example 1: Basic Processing
```bash
./photo-meta process ~/Downloads/vacation-photos ~/Photos/organized
```
This processes photos from `~/Downloads/vacation-photos` and organizes them under `~/Photos/organized/`.

#### Example 2: Year-Based Destination
```bash
./photo-meta process ~/vacation-photos /tmp/2025
```
Since destination ends with "2025", creates structure: `/tmp/2025/spain/palma/2025-09-02-palma.HEIC`

#### Example 3: General Destination
```bash
./photo-meta process ~/vacation-photos ~/Photos
```
Creates structure: `~/Photos/2025/spain/palma/2025-09-02-palma.HEIC`

## How It Works

1. **Scans Source Directory** - Recursively finds all photo files (JPG, JPEG, HEIC)
2. **GPS Extraction** - Uses exiftool to extract GPS coordinates from photo metadata
3. **Location Resolution** - Converts coordinates to location names using:
   - Offline mapping for 200+ major cities (fastest)
   - OpenStreetMap Nominatim API for other locations
4. **Smart Directory Creation** - Creates organized folder structure:
   - If destination ends with year (e.g., `/tmp/2025`): `country/city/`
   - Otherwise: `year/country/city/`
5. **File Processing** - Renames and moves files with duplicate handling

## Directory Structure Examples

### Input Files
```
vacation-photos/
├── IMG_001.HEIC (GPS: Palma, Spain)
├── IMG_002.JPG  (no GPS data)
└── IMG_003.HEIC (GPS: Manchester, UK)
```

### Output Structure (destination: `/organized`)
```
organized/
├── 2025/
│   ├── spain/
│   │   └── palma/
│   │       └── 2025-09-02-palma.HEIC
│   └── united-kingdom/
│       └── manchester/
│           └── 2025-08-31-manchester.HEIC
```

### Output Structure (destination: `/organized/2025`)
```
organized/2025/
├── spain/
│   └── palma/
│       └── 2025-09-02-palma.HEIC
└── united-kingdom/
    └── manchester/
        └── 2025-08-31-manchester.HEIC
```

## Supported Locations

### Offline Mapping (200+ cities)
The tool includes offline GPS coordinate mapping for major cities worldwide, including:

- **Europe**: London, Paris, Rome, Madrid, Berlin, Amsterdam, Vienna, Prague...
- **Asia**: Tokyo, Singapore, Bangkok, Seoul, Mumbai, Delhi, Beijing, Shanghai...
- **Americas**: New York, Los Angeles, Toronto, Mexico City, Buenos Aires...
- **Australia/Oceania**: Sydney, Melbourne, Auckland...
- **Africa**: Cairo, Cape Town, Marrakech...
- **Middle East**: Dubai, Istanbul, Tel Aviv...

### Online Geocoding
For locations not in the offline database, the tool uses OpenStreetMap's Nominatim API.

## Processing Summary

After processing, the tool displays:
```
📊 Processing Summary:
✅ Files processed with GPS data: 15
⚠️  Files skipped (no GPS data): 3
   Breakdown by type:
   - .JPG: 2 files
   - .JPEG: 1 files
```

## File Handling

### GPS Data Present
- Extracts date from photo metadata
- Converts GPS coordinates to location
- Creates organized directory structure
- Renames file with date and location
- Handles duplicates with numeric suffixes (`-1`, `-2`, etc.)

### No GPS Data
- Files are skipped (not moved or renamed)
- Reported in processing summary by file type
- Original files remain unchanged

## Interactive Prompts

If location parsing fails, the tool will prompt for manual input:
```
⚠️  Unable to determine country and city from location: unknown-location
Please provide the missing information:
Country: spain
City: palma
✅ Using: palma, spain
```

## Error Handling

- **Missing GPS data**: Files skipped, reported in summary
- **Invalid coordinates**: Validation with proper error messages
- **Network failures**: Falls back to offline mapping where possible
- **Duplicate files**: Automatic renaming with numeric suffixes
- **Permission errors**: Clear error messages with file paths

## Technical Details

### Location Name Processing
- Converts non-Latin characters to ASCII equivalents
- Standardizes city/country name formatting
- Handles multi-word countries (e.g., "united-kingdom")
- Maps regional variations to standard names

### File Operations
- Uses `os.Rename()` for efficient file moving
- Creates directory structure with `os.MkdirAll()`
- Validates GPS coordinate ranges
- Supports concurrent processing of multiple files

## Troubleshooting

### Common Issues

**"exiftool not found"**
- Install exiftool using package manager
- Ensure it's in your system PATH

**"No GPS data found"**
- Photo was taken without location services enabled
- GPS data may have been stripped during editing
- Check camera/phone location settings

**"Permission denied"**
- Ensure write permissions to destination directory
- Check source file read permissions

**"Network timeout"**
- Offline mapping will still work for major cities
- Check internet connection for unknown locations

## Building from Source

```bash
# Clone the repository
git clone <repository-url>
cd photo-meta

# Build the application
go build

# Run
./photo-meta process /source/path /destination/path
```

## License

This is a simplified version derived from PhotoXX project concepts.