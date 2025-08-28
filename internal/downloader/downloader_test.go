package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPClient_Download(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		wantContent    string
		wantErr        bool
	}{
		{
			name:           "successful download",
			serverResponse: "test content",
			serverStatus:   http.StatusOK,
			wantContent:    "test content",
			wantErr:        false,
		},
		{
			name:           "not found",
			serverResponse: "Not Found",
			serverStatus:   http.StatusNotFound,
			wantContent:    "",
			wantErr:        true,
		},
		{
			name:           "server error",
			serverResponse: "Internal Server Error",
			serverStatus:   http.StatusInternalServerError,
			wantContent:    "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("User-Agent") != "ghinstall/1.0" {
					t.Errorf("Expected User-Agent header 'ghinstall/1.0', got %q", r.Header.Get("User-Agent"))
				}
				
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := NewHTTPClient()
			reader, err := client.Download(context.Background(), server.URL)

			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPClient.Download() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			defer reader.Close()
			content, err := io.ReadAll(reader)
			if err != nil {
				t.Errorf("Failed to read response: %v", err)
				return
			}

			if string(content) != tt.wantContent {
				t.Errorf("HTTPClient.Download() content = %q, want %q", string(content), tt.wantContent)
			}
		})
	}
}

func TestHTTPClient_Download_WithRedirect(t *testing.T) {
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("redirected content"))
	}))
	defer redirectServer.Close()

	mainServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectServer.URL, http.StatusFound)
	}))
	defer mainServer.Close()

	client := NewHTTPClient()
	reader, err := client.Download(context.Background(), mainServer.URL)
	if err != nil {
		t.Errorf("HTTPClient.Download() with redirect failed: %v", err)
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Failed to read redirected response: %v", err)
		return
	}

	if string(content) != "redirected content" {
		t.Errorf("HTTPClient.Download() redirected content = %q, want %q", string(content), "redirected content")
	}
}

func TestHTTPClient_Download_TooManyRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
	}))
	defer server.Close()

	client := NewHTTPClient()
	_, err := client.Download(context.Background(), server.URL)
	if err == nil {
		t.Error("HTTPClient.Download() should fail with too many redirects")
		return
	}

	if !strings.Contains(err.Error(), "too many redirects") {
		t.Errorf("Expected 'too many redirects' error, got: %v", err)
	}
}

func TestHTTPClient_Download_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("delayed content"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Download(ctx, server.URL)
	if err == nil {
		t.Error("HTTPClient.Download() should fail with context deadline exceeded")
		return
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

func TestHTTPClient_Download_InvalidURL(t *testing.T) {
	client := NewHTTPClient()
	_, err := client.Download(context.Background(), "not a url")
	if err == nil {
		t.Error("HTTPClient.Download() should fail with invalid URL")
	}
}

func TestNewHTTPClientWithTimeout(t *testing.T) {
	timeout := 10 * time.Second
	client := NewHTTPClientWithTimeout(timeout)
	
	if client.client.Timeout != timeout {
		t.Errorf("NewHTTPClientWithTimeout() timeout = %v, want %v", client.client.Timeout, timeout)
	}
}

func TestResponseWrapper_Read_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("short"))
	}))
	defer server.Close()

	client := NewHTTPClient()
	reader, err := client.Download(context.Background(), server.URL)
	if err != nil {
		t.Errorf("HTTPClient.Download() failed: %v", err)
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		if !strings.Contains(err.Error(), server.URL) {
			t.Errorf("Error should contain URL, got: %v", err)
		}
	}

	if string(content) != "short" {
		t.Errorf("Got content %q, want %q", string(content), "short")
	}
}

func TestHTTPClient_Download_LargeFile(t *testing.T) {
	largeContent := strings.Repeat("a", 1024*1024)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(largeContent)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeContent))
	}))
	defer server.Close()

	client := NewHTTPClient()
	reader, err := client.Download(context.Background(), server.URL)
	if err != nil {
		t.Errorf("HTTPClient.Download() failed: %v", err)
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Failed to read large response: %v", err)
		return
	}

	if len(content) != len(largeContent) {
		t.Errorf("HTTPClient.Download() content length = %d, want %d", len(content), len(largeContent))
	}
}