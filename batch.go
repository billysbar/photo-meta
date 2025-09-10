package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// BatchMetadataUpdater handles batch metadata operations using ExifTool
type BatchMetadataUpdater struct {
	batchSize       int
	pendingUpdates  []*PhotoBatchInfo
	tempConfigFiles []string
	mu              sync.Mutex
	stats           BatchStats
}

// PhotoBatchInfo contains information for batch processing
type PhotoBatchInfo struct {
	FilePath    string
	NewLocation string
	NewDate     string
	Tags        map[string]string
}

// BatchStats tracks batch processing statistics
type BatchStats struct {
	TotalFiles      int
	ProcessedFiles  int
	BatchCount      int
	TotalBatches    int
	FailedFiles     int
	StartTime       time.Time
	LastBatchTime   time.Duration
	AverageBatchTime time.Duration
	mu              sync.Mutex
}

// NewBatchMetadataUpdater creates a new batch metadata updater
func NewBatchMetadataUpdater(batchSize int) *BatchMetadataUpdater {
	if batchSize <= 0 {
		batchSize = 50 // Default batch size
	}
	
	return &BatchMetadataUpdater{
		batchSize:       batchSize,
		pendingUpdates:  make([]*PhotoBatchInfo, 0, batchSize),
		tempConfigFiles: make([]string, 0),
		stats: BatchStats{
			StartTime: time.Now(),
		},
	}
}

// AddPhoto adds a photo to the batch for processing
func (bmu *BatchMetadataUpdater) AddPhoto(filePath, location, date string, tags map[string]string) {
	bmu.mu.Lock()
	defer bmu.mu.Unlock()
	
	bmu.pendingUpdates = append(bmu.pendingUpdates, &PhotoBatchInfo{
		FilePath:    filePath,
		NewLocation: location,
		NewDate:     date,
		Tags:        tags,
	})
	
	bmu.stats.TotalFiles++
	
	// Process batch if we've reached the batch size
	if len(bmu.pendingUpdates) >= bmu.batchSize {
		bmu.processBatch(context.Background())
	}
}

// Flush processes any remaining photos in the batch
func (bmu *BatchMetadataUpdater) Flush(ctx context.Context) error {
	bmu.mu.Lock()
	defer bmu.mu.Unlock()
	
	if len(bmu.pendingUpdates) > 0 {
		return bmu.processBatch(ctx)
	}
	
	return nil
}

// processBatch processes the current batch of photos
func (bmu *BatchMetadataUpdater) processBatch(ctx context.Context) error {
	if len(bmu.pendingUpdates) == 0 {
		return nil
	}
	
	batchStart := time.Now()
	batchSize := len(bmu.pendingUpdates)
	
	fmt.Printf("📦 Processing batch of %d files...\n", batchSize)
	
	// Create argument file for ExifTool
	argFile, err := bmu.createExifToolArgFile(bmu.pendingUpdates)
	if err != nil {
		return fmt.Errorf("failed to create ExifTool argument file: %v", err)
	}
	
	// Track temporary file for cleanup
	bmu.tempConfigFiles = append(bmu.tempConfigFiles, argFile)
	
	// Execute ExifTool with argument file
	err = bmu.processBatchWithArgFile(ctx, argFile)
	
	// Update statistics
	batchDuration := time.Since(batchStart)
	bmu.updateBatchStats(batchSize, batchDuration, err == nil)
	
	// Clear the pending updates
	bmu.pendingUpdates = bmu.pendingUpdates[:0]
	
	if err != nil {
		fmt.Printf("❌ Batch processing failed: %v\n", err)
		return err
	}
	
	fmt.Printf("✅ Batch completed in %v\n", batchDuration.Round(time.Millisecond))
	return nil
}

// createExifToolArgFile creates a temporary argument file for ExifTool
func (bmu *BatchMetadataUpdater) createExifToolArgFile(photos []*PhotoBatchInfo) (string, error) {
	tempFile, err := ioutil.TempFile("", "exiftool_batch_*.args")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()
	
	// Write ExifTool arguments to file
	for _, photo := range photos {
		// Add common arguments
		fmt.Fprintf(tempFile, "-overwrite_original\n")
		fmt.Fprintf(tempFile, "-P\n") // Preserve file modification date
		
		// Add GPS coordinates if available
		if photo.NewLocation != "" {
			// This is simplified - in reality you'd extract lat/lon from location
			fmt.Fprintf(tempFile, "-GPSLatitude=%s\n", "0.0") // Placeholder
			fmt.Fprintf(tempFile, "-GPSLongitude=%s\n", "0.0") // Placeholder
		}
		
		// Add date if available
		if photo.NewDate != "" {
			fmt.Fprintf(tempFile, "-AllDates=%s\n", photo.NewDate)
		}
		
		// Add custom tags
		for tag, value := range photo.Tags {
			fmt.Fprintf(tempFile, "-%s=%s\n", tag, value)
		}
		
		// Add file path (must be last for this file)
		fmt.Fprintf(tempFile, "%s\n", photo.FilePath)
	}
	
	return tempFile.Name(), nil
}

// processBatchWithArgFile executes ExifTool using an argument file
func (bmu *BatchMetadataUpdater) processBatchWithArgFile(ctx context.Context, argFile string) error {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	
	// Execute ExifTool
	cmd := exec.CommandContext(timeoutCtx, "exiftool", "-@", argFile)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		return fmt.Errorf("ExifTool execution failed: %v\nOutput: %s", err, string(output))
	}
	
	// Parse output for any warnings or errors
	outputStr := string(output)
	if strings.Contains(outputStr, "Warning") || strings.Contains(outputStr, "Error") {
		fmt.Printf("⚠️  ExifTool warnings/errors:\n%s\n", outputStr)
	}
	
	return nil
}

