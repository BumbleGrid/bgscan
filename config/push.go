package config

import (
	"fmt"
	"net/url"
	"strings"
)

func ValidatePushConfig(cfg Config) error {
	if cfg.Output != OutputPush {
		return nil
	}
	var missing []string
	if cfg.Endpoint == "" {
		missing = append(missing, "endpoint")
	}
	if cfg.Org == "" {
		missing = append(missing, "org")
	}
	if cfg.Document == "" {
		missing = append(missing, "document")
	}
	if cfg.Cluster == "" {
		missing = append(missing, "cluster")
	}
	if cfg.APIKey == "" {
		missing = append(missing, "api_key")
	}
	if len(missing) > 0 {
		return fmt.Errorf("push mode requires: %s", strings.Join(missing, ", "))
	}
	parsed, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("endpoint %q is not a valid URL: %w", cfg.Endpoint, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("endpoint %q must be http or https (got %q)", cfg.Endpoint, parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("endpoint %q has no host", cfg.Endpoint)
	}
	if !strings.HasPrefix(cfg.APIKey, "bg_sk_") {
		return fmt.Errorf("api_key must start with %q", "bg_sk_")
	}
	return nil
}
