package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// SignalHandler manages graceful shutdown on system signals
type SignalHandler struct {
	cancelMgr *CancellationManager
	signals   chan os.Signal
	done      chan struct{}
	once      sync.Once
}

// NewSignalHandler creates a new signal handler
func NewSignalHandler(cancelMgr *CancellationManager) *SignalHandler {
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	
	// Register for common termination signals
	signal.Notify(signals, 
		os.Interrupt,    // SIGINT (Ctrl+C)
		syscall.SIGTERM, // SIGTERM 
		syscall.SIGQUIT, // SIGQUIT
	)
	
	return &SignalHandler{
		cancelMgr: cancelMgr,
		signals:   signals,
		done:      done,
	}
}

// Start begins monitoring for signals
func (sh *SignalHandler) Start() {
	go func() {
		select {
		case sig := <-sh.signals:
			fmt.Printf("\n🛑 Received signal: %v\n", sig)
			fmt.Println("⏹️  Initiating graceful shutdown...")
			sh.cancelMgr.Cancel()
			
		case <-sh.done:
			return
		}
	}()
}

// Stop stops the signal handler
func (sh *SignalHandler) Stop() {
	sh.once.Do(func() {
		signal.Stop(sh.signals)
		close(sh.done)
	})
}

// GracefulShutdown handles graceful shutdown with timeout
func (sh *SignalHandler) GracefulShutdown(timeout time.Duration) error {
	fmt.Printf("⏳ Waiting up to %v for workers to complete...\n", timeout)
	
	// Create a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// Channel to signal when workers are done
	workersDone := make(chan struct{})
	
	// Wait for workers in a goroutine
	go func() {
		sh.cancelMgr.Wait()
		close(workersDone)
	}()
	
	// Wait for either completion or timeout
	select {
	case <-workersDone:
		fmt.Println("✅ All workers completed gracefully")
		return nil
		
	case <-ctx.Done():
		fmt.Println("⚠️  Graceful shutdown timeout reached")
		return fmt.Errorf("graceful shutdown timeout after %v", timeout)
	}
}

// Enhanced CancellationManager with better cancellation support
func (cm *CancellationManager) CancelWithReason(reason string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if !cm.cancelled {
		cm.cancelled = true
		fmt.Printf("🔴 Cancellation requested: %s\n", reason)
		cm.cancel()
	}
}

// IsGracefulShutdown checks if we're in graceful shutdown mode
func (cm *CancellationManager) IsGracefulShutdown() bool {
	return cm.IsCancelled()
}

// GetElapsed returns time since cancellation manager started
func (cm *CancellationManager) GetElapsed() time.Duration {
	return time.Since(cm.startTime)
}

// Enhanced progress reporter with cancellation awareness
func cancellableProgressReporter(progress *ProgressTracker, wg *sync.WaitGroup, 
	cancelMgr *CancellationManager) {
	defer wg.Done()
	
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	lastUpdate := ""
	
	for !progress.IsComplete() && cancelMgr.ShouldContinue() {
		select {
		case <-ticker.C:
			progressStr := progress.FormatProgress()
			
			// Add cancellation status if cancelled
			if cancelMgr.IsCancelled() {
				progressStr += " [CANCELLING...]"
			}
			
			// Only update if progress changed to reduce terminal spam
			if progressStr != lastUpdate {
				fmt.Printf("\r%s", progressStr)
				lastUpdate = progressStr
			}
			
		case <-cancelMgr.Context().Done():
			fmt.Printf("\r%s [CANCELLED]\n", progress.FormatProgress())
			return
		}
	}
	
	// Final progress update
	finalMsg := progress.FormatProgress()
	if cancelMgr.IsCancelled() {
		finalMsg += " [CANCELLED]"
	}
	fmt.Printf("\r%s\n", finalMsg)
}

// cancellableProgressBarReporter displays enhanced progress bar updates
func cancellableProgressBarReporter(progress *ProgressTracker, wg *sync.WaitGroup, 
	cancelMgr *CancellationManager) {
	defer wg.Done()
	
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	lastUpdate := ""
	
	for !progress.IsComplete() && cancelMgr.ShouldContinue() {
		select {
		case <-ticker.C:
			progressStr := progress.FormatProgressBar()
			
			// Add cancellation status if cancelled
			if cancelMgr.IsCancelled() {
				progressStr += " [CANCELLING...]"
			}
			
			// Only update if progress changed to reduce terminal spam
			if progressStr != lastUpdate {
				fmt.Printf("\r%s", progressStr)
				lastUpdate = progressStr
			}
			
		case <-cancelMgr.Context().Done():
			fmt.Printf("\r%s [CANCELLED]\n", progress.FormatProgressBar())
			return
		}
	}
	
	// Final progress update
	finalMsg := progress.FormatProgressBar()
	if cancelMgr.IsCancelled() {
		finalMsg += " [CANCELLED]"
	}
	fmt.Printf("\r%s\n", finalMsg)
}

