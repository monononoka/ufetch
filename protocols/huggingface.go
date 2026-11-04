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

// HuggingFace resolves hf:<search-term>, hf:<owner/model>, or
// hf:<owner/model/path/to/file> to files in a public model repository.
type HuggingFace struct {
	Client  *http.Client
	APIBase string // useful for tests or compatible API endpoints
	WebBase string // base URL used for resolved file downloads
}

func (HuggingFace) Scheme() string { return "hf" }

func (h HuggingFace) Resolve(ctx context.Context, query string) ([]Download, error) {
	query = strings.Trim(strings.TrimSpace(query), "/")
	if query == "" {
		return nil, fmt.Errorf("usage: hf:<model search, owner/model, or owner/model/path/to/file>")
	}

	apiBase := strings.TrimRight(h.APIBase, "/")
	if apiBase == "" {
		apiBase = "https://huggingface.co/api"
	}
	webBase := strings.TrimRight(h.WebBase, "/")
	if webBase == "" {
		webBase = "https://huggingface.co"
	}
	client := h.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	headers := http.Header{"User-Agent": {"ufetch"}}

	parts := strings.Split(query, "/")
	var repository, requestedFile string
	if len(parts) >= 2 {
		repository = parts[0] + "/" + parts[1]
		if len(parts) > 2 {
			requestedFile = strings.Join(parts[2:], "/")
		}
	} else {
		searchURL := apiBase + "/models?" + url.Values{
			"search":    {query},
			"limit":     {"10"},
			"sort":      {"downloads"},
			"direction": {"-1"},
		}.Encode()
		var matches []struct {
			ID        string `json:"id"`
			Downloads int64  `json:"downloads"`
		}
		if err := getJSON(ctx, client, searchURL, headers, &matches); err != nil {
			return nil, fmt.Errorf("search Hugging Face models: %w", err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no Hugging Face model found for %q", query)
		}
		repository = matches[0].ID
	}

	modelURL := apiBase + "/models/" + escapePath(repository) + "?blobs=true"
	var model struct {
		ID       string `json:"id"`
		SHA      string `json:"sha"`
		Siblings []struct {
			Filename string `json:"rfilename"`
			Size     int64  `json:"size"`
		} `json:"siblings"`
	}
	if err := getJSON(ctx, client, modelURL, headers, &model); err != nil {
		return nil, fmt.Errorf("get files for Hugging Face model %s: %w", repository, err)
	}
	if model.ID != "" {
		repository = model.ID
	}
	revision := model.SHA
	if revision == "" {
		revision = "main"
	}

	var downloads []Download
	for _, file := range model.Siblings {
		if file.Filename == "" {
			continue
		}
		if requestedFile != "" && file.Filename != requestedFile {
			continue
		}
		if requestedFile == "" && !isModelWeight(file.Filename) {
			continue
		}
		size := file.Size
		if size <= 0 {
			size = -1
		}
		downloads = append(downloads, Download{
			URL:         webBase + "/" + escapePath(repository) + "/resolve/" + url.PathEscape(revision) + "/" + escapePath(file.Filename) + "?download=true",
			Filename:    path.Base(file.Filename),
			Description: repository + " · " + file.Filename,
			Size:        size,
		})
	}
	if requestedFile != "" && len(downloads) == 0 {
		return nil, fmt.Errorf("file %q was not found in Hugging Face model %s", requestedFile, repository)
	}
	if len(downloads) == 0 {
		return nil, fmt.Errorf("no supported model weight files found in %s (supported: .safetensors, .gguf, .onnx, .bin)", repository)
	}
	sort.Slice(downloads, func(i, j int) bool {
		return downloads[i].Filename < downloads[j].Filename
	})
	return downloads, nil
}

func isModelWeight(filename string) bool {
	name := strings.ToLower(filename)
	if strings.Contains(name, ".index.") || strings.Contains(name, "optimizer") || strings.Contains(name, "training") {
		return false
	}
	for _, extension := range []string{".safetensors", ".gguf", ".onnx", ".bin"} {
		if strings.HasSuffix(name, extension) {
			return true
		}
	}
	return false
}

func escapePath(value string) string {
	parts := strings.Split(value, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
