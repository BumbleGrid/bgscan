package config

type Config struct {
	Kubeconfig       string
	Context          string
	Namespaces       []string
	Output           string
	ExtractorVersion string
}

func Load() (*Config, error) {
	return &Config{}, nil
}
