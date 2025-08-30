//go:build !windows

package extractor

import (
	"fmt"
	"os/exec"
	"syscall"
)

func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func (e *SystemExtractor) extractTarGz(archivePath, dst string) error {
	// Try system tar command
	if _, err := exec.LookPath("tar"); err == nil {
		return e.extractWithTar(archivePath, dst)
	}
	
	// Fall back to optimized extractor
	file, err := e.fs.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	optimized := NewOptimizedExtractor(e.fs)
	return optimized.Extract(file, dst)
}

func (e *SystemExtractor) extractWithTar(archivePath, dst string) error {
	cmd := exec.Command("tar", 
		"-xzf", archivePath,  // extract gzip compressed tar
		"-C", dst,            // change to directory
		"--no-same-owner",    // don't try to restore ownership
	)
	
	// Set process group for better signal handling
	setProcAttr(cmd)
	
	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar command failed: %w, output: %s", err, string(output))
	}
	
	return nil
}

func (e *SystemExtractor) extractZipLinux(archivePath, dst string) error {
	// Try unzip command first
	if _, err := exec.LookPath("unzip"); err == nil {
		cmd := exec.Command("unzip", "-q", "-o", archivePath, "-d", dst)
		setProcAttr(cmd)
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("unzip command failed: %w, output: %s", err, string(output))
		}
		return nil
	}
	
	// Fall back to optimized extractor
	file, err := e.fs.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	optimized := NewOptimizedExtractor(e.fs)
	return optimized.Extract(file, dst)
}

func (e *SystemExtractor) extractZipDarwin(archivePath, dst string) error {
	// macOS has built-in unzip
	cmd := exec.Command("unzip", "-q", "-o", archivePath, "-d", dst)
	setProcAttr(cmd)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unzip command failed: %w, output: %s", err, string(output))
	}
	return nil
}