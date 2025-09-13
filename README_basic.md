## 📋 Command Glossary

| Command | Purpose | Best For |
|---------|---------|----------|
| **`clean`** | Duplicate removal | Removing redundant files |
| **`process`** | GPS-based organization | Photos/videos with location data |
| **`datetime`** | Date-based filename matching | Files without GPS data (uses filename dates) |
| **`organize`** | Location-based filename organization | Files with location names in filenames |
| **`fallback`** | Simple filename date organization | Files with dates in filenames but no location matches |
| **`tiff`** | Timestamp repair | Fixing midnight timestamps (00:00:00) |
| **`cleanup`** | Empty directory removal | Cleaning up after processing |
| **`merge`** | Collection combining | Merging photo libraries |
| **`summary`** | Quick analysis | Initial directory assessment |
| **`report`** | Detailed reporting | Comprehensive analysis & documentation |



### 1. **CLEAN** - Intelligent Duplicate Detection & Removal

Detects and removes duplicate photos using SHA-256 hashing with intelligent file prioritization.

```bash
./photo-meta clean /path/to/photos
```

### 2. **PROCESS** - GPS-Based Photo Organization

Organizes photos and videos by extracting GPS location data and creating YEAR/COUNTRY/CITY structure.

```bash
./photo-meta process /path/to/photos /path/to/organized
# Add --info to generate PhotoXX-style directory summary
./photo-meta process /path/to/photos /path/to/organized --info
```

### 3. **DATETIME** - Date-Based File Matching

Matches files without GPS data to existing organized structure based on date/time extracted from filenames and metadata.

```bash
./photo-meta datetime /path/to/unorganized /path/to/organized
# Add --info to generate PhotoXX-style directory summary
./photo-meta datetime /path/to/unorganized /path/to/organized --info
# Clear GPS cache database for fresh scanning
./photo-meta datetime /path/to/unorganized /path/to/organized --reset-db
```

#### **Example filename patterns:**
```
2025-09-15-palma.HEIC          → Matches to 2025/spain/palma/
IMG_20250915_143022.jpg        → Matches to 2025 locations
vacation_2025-09-15.png        → Date extraction from filename
```

### 4. **ORGANIZE** - Location-Based Organization

Organizes files by extracting location information from filenames that contain location names or patterns.

```bash
./photo-meta organize /path/to/photos /path/to/organized
# Add --info to generate PhotoXX-style directory summary
./photo-meta organize /path/to/photos /path/to/organized --info
```

#### **Example filename patterns:**
```
palma-vacation-2025.HEIC       → Organizes to 2025/spain/palma/
london_trip_sept.jpg           → Organizes to 2025/united-kingdom/london/
paris-photos-day1.png          → Organizes to 2025/france/paris/
```

### 5. **FALLBACK** - Simple Date-Based Organization

Organizes files with extractable dates from filenames into a simple YYYY/Month directory structure when location-based organization isn't possible.

```bash
./photo-meta fallback /path/to/photos /path/to/organized
# Add --info to generate PhotoXX-style directory summary
./photo-meta fallback /path/to/photos /path/to/organized --info
```

#### **Example filename patterns:**
```
10-10-2018-DSC_0996.JPG       → 2018-10-10.JPG (2018/October/)
20250831120839.HEIC           → 2025-08-31.HEIC (2025/August/)
IMG_20250915.jpeg             → 2025-09-15.jpeg (2025/September/)
```

### 6. **TIFF** - Timestamp Repair & Correction

Fixes midnight timestamps (00:00:00) using EXIF ModifyDate and updates both EXIF timestamps and filename datetime.

```bash
./photo-meta tiff /target/path
# Add --dry-run for preview mode
./photo-meta tiff /target/path --dry-run
```

### 7. **CLEANUP** - Standalone Empty Directory Removal

Removes empty directories that contain no media files, providing a clean way to tidy up after processing operations.

```bash
./photo-meta cleanup /path/to/directory
```

### 8. **MERGE** - Smart Collection Combining

Merges photos from source directory into target directory while preserving YEAR/COUNTRY/CITY structure.

```bash
./photo-meta merge /path/to/source /path/to/target
```

### 9. **SUMMARY** - Quick Directory Analysis

Provides a quick overview of what's in a directory and what processing is needed.

```bash
./photo-meta summary /path/to/photos
```

### 10. **REPORT** - Comprehensive Analysis & Reporting

Generates detailed reports for directory analysis, duplicate detection, and statistics with optional file export.

```bash
./photo-meta report summary /path/to/photos
./photo-meta report duplicates /path/to/photos
./photo-meta report stats /path/to/photos
```