// updateBatchStats updates the batch processing statistics
func (bmu *BatchMetadataUpdater) updateBatchStats(batchSize int, duration time.Duration, success bool) {
	bmu.stats.mu.Lock()
	defer bmu.stats.mu.Unlock()
	
	bmu.stats.BatchCount++
	bmu.stats.LastBatchTime = duration
	
	if success {
		bmu.stats.ProcessedFiles += batchSize
	} else {
		bmu.stats.FailedFiles += batchSize
	}
	
	// Update average batch time
	if bmu.stats.BatchCount > 0 {
		totalTime := time.Since(bmu.stats.StartTime)
		bmu.stats.AverageBatchTime = totalTime / time.Duration(bmu.stats.BatchCount)
	}
}

// GetStats returns current batch processing statistics
func (bmu *BatchMetadataUpdater) GetStats() BatchStats {
	bmu.stats.mu.Lock()
	defer bmu.stats.mu.Unlock()
	
	// Return a copy
	return bmu.stats
}

// PrintStats prints detailed batch processing statistics
func (bmu *BatchMetadataUpdater) PrintStats() {
	stats := bmu.GetStats()
	
	fmt.Printf("\n📊 Batch Processing Statistics:\n")
	fmt.Printf("📁 Total files: %d\n", stats.TotalFiles)
	fmt.Printf("✅ Processed: %d\n", stats.ProcessedFiles)
	fmt.Printf("❌ Failed: %d\n", stats.FailedFiles)
	fmt.Printf("📦 Batches completed: %d\n", stats.BatchCount)
	fmt.Printf("⏱️  Average batch time: %v\n", stats.AverageBatchTime.Round(time.Millisecond))
	
	if stats.BatchCount > 0 {
		fmt.Printf("🚀 Files per batch: %.1f\n", float64(stats.ProcessedFiles)/float64(stats.BatchCount))
	}
	
	elapsed := time.Since(stats.StartTime)
	fmt.Printf("⏰ Total time: %v\n", elapsed.Round(time.Second))
	
	if stats.ProcessedFiles > 0 {
		avgTimePerFile := elapsed / time.Duration(stats.ProcessedFiles)
		fmt.Printf("📈 Average time per file: %v\n", avgTimePerFile.Round(time.Millisecond))
	}
}

// Close cleans up temporary files and resources
func (bmu *BatchMetadataUpdater) Close() error {
	bmu.mu.Lock()
	defer bmu.mu.Unlock()
	
	var errs []string
	
	// Clean up temporary argument files
	for _, tempFile := range bmu.tempConfigFiles {
		if err := os.Remove(tempFile); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Sprintf("failed to remove %s: %v", tempFile, err))
		}
	}
	
	bmu.tempConfigFiles = bmu.tempConfigFiles[:0]
	
	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errs, "; "))
	}
	
	return nil
}

// BatchProcessor manages the overall batch processing workflow
type BatchProcessor struct {
	updater  *BatchMetadataUpdater
	workers  int
	jobChan  chan *PhotoBatchInfo
	doneChan chan struct{}
	wg       sync.WaitGroup
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize, workers int) *BatchProcessor {
	return &BatchProcessor{
		updater:  NewBatchMetadataUpdater(batchSize),
		workers:  workers,
		jobChan:  make(chan *PhotoBatchInfo, workers*2), // Buffer for smooth processing
		doneChan: make(chan struct{}),
	}
}

// Start begins the batch processing workers
func (bp *BatchProcessor) Start(ctx context.Context) {
	for i := 0; i < bp.workers; i++ {
		bp.wg.Add(1)
		go bp.worker(ctx, i)
	}
}

// worker processes batch jobs
func (bp *BatchProcessor) worker(ctx context.Context, id int) {
	defer bp.wg.Done()
	
	for {
		select {
		case photo, ok := <-bp.jobChan:
			if !ok {
				return // Channel closed
			}
			
			// Add photo to batch
			bp.updater.AddPhoto(photo.FilePath, photo.NewLocation, 
				photo.NewDate, photo.Tags)
			
		case <-ctx.Done():
			return // Cancelled
		}
	}
}

// ProcessPhoto adds a photo to the processing queue
func (bp *BatchProcessor) ProcessPhoto(filePath, location, date string, tags map[string]string) {
	photo := &PhotoBatchInfo{
		FilePath:    filePath,
		NewLocation: location,
		NewDate:     date,
		Tags:        tags,
	}
	
	select {
	case bp.jobChan <- photo:
	case <-bp.doneChan:
		// Processor is shutting down
	}
}

// Stop stops the batch processor and flushes remaining batches
func (bp *BatchProcessor) Stop(ctx context.Context) error {
	close(bp.jobChan) // Stop accepting new jobs
	bp.wg.Wait()      // Wait for workers to finish
	close(bp.doneChan)
	
	// Flush any remaining batches
	err := bp.updater.Flush(ctx)
	
	// Print final statistics
	bp.updater.PrintStats()
	
	// Clean up
	if closeErr := bp.updater.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	
	return err
}

// Example usage function for integration
func ProcessPhotosWithBatch(photos []WorkJob, batchSize int) error {
	processor := NewBatchProcessor(batchSize, 4)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start batch processing
	processor.Start(ctx)
	
	// Process all photos
	for _, job := range photos {
		tags := map[string]string{
			"Software": "photo-meta",
			"ProcessedBy": "concurrent-batch",
		}
		
		processor.ProcessPhoto(job.PhotoPath, "", "", tags)
	}
	
	// Stop and flush
	return processor.Stop(ctx)
}