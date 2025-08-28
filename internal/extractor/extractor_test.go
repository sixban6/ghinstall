package extractor

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    string
		wantErr bool
	}{
		{
			name: "gzip format",
			data: []byte{0x1f, 0x8b, 0x08, 0x00},
			want: "tar.gz",
		},
		{
			name: "zip format",
			data: []byte{'P', 'K', 0x03, 0x04},
			want: "zip",
		},
		{
			name:    "unknown format",
			data:    []byte{0x00, 0x01, 0x02, 0x03},
			wantErr: true,
		},
		{
			name:    "too small",
			data:    []byte{0x1f},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectFormat(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("detectFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("detectFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiExtractor_Extract_TarGz(t *testing.T) {
	tmpDir := t.TempDir()
	extractor := New()

	tarGzData := createTestTarGz(t)
	reader := strings.NewReader(string(tarGzData))

	err := extractor.Extract(reader, tmpDir)
	if err != nil {
		t.Errorf("MultiExtractor.Extract() error = %v", err)
		return
	}

	testFiles := []struct {
		path    string
		content string
	}{
		{"test.txt", "Hello, World!"},
		{"subdir/nested.txt", "Nested file content"},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", tf.path, err)
			continue
		}
		if string(content) != tf.content {
			t.Errorf("File content mismatch for %s: got %q, want %q", tf.path, string(content), tf.content)
		}
	}
}

func TestMultiExtractor_Extract_Zip(t *testing.T) {
	tmpDir := t.TempDir()
	extractor := New()

	zipData := createTestZip(t)
	reader := bytes.NewReader(zipData)

	err := extractor.Extract(reader, tmpDir)
	if err != nil {
		t.Errorf("MultiExtractor.Extract() error = %v", err)
		return
	}

	testFiles := []struct {
		path    string
		content string
	}{
		{"test.txt", "Hello, World!"},
		{"subdir/nested.txt", "Nested file content"},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", tf.path, err)
			continue
		}
		if string(content) != tf.content {
			t.Errorf("File content mismatch for %s: got %q, want %q", tf.path, string(content), tf.content)
		}
	}
}

func TestMultiExtractor_Extract_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	extractor := New()

	unsupportedData := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	reader := bytes.NewReader(unsupportedData)

	err := extractor.Extract(reader, tmpDir)
	if err == nil {
		t.Error("MultiExtractor.Extract() should fail with unsupported format")
	}
}

func TestMultiExtractor_Extract_DirectoryTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	extractor := New()

	maliciousTarGz := createMaliciousTarGz(t)
	reader := strings.NewReader(string(maliciousTarGz))

	err := extractor.Extract(reader, tmpDir)
	if err == nil || !strings.Contains(err.Error(), "invalid file path") {
		t.Errorf("MultiExtractor.Extract() should fail with directory traversal attack, got: %v", err)
	}
}

func TestMultiExtractor_Extract_CurrentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	extractor := New()

	tarGzWithCurrentDir := createTarGzWithCurrentDir(t)
	reader := strings.NewReader(string(tarGzWithCurrentDir))

	err := extractor.Extract(reader, tmpDir)
	if err != nil {
		t.Errorf("MultiExtractor.Extract() should handle current directory entries, got: %v", err)
		return
	}

	// Verify that actual files were extracted
	testFile := filepath.Join(tmpDir, "test.txt")
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read extracted file: %v", err)
		return
	}
	if string(content) != "Hello, World!" {
		t.Errorf("File content mismatch: got %q, want %q", string(content), "Hello, World!")
	}
}

func TestTarGzExtractor_Extract(t *testing.T) {
	tmpDir := t.TempDir()
	extractor := NewTarGzExtractor()

	tarGzData := createTestTarGz(t)
	reader := strings.NewReader(string(tarGzData))

	err := extractor.Extract(reader, tmpDir)
	if err != nil {
		t.Errorf("TarGzExtractor.Extract() error = %v", err)
		return
	}

	fullPath := filepath.Join(tmpDir, "test.txt")
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Errorf("Failed to read extracted file: %v", err)
		return
	}
	if string(content) != "Hello, World!" {
		t.Errorf("File content mismatch: got %q, want %q", string(content), "Hello, World!")
	}
}

