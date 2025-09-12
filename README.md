# Photo Meta - High Performance Concurrent Photo Organizer

A powerful, concurrent photo organization tool that processes photos and videos with GPS data, organizes them into structured directories, and provides advanced management features including duplicate detection, datetime-based organization, and intelligent merging capabilities.

## 🚀 Key Features

- **🔥 High Performance**: Concurrent processing with configurable worker pools (1-16 workers)
- **📍 GPS-Based Organization**: Extracts GPS coordinates from photos/videos and organizes by location
- **📅 Date-Time Organization**: Matches files by date when GPS data is unavailable  
- **📅 Fallback Organization**: Organizes files by date into YYYY/Month structure when location matching fails
- **🔀 Smart Merging**: Combines photo collections while preserving structure
- **🧹 Duplicate Detection**: Intelligent duplicate removal with structure-based prioritization
- **🗑️ Empty Directory Cleanup**: Standalone tool for removing empty directories
- **📊 Progress Visualization**: Enhanced progress bars with ETA and real-time statistics
- **📋 Comprehensive Reporting**: Detailed analysis with summary, duplicates, and statistics reports
- **📄 PhotoXX-Style Info Files**: Automatic generation of `info_DIRNAME XX_timestamp.txt` directory summaries
- **🔍 Dry-Run Modes**: Safe preview modes including quick sampling (dry-run1)
- **🎥 Video Support**: Full video file processing with separate VIDEO-FILES organization
- **⏹️ Graceful Cancellation**: Ctrl+C support with proper cleanup
- **🌍 Global Coverage**: 200+ cities offline mapping + OpenStreetMap fallback

## 📋 Command Glossary

| Command | Purpose | Best For |
|---------|---------|----------|
| **`clean`** | Duplicate removal | Removing redundant files |
| **`process`** | GPS-based organization | Photos/videos with location data |
| **`datetime`** | Date-based matching | Files without GPS data |
| **`fallback`** | Simple date organization | Files with dates but no location matches |
| **`merge`** | Collection combining | Merging photo libraries |
| **`summary`** | Quick analysis | Initial directory assessment |
| **`report`** | Detailed reporting | Comprehensive analysis & documentation |
| **`cleanup`** | Empty directory removal | Cleaning up after processing |

### 🔧 Installation & Setup

```bash
# Install dependencies (macOS)
brew install exiftool

# Install dependencies (Ubuntu/Debian)  
sudo apt-get install exiftool

# Build the application
go build -o photo-meta .
```

---

## 📚 Complete Command Reference

### 1. **PROCESS** - GPS-Based Photo Organization

Organizes photos and videos by extracting GPS location data and creating YEAR/COUNTRY/CITY structure.

```bash
./photo-meta process /source/path /destination/path [OPTIONS]
```

#### **Options:**
- `--workers N` - Number of concurrent workers (1-16, default: 4)
- `--dry-run` - Preview all operations without moving files
- `--dry-run1` - Quick preview (1 file per type per directory)
- `--progress` - Show enhanced progress bar (default: enabled)
- `--no-progress` - Disable progress bar
- `--info` - Generate PhotoXX-style info_ directory summary file after processing

#### **Benefits:**
- ✅ **Automatic Organization**: Creates logical YEAR/COUNTRY/CITY folder structure
- ✅ **GPS Extraction**: Uses GPS metadata from photos/videos for precise location
- ✅ **Concurrent Processing**: Processes multiple files simultaneously for speed
- ✅ **Smart Naming**: Renames files to `YYYY-MM-DD-city.ext` format
- ✅ **Duplicate Handling**: Automatically handles name conflicts with counters
- ✅ **Video Support**: Organizes videos in separate VIDEO-FILES/ structure

#### **Examples:**
```bash
# Basic processing with progress
./photo-meta process ~/vacation-photos ~/organized --progress

# High-performance processing with 8 workers
./photo-meta process /large-collection /organized --workers 8

# Quick preview of what would happen
./photo-meta process ~/photos ~/organized --dry-run1

# Safe full preview
./photo-meta process ~/photos ~/organized --dry-run

# Process with automatic info summary generation
./photo-meta process ~/photos ~/organized --info
```

