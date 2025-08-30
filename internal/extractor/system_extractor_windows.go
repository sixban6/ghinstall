//go:build windows

package extractor

import "os/exec"

func setProcAttr(cmd *exec.Cmd) {
	// Windows doesn't support Setpgid, so we do nothing here
}

func (e *SystemExtractor) extractZipLinux(archivePath, dst string) error {
	// This shouldn't be called on Windows, but provide fallback
	return e.extractZipWindows(archivePath, dst)
}

func (e *SystemExtractor) extractZipDarwin(archivePath, dst string) error {
	// This shouldn't be called on Windows, but provide fallback
	return e.extractZipWindows(archivePath, dst)
}