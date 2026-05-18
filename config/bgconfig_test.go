package config

import (
	"os"
	"path/filepath"
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
