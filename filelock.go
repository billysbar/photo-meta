package main

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// FileLockManager manages file locks to prevent race conditions
type FileLockManager struct {
	locks map[string]*sync.RWMutex
	mu    sync.Mutex
}

// SafeFileOperation represents a locked file operation
type SafeFileOperation struct {
	manager     *FileLockManager
	lockedPaths []string
}

// NewFileLockManager creates a new file lock manager
func NewFileLockManager() *FileLockManager {
	return &FileLockManager{
		locks: make(map[string]*sync.RWMutex),
	}
}

// LockFile acquires a lock for a specific file path
func (flm *FileLockManager) LockFile(path string) {
	flm.mu.Lock()
	defer flm.mu.Unlock()
	
	// Normalize path for consistent locking
	normalPath := filepath.Clean(path)
	
	// Create lock if it doesn't exist
	if _, exists := flm.locks[normalPath]; !exists {
		flm.locks[normalPath] = &sync.RWMutex{}
	}
	
	// Unlock manager mutex before acquiring file lock to avoid deadlock
	lock := flm.locks[normalPath]
	flm.mu.Unlock()
	
	// Acquire the file lock
	lock.Lock()
	
	// Reacquire manager mutex
	flm.mu.Lock()
}

// UnlockFile releases a lock for a specific file path
func (flm *FileLockManager) UnlockFile(path string) {
	flm.mu.Lock()
	normalPath := filepath.Clean(path)
	
	if lock, exists := flm.locks[normalPath]; exists {
		flm.mu.Unlock()
		lock.Unlock()
		flm.mu.Lock()
	}
	
	flm.mu.Unlock()
}

// LockDirectory locks a file and its predicted target directory
func (flm *FileLockManager) LockDirectory(baseDir, filePath string) []string {
	var lockedPaths []string
	
	// Lock the original file
	flm.LockFile(filePath)
	lockedPaths = append(lockedPaths, filePath)
	
	// Lock predicted target directory based on file content hash
	if targetDir := flm.predictTargetDirectory(baseDir, filePath); targetDir != "" {
		targetLockPath := filepath.Join(targetDir, ".dir_lock")
		flm.LockFile(targetLockPath)
		lockedPaths = append(lockedPaths, targetLockPath)
	}
	
	return lockedPaths
}

// predictTargetDirectory predicts where a file might be moved to
func (flm *FileLockManager) predictTargetDirectory(baseDir, filePath string) string {
	// Create a hash-based prediction for the target directory
	// This is a simplified version - in PhotoXX it uses metadata extraction
	hash := md5.Sum([]byte(filePath))
	hashStr := fmt.Sprintf("%x", hash)
	
	// Use first few characters of hash to create subdirectory
	subDir := hashStr[:2]
	return filepath.Join(baseDir, subDir)
}

// NewSafeFileOperation creates a new safe file operation with locks
func (flm *FileLockManager) NewSafeFileOperation(baseDir, filePath string) *SafeFileOperation {
	lockedPaths := flm.LockDirectory(baseDir, filePath)
	
	return &SafeFileOperation{
		manager:     flm,
		lockedPaths: lockedPaths,
	}
}

// Release releases all locks held by this operation
func (sfo *SafeFileOperation) Release() {
	for _, path := range sfo.lockedPaths {
		sfo.manager.UnlockFile(path)
	}
	sfo.lockedPaths = nil
}

// GetLockedPaths returns the paths currently locked by this operation
func (sfo *SafeFileOperation) GetLockedPaths() []string {
	return append([]string{}, sfo.lockedPaths...) // Return copy
}

// Global file lock manager instance
var globalFileLockManager = NewFileLockManager()

// WithFilelocks executes a function with appropriate file locks
func WithFilelocks(baseDir, filePath string, fn func() error) error {
	operation := globalFileLockManager.NewSafeFileOperation(baseDir, filePath)
	defer operation.Release()
	
	return fn()
}

// WithSimpleFileLock executes a function with a simple file lock
func WithSimpleFileLock(filePath string, fn func() error) error {
	globalFileLockManager.LockFile(filePath)
	defer globalFileLockManager.UnlockFile(filePath)
	
	return fn()
}

// LockKey generates a consistent lock key for directory structures
func LockKey(parts ...string) string {
	// Join parts and normalize to create consistent lock keys
	joined := filepath.Join(parts...)
	normalized := filepath.Clean(joined)
	
	// Convert to lowercase for case-insensitive filesystems
	return strings.ToLower(normalized)
}

// DirectoryLock represents a directory-level lock
type DirectoryLock struct {
	manager *FileLockManager
	key     string
	locked  bool
}

// NewDirectoryLock creates a new directory lock
func NewDirectoryLock(manager *FileLockManager, directory string) *DirectoryLock {
	return &DirectoryLock{
		manager: manager,
		key:     LockKey(directory, ".directory_lock"),
		locked:  false,
	}
}

// Lock acquires the directory lock
func (dl *DirectoryLock) Lock() {
	if !dl.locked {
		dl.manager.LockFile(dl.key)
		dl.locked = true
	}
}

// Unlock releases the directory lock
func (dl *DirectoryLock) Unlock() {
	if dl.locked {
		dl.manager.UnlockFile(dl.key)
		dl.locked = false
	}
}

// IsLocked returns true if the lock is currently held
func (dl *DirectoryLock) IsLocked() bool {
	return dl.locked
}

// BatchLock manages multiple locks for batch operations
type BatchLock struct {
	manager *FileLockManager
	locks   []*DirectoryLock
}

// NewBatchLock creates a new batch lock manager
func NewBatchLock(manager *FileLockManager) *BatchLock {
	return &BatchLock{
		manager: manager,
		locks:   make([]*DirectoryLock, 0),
	}
}

// AddDirectory adds a directory to the batch lock
func (bl *BatchLock) AddDirectory(directory string) {
	lock := NewDirectoryLock(bl.manager, directory)
	bl.locks = append(bl.locks, lock)
}

// LockAll acquires all locks in the batch
func (bl *BatchLock) LockAll() {
	for _, lock := range bl.locks {
		lock.Lock()
	}
}

// UnlockAll releases all locks in the batch
func (bl *BatchLock) UnlockAll() {
	// Unlock in reverse order to prevent potential deadlocks
	for i := len(bl.locks) - 1; i >= 0; i-- {
		bl.locks[i].Unlock()
	}
}

// Count returns the number of locks in the batch
func (bl *BatchLock) Count() int {
	return len(bl.locks)
}

// WithBatchLocks executes a function with batch directory locks
func WithBatchLocks(directories []string, fn func() error) error {
	batch := NewBatchLock(globalFileLockManager)
	
	for _, dir := range directories {
		batch.AddDirectory(dir)
	}
	
	batch.LockAll()
	defer batch.UnlockAll()
	
	return fn()
}