package config

import "testing"

func TestNormalizeAutoArrangement(t *testing.T) {
	if got := NormalizeAutoArrangement(""); got != AutoArrangementNone {
		t.Fatalf("empty = %q, want %q", got, AutoArrangementNone)
	}
	if got := NormalizeAutoArrangement("  default  "); got != AutoArrangementDefault {
		t.Fatalf("trimmed = %q, want %q", got, AutoArrangementDefault)
	}
}

func TestValidateAutoArrangement_defaultRequiresWholeDocument(t *testing.T) {
	err := ValidateAutoArrangement(Config{AutoArrangement: AutoArrangementDefault, WholeDocument: false})
	if err == nil {
		t.Fatal("expected error when default without whole document")
	}
}

func TestValidateAutoArrangement_noneWithoutWholeDocument(t *testing.T) {
	if err := ValidateAutoArrangement(Config{AutoArrangement: AutoArrangementNone}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAutoArrangement_invalidValue(t *testing.T) {
	err := ValidateAutoArrangement(Config{AutoArrangement: "elk", WholeDocument: true})
	if err == nil {
		t.Fatal("expected error for unknown mode")
	}
}

func TestAutoArrangementStyleMapEnabled(t *testing.T) {
	cfg := Config{AutoArrangement: AutoArrangementDefault, WholeDocument: true}
	if !AutoArrangementStyleMapEnabled(cfg) {
		t.Fatal("expected style map enabled")
	}
	cfg.WholeDocument = false
	if AutoArrangementStyleMapEnabled(cfg) {
		t.Fatal("expected style map disabled without whole document")
	}
	cfg = Config{AutoArrangement: AutoArrangementNone, WholeDocument: true}
	if AutoArrangementStyleMapEnabled(cfg) {
		t.Fatal("expected style map disabled for none")
	}
}
