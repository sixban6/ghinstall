package extractor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// SystemExtractor uses system commands for better performance
type SystemExtractor struct {
	tempDir string
}

func NewSystem() *SystemExtractor {
	return &SystemExtractor{}
}

func (e *SystemExtractor) Extract(src io.Reader, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dst, err)
	}

	// Create temp file for the archive
	tempFile, err := e.createTempFile(src)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile)

	// Detect format and extract using system commands
	format, err := e.detectFormatFromFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to detect format: %w", err)
	}

	switch format {
	case "tar.gz", "tgz":
		return e.extractTarGzSystem(tempFile, dst)
	case "zip":
		return e.extractZipSystem(tempFile, dst)
	default:
		// Fallback to Go implementation
		file, err := os.Open(tempFile)
		if err != nil {
			return fmt.Errorf("failed to open temp file: %w", err)
		}
		defer file.Close()
		
		optimized := NewOptimized()
		return optimized.Extract(file, dst)
	}
}

func (e *SystemExtractor) createTempFile(src io.Reader) (string, error) {
	tempFile, err := os.CreateTemp("", "ghinstall-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Use larger buffer for faster copying
	buffer := make([]byte, 1024*1024) // 1MB buffer
	_, err = io.CopyBuffer(tempFile, src, buffer)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	return tempFile.Name(), nil
}

func (e *SystemExtractor) detectFormatFromFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := make([]byte, 4)
	_, err = file.Read(buf)
	if err != nil {
		return "", err
	}

	return detectFormatFromBytes(buf), nil
}

func (e *SystemExtractor) extractTarGzSystem(archivePath, dst string) error {
	switch runtime.GOOS {
	case "linux", "darwin":
		return e.extractWithTar(archivePath, dst)
	case "windows":
		return e.extractWithPowerShell(archivePath, dst)
	default:
		// Fallback to Go implementation
		file, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer file.Close()
		
		optimized := NewOptimized()
		return optimized.Extract(file, dst)
	}
}

func (e *SystemExtractor) extractWithTar(archivePath, dst string) error {
	// Check if tar command exists
	if _, err := exec.LookPath("tar"); err != nil {
		// Fallback to Go implementation
		file, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer file.Close()
		
		optimized := NewOptimized()
		return optimized.Extract(file, dst)
	}

	// Use system tar command with optimizations
	cmd := exec.Command("tar", 
		"-xzf", archivePath,  // extract gzip compressed tar
		"-C", dst,            // change to directory
		"--no-same-owner",    // don't try to restore ownership
	)
	
	// Set process group for better signal handling
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	
	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar command failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (e *SystemExtractor) extractWithPowerShell(archivePath, dst string) error {
	// Windows PowerShell command to extract tar.gz
	script := fmt.Sprintf(`
		try {
			if (Get-Command tar -ErrorAction SilentlyContinue) {
				tar -xzf "%s" -C "%s"
			} else {
				# Fallback: use .NET classes
				Add-Type -AssemblyName System.IO.Compression.FileSystem
				[System.IO.Compression.ZipFile]::ExtractToDirectory("%s", "%s")
			}
		} catch {
			exit 1
		}
	`, archivePath, dst, archivePath, dst)

	cmd := exec.Command("powershell", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("powershell extraction failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (e *SystemExtractor) extractZipSystem(archivePath, dst string) error {
	switch runtime.GOOS {
	case "linux":
		return e.extractZipLinux(archivePath, dst)
	case "darwin":
		return e.extractZipDarwin(archivePath, dst)
	case "windows":
		return e.extractZipWindows(archivePath, dst)
	default:
		// Fallback to Go implementation
		file, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer file.Close()
		
		optimized := NewOptimized()
		return optimized.Extract(file, dst)
	}
}

func (e *SystemExtractor) extractZipLinux(archivePath, dst string) error {
	// Try unzip command first
	if _, err := exec.LookPath("unzip"); err == nil {
		cmd := exec.Command("unzip", "-q", "-o", archivePath, "-d", dst)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("unzip command failed: %w, output: %s", err, string(output))
		}
		return nil
	}

	// Fallback to Go implementation
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	optimized := NewOptimized()
	return optimized.Extract(file, dst)
}

func (e *SystemExtractor) extractZipDarwin(archivePath, dst string) error {
	// macOS has built-in unzip
	cmd := exec.Command("unzip", "-q", "-o", archivePath, "-d", dst)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unzip command failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (e *SystemExtractor) extractZipWindows(archivePath, dst string) error {
	// Use PowerShell's Expand-Archive
	script := fmt.Sprintf(`
		try {
			Expand-Archive -Path "%s" -DestinationPath "%s" -Force
		} catch {
			exit 1
		}
	`, archivePath, dst)

	cmd := exec.Command("powershell", "-Command", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("powershell zip extraction failed: %w, output: %s", err, string(output))
	}

	return nil
}

// SystemExtractorWithFallback automatically falls back to Go implementation if system commands fail
type SystemExtractorWithFallback struct {
	system   *SystemExtractor
	fallback *OptimizedExtractor
}

func NewSystemWithFallback() *SystemExtractorWithFallback {
	return &SystemExtractorWithFallback{
		system:   NewSystem(),
		fallback: NewOptimized(),
	}
}

func (e *SystemExtractorWithFallback) Extract(src io.Reader, dst string) error {
	// Try system extractor first
	err := e.system.Extract(src, dst)
	if err != nil {
		// If system extraction fails, try reading from src again
		// This requires creating a temp file since src might be consumed
		if file, ok := src.(*os.File); ok {
			// If src is a file, seek back to beginning
			if _, seekErr := file.Seek(0, 0); seekErr == nil {
				return e.fallback.Extract(file, dst)
			}
		}
		
		// If we can't seek, return the original error
		return fmt.Errorf("system extraction failed and cannot retry with Go implementation: %w", err)
	}
	
	return nil
}