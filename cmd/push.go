package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/graph"
	"github.com/BumbleGrid/bgbase/scanner/k8s"
	"github.com/BumbleGrid/bgbase/validate"
	"github.com/BumbleGrid/bgscan/config"
	"github.com/BumbleGrid/bgscan/internal/push"
	"gopkg.in/yaml.v2"
)

func payloadKindFor(wholeDocument bool) string {
	if wholeDocument {
		return "whole_document"
	}
	return "floor0"
}

func validateLocally(validator validate.Validator, content floor.Content, doc *graph.BGSpecDocument, wholeDocument bool) error {
	if wholeDocument {
		if doc == nil {
			return fmt.Errorf("whole document missing for validation")
		}
		return validator.Validate(doc)
	}
	specDoc := k8s.NewBGSpecDocument(content)
	return validator.Validate(&specDoc)
}

func printPushResult(result push.PushResult, writer io.Writer) {
	if result.Idempotent {
		fmt.Fprintf(writer, "bgscan: push idempotent — id=%s node=%d edge=%d (no new extraction recorded)\n",
			result.ExtractionID, result.NodeCount, result.EdgeCount)
		return
	}
	fmt.Fprintf(writer, "bgscan: push ok — id=%s node=%d edge=%d idempotent=false\n",
		result.ExtractionID, result.NodeCount, result.EdgeCount)
}

func printPushDryRun(input push.PushInput, writer io.Writer) error {
	fmt.Fprintln(writer, "bgscan: push dry-run — review before sharing publicly (topology metadata included; api_key redacted)")
	block := map[string]any{
		"endpoint":          input.Endpoint,
		"org_slug":          input.OrgSlug,
		"document_slug":     input.DocumentSlug,
		"cluster_slug":      input.ClusterSlug,
		"api_key":           "bg_sk_<redacted>",
		"payload_kind":      input.PayloadKind,
		"whole_document":    input.WholeDocument,
		"extractor_version": input.ExtractorVersion,
		"idempotency_key":   input.Idempotency,
	}
	var payload any
	if err := json.Unmarshal(input.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload for dry-run: %w", err)
	}
	block["payload"] = payload
	out, err := yaml.Marshal(block)
	if err != nil {
		return fmt.Errorf("encode dry-run yaml: %w", err)
	}
	_, err = writer.Write(out)
	return err
}

func runPushMode(
	ctx context.Context,
	cfg config.Config,
	pushDryRun bool,
	payload []byte,
	content floor.Content,
	doc *graph.BGSpecDocument,
	writer io.Writer,
	pusher push.Pusher,
) error {
	if err := config.ValidatePushConfig(cfg); err != nil {
		return fmt.Errorf("push config: %w", err)
	}
	if cfg.LocalValidate {
		validator, vErr := validate.NewSchemaValidator()
		if vErr != nil {
			return fmt.Errorf("init local validator: %w", vErr)
		}
		if vErr := validateLocally(validator, content, doc, cfg.WholeDocument); vErr != nil {
			return fmt.Errorf("local schema validation: %w", vErr)
		}
	}
	idempotency := cfg.Idempotency
	if idempotency == "" {
		idempotency = push.DeriveIdempotencyKey(payload)
	}
	input := push.PushInput{
		Endpoint:         cfg.Endpoint,
		OrgSlug:          cfg.Org,
		DocumentSlug:     cfg.Document,
		ClusterSlug:      cfg.Cluster,
		APIKey:           cfg.APIKey,
		Idempotency:      idempotency,
		PayloadKind:      payloadKindFor(cfg.WholeDocument),
		WholeDocument:    cfg.WholeDocument,
		ExtractorVersion: cfg.ExtractorVersion,
		Payload:          payload,
	}
	if pushDryRun {
		return printPushDryRun(input, writer)
	}
	fmt.Fprintln(writer, "bgscan: pushing extraction…")
	result, pushErr := pusher.Push(ctx, input)
	if pushErr != nil {
		return pushErr
	}
	printPushResult(result, writer)
	return nil
}