#### **Output Structure:**
```
organized/
├── 2025/
│   ├── spain/
│   │   └── palma/
│   │       ├── 2025-09-02-palma.HEIC
│   │       └── 2025-09-03-palma.JPG
│   └── united-kingdom/
│       └── london/
│           └── 2025-08-15-london.HEIC
└── VIDEO-FILES/
    └── 2025/
        └── spain/
            └── palma/
                └── 2025-09-02-palma.MOV
```

---

### 2. **DATETIME** - Date-Based File Matching

Matches files without GPS data to existing organized structure based on date/time metadata.

```bash
./photo-meta datetime /source/path /destination/path [OPTIONS]
```

#### **Options:**
- `--workers N` - Number of concurrent workers (1-16, default: 4)
- `--dry-run` - Preview matching operations
- `--dry-run1` - Quick preview with samples
- `--progress` - Show progress bar (default: enabled)
- `--no-progress` - Disable progress bar
- `--info` - Generate PhotoXX-style info_ directory summary file after processing
- `--reset-db` - Clear the GPS cache database for fresh scanning

#### **Benefits:**
- ✅ **GPS-Free Organization**: Organizes files without GPS by matching dates
- ✅ **Intelligent Matching**: Uses existing location database from processed photos
- ✅ **Date Pattern Recognition**: Supports multiple filename date formats
- ✅ **Location Inference**: Smart location detection from target structure
- ✅ **Video Integration**: Properly handles video files in datetime matching
- ✅ **GPS Cache Database**: Speeds up subsequent scans by caching GPS detection results
- ✅ **Integrity Checking**: File hash validation ensures cache accuracy

#### **Examples:**
```bash
# Match files by datetime to existing structure
./photo-meta datetime ~/no-gps-photos ~/organized --progress

# Preview datetime matching
./photo-meta datetime ~/photos ~/organized --dry-run

# Quick sample of datetime matches
./photo-meta datetime ~/photos ~/organized --dry-run1

# Process with info summary generation
./photo-meta datetime ~/photos ~/organized --info

# Clear GPS cache database for fresh scanning
./photo-meta datetime ~/photos ~/organized --reset-db
```

#### **Use Cases:**
- 📱 **Old Phone Photos**: Photos without GPS metadata
- 📷 **Camera Imports**: DSLR photos that don't have GPS
- 🔄 **Backup Recovery**: Recovered files with lost GPS data
- 📅 **Date-Based Sorting**: When location isn't critical but date organization is
- 🚀 **Performance Optimization**: GPS cache significantly speeds up repeat processing

---

### 3. **FALLBACK** - Simple Date-Based Organization

Organizes files with extractable dates into a simple YYYY/Month directory structure when location-based organization isn't possible.

```bash
./photo-meta fallback /source/path /destination/path [OPTIONS]
```

#### **Options:**
- `--workers N` - Number of concurrent workers (1-16, default: 4)
- `--dry-run` - Preview organization operations
- `--dry-run1` - Quick preview with samples
- `--progress` - Show progress bar (default: enabled)
- `--no-progress` - Disable progress bar
- `--info` - Generate PhotoXX-style info_ directory summary file after processing

#### **Benefits:**
- ✅ **Date-Only Organization**: Uses only extractable dates from filenames
- ✅ **Simple Structure**: Clean YYYY/Month folder organization
- ✅ **Standardized Names**: Converts all files to YYYY-MM-DD.ext format
- ✅ **Multiple Date Formats**: Handles DD-MM-YYYY, YYYYMMDD, YYYYMMDDHHMMSS patterns
- ✅ **Automatic Processing**: No user prompts or intervention required
- ✅ **Video Separation**: Places videos in VIDEO-FILES/YYYY/Month structure

