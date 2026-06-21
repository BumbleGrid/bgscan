package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/BumbleGrid/bgbase/floor"
	"github.com/BumbleGrid/bgbase/graph"
	"github.com/BumbleGrid/bgbase/node"
	"github.com/BumbleGrid/bgbase/scanner/k8s"
	"github.com/BumbleGrid/bgscan/config"
	"github.com/BumbleGrid/bgscan/internal/push"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var cfg config.Config

var configPath string

var pushDryRun bool

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
		kubeContexts, currentContext, err := loadKubeconfigContextNames(cfg.Kubeconfig)
		if err != nil {
			return fmt.Errorf("load kubeconfig: %w", err)
		}
		scanContexts := resolveScanContexts(cfg.Context, currentContext, kubeContexts)
		var unreachable []unreachableContext
		if cfg.Context == "" {
			scanContexts, unreachable = filterReachableContexts(cmd.Context(), cfg.Kubeconfig, scanContexts)
		}
		if len(scanContexts) == 0 {
			if allLocalhostConnectionRefused(unreachable) {
				return fmt.Errorf("no reachable kubeconfig context to scan: cluster API servers use localhost addresses (127.0.0.1) that are not reachable from inside a Docker container; on Linux use docker run --network host")
			}
			return fmt.Errorf("no reachable kubeconfig context to scan")
		}
		trans := k8s.NewNodeTranslator()
		extractedAt := time.Now().UTC().Format(time.RFC3339)
		meta := node.Meta{ExtractorVersion: cfg.ExtractorVersion, ExtractedAt: extractedAt}
		var content floor.Content
		var extractErrors []error
		for _, scanContext := range scanContexts {
			clusterNodeID := clusterNodeIDFromContext(scanContext)
			client, err := k8s.NewClient(cfg.Kubeconfig, scanContext)
			if err != nil {
				extractErrors = append(extractErrors, fmt.Errorf("context %q: %w", scanContext, err))
				continue
			}
			res := k8s.NewEdgeResolver(k8s.EdgeResolverWithIstioLister(k8s.NewIstioCRDLister(client)))
			reader := k8s.NewReader(client)
			lister := k8s.NewListerForNamespaces(reader, cfg.Namespaces)
			tctx := k8s.K8sTranslateContext{
				Floor:         0,
				Meta:          meta,
				ClusterNodeID: clusterNodeID,
			}
			partial, err := k8s.Floor0Extractor(cmd.Context(), lister, trans, res, tctx)
			if err != nil {
				extractErrors = append(extractErrors, fmt.Errorf("context %q: %w", scanContext, err))
				continue
			}
			content.Nodes = append(content.Nodes, partial.Nodes...)
			content.Edges = append(content.Edges, partial.Edges...)
		}
		if len(extractErrors) > 0 {
			return fmt.Errorf("kubernetes extraction failed for %d of %d context(s): %w",
				len(extractErrors), len(scanContexts), errors.Join(extractErrors...))
		}
		if len(content.Nodes) == 0 {
			return fmt.Errorf("no cluster data extracted from %d context(s)", len(scanContexts))
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
		if err := config.ValidateAutoArrangement(cfg); err != nil {
			return err
		}
		var payload []byte
		var specDoc *graph.BGSpecDocument
		if cfg.WholeDocument {
			doc := k8s.NewBGSpecDocument(content)
			if config.AutoArrangementStyleMapEnabled(cfg) {
				styleMap := floor.AutoArrangeStyleMap(content)
				doc.StyleMap = &styleMap
			}
			specDoc = &doc
			payload, err = graph.MarshalBGSpecJSON(doc)
		} else {
			payload, err = graph.MarshalFloorContentJSON(content)
		}
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		if err := writeLocalOutput(cfg.LocalOutput, payload); err != nil {
			return err
		}
		if cfg.Output == config.OutputPush {
			return runPushMode(cmd.Context(), cfg, pushDryRun, payload, content, specDoc, os.Stderr, push.NewHTTPPusher(nil))
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
		"Kubeconfig context name (default: all kind-* contexts, or current context if none)")
	flagSet.StringSliceVar(&cfg.Namespaces, "namespaces", nil,
		"Comma-separated namespaces to scan (default: all accessible)")
	flagSet.StringVarP(&cfg.Output, "output", "o", "-",
		`Output path for the BGSpec JSON document ("-" for stdout)`)
	flagSet.StringVar(&cfg.ExtractorVersion, "extractor-version", config.DefaultExtractorVersion,
		"Value stamped into node/edge meta.extractorVersion")
	flagSet.BoolVar(&cfg.WholeDocument, "whole-document", false,
		"Emit full BGSpec document (floors 0–3) instead of floor 0 slice only")
	flagSet.StringVar(&cfg.AutoArrangement, "auto-arrangement", config.AutoArrangementDefault,
		`Node layout for whole-document output: "default" (auto grid) or "none" (omit styleMap); only with --whole-document`)
	flagSet.StringVar(&cfg.Endpoint, "endpoint", "", "Backend endpoint URL (push mode)")
	flagSet.StringVar(&cfg.Org, "org", "", "Organization slug (push mode)")
	flagSet.StringVar(&cfg.Document, "document", "", "Document slug (push mode)")
	flagSet.StringVar(&cfg.Cluster, "cluster", "", "Cluster slug (push mode)")
	flagSet.StringVar(&cfg.APIKey, "api-key", "", "Bearer API key, format bg_sk_* (push mode)")
	flagSet.StringVar(&cfg.Idempotency, "idempotency", "", "Idempotency-Key header (defaults to content-hash)")
	flagSet.BoolVar(&pushDryRun, "push-dry-run", false, "Resolve and print the push request without sending it")
	flagSet.StringVar(&cfg.LocalOutput, "local-output", "",
		"Also write the BGSpec JSON to this path (works with output: push)")
}

