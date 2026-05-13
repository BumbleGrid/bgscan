// Package config loads and holds bgscan runtime configuration: which
// cluster to talk to, which namespaces to scan, where to write output,
// and stamping fields (extractor version, extractedAt) that flow into
// node/edge meta blocks.
package config

// Config is the resolved bgscan configuration for a single run. It is
// populated from CLI flags, environment variables, and (optionally) a
// config file, in that precedence order.
type Config struct {
	// Kubeconfig is the path to a kubeconfig file. Empty means
	// in-cluster config (or default loading rules outside a cluster).
	Kubeconfig string

	// Context selects a kubeconfig context by name. Empty uses the
	// current-context.
	Context string

	// Namespaces restricts the scan to these namespaces. Empty scans
	// all namespaces the caller has access to.
	Namespaces []string

	// Output is the destination path for the BGSpec JSON document.
	// Empty or "-" means stdout.
	Output string

	// ExtractorVersion is stamped into node.meta.extractorVersion and
	// edge.meta.extractorVersion.
	ExtractorVersion string
}

// Load resolves a Config from CLI flags / env / file. The concrete
// loader will be plumbed in once cmd/root wires its flag set.
func Load() (*Config, error) {
	// TODO: implement precedence (flags > env > file > defaults).
	return &Config{}, nil
}