#### **Examples:**
```bash
# Quick preview of fallback organization
./photo-meta fallback ~/mixed-photos ~/organized --dry-run1

# Organize files by date when GPS/location isn't available
./photo-meta fallback ~/old-photos ~/organized --progress

# Preview full fallback organization
./photo-meta fallback ~/photos ~/organized --dry-run

# Process with info summary generation
./photo-meta fallback ~/photos ~/organized --info
```

#### **Output Structure:**
```
organized/
├── 2018/
│   └── October/
│       ├── 2018-10-10.JPG
│       ├── 2018-10-11.jpeg
│       └── 2018-10-16.jpg
├── 2020/
│   ├── March/
│   │   ├── 2020-03-09.jpeg
│   │   └── 2020-03-17.JPG
│   └── April/
│       └── 2020-04-22.jpeg
└── VIDEO-FILES/
    └── 2025/
        └── September/
            └── 2025-09-02.MOV
```

#### **Use Cases:**
- 📅 **Simple Date Organization**: When location doesn't matter, only chronological order
- 🔄 **Final Processing Step**: For files that datetime command can't match to locations
- 📱 **Mixed Sources**: Photos from multiple devices/sources with inconsistent metadata
- 🏃 **Quick Organization**: Fast way to get files into a basic organized structure

#### **Supported Date Patterns:**
- `DD-MM-YYYY-*` → 10-10-2018-DSC_0996.JPG → 2018-10-10.JPG
- `YYYYMMDDHHMMSS` → 20250831120839.HEIC → 2025-08-31.HEIC
- `YYYY-MM-DD-*` → Already standardized format
- `YYYYMMDD-*` → 20180310-IMG.jpg → 2018-03-10.jpg

#### **Date Validation:**
- ✅ **Year Range**: Only accepts years 1900-2030 (current year + 5)
- ✅ **Month Validation**: Validates months 01-12
- ✅ **Day Validation**: Validates days 01-31
- ❌ **Invalid Patterns**: Rejects files like `08941020123456.jpg` (invalid year 0894)

---

### 4. **CLEAN** - Intelligent Duplicate Detection & Removal

Detects and removes duplicate photos using SHA-256 hashing with intelligent file prioritization.

```bash
./photo-meta clean /target/path [OPTIONS]
```

#### **Options:**
- `--dry-run` - Preview duplicate removal (safe mode)
- `--dry-run1` - Quick summary showing first 3 duplicate groups
- `--verbose` - Detailed logging (disables progress bar)
- `--workers N` - Number of concurrent workers (1-16, default: 4)
- `--progress` - Show progress bar (default: enabled, disabled in verbose)
- `--no-progress` - Disable progress bar

#### **Benefits:**
- ✅ **SHA-256 Accuracy**: Cryptographically secure duplicate detection
- ✅ **Intelligent Selection**: Keeps best-structured files (proper naming/location)
- ✅ **Structure Scoring**: Prioritizes files with good folder organization
- ✅ **Safe Operation**: Multiple preview modes before actual deletion
- ✅ **Space Analysis**: Shows exactly how much space will be recovered
- ✅ **Smart Prioritization**: Removes "copy" files, keeps originals

#### **Examples:**
```bash
# Quick duplicate overview
./photo-meta clean ~/organized --dry-run1

# Full duplicate analysis
./photo-meta clean ~/organized --dry-run --verbose

# Actually clean duplicates with progress
./photo-meta clean ~/organized --progress

# High-performance cleaning
./photo-meta clean ~/large-collection --workers 8
```

#### **Duplicate Prioritization Logic:**
1. **Keeps**: Files with good structure (`YYYY-MM-DD-location.ext`)
2. **Keeps**: Files in proper year/country/city folders  
3. **Removes**: Files named with "copy", "(1)", "-1", etc.
4. **Keeps**: Newest files if structure is equivalent

---

### 5. **CLEANUP** - Standalone Empty Directory Removal

