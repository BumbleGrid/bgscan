package config

import (
	"fmt"

	"github.com/BumbleGrid/bgbase/scanner/k8s"
)

func ValidateScanFilters(ignoreNamespaces, ignoreWorkloads []string) error {
	if _, err := k8s.ParseScanFilter(ignoreNamespaces, ignoreWorkloads); err != nil {
		return fmt.Errorf("scan filters: %w", err)
	}
	return nil
}
