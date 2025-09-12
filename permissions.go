package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// PermissionError represents a file permission error with additional context
type PermissionError struct {
	Path      string
	Operation string
	Err       error
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("permission error during %s on %s: %v", e.Operation, e.Path, e.Err)
}

// isPermissionError checks if an error is related to file permissions
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check for common permission error types
	if os.IsPermission(err) {
		return true
	}
	
	// Check for syscall errors
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			return errno == syscall.EACCES || errno == syscall.EPERM
		}
	}
	
	// Check for common permission error strings
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "access is denied") ||
		strings.Contains(errStr, "operation not permitted")
}

// checkFilePermissions validates that we can read from source and write to destination
func checkFilePermissions(sourcePath, destPath string) error {
	// Check source file read permission
	_, err := os.Stat(sourcePath)
	if err != nil {
		if isPermissionError(err) {
			return &PermissionError{
				Path:      sourcePath,
				Operation: "read",
				Err:       err,
			}
		}
		return fmt.Errorf("failed to stat source file %s: %v", sourcePath, err)
	}
	
	// Check if source is readable
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		if isPermissionError(err) {
			return &PermissionError{
				Path:      sourcePath,
				Operation: "read",
				Err:       err,
			}
		}
		return fmt.Errorf("failed to open source file %s: %v", sourcePath, err)
	}
	sourceFile.Close()
	
	// Check destination directory write permission
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		if isPermissionError(err) {
			return &PermissionError{
				Path:      destDir,
				Operation: "create directory",
				Err:       err,
			}
		}
		return fmt.Errorf("failed to create destination directory %s: %v", destDir, err)
	}
	
	// Test write permission by creating a temporary file
	tempFile := filepath.Join(destDir, ".photo-meta-perm-test")
	testFile, err := os.Create(tempFile)
	if err != nil {
		if isPermissionError(err) {
			return &PermissionError{
				Path:      destDir,
				Operation: "write",
				Err:       err,
			}
		}
		return fmt.Errorf("failed to test write permission in %s: %v", destDir, err)
	}
	testFile.Close()
	os.Remove(tempFile) // Clean up test file
	
	// Check if destination already exists and is writable
	if _, err := os.Stat(destPath); err == nil {
		// File exists, check if we can write to it
		destFile, err := os.OpenFile(destPath, os.O_WRONLY, 0)
		if err != nil {
			if isPermissionError(err) {
				return &PermissionError{
					Path:      destPath,
					Operation: "write",
					Err:       err,
				}
			}
		} else {
			destFile.Close()
		}
	}
	
	return nil
}

// safeFileMove attempts to move a file with enhanced permission error handling
func safeFileMove(sourcePath, destPath string) error {
	// Pre-check permissions
	if err := checkFilePermissions(sourcePath, destPath); err != nil {
		return err
	}
	
	// Attempt the move
	if err := os.Rename(sourcePath, destPath); err != nil {
		if isPermissionError(err) {
			return &PermissionError{
				Path:      sourcePath,
				Operation: "move",
				Err:       err,
			}
		}
		
		// If rename fails (possibly cross-device), try copy+delete
		return safeCopyAndDelete(sourcePath, destPath)
	}
	
	return nil
}

// safeCopyAndDelete copies a file and deletes the original with permission checks
func safeCopyAndDelete(sourcePath, destPath string) error {
	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		if isPermissionError(err) {
			return &PermissionError{
				Path:      sourcePath,
				Operation: "read",
				Err:       err,
			}
		}
		return fmt.Errorf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()
	
	// Get source file info for permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %v", err)
	}
	
	// Create destination file
	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		if isPermissionError(err) {
			return &PermissionError{
				Path:      destPath,
				Operation: "create",
				Err:       err,
			}
		}
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()
	
	// Copy data
	_, err = sourceFile.WriteTo(destFile)
	if err != nil {
		if isPermissionError(err) {
			// Clean up partial file
			os.Remove(destPath)
			return &PermissionError{
				Path:      destPath,
				Operation: "write",
				Err:       err,
			}
		}
		os.Remove(destPath)
		return fmt.Errorf("failed to copy file data: %v", err)
	}
	
	// Sync to ensure data is written
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %v", err)
	}
	
	// Remove source file
	if err := os.Remove(sourcePath); err != nil {
		if isPermissionError(err) {
			return &PermissionError{
				Path:      sourcePath,
				Operation: "delete",
				Err:       err,
			}
		}
		return fmt.Errorf("failed to remove source file: %v", err)
	}
	
	return nil
}

// handlePermissionError provides user-friendly error handling for permission issues
func handlePermissionError(err error, suggestSudo bool) {
	if permErr, ok := err.(*PermissionError); ok {
		fmt.Printf("❌ Permission Error: Cannot %s %s\n", permErr.Operation, permErr.Path)
		fmt.Printf("   Reason: %v\n", permErr.Err)
		
		// Provide helpful suggestions
		switch permErr.Operation {
		case "read":
			fmt.Printf("💡 Suggestions:\n")
			fmt.Printf("   • Check if the file exists and is readable\n")
			fmt.Printf("   • Verify file ownership: ls -la %s\n", permErr.Path)
			if suggestSudo {
				fmt.Printf("   • Run with elevated permissions: sudo ./photo-meta ...\n")
			}
		case "write", "create", "create directory":
			fmt.Printf("💡 Suggestions:\n")
			fmt.Printf("   • Check if you have write permissions to the directory\n")
			fmt.Printf("   • Verify directory ownership: ls -la %s\n", filepath.Dir(permErr.Path))
			if suggestSudo {
				fmt.Printf("   • Run with elevated permissions: sudo ./photo-meta ...\n")
			}
			fmt.Printf("   • Try a different destination directory\n")
		case "delete":
			fmt.Printf("💡 Suggestions:\n")
			fmt.Printf("   • The file was copied successfully but couldn't be deleted from source\n")
			fmt.Printf("   • You may need to manually remove: %s\n", permErr.Path)
			fmt.Printf("   • Check file attributes: lsattr %s (Linux) or ls -lO %s (macOS)\n", permErr.Path, permErr.Path)
		case "move":
			fmt.Printf("💡 Suggestions:\n")
			fmt.Printf("   • Check permissions on both source and destination\n")
			fmt.Printf("   • Files may be on different filesystems (will attempt copy instead)\n")
		}
		fmt.Printf("\n")
	} else {
		fmt.Printf("❌ File operation error: %v\n", err)
	}
}