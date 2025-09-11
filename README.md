# Photo Meta - High Performance Concurrent Photo Organizer

A powerful, concurrent photo organization tool that processes photos and videos with GPS data, organizes them into structured directories, and provides advanced management features including duplicate detection, datetime-based organization, and intelligent merging capabilities.

## 🚀 Key Features

- **🔥 High Performance**: Concurrent processing with configurable worker pools (1-16 workers)
- **📍 GPS-Based Organization**: Extracts GPS coordinates from photos/videos and organizes by location
- **📅 Date-Time Organization**: Matches files by date when GPS data is unavailable
- **🔀 Smart Merging**: Combines photo collections while preserving structure
- **🧹 Duplicate Detection**: Intelligent duplicate removal with structure-based prioritization
- **📊 Progress Visualization**: Enhanced progress bars with ETA and real-time statistics
- **🔍 Dry-Run Modes**: Safe preview modes including quick sampling (dry-run1)
- **🎥 Video Support**: Full video file processing with separate VIDEO-FILES organization
- **⏹️ Graceful Cancellation**: Ctrl+C support with proper cleanup
- **🌍 Global Coverage**: 200+ cities offline mapping + OpenStreetMap fallback

## 📋 Command Glossary

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

#### **Benefits:**
- ✅ **GPS-Free Organization**: Organizes files without GPS by matching dates
- ✅ **Intelligent Matching**: Uses existing location database from processed photos
- ✅ **Date Pattern Recognition**: Supports multiple filename date formats
- ✅ **Location Inference**: Smart location detection from target structure
- ✅ **Video Integration**: Properly handles video files in datetime matching

#### **Examples:**
```bash
# Match files by datetime to existing structure
./photo-meta datetime ~/no-gps-photos ~/organized --progress

# Preview datetime matching
./photo-meta datetime ~/photos ~/organized --dry-run

# Quick sample of datetime matches
./photo-meta datetime ~/photos ~/organized --dry-run1
```

#### **Use Cases:**
- 📱 **Old Phone Photos**: Photos without GPS metadata
- 📷 **Camera Imports**: DSLR photos that don't have GPS
- 🔄 **Backup Recovery**: Recovered files with lost GPS data
- 📅 **Date-Based Sorting**: When location isn't critical but date organization is

---

### 3. **CLEAN** - Intelligent Duplicate Detection & Removal

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

### 4. **MERGE** - Smart Collection Combining

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
# 1. Quick assessment of what you have
./photo-meta process ~/messy-photos ~/organized --dry-run1

# 2. Full preview to understand structure
./photo-meta process ~/messy-photos ~/organized --dry-run

# 3. Process GPS-enabled photos
./photo-meta process ~/messy-photos ~/organized --workers 6 --progress

# 4. Handle remaining files by date matching
./photo-meta datetime ~/leftover-photos ~/organized --dry-run1
./photo-meta datetime ~/leftover-photos ~/organized --progress

# 5. Clean up any duplicates
./photo-meta clean ~/organized --dry-run1
./photo-meta clean ~/organized --progress

# 6. Merge additional collections as needed
./photo-meta merge ~/new-photos ~/organized --dry-run1
./photo-meta merge ~/new-photos ~/organized --progress
```

### **Large Collection Processing**

```bash
# For 10,000+ photos with high-performance system
./photo-meta process /massive-collection /organized \
  --workers 12 \
  --progress

# For network storage or slower systems
./photo-meta process /network-photos /organized \
  --workers 2 \
  --progress
```

### **Safe Duplicate Cleaning**

```bash
# 1. Get overview of duplicates
./photo-meta clean ~/organized --dry-run1

# 2. Detailed analysis
./photo-meta clean ~/organized --dry-run --verbose

# 3. Clean duplicates
./photo-meta clean ~/organized --progress
```

---

## 🗂️ Supported File Formats

### **Photo Formats (25+ formats)**
- **Common**: JPG, JPEG, HEIC, HEIF, TIFF, PNG
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

## 📄 License

This project is derived from PhotoXX concepts with significant enhancements for concurrent processing, advanced duplicate detection, and comprehensive photo/video organization capabilities.