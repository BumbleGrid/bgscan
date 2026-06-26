package cmd

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

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