Removes empty directories that contain no media files, providing a clean way to tidy up after processing operations.

```bash
./photo-meta cleanup /target/path [OPTIONS]
```

#### **Options:**
- `--dry-run` - Preview what directories would be removed
- `--dry-run1` - Same as dry-run for cleanup operations

#### **Benefits:**
- ✅ **Standalone Operation**: Can be run independently without other processing
- ✅ **Intelligent Detection**: Only removes directories with no media files
- ✅ **Multi-Pass Removal**: Handles nested empty directories properly
- ✅ **Safe Operation**: Ignores hidden files (.DS_Store) when determining emptiness
- ✅ **Detailed Logging**: Shows exactly which directories are removed
- ✅ **Non-Media Aware**: Considers directories empty if they only contain non-media files

#### **Examples:**
```bash
# Preview what would be cleaned up
./photo-meta cleanup ~/organized --dry-run

# Remove empty directories after processing
./photo-meta cleanup ~/organized

# Clean up after batch processing
./photo-meta cleanup /photo-library
```

#### **Use Cases:**
- 🧹 **Post-Processing Cleanup**: Remove empty directories left after organizing photos
- 📁 **Directory Maintenance**: Keep organized collections clean and tidy
- 🔄 **Batch Operations**: Clean up after multiple processing operations
- 💾 **Storage Optimization**: Remove unnecessary directory structure

#### **How It Works:**
1. **Scans** target directory for empty directories
2. **Analyzes** each directory to check for media files only
3. **Removes** directories with no photos or videos (ignores .DS_Store, text files, etc.)
4. **Multi-Pass** processing removes nested empty directories
5. **Reports** exactly which directories were removed

#### **Example Output:**
```bash
🧹 Standalone Empty Directory Cleanup
🔍 Target: /photo-library
🧹 Cleaning up empty directories in: /photo-library
🗑️  Removed empty directory: 2024/spain/empty-folder
🗑️  Removed empty directory: 2025/temp/processing
✅ Cleanup complete: Removed 2 empty directories total
```

---

### 6. **MERGE** - Smart Collection Combining

Merges photos from source directory into target directory while preserving YEAR/COUNTRY/CITY structure.

```bash
./photo-meta merge /source/path /target/path [OPTIONS]
```

#### **Options:**
- `--workers N` - Number of concurrent workers (1-16, default: 4)
- `--dry-run` - Preview merge operations without copying files
- `--dry-run1` - Quick preview (1 file per type per directory)
- `--progress` - Show progress bar (default: enabled)
- `--no-progress` - Disable progress bar

#### **Benefits:**
- ✅ **Non-Destructive**: Copies files (preserves originals in source)
- ✅ **Structure Preservation**: Maintains organized YEAR/COUNTRY/CITY layout
- ✅ **GPS & Inference**: Uses GPS data or infers from target structure
- ✅ **Duplicate Avoidance**: Smart detection prevents overwriting existing files
- ✅ **Video Integration**: Properly handles video files in VIDEO-FILES structure
- ✅ **Incremental Updates**: Perfect for adding new photos to existing collections

#### **Examples:**
```bash
# Preview merge operation
./photo-meta merge ~/new-photos ~/organized --dry-run1

# Merge new collection into organized structure
./photo-meta merge ~/vacation-2025 ~/photo-library --progress

# High-performance merge with 8 workers
./photo-meta merge ~/large-import ~/organized --workers 8 --progress
```

#### **Use Cases:**
- 📁 **Collection Combining**: Merge multiple photo folders into one organized structure
- 🔄 **Incremental Imports**: Add new photos to existing organized collection
- 💾 **Backup Integration**: Merge recovered/backup photos into main library
- 🎯 **Selective Organization**: Keep originals while building organized copies

---

### 7. **SUMMARY** - Quick Directory Analysis

Provides a quick overview of what's in a directory and what processing is needed.

```bash
./photo-meta summary /source/path
```