func writeLocalOutput(path string, payload []byte) error {
	if path == "" {
		return nil
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write local output %q: %w", path, err)
	}
	return nil
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
	if !fs.Changed("auto-arrangement") {
		cfg.AutoArrangement = fileCfg.AutoArrangement
	}
	if !fs.Changed("endpoint") {
		cfg.Endpoint = fileCfg.Endpoint
	}
	if !fs.Changed("org") {
		cfg.Org = fileCfg.Org
	}
	if !fs.Changed("document") {
		cfg.Document = fileCfg.Document
	}
	if !fs.Changed("cluster") {
		cfg.Cluster = fileCfg.Cluster
	}
	if !fs.Changed("api-key") {
		cfg.APIKey = fileCfg.APIKey
	}
	if !fs.Changed("idempotency") {
		cfg.Idempotency = fileCfg.Idempotency
	}
	if !fs.Changed("local-output") {
		cfg.LocalOutput = fileCfg.LocalOutput
	}
	cfg.LocalValidate = fileCfg.LocalValidate
	return nil
}

func Execute() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadKubeconfigContextNames(kubeconfigPath string) ([]string, string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}
	raw, err := loadingRules.Load()
	if err != nil {
		return nil, "", err
	}
	names := make([]string, 0, len(raw.Contexts))
	for name := range raw.Contexts {
		names = append(names, name)
	}
	return names, raw.CurrentContext, nil
}

func resolveScanContexts(explicit, current string, allNames []string) []string {
	if explicit != "" {
		return []string{explicit}
	}
	kindContexts := kindContextNames(allNames)
	if len(kindContexts) > 0 {
		return kindContexts
	}
	if current != "" {
		return []string{current}
	}
	return nil
}

func kindContextNames(allNames []string) []string {
	var names []string
	for _, name := range allNames {
		if strings.HasPrefix(name, "kind-") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func clusterNodeIDFromContext(contextName string) string {
	label := strings.TrimPrefix(contextName, "kind-")
	if label == "" {
		label = contextName
	}
	return "cluster/" + label
}

type unreachableContext struct {
	name      string
	serverURL string
	reason    string
}

func filterReachableContexts(ctx context.Context, kubeconfig string, contexts []string) ([]string, []unreachableContext) {
	reachable := make([]string, 0, len(contexts))
	var unreachable []unreachableContext
	for _, scanContext := range contexts {
		reachErr := kubeContextReachable(ctx, kubeconfig, scanContext)
		if reachErr == nil {
			reachable = append(reachable, scanContext)
			continue
		}
		serverURL := clusterServerURL(kubeconfig, scanContext)
		unreachable = append(unreachable, unreachableContext{
			name:      scanContext,
			serverURL: serverURL,
			reason:    reachErr.Error(),
		})
		fmt.Fprintf(os.Stderr, "bgscan: ignoring unreachable context %q\n", scanContext)
	}
	return reachable, unreachable
}

func allLocalhostConnectionRefused(unreachable []unreachableContext) bool {
	if len(unreachable) == 0 {
		return false
	}
	for _, entry := range unreachable {
		if !strings.Contains(entry.serverURL, "127.0.0.1") && !strings.Contains(entry.serverURL, "localhost") {
			return false
		}
		if !strings.Contains(entry.reason, "connection refused") {
			return false
		}
	}
	return true
}

func kubeContextReachable(ctx context.Context, kubeconfig, scanContext string) error {
	client, err := k8s.NewClient(kubeconfig, scanContext)
	if err != nil {
		return err
	}
	reader := k8s.NewReader(client)
	_, listErr := reader.ListNamespaces(ctx)
	return listErr
}

func clusterServerURL(kubeconfigPath, contextName string) string {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}
	raw, err := loadingRules.Load()
	if err != nil {
		return ""
	}
	ctxCfg, ok := raw.Contexts[contextName]
	if !ok || ctxCfg == nil {
		return ""
	}
	cluster, ok := raw.Clusters[ctxCfg.Cluster]
	if !ok || cluster == nil {
		return ""
	}
	return cluster.Server
}
