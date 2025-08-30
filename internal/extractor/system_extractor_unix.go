//go:build !windows

package extractor

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func setProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	optimized := NewOptimized()
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