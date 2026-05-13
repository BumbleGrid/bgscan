// Package cmd defines the bgscan CLI surface.
//
// The root command wires together configuration loading, the Kubernetes
// reader, the BGSpec mapper, and the output writer. Subcommands (when
// added) live in this same package.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/BumbleGrid/bgscan/config"
	"github.com/spf13/cobra"
)

var cfg config.Config

// configPath is set by --config; empty means "auto: bgconfig.yaml next to the binary if present".
var configPath string

var rootCmd = &cobra.Command{
	Use:   "bgscan",
	Short: "Scan a Kubernetes cluster and emit BGSpec Floor 0 JSON",
	Long: `bgscan reads workloads, services, ingresses, config/secret sources,
PVCs, network policies, and autoscalers from a Kubernetes cluster and
writes a BGSpec Floor 0 document (nodes + edges) that validates against
bgspec.schema.json.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return applyBGConfig(cmd)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: wire cfg -> internal/k8s.Client -> internal/k8s.Reader
		// -> internal/mapper -> internal/output.Writer.
		return fmt.Errorf("scan pipeline not implemented yet")
	},
}

func init() {
	f := rootCmd.Flags()
	f.StringVar(&configPath, "config", "",
		fmt.Sprintf("Path to %s (default: same directory as this binary, if that file exists)", config.BGConfigFileName))
	f.StringVar(&cfg.Kubeconfig, "kubeconfig", "",
		"Path to kubeconfig file (default: in-cluster, then ~/.kube/config)")
	f.StringVar(&cfg.Context, "context", "",
		"Kubeconfig context name (default: current context)")
	f.StringSliceVar(&cfg.Namespaces, "namespaces", nil,
		"Comma-separated namespaces to scan (default: all accessible)")
	f.StringVarP(&cfg.Output, "output", "o", "-",
		`Output path for the BGSpec JSON document ("-" for stdout)`)
	f.StringVar(&cfg.ExtractorVersion, "extractor-version", "0.1.0",
		"Value stamped into node/edge meta.extractorVersion")
}

func applyBGConfig(cmd *cobra.Command) error {
	path, err := config.ResolveBGConfigPath(configPath)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		if configPath != "" {
			return fmt.Errorf("config file %q: %w", path, err)
		}
		return nil
	}
	fileCfg, err := config.LoadBGConfig(path)
	if err != nil {
		return err
	}
	fs := cmd.Flags()
	if !fs.Changed("kubeconfig") {
		cfg.Kubeconfig = fileCfg.Kubeconfig
	}
	if !fs.Changed("context") {
		cfg.Context = fileCfg.Context
	}
	if !fs.Changed("namespaces") {
		cfg.Namespaces = fileCfg.Namespaces
	}
	if !fs.Changed("output") && !fs.Changed("o") {
		cfg.Output = fileCfg.Output
	}
	if !fs.Changed("extractor-version") {
		cfg.ExtractorVersion = fileCfg.ExtractorVersion
	}
	return nil
}

// Execute runs the root command. It is the single entry point invoked
// from main and is responsible for parsing flags, building the pipeline,
// and surfacing exit codes.
func Execute() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
