package protocols

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"
)

// GitHub resolves app:owner/repo or app:search-term to compatible assets from
// the repository's latest stable GitHub Release.
type GitHub struct {
	Client  *http.Client
	APIBase string // useful for self-hosted API-compatible servers and tests
}

func (GitHub) Scheme() string { return "app" }

func (g GitHub) Resolve(ctx context.Context, query string) ([]Download, error) {
	query = strings.TrimSpace(strings.Trim(query, "/"))
	if query == "" {
		return nil, fmt.Errorf("usage: app:<owner/repository or search term>")
	}

	base := strings.TrimRight(g.APIBase, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	headers := http.Header{
		"Accept":     {"application/vnd.github+json"},
		"User-Agent": {"ufetch"},
	}

	repository := query
	if !strings.Contains(query, "/") {
		searchURL := base + "/search/repositories?" + url.Values{
			"q":        {query},
			"sort":     {"stars"},
			"order":    {"desc"},
			"per_page": {"1"},
		}.Encode()
		var search struct {
			Items []struct {
				FullName string `json:"full_name"`
			} `json:"items"`
		}
		if err := getJSON(ctx, client, searchURL, headers, &search); err != nil {
			return nil, fmt.Errorf("search GitHub repositories: %w", err)
		}
		if len(search.Items) == 0 {
			return nil, fmt.Errorf("no GitHub repository found for %q; use app:owner/repository for an exact repository", query)
		}
		repository = search.Items[0].FullName
	}

	parts := strings.Split(repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("GitHub repository must be written as owner/repository")
	}
	endpoint := base + "/repos/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/releases/latest"
	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Size               int64  `json:"size"`
		} `json:"assets"`
	}
	if err := getJSON(ctx, client, endpoint, headers, &release); err != nil {
		return nil, fmt.Errorf("get latest release for %s: %w", repository, err)
	}

	var downloads []Download
	for _, asset := range release.Assets {
		if asset.Name == "" || asset.BrowserDownloadURL == "" || !compatibleAsset(asset.Name, runtime.GOOS, runtime.GOARCH) {
			continue
		}
		description := asset.Name + " · " + repository
		size := asset.Size
		if size <= 0 {
			size = -1
		}
		if release.TagName != "" {
			description += " " + release.TagName
		}
		downloads = append(downloads, Download{
			URL:         asset.BrowserDownloadURL,
			Filename:    asset.Name,
			Description: description,
			Size:        size,
		})
	}
	if len(downloads) == 0 {
		return nil, fmt.Errorf("latest release for %s has no assets compatible with %s/%s", repository, runtime.GOOS, runtime.GOARCH)
	}
	return downloads, nil
}

func compatibleAsset(name, goos, goarch string) bool {
	n := strings.ToLower(name)
	for _, marker := range []string{"source code", "checksums", "checksum", ".sha256", ".sha512", ".sig", ".asc", "debug symbols"} {
		if strings.Contains(n, marker) {
			return false
		}
	}

	if strings.Contains(n, "android") || strings.Contains(n, "freebsd") || strings.Contains(n, "openbsd") || strings.Contains(n, "netbsd") || strings.HasSuffix(n, ".apk") {
		return false
	}

	assetOS := ""
	switch {
	case strings.Contains(n, "windows") || strings.Contains(n, "win64") || strings.Contains(n, "win32") || strings.Contains(n, "win-") || strings.Contains(n, "win_") || strings.HasSuffix(n, ".exe") || strings.HasSuffix(n, ".msi"):
		assetOS = "windows"
	case strings.Contains(n, "darwin") || strings.Contains(n, "macos") || strings.Contains(n, "mac os") || strings.Contains(n, "osx"):
		assetOS = "darwin"
	case strings.Contains(n, "linux"):
		assetOS = "linux"
	}
	if assetOS != "" && assetOS != goos {
		return false
	}

	assetArch := ""
	switch {
	case strings.Contains(n, "aarch64") || strings.Contains(n, "arm64"):
		assetArch = "arm64"
	case strings.Contains(n, "x86_64") || strings.Contains(n, "amd64") || strings.Contains(n, "x64") || strings.Contains(n, "win64"):
		assetArch = "amd64"
	case strings.Contains(n, "i386") || strings.Contains(n, "i686") || strings.Contains(n, "386") || strings.Contains(n, "x86") || strings.Contains(n, "win32"):
		assetArch = "386"
	case strings.Contains(n, "armv7") || strings.Contains(n, "armhf"):
		assetArch = "arm"
	}
	if assetArch != "" && assetArch != goarch {
		return false
	}
	return true
}
