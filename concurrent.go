package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkJob represents a single photo processing job
type WorkJob struct {
	PhotoPath string
	DestPath  string
	JobType   string // "process", "datetime", "clean"
}

// WorkResult represents the result of processing a job
type WorkResult struct {
	Job       WorkJob
	Success   bool
	Error     error
	Message   string
	Duration  time.Duration
}

// ProgressTracker tracks processing progress with thread safety
type ProgressTracker struct {
	Total     int
	Completed int
	Failed    int
	Skipped   int
	StartTime time.Time
	mu        sync.Mutex
}

// CancellationManager handles graceful shutdown and cancellation
type CancellationManager struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	cancelled bool
	mu        sync.Mutex
	startTime time.Time
}

// NewCancellationManager creates a new cancellation manager
func NewCancellationManager() *CancellationManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &CancellationManager{
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}
}

// Context returns the cancellation context
func (cm *CancellationManager) Context() context.Context {
	return cm.ctx
}

// Cancel cancels all operations
func (cm *CancellationManager) Cancel() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if !cm.cancelled {
		cm.cancelled = true
		cm.cancel()
	}
}

// AddWorker increments the worker wait group
func (cm *CancellationManager) AddWorker() {
	cm.wg.Add(1)
}

// WorkerDone marks a worker as completed
func (cm *CancellationManager) WorkerDone() {
	cm.wg.Done()
}

// Wait waits for all workers to complete
func (cm *CancellationManager) Wait() {
	cm.wg.Wait()
}

// ShouldContinue checks if processing should continue
func (cm *CancellationManager) ShouldContinue() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return !cm.cancelled
}

// IsCancelled returns true if cancellation was requested
func (cm *CancellationManager) IsCancelled() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.cancelled
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		Total:     total,
		StartTime: time.Now(),
	}
}

// Update increments the completed count
func (pt *ProgressTracker) Update(success bool) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	pt.Completed++
	if !success {
		pt.Failed++
	}
}

// Skip increments the skipped count
func (pt *ProgressTracker) Skip() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	pt.Skipped++
}

// GetStats returns current progress statistics
func (pt *ProgressTracker) GetStats() (total, completed, failed, skipped int, elapsed time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	return pt.Total, pt.Completed, pt.Failed, pt.Skipped, time.Since(pt.StartTime)
}

// IsComplete checks if all jobs are processed
func (pt *ProgressTracker) IsComplete() bool {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	return pt.Completed+pt.Skipped >= pt.Total
}

// EstimateTimeRemaining calculates ETA based on current progress
func (pt *ProgressTracker) EstimateTimeRemaining() time.Duration {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	if pt.Completed == 0 {
		return 0
	}
	
	elapsed := time.Since(pt.StartTime)
	remaining := pt.Total - pt.Completed - pt.Skipped
	
	if remaining <= 0 {
		return 0
	}
	
	avgTimePerJob := elapsed / time.Duration(pt.Completed)
	return time.Duration(remaining) * avgTimePerJob
}

// FormatProgress returns a formatted progress string
func (pt *ProgressTracker) FormatProgress() string {
	total, completed, failed, skipped, elapsed := pt.GetStats()
	
	percentage := 0.0
	if total > 0 {
		percentage = float64(completed+skipped) / float64(total) * 100
	}
	
	eta := pt.EstimateTimeRemaining()
	etaStr := ""
	if eta > 0 {
		etaStr = fmt.Sprintf(" (ETA: %v)", eta.Round(time.Second))
	}
	
	return fmt.Sprintf("Progress: %d/%d (%.1f%%) | Success: %d | Failed: %d | Skipped: %d | Elapsed: %v%s",
		completed+skipped, total, percentage, completed-failed, failed, skipped, 
		elapsed.Round(time.Second), etaStr)
}

// processJobsConcurrently processes jobs using a worker pool with cancellation support
func processJobsConcurrently(jobs []WorkJob, numWorkers int) error {
	return ProcessJobsWithCancellation(jobs, numWorkers)
}

// cancellableWorker processes jobs with cancellation support
func cancellableWorker(id int, jobs <-chan WorkJob, results chan<- WorkResult, 
	cancelMgr *CancellationManager, progress *ProgressTracker) {
	
	defer cancelMgr.WorkerDone()
	
	for {
		select {
		case job, ok := <-jobs:
			if !ok {
				return // Channel closed
			}
			
			// Check cancellation before processing
			if !cancelMgr.ShouldContinue() {
				return
			}
			
			// Process the job
			result := processJob(job, cancelMgr.Context())
			
			// Send result if not cancelled
			select {
			case results <- result:
			case <-cancelMgr.Context().Done():
				return
			}
			
		case <-cancelMgr.Context().Done():
			return // Cancelled
		}
	}
}

// processJob processes a single job with the appropriate handler
func processJob(job WorkJob, ctx context.Context) WorkResult {
	startTime := time.Now()
	
	result := WorkResult{
		Job:      job,
		Success:  false,
		Duration: 0,
	}
	
	var err error
	switch job.JobType {
	case "process":
		err = processPhoto(job.PhotoPath, job.DestPath)
		if err != nil {
			if isNoGPSError(err) {
				result.Message = "No GPS data"
				result.Success = false // Still mark as unsuccessful for stats
			} else {
				result.Error = err
			}
		} else {
			result.Success = true
			result.Message = "Processed successfully"
		}
		
	case "datetime":
		// Implement datetime matching logic here
		result.Message = "DateTime matching not yet implemented"
		result.Success = false
		
	case "clean":
		// Implement clean logic here  
		result.Message = "Clean operation not yet implemented"
		result.Success = false
		
	default:
		result.Error = fmt.Errorf("unknown job type: %s", job.JobType)
	}
	
	result.Duration = time.Since(startTime)
	return result
}

// progressReporter displays progress updates
func progressReporter(progress *ProgressTracker, wg *sync.WaitGroup, cancelMgr *CancellationManager) {
	defer wg.Done()
	
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	for !progress.IsComplete() && cancelMgr.ShouldContinue() {
		select {
		case <-ticker.C:
			fmt.Printf("\r%s", progress.FormatProgress())
		case <-cancelMgr.Context().Done():
			return
		}
	}
	
	// Final progress update
	fmt.Printf("\r%s\n", progress.FormatProgress())
}

// printProcessingSummary prints a detailed summary of processing results
func printProcessingSummary(results []WorkResult, progress *ProgressTracker) {
	total, completed, failed, skipped, elapsed := progress.GetStats()
	
	fmt.Printf("\n📊 Processing Summary:\n")
	fmt.Printf("✅ Total processed: %d/%d\n", completed, total)
	fmt.Printf("❌ Failed: %d\n", failed)
	fmt.Printf("⏭️  Skipped: %d\n", skipped)
	fmt.Printf("⏱️  Total time: %v\n", elapsed.Round(time.Second))
	
	if completed > 0 {
		avgTime := elapsed / time.Duration(completed)
		fmt.Printf("📈 Average time per file: %v\n", avgTime.Round(time.Millisecond))
	}
	
	// Group errors by type
	errorCounts := make(map[string]int)
	for _, result := range results {
		if !result.Success && result.Error != nil {
			errorType := "Unknown error"
			if isNoGPSError(result.Error) {
				errorType = "No GPS data"
			} else {
				errorType = result.Error.Error()
			}
			errorCounts[errorType]++
		}
	}
	
	if len(errorCounts) > 0 {
		fmt.Printf("\n❌ Error breakdown:\n")
		for errorType, count := range errorCounts {
			fmt.Printf("   - %s: %d files\n", errorType, count)
		}
	}
}