// ProcessJobsWithCancellation processes jobs with full cancellation support
func ProcessJobsWithCancellation(jobs []WorkJob, numWorkers int, showProgress bool) error {
	// Validate worker count (1-16 workers)
	if numWorkers < 1 {
		numWorkers = 1
	} else if numWorkers > 16 {
		numWorkers = 16
	}
	
	// Initialize tracking and cancellation
	progress := NewProgressTracker(len(jobs))
	cancelMgr := NewCancellationManager()
	signalHandler := NewSignalHandler(cancelMgr)
	
	// Start signal monitoring
	signalHandler.Start()
	defer signalHandler.Stop()
	
	// Create channels
	jobChan := make(chan WorkJob, len(jobs))
	resultChan := make(chan WorkResult, len(jobs))
	
	// Start workers
	fmt.Printf("🚀 Starting %d workers to process %d jobs...\n", numWorkers, len(jobs))
	
	for i := 0; i < numWorkers; i++ {
		cancelMgr.AddWorker()
		go cancellableWorker(i, jobChan, resultChan, cancelMgr, progress)
	}
	
	// Start progress reporter
	var progressWg sync.WaitGroup
	if showProgress {
		progressWg.Add(1)
		go cancellableProgressBarReporter(progress, &progressWg, cancelMgr)
	}
	
	// Send jobs
	go func() {
		defer close(jobChan)
		for _, job := range jobs {
			select {
			case jobChan <- job:
			case <-cancelMgr.Context().Done():
				fmt.Println("🔴 Job distribution cancelled")
				return
			}
		}
	}()
	
	// Collect results
	var results []WorkResult
	resultsComplete := make(chan struct{})
	
	go func() {
		defer close(resultsComplete)
		for i := 0; i < len(jobs); i++ {
			select {
			case result := <-resultChan:
				results = append(results, result)
				progress.Update(result.Success)
				
			case <-cancelMgr.Context().Done():
				fmt.Println("🔴 Result collection cancelled")
				return
			}
		}
	}()
	
	// Wait for completion or cancellation
	select {
	case <-resultsComplete:
		// Normal completion
		cancelMgr.Wait()
		if showProgress {
			progressWg.Wait()
		}
		
	case <-cancelMgr.Context().Done():
		// Cancellation requested
		fmt.Println("🔴 Processing cancelled, waiting for graceful shutdown...")
		
		// Attempt graceful shutdown
		if err := signalHandler.GracefulShutdown(30 * time.Second); err != nil {
			fmt.Printf("⚠️  %v\n", err)
		}
		if showProgress {
			progressWg.Wait()
		}
	}
	
	// Print final summary
	printCancellableProcessingSummary(results, progress, cancelMgr)
	
	// Return appropriate error if cancelled
	if cancelMgr.IsCancelled() {
		return fmt.Errorf("processing was cancelled")
	}
	
	return nil
}

// Enhanced summary with cancellation information
func printCancellableProcessingSummary(results []WorkResult, progress *ProgressTracker, 
	cancelMgr *CancellationManager) {
	
	total, completed, failed, skipped, elapsed := progress.GetStats()
	
	// Count file types
	photoCount := 0
	videoCount := 0
	photoSuccess := 0
	videoSuccess := 0
	photoFailed := 0
	videoFailed := 0
	
	for _, result := range results {
		if isVideoFile(result.Job.PhotoPath) {
			videoCount++
			if result.Success {
				videoSuccess++
			} else {
				videoFailed++
			}
		} else {
			photoCount++
			if result.Success {
				photoSuccess++
			} else {
				photoFailed++
			}
		}
	}
	
	fmt.Printf("\n📊 Processing Summary:\n")
	
	if cancelMgr.IsCancelled() {
		fmt.Printf("🔴 Status: CANCELLED\n")
	} else {
		fmt.Printf("✅ Status: COMPLETED\n")
	}
	
	fmt.Printf("📈 Progress: %d/%d processed\n", completed, total)
	fmt.Printf("✅ Total successful: %d\n", completed-failed)
	fmt.Printf("📷 Photos successful: %d/%d\n", photoSuccess, photoCount)
	fmt.Printf("🎥 Videos successful: %d/%d\n", videoSuccess, videoCount)
	fmt.Printf("❌ Total failed: %d\n", failed)
	fmt.Printf("⏭️  Skipped: %d\n", skipped)
	fmt.Printf("⏱️  Total time: %v\n", elapsed.Round(time.Second))
	
	if completed > 0 {
		avgTime := elapsed / time.Duration(completed)
		fmt.Printf("📊 Average time per file: %v\n", avgTime.Round(time.Millisecond))
	}
	
	// Show completion percentage
	percentage := 0.0
	if total > 0 {
		percentage = float64(completed+skipped) / float64(total) * 100
	}
	fmt.Printf("📋 Completion: %.1f%%\n", percentage)
	
	// Group errors by type and file type
	errorCounts := make(map[string]int)
	photoErrors := make(map[string]int)
	videoErrors := make(map[string]int)
	
	for _, result := range results {
		if !result.Success {
			errorType := "Unknown error"
			if result.Error != nil {
				if isNoGPSError(result.Error) {
					errorType = "No GPS data"
				} else {
					errorType = result.Error.Error()
				}
			} else if result.Message != "" {
				errorType = result.Message
			}
			
			errorCounts[errorType]++
			
			if isVideoFile(result.Job.PhotoPath) {
				videoErrors[errorType]++
			} else {
				photoErrors[errorType]++
			}
		}
	}
	
	if len(errorCounts) > 0 {
		fmt.Printf("\n❌ Error breakdown:\n")
		for errorType, count := range errorCounts {
			photoCount := photoErrors[errorType]
			videoCount := videoErrors[errorType]
			fmt.Printf("   - %s: %d files", errorType, count)
			if photoCount > 0 && videoCount > 0 {
				fmt.Printf(" (%d photos, %d videos)", photoCount, videoCount)
			} else if videoCount > 0 {
				fmt.Printf(" (%d videos)", videoCount)
			} else if photoCount > 0 {
				fmt.Printf(" (%d photos)", photoCount)
			}
			fmt.Println()
		}
	}
	
	if videoSuccess > 0 {
		fmt.Printf("\n🎥 Videos organized in VIDEO-FILES/ directory structure\n")
	}
	
	if cancelMgr.IsCancelled() {
		fmt.Printf("\n💡 Tip: You can resume processing by running the command again\n")
	}
}