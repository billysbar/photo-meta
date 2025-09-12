## 📋 Command Glossary

| Command | Purpose | Best For |
|---------|---------|----------|
| **`clean`** | Duplicate removal | Removing redundant files |
| **`process`** | GPS-based organization | Photos/videos with location data |
| **`datetime`** | Date-based matching | Files without GPS data |
| **`fallback`** | Simple date organization | Files with dates but no location matches |
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

Matches files without GPS data to existing organized structure based on date/time metadata.

```bash
./photo-meta datetime /path/to/unorganized /path/to/organized
# Add --info to generate PhotoXX-style directory summary
./photo-meta datetime /path/to/unorganized /path/to/organized --info
```

### 4. **FALLBACK** - Simple Date-Based Organization

Organizes files with extractable dates into a simple YYYY/Month directory structure when location-based organization isn't possible.

```bash
./photo-meta fallback /path/to/photos /path/to/organized
# Add --info to generate PhotoXX-style directory summary
./photo-meta fallback /path/to/photos /path/to/organized --info
```

### 5. **CLEANUP** - Standalone Empty Directory Removal

Removes empty directories that contain no media files, providing a clean way to tidy up after processing operations.

```bash
./photo-meta cleanup /path/to/directory
```

### 6. **MERGE** - Smart Collection Combining

Merges photos from source directory into target directory while preserving YEAR/COUNTRY/CITY structure.

```bash
./photo-meta merge /path/to/source /path/to/target
```

### 7. **SUMMARY** - Quick Directory Analysis

Provides a quick overview of what's in a directory and what processing is needed.

```bash
./photo-meta summary /path/to/photos
```

### 8. **REPORT** - Comprehensive Analysis & Reporting

Generates detailed reports for directory analysis, duplicate detection, and statistics with optional file export.

```bash
./photo-meta report /path/to/photos
```
