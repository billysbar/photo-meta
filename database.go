package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// GPSCache represents a JSON-based database for caching GPS processing results
type GPSCache struct {
	data map[string]FileStatus
	path string
	mu   sync.RWMutex
}

// FileStatus represents the GPS processing status of a file
type FileStatus struct {
	FilePath     string
	FileSize     int64
	ModTime      time.Time
	HasGPS       bool
	ProcessedAt  time.Time
	FileHash     string // MD5 hash for integrity checking
}

// NewGPSCache creates or opens a GPS cache database
func NewGPSCache(dbPath string) (*GPSCache, error) {
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %v", err)
	}

	cache := &GPSCache{
		data: make(map[string]FileStatus),
		path: dbPath,
	}

	// Load existing data if file exists
	if err := cache.load(); err != nil {
		return nil, fmt.Errorf("failed to load cache: %v", err)
	}

	return cache, nil
}

// load reads the JSON cache from disk
func (c *GPSCache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If file doesn't exist, start with empty cache
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}

	// Handle empty file
	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, &c.data)
}

// save writes the JSON cache to disk
func (c *GPSCache) save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

// IsFileProcessed checks if a file has been processed and returns its GPS status
func (c *GPSCache) IsFileProcessed(filePath string) (hasGPS bool, cached bool, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Get file info to check if it has changed
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, false, err
	}

	status, exists := c.data[filePath]
	if !exists {
		// File not in cache
		return false, false, nil
	}

	// Check if file has changed (size or modification time)
	if status.FileSize != fileInfo.Size() || status.ModTime.Unix() != fileInfo.ModTime().Unix() {
		// File has changed, remove old entry and return as not cached
		// We need to unlock to call RemoveFile (which needs write lock)
		c.mu.RUnlock()
		c.RemoveFile(filePath)
		c.mu.RLock()
		return false, false, nil
	}

	// File is cached and unchanged
	return status.HasGPS, true, nil
}

// RecordFile records the GPS processing result for a file
func (c *GPSCache) RecordFile(filePath string, hasGPS bool, fileHash string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.data[filePath] = FileStatus{
		FilePath:    filePath,
		FileSize:    fileInfo.Size(),
		ModTime:     fileInfo.ModTime(),
		HasGPS:      hasGPS,
		ProcessedAt: time.Now(),
		FileHash:    fileHash,
	}
	c.mu.Unlock()

	// Save to disk
	return c.save()
}

// RemoveFile removes a file from the cache
func (c *GPSCache) RemoveFile(filePath string) error {
	c.mu.Lock()
	delete(c.data, filePath)
	c.mu.Unlock()

	// Save to disk
	return c.save()
}

// GetCacheStats returns statistics about the cache
func (c *GPSCache) GetCacheStats() (totalFiles, filesWithGPS, filesWithoutGPS int, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalFiles = len(c.data)
	
	for _, status := range c.data {
		if status.HasGPS {
			filesWithGPS++
		}
	}

	filesWithoutGPS = totalFiles - filesWithGPS
	return totalFiles, filesWithGPS, filesWithoutGPS, nil
}

// Clear removes all entries from the cache
func (c *GPSCache) Clear() error {
	c.mu.Lock()
	c.data = make(map[string]FileStatus)
	c.mu.Unlock()

	// Save empty cache to disk
	return c.save()
}

// Close saves any pending changes and closes the cache
func (c *GPSCache) Close() error {
	// Save any pending changes to disk
	return c.save()
}

// CleanupStaleEntries removes entries for files that no longer exist
func (c *GPSCache) CleanupStaleEntries() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var staleFiles []string
	for filePath := range c.data {
		// Check if file still exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			staleFiles = append(staleFiles, filePath)
		}
	}

	// Remove stale entries
	removedCount := 0
	for _, filePath := range staleFiles {
		delete(c.data, filePath)
		removedCount++
	}

	if removedCount > 0 {
		// Save changes to disk
		if err := c.save(); err != nil {
			log.Printf("Warning: failed to save cache after cleanup: %v", err)
		}
	}

	return removedCount, nil
}

// getDefaultCachePath returns the default cache database path
func getDefaultCachePath() string {
	// Use a temporary directory for the cache database
	tempDir := os.TempDir()
	return filepath.Join(tempDir, "photo-meta-cache.json")
}

// Global cache instance
var gpsCache *GPSCache

// InitGPSCache initializes the global GPS cache
func InitGPSCache() error {
	if gpsCache != nil {
		return nil // Already initialized
	}

	cachePath := getDefaultCachePath()
	cache, err := NewGPSCache(cachePath)
	if err != nil {
		return err
	}

	gpsCache = cache
	return nil
}

// CloseGPSCache closes the global GPS cache
func CloseGPSCache() error {
	if gpsCache == nil {
		return nil
	}

	err := gpsCache.Close()
	gpsCache = nil
	return err
}

// GetGPSCache returns the global GPS cache instance
func GetGPSCache() *GPSCache {
	return gpsCache
}