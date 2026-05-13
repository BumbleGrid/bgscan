package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/BumbleGrid/bgscan/config"
	"github.com/spf13/cobra"
)

var cfg config.Config

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
		return fmt.Errorf("scan pipeline not implemented yet")
	},
}

func init() {
	flagSet := rootCmd.Flags()
	flagSet.StringVar(&configPath, "config", "",
		fmt.Sprintf("Path to %s (default: same directory as this binary, if that file exists)", config.BGConfigFileName))
	flagSet.StringVar(&cfg.Kubeconfig, "kubeconfig", "",
		"Path to kubeconfig file (default: in-cluster, then ~/.kube/config)")
	flagSet.StringVar(&cfg.Context, "context", "",
		"Kubeconfig context name (default: current context)")
	flagSet.StringSliceVar(&cfg.Namespaces, "namespaces", nil,
		"Comma-separated namespaces to scan (default: all accessible)")
	flagSet.StringVarP(&cfg.Output, "output", "o", "-",
		`Output path for the BGSpec JSON document ("-" for stdout)`)
	flagSet.StringVar(&cfg.ExtractorVersion, "extractor-version", "0.1.0",
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

func Execute() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
