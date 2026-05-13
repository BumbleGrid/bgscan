// Package k8s constructs Kubernetes API clients and exposes typed readers
// over the cluster. It is the only package in bgscan that depends on
// client-go; everything downstream consumes plain Go slices of API types.
package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is the bgscan-facing handle to a Kubernetes cluster. It wraps a
// kubernetes.Interface (clientset) plus the configuration needed to scope
// reads (namespaces, kubeconfig context, request timeouts).
type Client struct {
	cs kubernetes.Interface
}

// NewClient builds a Client from the provided kubeconfig path and context
// name. An empty kubeconfigPath uses the default loading rules ($KUBECONFIG,
// ~/.kube/config) together with optional in-cluster overrides, matching
// kubectl behavior.
func NewClient(kubeconfigPath, contextName string) (*Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}
	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	restConfig, err := cc.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build rest config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}
	return &Client{cs: clientset}, nil
}
