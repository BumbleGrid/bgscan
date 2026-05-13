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

// EdgeResolver is the default K8sEdgeResolver. It operates on the full
// node slice produced by a K8sNodeTranslator and emits one edge.Data per
// resolved relationship.
type EdgeResolver struct{}

// NewEdgeResolver returns a ready-to-use EdgeResolver.
func NewEdgeResolver() *EdgeResolver {
	return &EdgeResolver{}
}

// TODO: implement ResolveEdges per the inference rules above, sourcing
// hints from bgextract/document.K8sResolverHints attached to each node
// during translation.
