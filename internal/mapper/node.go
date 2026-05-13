// Package mapper — node.go: Kubernetes → BGSpec Floor 0 node translation.
//
// This file owns the per-kind mappings (Deployment, StatefulSet, Service,
// Ingress, ConfigMap, Secret, PVC, NetworkPolicy, HPA, Namespace, PV,
// IngressClass, …) into node.Data values from bgbase/node, populating
// the kubernetes block from bgextract/document (K8sNode, PortSpec, etc.).
//
// Source of truth for emitted shapes: bgspec/floor-0-node.json. Renderer
// hints (color, shape, position) belong under data.style; operational
// metadata (team, repo, tags, extractedAt) belongs under data.meta.
package mapper

// NodeTranslator is the default K8sNodeTranslator. It is stateless and
// safe to reuse across scans; per-scan context (namespace scoping,
// extractor version, timestamps) is passed in via K8sTranslateContext.
type NodeTranslator struct{}

// NewNodeTranslator returns a ready-to-use NodeTranslator.
func NewNodeTranslator() *NodeTranslator {
	return &NodeTranslator{}
}

// TODO: implement the Translate* methods declared by K8sNodeTranslator,
// each returning []node.Data with bgKind, infraProvider="kubernetes",
// and a populated data.k8s block.
