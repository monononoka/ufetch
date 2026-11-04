package protocols

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
)

// HTTP resolves direct HTTP and HTTPS URLs. It does not perform network
// requests; the downloader owns all transfer behavior.
type HTTP struct{}

func (HTTP) Scheme() string { return "http" }

func (HTTP) Resolve(_ context.Context, raw string) ([]Download, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if (scheme != "http" && scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("expected an absolute http:// or https:// URL")
	}

	name := path.Base(u.Path)
	if name == "." || name == "/" || name == "" || strings.TrimSpace(name) == "" {
		name = "download.bin"
	}
	return []Download{{URL: u.String(), Filename: name, Description: "Direct HTTP(S) download", Size: -1}}, nil
}
