package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// BGConfigFileName is the default config filename searched next to the binary.
const BGConfigFileName = "bgconfig.yaml"

var requiredBGConfigKeys = []string{
	"kubeconfig",
	"context",
	"namespaces",
	"output",
	"extractor_version",
}

type bgConfigFile struct {
	Kubeconfig       string   `yaml:"kubeconfig"`
	Context          string   `yaml:"context"`
	Namespaces       []string `yaml:"namespaces"`
	Output           string   `yaml:"output"`
	ExtractorVersion string   `yaml:"extractor_version"`
}

// LoadBGConfig reads path, ensures every expected key is present in the document,
// and returns a Config. Omitted keys are rejected even if YAML would treat them as zero.
func LoadBGConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("bgconfig %s: %w", path, err)
	}
	for _, k := range requiredBGConfigKeys {
		if _, ok := raw[k]; !ok {
			return Config{}, fmt.Errorf("bgconfig %s: missing required key %q (need all of: %v)", path, k, requiredBGConfigKeys)
		}
	}
	var f bgConfigFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return Config{}, fmt.Errorf("bgconfig %s: %w", path, err)
	}
	return Config{
		Kubeconfig:       f.Kubeconfig,
		Context:          f.Context,
		Namespaces:       f.Namespaces,
		Output:           f.Output,
		ExtractorVersion: f.ExtractorVersion,
	}, nil
}

// ResolveBGConfigPath returns an explicit --config path unchanged, or the path
// to BGConfigFileName beside the executable when that file exists. When no file
// applies, it returns ("", nil).
func ResolveBGConfigPath(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Clean(explicit), nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	cand := filepath.Join(filepath.Dir(exe), BGConfigFileName)
	st, err := os.Stat(cand)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if st.IsDir() {
		return "", nil
	}
	return cand, nil
}
