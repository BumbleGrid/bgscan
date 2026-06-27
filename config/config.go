package config

const (
	AutoArrangementDefault = "default"
	AutoArrangementNone    = "none"

	OutputPush = "push"
)

var DefaultExtractorVersion = "__BGSCAN_VERSION__"

type Config struct {
	Kubeconfig       string
	Context          string
	Namespaces       []string
	IgnoreNamespaces []string
	IgnoreWorkloads  []string
	Output           string
	ExtractorVersion string
	WholeDocument    bool
	AutoArrangement  string

	Endpoint      string
	Org           string
	Document      string
	Cluster       string
	APIKey        string
	Idempotency   string
	LocalValidate bool
	LocalOutput   string
}

func Load() (*Config, error) {
	return &Config{}, nil
}
