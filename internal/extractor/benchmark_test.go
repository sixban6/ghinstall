package extractor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Create a sample tar.gz for testing
func createSampleTarGz(t testing.TB) []byte {
	t.Helper()
	
	// This is a minimal valid tar.gz file (empty archive)
	// In real benchmarks, you'd use actual files
	tarGzData := []byte{
		0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
		0xed, 0xc1, 0x01, 0x0d, 0x00, 0x00, 0x00, 0xc2, 0xa0, 0xf7,
		0x4f, 0x6d, 0x0e, 0x37, 0xa0, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	return tarGzData
}

func BenchmarkExtractors(b *testing.B) {
	sampleData := createSampleTarGz(b)
	
	extractors := map[string]Extractor{
		"Legacy":           NewLegacy(),
		"Optimized":        NewOptimized(),
		"System":           NewSystem(),
		"SystemFallback":   NewSystemWithFallback(),
	}
	
	for name, extractor := range extractors {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				
				// Create temp directory
				tempDir, err := os.MkdirTemp("", "benchmark-*")
				if err != nil {
					b.Fatal(err)
				}
				defer os.RemoveAll(tempDir)
				
				// Create reader
				reader := bytes.NewReader(sampleData)
				
				b.StartTimer()
				
				// Extract
				err = extractor.Extract(reader, tempDir)
				if err != nil {
					b.Errorf("Extraction failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkLargeFileExtraction(b *testing.B) {
	// Skip if no large test files available
	testFile := os.Getenv("LARGE_TEST_FILE") 
	if testFile == "" {
		b.Skip("Set LARGE_TEST_FILE environment variable to run large file benchmarks")
	}
	
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		b.Skipf("Test file %s not found", testFile)
	}
	
	extractors := map[string]Extractor{
		"Optimized":      NewOptimized(),
		"System":         NewSystem(),
		"SystemFallback": NewSystemWithFallback(),
	}
	
	for name, extractor := range extractors {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				
				// Create temp directory
				tempDir, err := os.MkdirTemp("", "benchmark-large-*")
				if err != nil {
					b.Fatal(err)
				}
				defer os.RemoveAll(tempDir)
				
				// Open test file
				file, err := os.Open(testFile)
				if err != nil {
					b.Fatal(err)
				}
				defer file.Close()
				
				b.StartTimer()
				
				// Extract
				err = extractor.Extract(file, tempDir)
				if err != nil {
					b.Errorf("Extraction failed: %v", err)
				}
				
				// Seek back for next iteration
				file.Seek(0, 0)
			}
		})
	}
}

func BenchmarkMemoryUsage(b *testing.B) {
	sampleData := createSampleTarGz(b)
	
	extractors := map[string]Extractor{
		"Legacy":     NewLegacy(),
		"Optimized":  NewOptimized(),
		"System":     NewSystem(),
	}
	
	for name, extractor := range extractors {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				tempDir, err := os.MkdirTemp("", "benchmark-mem-*")
				if err != nil {
					b.Fatal(err)
				}
				defer os.RemoveAll(tempDir)
				
				reader := bytes.NewReader(sampleData)
				
				err = extractor.Extract(reader, tempDir)
				if err != nil {
					b.Errorf("Extraction failed: %v", err)
				}
			}
		})
	}
}

// Benchmark concurrent extractions
func BenchmarkConcurrentExtraction(b *testing.B) {
	sampleData := createSampleTarGz(b)
	extractor := NewSystemWithFallback()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tempDir, err := os.MkdirTemp("", "benchmark-concurrent-*")
			if err != nil {
				b.Error(err)
				continue
			}
			defer os.RemoveAll(tempDir)
			
			reader := bytes.NewReader(sampleData)
			
			err = extractor.Extract(reader, tempDir)
			if err != nil {
				b.Errorf("Concurrent extraction failed: %v", err)
			}
		}
	})
}

// Helper function to create actual test files for more realistic benchmarks
func createRealisticTarGz(t testing.TB, numFiles, fileSize int) []byte {
	t.Helper()
	
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "create-tar-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files
	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		content := bytes.Repeat([]byte("test data "), fileSize/10)
		
		if err := os.WriteFile(filename, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	
	// Create tar.gz from temp directory
	var buf bytes.Buffer
	
	// Use system tar if available for creating test data
	cmd := exec.Command("tar", "-czf", "-", "-C", tempDir, ".")
	cmd.Stdout = &buf
	
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create test tar.gz: %v", err)
	}
	
	return buf.Bytes()
}

func BenchmarkRealisticWorkload(b *testing.B) {
	// Create a more realistic tar.gz with multiple files
	testData := createRealisticTarGz(b, 10, 1024) // 10 files, ~1KB each
	
	extractors := map[string]Extractor{
		"Optimized": NewOptimized(),
		"System":    NewSystem(),
	}
	
	for name, extractor := range extractors {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				
				tempDir, err := os.MkdirTemp("", "benchmark-realistic-*")
				if err != nil {
					b.Fatal(err)
				}
				defer os.RemoveAll(tempDir)
				
				reader := bytes.NewReader(testData)
				
				b.StartTimer()
				
				err = extractor.Extract(reader, tempDir)
				if err != nil {
					b.Errorf("Extraction failed: %v", err)
				}
			}
		})
	}
}