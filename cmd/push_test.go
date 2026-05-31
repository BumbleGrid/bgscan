package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/graph"
	"github.com/BumbleGrid/bgbase/validate"
	"github.com/BumbleGrid/bgscan/config"
	"github.com/BumbleGrid/bgscan/internal/push"
)

func TestPrintPushDryRun_redactsAPIKey(t *testing.T) {
	var buf bytes.Buffer
	input := push.PushInput{
		Endpoint:         "https://api.example.com",
		OrgSlug:          "acme",
		DocumentSlug:     "infra",
		ClusterSlug:      "prod",
		APIKey:           "bg_sk_super_secret_value",
		Idempotency:      "content:abc",
		PayloadKind:      "floor0",
		ExtractorVersion: "0.1.0",
		Payload:          []byte(`{"label":"Infrastructure","nodes":[],"edges":[]}`),
	}
	if err := printPushDryRun(input, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "bg_sk_super_secret_value") {
		t.Fatalf("dry-run leaked api key: %s", out)
	}
	if !strings.Contains(out, `bg_sk_<redacted>`) {
		t.Fatalf("dry-run missing redacted marker: %s", out)
	}
}

func TestRunPushMode_localValidationFailure(t *testing.T) {
	cfg := config.Config{
		Output:           config.OutputPush,
		Endpoint:         "https://api.example.com",
		Org:              "acme",
		Document:         "infra",
		Cluster:          "prod",
		APIKey:           "bg_sk_test",
		LocalValidate:    true,
		WholeDocument:    true,
		ExtractorVersion: "0.1.0",
	}
	doc := &graph.BGSpecDocument{BGSpec: "not-a-version"}
	payload := []byte(`{"bgspec":"not-a-version"}`)
	content := floor.Content{Label: "Infrastructure"}
	var buf bytes.Buffer
	err := runPushMode(context.Background(), cfg, false, payload, content, doc, &buf, push.NewHTTPPusher(nil))
	if err == nil || !strings.Contains(err.Error(), "local schema validation") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunPushMode_pushDryRun(t *testing.T) {
	cfg := config.Config{
		Output:           config.OutputPush,
		Endpoint:         "https://api.example.com",
		Org:              "acme",
		Document:         "infra",
		Cluster:          "prod",
		APIKey:           "bg_sk_test",
		LocalValidate:    false,
		ExtractorVersion: "0.1.0",
	}
	payload := []byte(`{"label":"Infrastructure","nodes":[],"edges":[]}`)
	content := floor.Content{Label: "Infrastructure"}
	var buf bytes.Buffer
	if err := runPushMode(context.Background(), cfg, true, payload, content, nil, &buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `bg_sk_<redacted>`) {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestValidateLocally_invalidWholeDocument(t *testing.T) {
	validator, err := validate.NewSchemaValidator()
	if err != nil {
		t.Fatal(err)
	}
	doc := &graph.BGSpecDocument{BGSpec: "not-a-version"}
	if err := validateLocally(validator, floor.Content{}, doc, true); err == nil {
		t.Fatal("expected validation error for invalid whole document")
	}
}

func TestValidateLocally_rejectsNilWholeDocument(t *testing.T) {
	validator, err := validate.NewSchemaValidator()
	if err != nil {
		t.Fatal(err)
	}
	err = validateLocally(validator, floor.Content{}, nil, true)
	if err == nil || !strings.Contains(err.Error(), "whole document missing") {
		t.Fatalf("err = %v", err)
	}
}
