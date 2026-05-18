package config

const (
	AutoArrangementDefault = "default"
	AutoArrangementNone    = "none"
)

type Config struct {
	Kubeconfig       string
	Context          string
	Namespaces       []string
	Output           string
	ExtractorVersion string
	WholeDocument    bool
	AutoArrangement  string
}

func Load() (*Config, error) {
	return &Config{}, nil
}
