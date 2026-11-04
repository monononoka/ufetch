package protocols

import (
	"context"
	"fmt"
	"strings"
)

// Router dispatches queries to registered protocol implementations.
type Router struct {
	protocols map[string]Protocol
	http      HTTP
}

func NewRouter(protocols ...Protocol) *Router {
	r := &Router{protocols: make(map[string]Protocol), http: HTTP{}}
	for _, p := range protocols {
		r.Register(p)
	}
	return r
}

// Register adds or replaces a protocol by its scheme (for example, "hf").
func (r *Router) Register(p Protocol) {
	if p == nil {
		return
	}
	scheme := strings.ToLower(strings.TrimSpace(p.Scheme()))
	if scheme == "" || scheme == "http" || scheme == "https" {
		return
	}
	r.protocols[scheme] = p
}

func (r *Router) Resolve(ctx context.Context, input string) ([]Download, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty download request")
	}

	lower := strings.ToLower(input)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return r.http.Resolve(ctx, input)
	}

	scheme, query, ok := strings.Cut(input, ":")
	if !ok || scheme == "" || query == "" {
		return nil, fmt.Errorf("expected an http(s) URL or a protocol query such as hf:model")
	}
	p, ok := r.protocols[strings.ToLower(scheme)]
	if !ok {
		return nil, fmt.Errorf("unsupported protocol %q", scheme)
	}
	return p.Resolve(ctx, query)
}