#### **Benefits:**
- ✅ **Quick Assessment**: Instant overview of photos, videos, and file types
- ✅ **GPS Analysis**: Shows files with/without GPS data
- ✅ **Processing Guidance**: Recommends which commands to use next
- ✅ **Date Analysis**: Shows extractable dates and date ranges
- ✅ **Directory Structure**: File counts per subdirectory

#### **Examples:**
```bash
# Quick overview of directory contents
./photo-meta summary ~/vacation-photos

# Assess processing needs for large collection
./photo-meta summary /massive-photo-collection
```

---

### 8. **REPORT** - Comprehensive Analysis & Reporting

Generates detailed reports for directory analysis, duplicate detection, and statistics with optional file export.

```bash
./photo-meta report <type> /source/path [OPTIONS]
```

#### **Report Types:**
- **`summary`** - Comprehensive directory analysis with processing status
- **`duplicates`** - Find and analyze duplicate files with quality scoring
- **`stats`** - General file statistics and extension breakdown

#### **Options:**
- `--save` - Export report to timestamped .txt file
- `--progress` - Show scanning progress (default: enabled)
- `--verbose` - Detailed output with additional information

#### **Benefits:**
- ✅ **Detailed Analysis**: In-depth directory structure and file analysis
- ✅ **Processing Status**: Shows which files are processed vs unprocessed
- ✅ **Duplicate Intelligence**: SHA-256 hashing with quality-based prioritization
- ✅ **Space Analysis**: Calculate potential space savings from duplicates
- ✅ **Export Capability**: Save reports as timestamped text files
- ✅ **Non-Destructive**: All reports are read-only analysis

#### **Examples:**
```bash
# Quick statistics overview
./photo-meta report stats ~/photos

# Comprehensive directory analysis with export
./photo-meta report summary ~/photos --save --progress

# Find duplicates with quality analysis
./photo-meta report duplicates ~/photos --save

# Analyze processing completion status
./photo-meta report summary ~/organized-photos --verbose
```

#### **Report Outputs:**

**Summary Report:**
- Processing completion percentage
- Processed vs unprocessed files by location
- Files that would be moved to VIDEO-FILES
- Directory structure with file counts
- Date ranges and file type breakdown

**Duplicates Report:**
- Duplicate groups with SHA-256 hashes
- Quality scores for intelligent prioritization
- Wasted space calculations
- File modification dates and paths
- Recommendations for which files to keep

**Stats Report:**
- File type statistics (photos, videos, other)
- Extension breakdown sorted by frequency
- Total file counts and sizes
- Quick analysis timing

---

## ⚙️ Performance & Configuration

### **Worker Configuration**
- **1-2 workers**: USB drives, slow storage, limited CPU
- **4-6 workers**: Standard HDDs, moderate collections
- **8-12 workers**: SSDs, large collections, powerful CPUs  
- **12-16 workers**: NVMe SSDs, massive collections, high-end systems

### **Progress Bar Features**
- 📊 **Visual Progress**: 40-character progress bar with Unicode blocks
- ⏱️ **ETA Calculation**: Real-time estimated time remaining
- 📈 **Statistics**: Files processed, success rate, failures
- 🎯 **File Type Breakdown**: Separate counts for photos/videos
- ⚡ **Live Updates**: 500ms refresh rate for smooth feedback

### **Memory Usage**
- **Base**: ~10-50MB depending on collection size
- **Per Worker**: ~5-10MB additional memory per concurrent worker
- **Large Collections**: Automatically optimizes for 10k+ files

### **PhotoXX-Style Info Files**
- **Purpose**: Generate directory summary files compatible with PhotoXX format
- **Naming**: `info_DIRNAME XX_YYYY-MM-DD_HH-MM-SS.txt`
- **Contents**: Directory analysis, processing recommendations, file statistics
- **Usage**: Add `--info` flag to process, datetime, or fallback commands
- **Behavior**: Only generated in live mode (not in dry-run modes)

---

## 🔍 Dry-Run Modes Comparison

