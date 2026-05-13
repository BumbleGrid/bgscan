// Package k8s — reader.go: typed list operations over a Kubernetes cluster.
//
// Reader is the bgscan equivalent of bgextract's K8sLister: it returns
// raw Kubernetes API objects with no BGSpec translation. Translation is
// the mapper package's job.
package k8s

// Reader lists Kubernetes resources used to build a BGSpec Floor 0 graph
// (workloads, services, ingresses, config/secret sources, PVCs, network
// policies, autoscalers, namespaces, persistent volumes, ingress classes).
type Reader struct {
	// TODO: embed *Client (or hold kubernetes.Interface) so each List*
	// method can dispatch against the right typed clientset group.
}

// NewReader returns a Reader bound to the given Client.
func NewReader(c *Client) *Reader {
	return &Reader{}
}

// TODO: ListNamespaces, ListDeployments, ListStatefulSets, ListDaemonSets,
// ListReplicaSets, ListCronJobs, ListJobs, ListServices, ListIngresses,
// ListConfigMaps, ListSecrets, ListPersistentVolumeClaims,
// ListPersistentVolumes, ListIngressClasses, ListNetworkPolicies,
// ListHorizontalPodAutoscalersV2 — each scoped to a namespace where
// applicable.
