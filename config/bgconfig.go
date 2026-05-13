package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

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

func LoadBGConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("bgconfig %s: %w", path, err)
	}
	for _, cfgKey := range requiredBGConfigKeys {
		if _, ok := raw[cfgKey]; !ok {
			return Config{}, fmt.Errorf("bgconfig %s: missing required key %q (need all of: %v)", path, cfgKey, requiredBGConfigKeys)
		}
	}
	var parsed bgConfigFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return Config{}, fmt.Errorf("bgconfig %s: %w", path, err)
	}
	return Config{
		Kubeconfig:       parsed.Kubeconfig,
		Context:          parsed.Context,
		Namespaces:       parsed.Namespaces,
		Output:           parsed.Output,
		ExtractorVersion: parsed.ExtractorVersion,
	}, nil
}

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
	candidate := filepath.Join(filepath.Dir(exe), BGConfigFileName)
	fileInfo, err := os.Stat(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if fileInfo.IsDir() {
		return "", nil
	}
	return candidate, nil
}