| Mode | Files Processed | Purpose | Best For |
|------|----------------|---------|----------|
| **Normal** | All files | Full processing | Production use |
| **--dry-run** | All files | Complete preview | Safety verification |
| **--dry-run1** | 1 per type/dir | Quick overview | Initial assessment |

### **When to Use Each Mode:**

🔍 **--dry-run1**: 
- First time using the tool
- Quick assessment of large collections
- Checking if GPS data exists
- Understanding potential organization structure

🔍 **--dry-run**:
- Before major operations
- Verifying duplicate detection logic
- Checking merge behavior
- Final safety check before processing

⚡ **Normal Mode**:
- After dry-run verification
- Production processing
- Automated workflows
- When confident about operations

---

## 🎯 Workflow Examples

### **Complete Photo Organization Workflow**

```bash
# 1. Initial assessment of what you have
./photo-meta summary ~/messy-photos

# 2. Quick assessment with detailed reporting
./photo-meta report summary ~/messy-photos --save

# 3. Quick process preview to understand structure
./photo-meta process ~/messy-photos ~/organized --dry-run1

# 4. Full preview to verify operations
./photo-meta process ~/messy-photos ~/organized --dry-run

# 5. Process GPS-enabled photos with info summary
./photo-meta process ~/messy-photos ~/organized --workers 6 --progress --info

# 6. Handle remaining files by date matching
./photo-meta datetime ~/leftover-photos ~/organized --dry-run1
./photo-meta datetime ~/leftover-photos ~/organized --progress --info

# 7. Fallback organization for any remaining dated files
./photo-meta fallback ~/still-leftover-photos ~/organized --dry-run1
./photo-meta fallback ~/still-leftover-photos ~/organized --progress --info

# 8. Check for duplicates before cleaning
./photo-meta report duplicates ~/organized --save

# 9. Clean up any duplicates
./photo-meta clean ~/organized --dry-run1
./photo-meta clean ~/organized --progress

# 10. Remove any empty directories left behind
./photo-meta cleanup ~/organized --dry-run
./photo-meta cleanup ~/organized

# 11. Generate final statistics report
./photo-meta report stats ~/organized --save

# 12. Merge additional collections as needed
./photo-meta merge ~/new-photos ~/organized --dry-run1
./photo-meta merge ~/new-photos ~/organized --progress
```

### **Large Collection Processing**

```bash
# For 10,000+ photos with high-performance system and info generation
./photo-meta process /massive-collection /organized \
  --workers 12 \
  --progress \
  --info

# For network storage or slower systems
./photo-meta process /network-photos /organized \
  --workers 2 \
  --progress \
  --info
```

### **Safe Duplicate Cleaning**

```bash
# 1. Generate comprehensive duplicate analysis report
./photo-meta report duplicates ~/organized --save --verbose

# 2. Get overview of duplicates for cleaning
./photo-meta clean ~/organized --dry-run1

# 3. Detailed cleaning analysis
./photo-meta clean ~/organized --dry-run --verbose

# 4. Clean duplicates
./photo-meta clean ~/organized --progress

# 5. Clean up empty directories
./photo-meta cleanup ~/organized
```

### **Comprehensive Reporting Workflow**

```bash
# 1. Quick directory assessment
./photo-meta summary ~/photos

# 2. Detailed processing status analysis
./photo-meta report summary ~/photos --save --progress

# 3. Duplicate analysis with quality scoring
./photo-meta report duplicates ~/photos --save

# 4. Final statistics and file breakdown
./photo-meta report stats ~/photos --save

# All reports are saved with timestamps for tracking
ls ~/photos/summary_* ~/photos/duplicates_* ~/photos/stats_*
```

---

## 🗂️ Supported File Formats

### **Photo Formats (25+ formats)**
- **Common**: JPG, JPEG, HEIC, HEIF, TIFF, TIF, PNG
- **RAW**: CR2 (Canon), NEF (Nikon), ARW (Sony), ORF (Olympus), RW2 (Panasonic), RAF (Fuji), DNG, and more

