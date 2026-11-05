package protocols

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

// PyPI resolves pypi:<project> or pypi:<project>@<version> through the
// official Python Package Index. It downloads a distribution archive only.
type PyPI struct {
	Client  *http.Client
	APIBase string // useful for compatible indexes and tests
}

func (PyPI) Scheme() string { return "pypi" }

func (p PyPI) Resolve(ctx context.Context, query string) ([]Download, error) {
	name, version, err := parsePackageSpec(strings.TrimSpace(query))
	if err != nil {
		return nil, fmt.Errorf("usage: pypi:<project> or pypi:<project>@<version>")
	}
	if strings.Contains(name, "/") {
		return nil, fmt.Errorf("PyPI project names cannot contain slashes")
	}
	project := normalizePyPIName(name)
	apiBase := strings.TrimRight(p.APIBase, "/")
	if apiBase == "" {
		apiBase = "https://pypi.org/pypi"
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	endpoint := apiBase + "/" + url.PathEscape(project) + "/json"
	if version != "latest" {
		endpoint = apiBase + "/" + url.PathEscape(project) + "/" + url.PathEscape(version) + "/json"
	}
	var metadata struct {
		Info struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"info"`
		URLs []struct {
			Filename    string `json:"filename"`
			URL         string `json:"url"`
			PackageType string `json:"packagetype"`
			Size        int64  `json:"size"`
			Python      string `json:"python_version"`
			Yanked      bool   `json:"yanked"`
		} `json:"urls"`
	}
	if err := getJSON(ctx, client, endpoint, http.Header{"User-Agent": {"ufetch"}}, &metadata); err != nil {
		return nil, fmt.Errorf("get PyPI project %s@%s: %w", name, version, err)
	}

	resolvedName := metadata.Info.Name
	if resolvedName == "" {
		resolvedName = name
	}
	resolvedVersion := metadata.Info.Version
	if resolvedVersion == "" {
		resolvedVersion = version
	}
	var downloads []Download
	for _, file := range metadata.URLs {
		if file.Yanked || file.Filename == "" || file.URL == "" {
			continue
		}
		if err := requireHTTPSURL(file.URL); err != nil {
			continue
		}
		description := fmt.Sprintf("PyPI %s %s · %s", resolvedName, resolvedVersion, file.PackageType)
		if file.Python != "" {
			description += " · Python " + file.Python
		}
		size := file.Size
		if size <= 0 {
			size = -1
		}
		downloads = append(downloads, Download{
			URL:         file.URL,
			Filename:    path.Base(file.Filename),
			Description: description,
			Size:        size,
		})
	}
	if len(downloads) == 0 {
		return nil, fmt.Errorf("PyPI project %s@%s has no downloadable, non-yanked HTTPS files", resolvedName, resolvedVersion)
	}
	sort.Slice(downloads, func(i, j int) bool {
		return downloads[i].Filename < downloads[j].Filename
	})
	return downloads, nil
}

func normalizePyPIName(name string) string {
	name = strings.ToLower(name)
	return strings.NewReplacer("_", "-", ".", "-").Replace(name)
}

func requireHTTPSURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("expected an absolute HTTPS URL")
	}
	return nil
}
