// Package mapper — edge.go: BGSpec Floor 0 edge resolution from a mapped
// node set.
//
// Inference rules (must match specs/floor-0-edge.json):
//   - Service.spec.selector vs Workload.metadata.labels  → Exposes (inferred=true)
//   - Ingress rule backends                              → Routes  (inferred=false)
//   - Workload volumes (ConfigMap/Secret/PVC)            → Mounts  (inferred=false)
//   - Job ownerReferences{kind=CronJob}                  → ScheduledBy (inferred=false)
//
// Cross-namespace matches must be rejected unless both endpoints are
// cluster-scoped (no namespace).
package mapper

// EdgeResolverImpl is the default EdgeResolver. It operates on the full
// node slice produced by NodeMapper and emits one edge.Wrapper per
// resolved relationship.
type EdgeResolverImpl struct{}

// NewEdgeResolver returns a ready-to-use EdgeResolverImpl.
func NewEdgeResolver() *EdgeResolverImpl {
	return &EdgeResolverImpl{}
}

// TODO: implement ResolveEdges per the inference rules above, sourcing
// hints from bgextract/document.K8sResolverHints attached to each node
// during mapping.
