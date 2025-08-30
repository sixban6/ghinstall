//go:build windows

package extractor

import (
	"fmt"
	"os/exec"
)

func setProcAttr(cmd *exec.Cmd) {
	// Windows doesn't support Setpgid, so we do nothing here
}

func (e *SystemExtractor) extractTarGz(archivePath, dst string) error {
	// Windows typically doesn't have tar command, fall back to optimized extractor
	file, err := e.fs.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	optimized := NewOptimizedExtractor(e.fs)
	return optimized.Extract(file, dst)
}

func (e *SystemExtractor) extractWithTar(archivePath, dst string) error {
	// Try tar command if available (newer Windows versions may have it)
	if _, err := exec.LookPath("tar"); err == nil {
		cmd := exec.Command("tar", 
			"-xzf", archivePath,  // extract gzip compressed tar
			"-C", dst,            // change to directory
		)
		
		// Set process attributes (no-op on Windows)
		setProcAttr(cmd)
		
		// Capture both stdout and stderr
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("tar command failed: %w, output: %s", err, string(output))
		}
		return nil
	}
	
	// Fall back to optimized extractor
	return e.extractTarGz(archivePath, dst)
}

func (e *SystemExtractor) extractZipLinux(archivePath, dst string) error {
	// This shouldn't be called on Windows, but provide fallback
	return e.extractZipWindows(archivePath, dst)
}

func (e *SystemExtractor) extractZipDarwin(archivePath, dst string) error {
	// This shouldn't be called on Windows, but provide fallback
	return e.extractZipWindows(archivePath, dst)
}