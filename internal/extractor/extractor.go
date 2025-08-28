package extractor

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Extractor interface {
	Extract(src io.Reader, dst string) error
}

type MultiExtractor struct{}

func New() Extractor {
	// Return system extractor with fallback for best performance
	//return NewSystemWithFallback()
	return NewOptimized()
}

func NewLegacy() *MultiExtractor {
	return &MultiExtractor{}
}

func (e *MultiExtractor) Extract(src io.Reader, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dst, err)
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return fmt.Errorf("failed to read source data: %w", err)
	}

	format, err := detectFormat(data)
	if err != nil {
		return fmt.Errorf("failed to detect archive format: %w", err)
	}

	switch format {
	case "tar.gz", "tgz":
		return e.extractTarGz(data, dst)
	case "zip":
		return e.extractZip(data, dst)
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

func detectFormat(data []byte) (string, error) {
	if len(data) < 4 {
		return "", fmt.Errorf("file too small to detect format")
	}

	if data[0] == 0x1f && data[1] == 0x8b {
		return "tar.gz", nil
	}

	if data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04 {
		return "zip", nil
	}

	return "", fmt.Errorf("unknown archive format")
}

func (e *MultiExtractor) extractTarGz(data []byte, dst string) error {
	gzReader, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		if err := e.extractTarEntry(tarReader, header, dst); err != nil {
			return fmt.Errorf("failed to extract tar entry %s: %w", header.Name, err)
		}
	}

	return nil
}

func (e *MultiExtractor) extractTarEntry(reader *tar.Reader, header *tar.Header, dst string) error {
	path := filepath.Join(dst, header.Name)
	cleanedDst := filepath.Clean(dst)
	cleanedPath := filepath.Clean(path)

	// Skip the current directory entry
	if header.Name == "./" || header.Name == "." {
		return nil
	}

	// Prevent directory traversal attacks
	if !strings.HasPrefix(cleanedPath, cleanedDst) ||
		(cleanedPath != cleanedDst && !strings.HasPrefix(cleanedPath, cleanedDst+string(os.PathSeparator))) {
		return fmt.Errorf("invalid file path: %s", header.Name)
	}

	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(path, os.FileMode(header.Mode))
	case tar.TypeReg:
		return e.extractFile(reader, path, os.FileMode(header.Mode))
	case tar.TypeSymlink:
		linkTarget := header.Linkname
		if !strings.HasPrefix(filepath.Join(dst, linkTarget), dst) {
			return fmt.Errorf("invalid symlink target: %s", linkTarget)
		}
		return os.Symlink(linkTarget, path)
	default:
		return nil
	}
}

func (e *MultiExtractor) extractZip(data []byte, dst string) error {
	reader, err := zip.NewReader(strings.NewReader(string(data)), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	for _, file := range reader.File {
		if err := e.extractZipEntry(file, dst); err != nil {
			return fmt.Errorf("failed to extract zip entry %s: %w", file.Name, err)
		}
	}

	return nil
}

func (e *MultiExtractor) extractZipEntry(file *zip.File, dst string) error {
	path := filepath.Join(dst, file.Name)
	cleanedDst := filepath.Clean(dst)
	cleanedPath := filepath.Clean(path)

	// Skip the current directory entry
	if file.Name == "./" || file.Name == "." {
		return nil
	}

	// Prevent directory traversal attacks
	if !strings.HasPrefix(cleanedPath, cleanedDst) ||
		(cleanedPath != cleanedDst && !strings.HasPrefix(cleanedPath, cleanedDst+string(os.PathSeparator))) {
		return fmt.Errorf("invalid file path: %s", file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(path, file.FileInfo().Mode())
	}

	fileReader, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip file entry: %w", err)
	}
	defer fileReader.Close()

	return e.extractFile(fileReader, path, file.FileInfo().Mode())
}

func (e *MultiExtractor) extractFile(reader io.Reader, path string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for file %s: %w", path, err)
	}

	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, reader)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

type TarGzExtractor struct{}

func NewTarGzExtractor() *TarGzExtractor {
	return &TarGzExtractor{}
}

func (e *TarGzExtractor) Extract(src io.Reader, dst string) error {
	optimized := NewOptimized()
	return optimized.Extract(src, dst)
}

type ZipExtractor struct{}

func NewZipExtractor() *ZipExtractor {
	return &ZipExtractor{}
}

func (e *ZipExtractor) Extract(src io.Reader, dst string) error {
	optimized := NewOptimized()
	return optimized.Extract(src, dst)
}
