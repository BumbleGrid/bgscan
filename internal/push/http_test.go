package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func successBody(idempotent bool) []byte {
	code := "EXTRACTION_CREATE_SUCCESS"
	idempotentFlag := false
	if idempotent {
		code = "EXTRACTION_CREATE_IDEMPOTENT"
		idempotentFlag = true
	}
	body, _ := json.Marshal(map[string]any{
		"response_code": code,
		"data": map[string]any{
			"id":           "01HXXX",
			"status":       "received",
			"s3_key":       "extractions/doc/cluster/hash.json",
			"content_hash": "abc123",
			"idempotent":   idempotentFlag,
			"node_count":   47,
			"edge_count":   61,
		},
	})
	return body
}

func errorBody(code, message string) []byte {
	body, _ := json.Marshal(map[string]any{
		"error_code": code,
		"data":       map[string]string{"message": message},
	})
	return body
}

func baseInput(endpoint string) PushInput {
	return PushInput{
		Endpoint:         endpoint,
		OrgSlug:          "paperpath-inc",
		DocumentSlug:     "payments-domain",
		ClusterSlug:      "prod-eu",
		APIKey:           "bg_sk_test_secret",
		Idempotency:      DeriveIdempotencyKey([]byte(`{"nodes":[]}`)),
		PayloadKind:      "floor0",
		WholeDocument:    false,
		ExtractorVersion: "0.1.0",
		Payload:          []byte(`{"label":"Infrastructure","nodes":[],"edges":[]}`),
	}
}

func TestHTTPPusher_happyPath(t *testing.T) {
	var captured struct {
		auth    string
		body    map[string]any
		headers http.Header
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		captured.auth = req.Header.Get("Authorization")
		captured.headers = req.Header.Clone()
		if err := json.NewDecoder(req.Body).Decode(&captured.body); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(successBody(false))
	}))
	defer server.Close()

	input := baseInput(server.URL)
	result, err := NewHTTPPusher(nil).Push(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.NodeCount != 47 || result.EdgeCount != 61 {
		t.Fatalf("counts = %d/%d, want 47/61", result.NodeCount, result.EdgeCount)
	}
	if result.Idempotent || result.ExtractionID != "01HXXX" {
		t.Fatalf("result = %+v", result)
	}
	if captured.auth != "Bearer bg_sk_test_secret" {
		t.Fatalf("Authorization = %q", captured.auth)
	}
	if captured.body["org_slug"] != input.OrgSlug ||
		captured.body["document_slug"] != input.DocumentSlug ||
		captured.body["cluster_slug"] != input.ClusterSlug ||
		captured.body["payload_kind"] != input.PayloadKind ||
		captured.body["extractor_version"] != input.ExtractorVersion {
		t.Fatalf("body = %+v", captured.body)
	}
	for _, header := range []string{"Authorization", "Content-Type", "Idempotency-Key", "User-Agent"} {
		if captured.headers.Get(header) == "" {
			t.Fatalf("missing header %q", header)
		}
	}
	if captured.headers.Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q", captured.headers.Get("Content-Type"))
	}
	if captured.headers.Get("User-Agent") != "bgscan/0.1.0" {
		t.Fatalf("User-Agent = %q", captured.headers.Get("User-Agent"))
	}
}

func TestHTTPPusher_idempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(successBody(true))
	}))
	defer server.Close()

	result, err := NewHTTPPusher(nil).Push(context.Background(), baseInput(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Idempotent {
		t.Fatal("expected idempotent result")
	}
}

func TestHTTPPusher_authErrors(t *testing.T) {
	codes := []string{
		"EXTRACTION_AUTH_ERROR_UNKNOWN",
		"EXTRACTION_AUTH_ERROR_REVOKED",
		"EXTRACTION_AUTH_ERROR_EXPIRED",
		"EXTRACTION_AUTH_ERROR_BAD_FORMAT",
	}
	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			var hits atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				hits.Add(1)
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write(errorBody(code, "auth failed"))
			}))
			defer server.Close()

			_, err := NewHTTPPusher(nil).Push(context.Background(), baseInput(server.URL))
			if !errors.Is(err, ErrAuth) {
				t.Fatalf("errors.Is(err, ErrAuth) = false, err = %v", err)
			}
			if hits.Load() != 1 {
				t.Fatalf("server hits = %d, want 1", hits.Load())
			}
		})
	}
}

func TestHTTPPusher_scopeForbidden(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write(errorBody("EXTRACTION_CREATE_ERROR_SCOPE_FORBIDDEN", "forbidden"))
	}))
	defer server.Close()

	_, err := NewHTTPPusher(nil).Push(context.Background(), baseInput(server.URL))
	if !errors.Is(err, ErrScope) {
		t.Fatalf("err = %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d", hits.Load())
	}
}

func TestHTTPPusher_clusterNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(errorBody("EXTRACTION_CREATE_ERROR_CLUSTER_NOT_FOUND", "cluster not found"))
	}))
	defer server.Close()

	_, err := NewHTTPPusher(nil).Push(context.Background(), baseInput(server.URL))
	if !errors.Is(err, ErrCluster) {
		t.Fatalf("err = %v", err)
	}
}

func TestHTTPPusher_schemaValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write(errorBody("EXTRACTION_CREATE_ERROR_SCHEMA_VALIDATION", "schema invalid"))
	}))
	defer server.Close()

	err := func() error {
		_, pushErr := NewHTTPPusher(nil).Push(context.Background(), baseInput(server.URL))
		return pushErr
	}()
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "schema invalid") {
		t.Fatalf("err = %v", err)
	}
}

