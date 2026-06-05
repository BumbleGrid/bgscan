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
	WholeDocument    bool     `yaml:"whole_document"`
	AutoArrangement  string   `yaml:"auto_arrangement"`

	Endpoint      string `yaml:"endpoint"`
	Org           string `yaml:"org"`
	Document      string `yaml:"document"`
	APIKey        string `yaml:"api_key"`
	Idempotency   string `yaml:"idempotency"`
	LocalValidate *bool  `yaml:"local_validate"`
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
	clusterSlug, err := clusterSlugFromRaw(raw)
	if err != nil {
		return Config{}, fmt.Errorf("bgconfig %s: %w", path, err)
	}
	var parsed bgConfigFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return Config{}, fmt.Errorf("bgconfig %s: %w", path, err)
	}
	localValidate := parsed.LocalValidate != nil && *parsed.LocalValidate
	if parsed.Output == OutputPush && parsed.LocalValidate == nil {
		localValidate = true
	}
	cfg := Config{
		Kubeconfig:       parsed.Kubeconfig,
		Context:          parsed.Context,
		Namespaces:       parsed.Namespaces,
		Output:           parsed.Output,
		ExtractorVersion: parsed.ExtractorVersion,
		WholeDocument:    parsed.WholeDocument,
		AutoArrangement:  parsed.AutoArrangement,
		Endpoint:         parsed.Endpoint,
		Org:              parsed.Org,
		Document:         parsed.Document,
		Cluster:          clusterSlug,
		APIKey:           parsed.APIKey,
		Idempotency:      parsed.Idempotency,
		LocalValidate:    localValidate,
	}
	if err := ValidatePushConfig(cfg); err != nil {
		return Config{}, fmt.Errorf("bgconfig %s: %w", path, err)
	}
	return cfg, nil
}

func clusterSlugFromRaw(raw map[string]interface{}) (string, error) {
	value, ok := raw["cluster"]
	if !ok {
		return "", nil
	}
	switch typed := value.(type) {
	case string:
		return typed, nil
	case map[interface{}]interface{}:
		if slug, ok := typed["slug"].(string); ok && slug != "" {
			return slug, nil
		}
		if name, ok := typed["name"].(string); ok && name != "" {
			return name, nil
		}
		return "", fmt.Errorf("cluster map missing slug and name")
	default:
		return "", fmt.Errorf("cluster has unsupported type %T", value)
	}
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
