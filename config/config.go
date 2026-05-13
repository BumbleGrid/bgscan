package config

type Config struct {
	Kubeconfig       string
	Context          string
	Namespaces       []string
	Output           string
	ExtractorVersion string
	WholeDocument    bool
}

func Load() (*Config, error) {
	return &Config{}, nil
}