func TestHTTPPusher_idempotencyConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write(errorBody("EXTRACTION_CREATE_ERROR_IDEMPOTENCY_CONFLICT", "conflict"))
	}))
	defer server.Close()

	_, err := NewHTTPPusher(nil).Push(context.Background(), baseInput(server.URL))
	if !errors.Is(err, ErrIdempotency) {
		t.Fatalf("err = %v", err)
	}
}

func TestHTTPPusher_payloadTooLarge(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write(errorBody("EXTRACTION_CREATE_ERROR_PAYLOAD_TOO_LARGE", "too large"))
	}))
	defer server.Close()

	_, err := NewHTTPPusher(nil).Push(context.Background(), baseInput(server.URL))
	if !errors.Is(err, ErrPayloadSize) {
		t.Fatalf("err = %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d", hits.Load())
	}
}

func TestHTTPPusher_retry503(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if hits.Add(1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write(errorBody("EXTRACTION_CREATE_ERROR_INTERNAL", "busy"))
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(successBody(false))
	}))
	defer server.Close()

	input := baseInput(server.URL)
	input.Backoff = time.Millisecond
	result, err := NewHTTPPusher(nil).Push(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExtractionID != "01HXXX" {
		t.Fatalf("result = %+v", result)
	}
	if hits.Load() != 3 {
		t.Fatalf("hits = %d, want 3", hits.Load())
	}
}

func TestHTTPPusher_retryExhaustion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(errorBody("EXTRACTION_CREATE_ERROR_INTERNAL", "busy"))
	}))
	defer server.Close()

	input := baseInput(server.URL)
	input.MaxRetries = 3
	input.Backoff = time.Millisecond
	_, err := NewHTTPPusher(nil).Push(context.Background(), input)
	var pushErr *PushError
	if !errors.As(err, &pushErr) || pushErr.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("err = %v", err)
	}
}

func TestHTTPPusher_networkRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(successBody(false))
	}))
	defer server.Close()

	client := &retryOnceClient{
		failErr: &netError{timeout: true},
		success: server.Client(),
	}
	input := baseInput(server.URL)
	input.Backoff = time.Millisecond
	input.HTTPClient = client
	_, err := NewHTTPPusher(input.HTTPClient).Push(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if client.calls != 2 {
		t.Fatalf("calls = %d, want 2", client.calls)
	}
}

func TestHTTPPusher_networkNonRetriable(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hits.Add(1)
	}))
	defer server.Close()

	client := &fixedErrorClient{err: fmt.Errorf("x509: unknown authority")}
	input := baseInput(server.URL)
	input.HTTPClient = client
	_, err := NewHTTPPusher(input.HTTPClient).Push(context.Background(), input)
	if err == nil {
		t.Fatal("expected error")
	}
	if client.calls != 1 {
		t.Fatalf("calls = %d", client.calls)
	}
}

func TestHTTPPusher_contextCancelDuringBackoff(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(errorBody("EXTRACTION_CREATE_ERROR_INTERNAL", "busy"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	input := baseInput(server.URL)
	input.MaxRetries = 3
	input.Backoff = 200 * time.Millisecond
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_, err := NewHTTPPusher(nil).Push(ctx, input)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if hits.Load() >= 4 {
		t.Fatalf("hits = %d", hits.Load())
	}
}

func TestHTTPPusher_payloadSizeCap(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		hits.Add(1)
	}))
	defer server.Close()

	input := baseInput(server.URL)
	input.Payload = make([]byte, maxPayloadBytes+1)
	_, err := NewHTTPPusher(nil).Push(context.Background(), input)
	if !errors.Is(err, ErrPayloadSize) {
		t.Fatalf("err = %v", err)
	}
	if hits.Load() != 0 {
		t.Fatalf("hits = %d", hits.Load())
	}
}

func TestHTTPPusher_bodyDecodeFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "<html>bad gateway</html>")
	}))
	defer server.Close()

	_, err := NewHTTPPusher(nil).Push(context.Background(), baseInput(server.URL))
	if errors.Is(err, ErrAuth) || errors.Is(err, ErrScope) {
		t.Fatalf("unexpected sentinel: %v", err)
	}
	if !strings.Contains(err.Error(), "<html>") {
		t.Fatalf("err = %v", err)
	}
}

func TestHTTPPusher_explicitIdempotencyKey(t *testing.T) {
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		captured = req.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(successBody(false))
	}))
	defer server.Close()

	input := baseInput(server.URL)
	input.Idempotency = "release-2026-05-27"
	_, err := NewHTTPPusher(nil).Push(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if captured != "release-2026-05-27" {
		t.Fatalf("Idempotency-Key = %q", captured)
	}
}

type netError struct {
	timeout bool
}

func (err *netError) Error() string   { return "timeout" }
func (err *netError) Timeout() bool   { return err.timeout }
func (err *netError) Temporary() bool { return err.timeout }

type retryOnceClient struct {
	failErr error
	success HTTPDoer
	calls   int
}

func (client *retryOnceClient) Do(req *http.Request) (*http.Response, error) {
	client.calls++
	if client.calls == 1 {
		return nil, client.failErr
	}
	return client.success.Do(req)
}

type fixedErrorClient struct {
	err   error
	calls int
}

func (client *fixedErrorClient) Do(req *http.Request) (*http.Response, error) {
	client.calls++
	return nil, client.err
}
