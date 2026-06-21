package cmd

import (
	"bytes"
	"strings"
	"testing"
)

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
