package push

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Pusher interface {
	Push(ctx context.Context, input PushInput) (PushResult, error)
}

type PushInput struct {
	Endpoint     string
	OrgSlug      string
	DocumentSlug string
	ClusterSlug  string
	APIKey       string
	Idempotency  string

	PayloadKind      string
	WholeDocument    bool
	ExtractorVersion string
	Payload          []byte

	HTTPClient HTTPDoer
	MaxRetries int
	Timeout    time.Duration
	Backoff    time.Duration
}

type PushResult struct {
	HTTPStatus   int
	Code         string
	ExtractionID string
	S3Key        string
	ContentHash  string
	NodeCount    int
	EdgeCount    int
	Idempotent   bool
	Raw          []byte
}

type PushError struct {
	HTTPStatus int
	Code       string
	Message    string
	Retriable  bool
}

func (pushErr *PushError) Error() string {
	return fmt.Sprintf("push failed: HTTP %d %s: %s", pushErr.HTTPStatus, pushErr.Code, pushErr.Message)
}

var (
	ErrAuth        = errors.New("push: bearer token rejected")
	ErrScope       = errors.New("push: token does not authorize this cluster")
	ErrValidation  = errors.New("push: payload failed schema validation")
	ErrIdempotency = errors.New("push: idempotency key reused with different content")
	ErrCluster     = errors.New("push: cluster slug not found")
	ErrPayloadSize = errors.New("push: payload exceeds backend limit")
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}
