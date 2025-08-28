package extractor

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// OptimizedExtractor provides better performance by avoiding unnecessary memory copies
type OptimizedExtractor struct {
	bufferSize int
}

func NewOptimized() *OptimizedExtractor {
	return &OptimizedExtractor{
		bufferSize: 64 * 1024, // 64KB buffer
	}
}

func (e *OptimizedExtractor) Extract(src io.Reader, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dst, err)
	}

	// Use buffered reader for better I/O performance
	bufferedSrc := bufio.NewReaderSize(src, e.bufferSize)
	
	// Read only the first few bytes to detect format
	peek, err := bufferedSrc.Peek(4)
	if err != nil {
		return fmt.Errorf("failed to peek archive data: %w", err)
	}

	format := detectFormatFromBytes(peek)
	
	switch format {
	case "tar.gz", "tgz":
		return e.extractTarGzStream(bufferedSrc, dst)
	case "zip":
		// Zip requires seeking, so we need to read all data
		return e.extractZipFromReader(bufferedSrc, dst)
	default:
		return fmt.Errorf("unsupported archive format")
	}
}

func detectFormatFromBytes(data []byte) string {
	if len(data) < 4 {
		return ""
	}

	if data[0] == 0x1f && data[1] == 0x8b {
		return "tar.gz"
	}

	if data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04 {
		return "zip"
	}

	return ""
}

// Optimized tar.gz extraction using streaming
func (e *OptimizedExtractor) extractTarGzStream(src io.Reader, dst string) error {
	gzReader, err := gzip.NewReader(src)
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

		if err := e.extractTarEntryOptimized(tarReader, header, dst); err != nil {
			return fmt.Errorf("failed to extract tar entry %s: %w", header.Name, err)
		}
	}

	return nil
}

func (e *OptimizedExtractor) extractTarEntryOptimized(reader *tar.Reader, header *tar.Header, dst string) error {
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
		return e.extractFileOptimized(reader, path, os.FileMode(header.Mode))
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

// Optimized file extraction with buffered I/O
func (e *OptimizedExtractor) extractFileOptimized(reader io.Reader, path string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for file %s: %w", path, err)
	}

	outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer outFile.Close()

	// Use buffered writer for better performance
	bufferedWriter := bufio.NewWriterSize(outFile, e.bufferSize)
	defer bufferedWriter.Flush()

	// Use larger buffer for copying
	buffer := make([]byte, e.bufferSize)
	_, err = io.CopyBuffer(bufferedWriter, reader, buffer)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// For ZIP files, we still need to read all data due to seeking requirements
func (e *OptimizedExtractor) extractZipFromReader(src io.Reader, dst string) error {
	// Use bytes.Buffer for better memory management
	var buf bytes.Buffer
	_, err := buf.ReadFrom(src)
	if err != nil {
		return fmt.Errorf("failed to read zip data: %w", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	for _, file := range reader.File {
		if err := e.extractZipEntryOptimized(file, dst); err != nil {
			return fmt.Errorf("failed to extract zip entry %s: %w", file.Name, err)
		}
	}

	return nil
}

func (e *OptimizedExtractor) extractZipEntryOptimized(file *zip.File, dst string) error {
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

	return e.extractFileOptimized(fileReader, path, file.FileInfo().Mode())
}