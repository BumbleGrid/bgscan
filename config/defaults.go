package config

import (
	_ "embed"
	"fmt"
	"sync"

	"gopkg.in/yaml.v2"
)

//go:embed default_scan_filters.yaml
var defaultScanFiltersYAML []byte

type defaultScanFilters struct {
	IgnoreNamespaces []string `yaml:"ignore_namespaces"`
	IgnoreWorkloads  []string `yaml:"ignore_workloads"`
}

var (
	defaultScanFiltersOnce sync.Once
	defaultScanFiltersData defaultScanFilters
	defaultScanFiltersErr  error
)

func LoadDefaultScanFilters() (defaultScanFilters, error) {
	defaultScanFiltersOnce.Do(func() {
		if err := yaml.Unmarshal(defaultScanFiltersYAML, &defaultScanFiltersData); err != nil {
			defaultScanFiltersErr = fmt.Errorf("parse embedded default_scan_filters.yaml: %w", err)
		}
	})
	return defaultScanFiltersData, defaultScanFiltersErr
}
