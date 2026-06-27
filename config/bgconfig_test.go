package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBGConfig_wholeDocumentDefaultFalseWhenOmitted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: "-"
extractor_version: "0.0.0"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBGConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WholeDocument {
		t.Fatal("expected WholeDocument false when key omitted")
	}
}

func TestLoadBGConfig_autoArrangement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: "-"
extractor_version: "0.0.0"
auto_arrangement: none
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBGConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AutoArrangement != AutoArrangementNone {
		t.Fatalf("AutoArrangement = %q, want %q", cfg.AutoArrangement, AutoArrangementNone)
	}
}

func TestLoadBGConfig_wholeDocumentTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: "-"
extractor_version: "0.0.0"
whole_document: true
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBGConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.WholeDocument {
		t.Fatal("expected WholeDocument true")
	}
}

func TestLoadBGConfig_existingFileWithoutPushKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: "-"
extractor_version: "0.1.0"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBGConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "-" {
		t.Fatalf("Output = %q", cfg.Output)
	}
}

func TestValidatePushConfig_missingKeys(t *testing.T) {
	err := ValidatePushConfig(Config{Output: OutputPush})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, key := range []string{"endpoint", "org", "document", "cluster", "api_key"} {
		if !strings.Contains(err.Error(), key) {
			t.Fatalf("error %q missing %q", err, key)
		}
	}
}

func TestLoadBGConfig_pushLocalValidateFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: push
extractor_version: "0.1.0"
endpoint: "https://api.example.com"
org: acme
document: infra
cluster: prod
api_key: bg_sk_test
local_validate: false
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBGConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LocalValidate {
		t.Fatal("expected LocalValidate false")
	}
}

func TestLoadBGConfig_pushLocalValidateDefaultTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: push
extractor_version: "0.1.0"
endpoint: "https://api.example.com"
org: acme
document: infra
cluster: prod
api_key: bg_sk_test
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBGConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.LocalValidate {
		t.Fatal("expected LocalValidate true for push mode default")
	}
}

func TestLoadBGConfig_nonPushLocalValidateDefaultFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: "/tmp/out.json"
extractor_version: "0.1.0"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBGConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LocalValidate {
		t.Fatal("expected LocalValidate false for non-push output")
	}
}

func TestValidatePushConfig_badEndpointScheme(t *testing.T) {
	err := ValidatePushConfig(Config{
		Output:   OutputPush,
		Endpoint: "ftp://oops",
		Org:      "acme",
		Document: "doc",
		Cluster:  "prod",
		APIKey:   "bg_sk_test",
	})
	if err == nil || !strings.Contains(err.Error(), "http or https") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidatePushConfig_badAPIKeyPrefix(t *testing.T) {
	err := ValidatePushConfig(Config{
		Output:   OutputPush,
		Endpoint: "https://api.example.com",
		Org:      "acme",
		Document: "doc",
		Cluster:  "prod",
		APIKey:   "no-prefix",
	})
	if err == nil || !strings.Contains(err.Error(), "bg_sk_") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadBGConfig_ignoreNamespacesAbsentUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: "-"
extractor_version: "0.0.0"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBGConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	defaults, err := LoadDefaultScanFilters()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.IgnoreNamespaces) != len(defaults.IgnoreNamespaces) {
		t.Fatalf("IgnoreNamespaces = %v, want %v", cfg.IgnoreNamespaces, defaults.IgnoreNamespaces)
	}
	for idx := range defaults.IgnoreNamespaces {
		if cfg.IgnoreNamespaces[idx] != defaults.IgnoreNamespaces[idx] {
			t.Fatalf("IgnoreNamespaces = %v, want %v", cfg.IgnoreNamespaces, defaults.IgnoreNamespaces)
		}
	}
}

func TestLoadBGConfig_ignoreNamespacesEmptyDisablesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: "-"
extractor_version: "0.0.0"
ignore_namespaces: []
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBGConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IgnoreNamespaces == nil || len(cfg.IgnoreNamespaces) != 0 {
		t.Fatalf("IgnoreNamespaces = %v, want empty slice", cfg.IgnoreNamespaces)
	}
}

func TestLoadBGConfig_invalidIgnoreWorkloadPattern(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: "-"
extractor_version: "0.0.0"
ignore_workloads:
  - "bad-pattern"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadBGConfig(path)
	if err == nil || !strings.Contains(err.Error(), "scan filters") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadBGConfig_ignoreWorkloadsAbsentUsesEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: ""
context: ""
namespaces: []
output: "-"
extractor_version: "0.0.0"
ignore_namespaces: []
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBGConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IgnoreWorkloads == nil || len(cfg.IgnoreWorkloads) != 0 {
		t.Fatalf("IgnoreWorkloads = %v, want empty slice", cfg.IgnoreWorkloads)
	}
}
