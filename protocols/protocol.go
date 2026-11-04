package protocols

import "context"

// Download describes one resource resolved by a protocol.
type Download struct {
	URL         string
	Filename    string
	Description string
	Size        int64 // -1 when the size is unknown
}

// Protocol resolves a user query (for example, the text after "hf:") into
// one or more downloadable resources.
type Protocol interface {
	Scheme() string
	Resolve(ctx context.Context, query string) ([]Download, error)
}
