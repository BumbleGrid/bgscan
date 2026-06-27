package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BumbleGrid/bgscan/config"
	"github.com/spf13/cobra"
)

func testApplyBGConfig(t *testing.T, args ...string) {
	t.Helper()
	cfg = config.Config{}
	configPath = ""
	cmd := &cobra.Command{}
	registerScanFlags(cmd.PersistentFlags())
	if len(args) > 0 {
		if err := cmd.ParseFlags(args); err != nil {
			t.Fatal(err)
		}
	}
	if err := applyBGConfig(cmd); err != nil {
		t.Fatal(err)
	}
}

func TestApplyBGConfig_ignoreFlagsFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: /tmp/kube
context: dev
namespaces: [apps]
output: "-"
extractor_version: "1.0.0"
ignore_namespaces: [kube-system]
ignore_workloads: ["apps/deployments/noise*"]
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	testApplyBGConfig(t, "--config", path)
	if !reflect.DeepEqual(cfg.IgnoreNamespaces, []string{"kube-system"}) {
		t.Fatalf("IgnoreNamespaces = %v", cfg.IgnoreNamespaces)
	}
	if !reflect.DeepEqual(cfg.IgnoreWorkloads, []string{"apps/deployments/noise*"}) {
		t.Fatalf("IgnoreWorkloads = %v", cfg.IgnoreWorkloads)
	}
}

func TestApplyBGConfig_ignoreFlagsCLIOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bgconfig.yaml")
	content := `kubeconfig: /tmp/kube
context: dev
namespaces: []
output: "-"
extractor_version: "1.0.0"
ignore_namespaces: [kube-system]
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	testApplyBGConfig(t, "--config", path, "--ignore-namespaces=custom")
	if !reflect.DeepEqual(cfg.IgnoreNamespaces, []string{"custom"}) {
		t.Fatalf("IgnoreNamespaces = %v", cfg.IgnoreNamespaces)
	}
}

func TestApplyBGConfig_noConfigUsesDefaultIgnoreNamespaces(t *testing.T) {
	testApplyBGConfig(t)
	defaults, err := config.LoadDefaultScanFilters()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.IgnoreNamespaces, defaults.IgnoreNamespaces) {
		t.Fatalf("IgnoreNamespaces = %v, want defaults %v", cfg.IgnoreNamespaces, defaults.IgnoreNamespaces)
	}
}

func TestResolveScanContexts_inClusterUsesImplicitContext(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	got := resolveScanContexts("", "", nil)
	want := []string{""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveScanContexts() = %v, want %v", got, want)
	}
}

func TestResolveScanContexts_nonKindContextsWithoutInClusterReturnsNil(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	got := resolveScanContexts("", "", []string{"prod", "staging"})
	if got != nil {
		t.Fatalf("resolveScanContexts() = %v, want nil", got)
	}
}

func TestResolveScanContexts_prefersKindContexts(t *testing.T) {
	got := resolveScanContexts("", "", []string{"prod", "kind-dev", "kind-staging"})
	want := []string{"kind-dev", "kind-staging"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveScanContexts() = %v, want %v", got, want)
	}
}

func TestClusterNodeIDForScan_usesClusterSlugWhenContextEmpty(t *testing.T) {
	got := clusterNodeIDForScan("", "complex")
	want := "cluster/complex"
	if got != want {
		t.Fatalf("clusterNodeIDForScan() = %q, want %q", got, want)
	}
}

func TestRootHelp_listsRunSubcommand(t *testing.T) {
	rootCmd.SetArgs([]string{"--help"})
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "run") {
		t.Fatalf("root help missing run subcommand:\n%s", out)
	}
}

func TestRunHelp_listsScanFlags(t *testing.T) {
	rootCmd.SetArgs([]string{"run", "--help"})
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, flag := range []string{"--config", "--kubeconfig", "--push-dry-run", "--output"} {
		if !strings.Contains(out, flag) {
			t.Fatalf("run help missing %q:\n%s", flag, out)
		}
	}
}
