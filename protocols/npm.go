package protocols

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// NPM resolves npm:<package> or npm:<package>@<version> through the public
// npm registry. It downloads the package archive; it does not install it.
type NPM struct {
	Client   *http.Client
	Registry string // useful for compatible registries and tests
}

func (NPM) Scheme() string { return "npm" }

func (n NPM) Resolve(ctx context.Context, query string) ([]Download, error) {
	name, version, err := parsePackageSpec(strings.TrimSpace(query))
	if err != nil {
		return nil, err
	}
	registry := strings.TrimRight(n.Registry, "/")
	if registry == "" {
		registry = "https://registry.npmjs.org"
	}
	client := n.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	endpoint := registry + "/" + url.PathEscape(name) + "/" + url.PathEscape(version)
	var metadata struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Dist    struct {
			Tarball string `json:"tarball"`
		} `json:"dist"`
	}
	if err := getJSON(ctx, client, endpoint, http.Header{"User-Agent": {"ufetch"}}, &metadata); err != nil {
		return nil, fmt.Errorf("get npm package %s@%s: %w", name, version, err)
	}
	if err := requireHTTPSURL(metadata.Dist.Tarball); err != nil {
		return nil, fmt.Errorf("npm registry returned an invalid package URL: %w", err)
	}

	u, _ := url.Parse(metadata.Dist.Tarball)
	filename := path.Base(u.Path)
	if filename == "." || filename == "/" || filename == "" {
		return nil, fmt.Errorf("npm registry did not provide a package archive filename")
	}
	packageName := metadata.Name
	if packageName == "" {
		packageName = name
	}
	resolvedVersion := metadata.Version
	if resolvedVersion == "" {
		resolvedVersion = version
	}
	return []Download{{
		URL:         metadata.Dist.Tarball,
		Filename:    filename,
		Description: "npm " + packageName + "@" + resolvedVersion,
		Size:        -1,
	}}, nil
}

func parsePackageSpec(spec string) (name, version string, err error) {
	if spec == "" || strings.ContainsAny(spec, " \t\r\n") {
		return "", "", fmt.Errorf("usage: npm:<package> or npm:<package>@<version>")
	}
	version = "latest"
	separator := strings.LastIndex(spec, "@")
	if strings.HasPrefix(spec, "@") {
		if slash := strings.Index(spec, "/"); slash < 2 {
			return "", "", fmt.Errorf("scoped npm package must be written as @scope/name")
		} else if separator > slash {
			name, version = spec[:separator], spec[separator+1:]
		} else {
			name = spec
		}
	} else if separator > 0 {
		name, version = spec[:separator], spec[separator+1:]
	} else {
		name = spec
	}
	if name == "" || version == "" || strings.HasSuffix(name, "/") {
		return "", "", fmt.Errorf("usage: npm:<package> or npm:<package>@<version>")
	}
	if strings.Count(name, "/") > 0 && (!strings.HasPrefix(name, "@") || strings.Count(name, "/") != 1) {
		return "", "", fmt.Errorf("scoped npm package must be written as @scope/name")
	}
	return name, version, nil
}