func TestZipExtractor_Extract(t *testing.T) {
	tmpDir := t.TempDir()
	extractor := NewZipExtractor()

	zipData := createTestZip(t)
	reader := bytes.NewReader(zipData)

	err := extractor.Extract(reader, tmpDir)
	if err != nil {
		t.Errorf("ZipExtractor.Extract() error = %v", err)
		return
	}

	fullPath := filepath.Join(tmpDir, "test.txt")
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Errorf("Failed to read extracted file: %v", err)
		return
	}
	if string(content) != "Hello, World!" {
		t.Errorf("File content mismatch: got %q, want %q", string(content), "Hello, World!")
	}
}

func createTestTarGz(t *testing.T) []byte {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	files := []struct {
		name    string
		content string
		mode    int64
	}{
		{"test.txt", "Hello, World!", 0644},
		{"subdir/", "", 0755},
		{"subdir/nested.txt", "Nested file content", 0644},
	}

	for _, file := range files {
		if strings.HasSuffix(file.name, "/") {
			header := &tar.Header{
				Name:     file.name,
				Mode:     file.mode,
				Typeflag: tar.TypeDir,
			}
			if err := tarWriter.WriteHeader(header); err != nil {
				t.Fatalf("Failed to write tar header: %v", err)
			}
		} else {
			header := &tar.Header{
				Name: file.name,
				Mode: file.mode,
				Size: int64(len(file.content)),
			}
			if err := tarWriter.WriteHeader(header); err != nil {
				t.Fatalf("Failed to write tar header: %v", err)
			}
			if _, err := tarWriter.Write([]byte(file.content)); err != nil {
				t.Fatalf("Failed to write tar content: %v", err)
			}
		}
	}

	tarWriter.Close()
	gzWriter.Close()

	return buf.Bytes()
}

func createTestZip(t *testing.T) []byte {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	files := []struct {
		name    string
		content string
	}{
		{"test.txt", "Hello, World!"},
		{"subdir/nested.txt", "Nested file content"},
	}

	for _, file := range files {
		writer, err := zipWriter.Create(file.name)
		if err != nil {
			t.Fatalf("Failed to create zip entry: %v", err)
		}
		if _, err := writer.Write([]byte(file.content)); err != nil {
			t.Fatalf("Failed to write zip content: %v", err)
		}
	}

	zipWriter.Close()
	return buf.Bytes()
}

func createMaliciousTarGz(t *testing.T) []byte {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	header := &tar.Header{
		Name: "../../../etc/passwd",
		Mode: 0644,
		Size: 12,
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write malicious tar header: %v", err)
	}
	if _, err := tarWriter.Write([]byte("malicious")); err != nil {
		t.Fatalf("Failed to write malicious tar content: %v", err)
	}

	tarWriter.Close()
	gzWriter.Close()

	return buf.Bytes()
}

func createTarGzWithCurrentDir(t *testing.T) []byte {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	entries := []struct {
		name    string
		content string
		mode    int64
		isDir   bool
	}{
		{"./", "", 0755, true},
		{"test.txt", "Hello, World!", 0644, false},
		{"subdir/", "", 0755, true},
		{"subdir/nested.txt", "Nested file content", 0644, false},
	}

	for _, entry := range entries {
		if entry.isDir {
			header := &tar.Header{
				Name:     entry.name,
				Mode:     entry.mode,
				Typeflag: tar.TypeDir,
			}
			if err := tarWriter.WriteHeader(header); err != nil {
				t.Fatalf("Failed to write tar header for %s: %v", entry.name, err)
			}
		} else {
			header := &tar.Header{
				Name: entry.name,
				Mode: entry.mode,
				Size: int64(len(entry.content)),
			}
			if err := tarWriter.WriteHeader(header); err != nil {
				t.Fatalf("Failed to write tar header for %s: %v", entry.name, err)
			}
			if _, err := tarWriter.Write([]byte(entry.content)); err != nil {
				t.Fatalf("Failed to write tar content for %s: %v", entry.name, err)
			}
		}
	}

	tarWriter.Close()
	gzWriter.Close()

	return buf.Bytes()
}