package extractor

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ------------------------------------------------------------------
// 公共接口
// ------------------------------------------------------------------

type Extractor interface {
	Extract(src io.Reader, dst string) error
}

// New 返回一个默认实现（目前就是 MultiExtractor）
func New() Extractor { return &MultiExtractor{} }

// ------------------------------------------------------------------
// 多格式实现
// ------------------------------------------------------------------

type MultiExtractor struct{}

func (m *MultiExtractor) Extract(src io.Reader, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("create dst dir: %w", err)
	}

	// 预读前 512 字节做格式嗅探
	peek, err := peekReader(src, 512)
	if err != nil {
		return fmt.Errorf("peek header: %w", err)
	}

	// 根据嗅探结果选择子提取器
	var sub Extractor
	switch detectFormat(peek) {
	case "zip":
		sub = &zipExtractor{}
	case "tar":
		sub = &tarExtractor{compressed: false}
	case "tar.gz", "tgz":
		sub = &tarExtractor{compressed: true}
	default:
		return errors.New("unsupported archive format")
	}

	// 把预读的数据塞回去，再交给子提取器
	full := io.MultiReader(bytes.NewReader(peek), src)
	return sub.Extract(full, dst)
}

// ------------------------------------------------------------------
// 格式嗅探
// ------------------------------------------------------------------

func detectFormat(header []byte) string {
	if len(header) < 4 {
		return ""
	}
	// zip 签名
	if bytes.HasPrefix(header, []byte("PK\x03\x04")) {
		return "zip"
	}
	// gzip 签名
	if bytes.HasPrefix(header, []byte{0x1f, 0x8b}) {
		return "tar.gz"
	}
	// ustar 或 老 tar
	if bytes.HasPrefix(header[257:], []byte("ustar")) || header[0] == 0 {
		return "tar"
	}
	return ""
}

// ------------------------------------------------------------------
// 工具
// ------------------------------------------------------------------

// 预读 n 字节
func peekReader(r io.Reader, n int) ([]byte, error) {
	return bufio.NewReader(r).Peek(n)
}

// 安全路径校验：保证 path 必须位于 root 之内
func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// ------------------------------------------------------------------
// zip 专用实现
// ------------------------------------------------------------------

type zipExtractor struct{}

func (z *zipExtractor) Extract(src io.Reader, dst string) error {
	// zip.NewReader 需要 io.ReaderAt + size
	// 先写入临时文件，再打开
	tmp, err := writeToTemp(src)
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	stat, _ := tmp.Stat()
	r, err := zip.NewReader(tmp, stat.Size())
	if err != nil {
		return fmt.Errorf("zip reader: %w", err)
	}

	for _, f := range r.File {
		if err := z.extractFile(f, dst); err != nil {
			return fmt.Errorf("extract %s: %w", f.Name, err)
		}
	}
	return nil
}

func (z *zipExtractor) extractFile(f *zip.File, dst string) error {
	path := filepath.Join(dst, filepath.FromSlash(f.Name))
	if !inside(dst, path) {
		return fmt.Errorf("illegal path: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, f.Mode())
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	return copyFile(rc, path, f.Mode())
}

// ------------------------------------------------------------------
// tar / tar.gz 专用实现
// ------------------------------------------------------------------

type tarExtractor struct {
	compressed bool
}

func (t *tarExtractor) Extract(src io.Reader, dst string) error {
	if t.compressed {
		gzr, err := gzip.NewReader(src)
		if err != nil {
			return fmt.Errorf("gzip: %w", err)
		}
		defer gzr.Close()
		src = gzr
	}

	tr := tar.NewReader(src)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}
		if err := t.extractEntry(tr, hdr, dst); err != nil {
			return fmt.Errorf("tar entry %s: %w", hdr.Name, err)
		}
	}
	return nil
}

func (t *tarExtractor) extractEntry(tr *tar.Reader, hdr *tar.Header, dst string) error {
	path := filepath.Join(dst, filepath.FromSlash(hdr.Name))
	if !inside(dst, path) {
		return fmt.Errorf("illegal path: %s", hdr.Name)
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(path, hdr.FileInfo().Mode())
	case tar.TypeReg:
		return copyFile(tr, path, hdr.FileInfo().Mode())
	case tar.TypeSymlink:
		target := filepath.FromSlash(hdr.Linkname)
		absTarget := filepath.Join(filepath.Dir(path), target)
		if !inside(dst, absTarget) {
			return fmt.Errorf("illegal symlink target: %s", target)
		}
		return os.Symlink(target, path)
	default:
		return nil
	}
}

// ------------------------------------------------------------------
// 通用文件写出
// ------------------------------------------------------------------

func copyFile(r io.Reader, path string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func writeToTemp(r io.Reader) (*os.File, error) {
	tmp, err := os.CreateTemp("", "extract-*.tmp")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(tmp, r)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return tmp, nil
}
