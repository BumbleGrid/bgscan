package config

import (
	"fmt"
	"strings"
)

func NormalizeAutoArrangement(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return AutoArrangementNone
	}
	return trimmed
}

func ValidateAutoArrangement(cfg Config) error {
	mode := NormalizeAutoArrangement(cfg.AutoArrangement)
	switch mode {
	case AutoArrangementNone:
		return nil
	case AutoArrangementDefault:
		if !cfg.WholeDocument {
			return fmt.Errorf("auto-arrangement %q requires --whole-document", mode)
		}
		return nil
	default:
		return fmt.Errorf(
			"auto-arrangement %q: want %q or %q",
			cfg.AutoArrangement,
			AutoArrangementDefault,
			AutoArrangementNone,
		)
	}
}

func AutoArrangementStyleMapEnabled(cfg Config) bool {
	return cfg.WholeDocument && NormalizeAutoArrangement(cfg.AutoArrangement) == AutoArrangementDefault
}
