package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client interface {
	Download(ctx context.Context, url string) (io.ReadCloser, error)
}

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 5 * time.Minute,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func NewHTTPClientWithTimeout(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func (c *HTTPClient) Download(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	req.Header.Set("User-Agent", "ghinstall/1.0")
	req.Header.Set("Accept", "*/*")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d for %s", resp.StatusCode, url)
	}

	return &responseWrapper{
		ReadCloser: resp.Body,
		url:        url,
	}, nil
}

type responseWrapper struct {
	io.ReadCloser
	url string
}

func (w *responseWrapper) Close() error {
	return w.ReadCloser.Close()
}

func (w *responseWrapper) Read(p []byte) (n int, err error) {
	n, err = w.ReadCloser.Read(p)
	if err != nil && err != io.EOF {
		err = fmt.Errorf("failed to read from %s: %w", w.url, err)
	}
	return n, err
}