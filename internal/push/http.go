package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
	defaultBackoff    = 1 * time.Second
	maxPayloadBytes   = 10 * 1024 * 1024
)

type httpPusher struct {
	client HTTPDoer
}

func NewHTTPPusher(client HTTPDoer) Pusher {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &httpPusher{client: client}
}

func (pusher *httpPusher) Push(ctx context.Context, input PushInput) (PushResult, error) {
	if len(input.Payload) > maxPayloadBytes {
		return PushResult{}, fmt.Errorf("%w: payload is %d bytes; backend limit is %d",
			ErrPayloadSize, len(input.Payload), maxPayloadBytes)
	}
	requestBody, err := json.Marshal(map[string]any{
		"org_slug":          input.OrgSlug,
		"document_slug":     input.DocumentSlug,
		"cluster_slug":      input.ClusterSlug,
		"payload_kind":      input.PayloadKind,
		"whole_document":    input.WholeDocument,
		"extractor_version": input.ExtractorVersion,
		"payload":           json.RawMessage(input.Payload),
	})
	if err != nil {
		return PushResult{}, fmt.Errorf("encode request: %w", err)
	}

	maxRetries := input.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	backoff := input.Backoff
	if backoff <= 0 {
		backoff = defaultBackoff
	}

	targetURL := strings.TrimRight(input.Endpoint, "/") + "/api/v1/extractions"

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return PushResult{}, ctx.Err()
			case <-time.After(backoff * time.Duration(1<<uint(attempt-1))):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(requestBody))
		if err != nil {
			return PushResult{}, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+input.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", input.Idempotency)
		req.Header.Set("User-Agent", "bgscan/"+input.ExtractorVersion)
		resp, err := pusher.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http do: %w", err)
			if attempt < maxRetries && isNetworkRetriable(err) {
				continue
			}
			return PushResult{}, lastErr
		}
		result, pushErr, retriable, err := parseResponse(resp)
		if err != nil {
			lastErr = err
			if attempt < maxRetries && retriable {
				continue
			}
			return PushResult{}, lastErr
		}
		if pushErr != nil {
			if attempt < maxRetries && pushErr.Retriable {
				lastErr = pushErr
				continue
			}
			return PushResult{}, wrapSentinel(pushErr)
		}
		return result, nil
	}
	return PushResult{}, lastErr
}

func parseResponse(resp *http.Response) (PushResult, *PushError, bool, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PushResult{}, nil, true, fmt.Errorf("read body: %w", err)
	}
	switch {
	case resp.StatusCode == http.StatusCreated, resp.StatusCode == http.StatusOK:
		var envelope struct {
			ResponseCode string          `json:"response_code"`
			Data         json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			return PushResult{}, nil, false,
				fmt.Errorf("decode response envelope: %w (status=%d body=%.200q)", err, resp.StatusCode, body)
		}
		var data struct {
			ID          string `json:"id"`
			Status      string `json:"status"`
			S3Key       string `json:"s3_key"`
			ContentHash string `json:"content_hash"`
			Idempotent  bool   `json:"idempotent"`
			NodeCount   int    `json:"node_count"`
			EdgeCount   int    `json:"edge_count"`
		}
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			return PushResult{}, nil, false, fmt.Errorf("decode success data: %w", err)
		}
		return PushResult{
			HTTPStatus:   resp.StatusCode,
			Code:         envelope.ResponseCode,
			ExtractionID: data.ID,
			S3Key:        data.S3Key,
			ContentHash:  data.ContentHash,
			NodeCount:    data.NodeCount,
			EdgeCount:    data.EdgeCount,
			Idempotent:   data.Idempotent || envelope.ResponseCode == "EXTRACTION_CREATE_IDEMPOTENT",
			Raw:          body,
		}, nil, false, nil
	default:
		retriable := resp.StatusCode >= 500 && resp.StatusCode <= 599
		var envelope struct {
			ErrorCode string `json:"error_code"`
			Data      struct {
				Message      string `json:"message"`
				ErrorMessage string `json:"error_message"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			return PushResult{}, nil, retriable,
				fmt.Errorf("decode error envelope: %w (status=%d body=%.200q)", err, resp.StatusCode, body)
		}
		errMsg := envelope.Data.Message
		if errMsg == "" {
			errMsg = envelope.Data.ErrorMessage
		}
		return PushResult{}, &PushError{
			HTTPStatus: resp.StatusCode,
			Code:       envelope.ErrorCode,
			Message:    errMsg,
			Retriable:  retriable,
		}, retriable, nil
	}
}

func wrapSentinel(pushErr *PushError) error {
	switch pushErr.Code {
	case "EXTRACTION_AUTH_ERROR_BAD_FORMAT",
		"EXTRACTION_AUTH_ERROR_UNKNOWN",
		"EXTRACTION_AUTH_ERROR_REVOKED",
		"EXTRACTION_AUTH_ERROR_EXPIRED":
		return fmt.Errorf("%w: %s", ErrAuth, pushErr.Error())
	case "EXTRACTION_CREATE_ERROR_SCOPE_FORBIDDEN":
		return fmt.Errorf("%w: %s", ErrScope, pushErr.Error())
	case "EXTRACTION_CREATE_ERROR_SCHEMA_VALIDATION":
		return fmt.Errorf("%w: %s", ErrValidation, pushErr.Error())
	case "EXTRACTION_CREATE_ERROR_IDEMPOTENCY_CONFLICT":
		return fmt.Errorf("%w: %s", ErrIdempotency, pushErr.Error())
	case "EXTRACTION_CREATE_ERROR_CLUSTER_NOT_FOUND":
		return fmt.Errorf("%w: %s", ErrCluster, pushErr.Error())
	case "EXTRACTION_CREATE_ERROR_PAYLOAD_TOO_LARGE":
		return fmt.Errorf("%w: %s", ErrPayloadSize, pushErr.Error())
	}
	return pushErr
}

func isNetworkRetriable(err error) bool {
	if err == nil {
		return false
	}
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "connection reset") ||
		errors.Is(err, io.EOF)
}
