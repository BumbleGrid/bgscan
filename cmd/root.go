package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/graph"
	"github.com/BumbleGrid/bgbase/node"
	"github.com/BumbleGrid/bgbase/scanner/k8s"
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
		client, err := k8s.NewClient(cfg.Kubeconfig, cfg.Context)
		if err != nil {
			return err
		}
		reader := k8s.NewReader(client)
		lister := k8s.NewListerForNamespaces(reader, cfg.Namespaces)
		trans := k8s.NewNodeTranslator()
		res := k8s.NewEdgeResolver()
		extractedAt := time.Now().UTC().Format(time.RFC3339)
		tctx := k8s.K8sTranslateContext{
			Floor:         0,
			Meta:          node.Meta{ExtractorVersion: cfg.ExtractorVersion, ExtractedAt: extractedAt},
			ClusterNodeID: "cluster/main",
		}
		content, err := k8s.Floor0Extractor(cmd.Context(), lister, trans, res, tctx)
		if err != nil {
			return err
		}
		if content.Label == "" {
			content.Label = "Infrastructure"
		}
		if content.Description == "" {
			content.Description = "Kubernetes cluster resources (Floor 0)."
		}
		content.Meta = &floor.BlockMeta{
			ExtractedAt:      extractedAt,
			ExtractorVersion: cfg.ExtractorVersion,
		}
		var payload []byte
		if cfg.WholeDocument {
			doc := k8s.NewBGSpecDocument(content)
			payload, err = graph.MarshalBGSpecJSON(doc)
		} else {
			payload, err = graph.MarshalFloorContentJSON(content)
		}
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		if cfg.Output == "" || cfg.Output == "-" {
			if _, werr := os.Stdout.Write(payload); werr != nil {
				return werr
			}
			_, werr := os.Stdout.Write([]byte("\n"))
			return werr
		}
		return os.WriteFile(cfg.Output, payload, 0o644)
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
	flagSet.BoolVar(&cfg.WholeDocument, "whole-document", false,
		"Emit full BGSpec document (floors 0–3) instead of floor 0 slice only")
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
	if !fs.Changed("whole-document") {
		cfg.WholeDocument = fileCfg.WholeDocument
	}
	return nil
}

func Execute() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
