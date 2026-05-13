// Package mapper — node.go: Kubernetes → BGSpec Floor 0 node translation.
//
// This file owns the per-kind mappings (Deployment, StatefulSet, Service,
// Ingress, ConfigMap, Secret, PVC, NetworkPolicy, HPA, Namespace, PV,
// IngressClass, …) into node.Wrapper values from bgbase/node, populating
// the kubernetes block from bgextract/document (K8sNode, PortSpec, etc.).
//
// Source of truth for emitted shapes: bgspec/floor-0-node.json. Renderer
// hints (color, shape, position) belong under data.style; operational
// metadata (team, repo, tags, extractedAt) belongs under data.meta.
package mapper

// NodeMapperImpl is the default NodeMapper. It is stateless and safe to
// reuse across scans; per-scan context (namespace scoping, extractor
// version, timestamps) is passed in via method arguments.
type NodeMapperImpl struct{}

// NewNodeMapper returns a ready-to-use NodeMapperImpl.
func NewNodeMapper() *NodeMapperImpl {
	return &NodeMapperImpl{}
}

// TODO: implement the Map* methods declared by NodeMapper, each returning
// []node.Wrapper with bgKind, infraProvider="kubernetes", and a populated
// data.k8s block.