### **Video Formats**
- **Common**: MP4, MOV, AVI, MKV, WMV
- **Professional**: MTS, M2TS, MXF, ProRes
- **Mobile**: 3GP, 3G2, WebM

---

## 🚨 Error Handling & Troubleshooting

### **Common Issues & Solutions**

**"exiftool not found"**
```bash
# macOS
brew install exiftool

# Ubuntu/Debian
sudo apt-get install exiftool
```

**"No GPS data found"**
- Use `datetime` command for non-GPS files
- Check if location services were enabled when photos were taken
- Consider using merge with location inference

**Performance Issues**
- Reduce `--workers` count for slower storage
- Use `--no-progress` to reduce terminal overhead
- Process in smaller batches for very large collections

**Permission Errors**
- Ensure read access to source directories
- Ensure write access to destination directories  
- Check file ownership and permissions

### **Graceful Cancellation**
- **Ctrl+C**: Initiates graceful shutdown
- **30-second timeout**: Allows workers to finish current files
- **Progress preservation**: Shows final statistics even when cancelled
- **Resume capability**: Can restart processing from where it left off

---

## 📈 Performance Benchmarks

### **Typical Processing Speeds** (varies by hardware/storage)

| Collection Size | Workers | SSD Time | HDD Time |
|----------------|---------|----------|----------|
| 100 photos | 4 | ~10s | ~30s |
| 1,000 photos | 6 | ~1m | ~3m |  
| 10,000 photos | 8 | ~8m | ~25m |
| 50,000+ photos | 12 | ~35m | ~2h |

*Note: Times include GPS extraction, location resolution, and file operations*

---

## 🏗️ Building from Source

```bash
# Prerequisites
go version  # Requires Go 1.21+
brew install exiftool  # or equivalent for your OS

# Clone and build
git clone <repository-url>
cd photo-meta
go build -o photo-meta .

# Verify installation
./photo-meta --help
```

---

## 📝 Recent Updates

### v1.2 - GPS Cache Database & Performance Improvements
- **🆕 GPS Cache Database**: Intelligent caching of GPS detection results for significantly faster subsequent scans
- **🚀 Performance Optimization**: Cache reduces GPS scanning time by up to 90% on repeat processing
- **🔧 Cache Management**: Built-in cache validation with file hash integrity checking
- **🗑️ Cache Control**: `--reset-db` flag allows clearing cache when needed
- **📊 Cache Statistics**: Real-time cache hit/miss statistics during processing
- **💾 Persistent Storage**: Cache survives between sessions using JSON-based storage

### v1.1 - Enhanced Date Validation & Cleanup
- **🆕 Standalone Cleanup Command**: Added `cleanup` command for removing empty directories independently
- **🔧 Improved Date Validation**: Enhanced fallback command with robust year validation (1900-2030)
- **🛡️ Better Error Prevention**: Prevents creation of invalid year directories (e.g., "0894")
- **📖 Documentation**: Comprehensive README updates with new workflows and examples

### Key Improvements:
- ✅ **GPS Cache Intelligence**: Automatically detects file changes and invalidates stale cache entries
- ✅ **File Integrity**: MD5 hash validation ensures cache accuracy across file modifications
- ✅ **Memory Efficiency**: Thread-safe cache operations with minimal memory overhead
- ✅ **Cache Cleanup**: Automatic removal of stale entries for non-existent files
- ✅ **Date Pattern Validation**: Validates year, month, and day ranges before processing
- ✅ **Empty Directory Management**: Intelligent cleanup that preserves directories with non-media files
- ✅ **Multi-Pass Cleanup**: Handles nested empty directories properly
- ✅ **Enhanced Safety**: Better validation prevents invalid directory structures

---

## 📄 License

This project is derived from PhotoXX concepts with significant enhancements for concurrent processing, advanced duplicate detection, and comprehensive photo/video organization capabilities.