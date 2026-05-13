// Package k8s constructs Kubernetes API clients and exposes typed readers
// over the cluster. It is the only package in bgscan that depends on
// client-go; everything downstream consumes plain Go slices of API types.
package k8s

// Client is the bgscan-facing handle to a Kubernetes cluster. It wraps a
// kubernetes.Interface (clientset) plus the configuration needed to scope
// reads (namespaces, kubeconfig context, request timeouts).
type Client struct {
	// TODO: hold *rest.Config, kubernetes.Interface, default namespace,
	// and per-call timeouts here.
}

// NewClient builds a Client from the provided kubeconfig path and context
// name. An empty kubeconfigPath falls back to in-cluster config and then
// to the default loading rules ($KUBECONFIG, ~/.kube/config).
func NewClient(kubeconfigPath, contextName string) (*Client, error) {
	// TODO: build *rest.Config via clientcmd / rest.InClusterConfig and
	// instantiate kubernetes.NewForConfig.
	return nil, nil
}
