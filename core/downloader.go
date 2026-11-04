package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ProgressFunc receives the total number of bytes currently stored and the
// expected total size. expected is -1 when the server does not provide it.
type ProgressFunc func(downloaded, expected int64)

// Downloader downloads files using only the Go standard library.
type Downloader struct {
	Client *http.Client
}

func NewDownloader() *Downloader {
	return &Downloader{Client: &http.Client{Timeout: 0}}
}

// Download writes to destination + ".part" and renames it on success. It
// resumes an existing partial file only when the server honors HTTP Range;
// if Range is ignored, the response is safely treated as a fresh download.
func (d *Downloader) Download(ctx context.Context, rawURL, destination string, progress ProgressFunc) error {
	if d.Client == nil {
		d.Client = http.DefaultClient
	}
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("destination already exists: %s", destination)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check destination: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}
	partPath := destination + ".part"
	metaPath := partPath + ".url"

	var offset int64
	if info, err := os.Stat(partPath); err == nil {
		if info.IsDir() {
			return fmt.Errorf("partial download path is a directory: %s", partPath)
		}
		if savedURL, readErr := os.ReadFile(metaPath); readErr == nil && string(savedURL) == rawURL {
			offset = info.Size()
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check partial file: %w", err)
	}
	// The sidecar prevents accidentally appending bytes from a different URL.
	if err := os.WriteFile(metaPath, []byte(rawURL), 0600); err != nil {
		return fmt.Errorf("save resume metadata: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept-Encoding", "identity")
	if offset > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(offset, 10)+"-")
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request download: %w", err)
	}
	defer resp.Body.Close()

	appendMode := false
	switch resp.StatusCode {
	case http.StatusOK:
		// A 200 response means the server ignored Range; restart from byte 0.
		offset = 0
	case http.StatusPartialContent:
		start, ok := contentRangeStart(resp.Header.Get("Content-Range"))
		if !ok || start != offset {
			return fmt.Errorf("server returned an invalid Content-Range for offset %d", offset)
		}
		appendMode = offset > 0
	default:
		return fmt.Errorf("download request failed: %s", resp.Status)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(partPath, flags, 0600)
	if err != nil {
		return fmt.Errorf("open partial file: %w", err)
	}
	defer f.Close()

	expected := int64(-1)
	if resp.ContentLength >= 0 {
		expected = offset + resp.ContentLength
	}
	if progress != nil {
		progress(offset, expected)
	}
	lastUpdate := time.Now()
	var copied int64
	written, copyErr := io.Copy(progressWriter{dst: f, onWrite: func(n int64) {
		copied += n
		if progress != nil && time.Since(lastUpdate) >= 150*time.Millisecond {
			progress(offset+copied, expected)
			lastUpdate = time.Now()
		}
	}}, resp.Body)
	if syncErr := f.Sync(); copyErr == nil && syncErr != nil {
		copyErr = syncErr
	}
	if closeErr := f.Close(); copyErr == nil && closeErr != nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return fmt.Errorf("write download: %w", copyErr)
	}
	if progress != nil {
		progress(offset+written, expected)
	}
	if err := os.Rename(partPath, destination); err != nil {
		return fmt.Errorf("finalize download: %w", err)
	}
	_ = os.Remove(metaPath)
	return nil
}

type progressWriter struct {
	dst     io.Writer
	onWrite func(int64)
}

func (w progressWriter) Write(p []byte) (int, error) {
	n, err := w.dst.Write(p)
	if n > 0 && w.onWrite != nil {
		w.onWrite(int64(n))
	}
	return n, err
}

func contentRangeStart(value string) (int64, bool) {
	if !strings.HasPrefix(value, "bytes ") {
		return 0, false
	}
	bounds := strings.SplitN(strings.TrimPrefix(value, "bytes "), "-", 2)
	if len(bounds) != 2 {
		return 0, false
	}
	start, err := strconv.ParseInt(bounds[0], 10, 64)
	return start, err == nil
}
