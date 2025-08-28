package extractor

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	log "github.com/sixban6/ghinstall/internal/logger"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Extractor interface {
	Extract(src io.Reader, dst string) error
}

type MultiExtractor struct {
	cacheFirst bool // true = 先落盘再解压
}

func (m MultiExtractor) WithCache() *MultiExtractor {
	m.cacheFirst = true
	return &m
}

func New() Extractor {
	// Return system extractor with fallback for best performance
	//return NewSystemWithFallback()
	return NewLegacy()
	//return NewOptimized()
}

func NewLegacy() *MultiExtractor {
	return &MultiExtractor{cacheFirst: true}
}

func fileSize(f *os.File) int64 {
	if fi, err := f.Stat(); err == nil {
		return fi.Size()
	}
	return 0
}

func (e *MultiExtractor) Extract(src io.Reader, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dst, err)
	}

	// 如果启用了先缓存
	var tmp *os.File
	var err error
	if e.cacheFirst {
		start := time.Now()
		tmp, err = writeToTemp(src) // 复用已有函数
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()
		log.Info("Download Finished，Cost %v，Size %d Bytes\n", time.Since(start), fileSize(tmp))
		// 把文件重新变成 Reader
		src = tmp
	}
	var format string
	format, err = detectFormat(tmp, fileSize(tmp))

	if err != nil {
		return fmt.Errorf("failed to detect archive format: %w", err)
	}

	switch format {
	case "tar.gz", "tgz":
		return e.extractTarGz(tmp, dst)
	case "zip":
		return e.extractZip(tmp, dst)
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

func writeToTemp(r io.Reader) (*os.File, error) {
	tmp, err := os.CreateTemp("", "extract-*.tmp")
	if err != nil {
		return nil, err
	}

	// 512 KiB 缓冲
	buf := make([]byte, 512*1024)
	if _, err := io.CopyBuffer(tmp, r, buf); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, err
	}

	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return tmp, nil
}

func detectFormat(r io.ReaderAt, size int64) (string, error) {
	buf := make([]byte, 4)
	if _, err := r.ReadAt(buf, 0); err != nil {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}
	if bytes.HasPrefix(buf, []byte{0x1f, 0x8b}) {
		return "tar.gz", nil
	}
	if bytes.HasPrefix(buf, []byte{'P', 'K', 0x03, 0x04}) {
		return "zip", nil
	}
	return "", fmt.Errorf("unknown archive format")
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

// 解压 tar.gz
func (e *MultiExtractor) extractTarGz(f *os.File, dst string) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek tar.gz file: %w", err)
	}

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		if err := e.extractTarEntry(tr, hdr, dst); err != nil {
			return fmt.Errorf("extract tar entry %s: %w", hdr.Name, err)
		}
	}
	return nil
}

// 解压 zip
func (e *MultiExtractor) extractZip(f *os.File, dst string) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek zip file: %w", err)
	}

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat zip file: %w", err)
	}

	zr, err := zip.NewReader(f, stat.Size())
	if err != nil {
		return fmt.Errorf("create zip reader: %w", err)
	}

	for _, file := range zr.File {
		if err := e.extractZipEntry(file, dst); err != nil {
			return fmt.Errorf("extract zip entry %s: %w", file.Name, err)
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
