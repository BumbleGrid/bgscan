package config_test

import (
	"strings"
	"testing"

	"github.com/BumbleGrid/bgscan/config"
)

func TestValidateScanFilters_validWorkloadPattern(t *testing.T) {
	err := config.ValidateScanFilters(
		[]string{"kube-system"},
		[]string{"apps/deployments/noise*"},
	)
	if err != nil {
		t.Fatalf("ValidateScanFilters() = %v", err)
	}
}

func TestValidateScanFilters_invalidWorkloadKind(t *testing.T) {
	err := config.ValidateScanFilters(nil, []string{"apps/pods/noise"})
	if err == nil || !strings.Contains(err.Error(), "scan filters") {
		t.Fatalf("ValidateScanFilters() = %v", err)
	}
}

func TestLoadDefaultScanFilters_embeddedYAML(t *testing.T) {
	filters, err := config.LoadDefaultScanFilters()
	if err != nil {
		t.Fatal(err)
	}
	if len(filters.IgnoreNamespaces) < 5 {
		t.Fatalf("IgnoreNamespaces = %v, want built-in defaults", filters.IgnoreNamespaces)
	}
	if len(filters.IgnoreWorkloads) != 0 {
		t.Fatalf("IgnoreWorkloads = %v, want empty", filters.IgnoreWorkloads)
	}
}
