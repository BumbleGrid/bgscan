// Package mapper translates raw Kubernetes API objects into BGSpec Floor 0
// node and edge values (see bgspec/floor-0-node.json and floor-0-edge.json).
//
// contracts.go declares the interfaces that node.go and edge.go implement,
// so callers (cmd, future subcommands, tests) depend on behavior rather
// than concrete types.
package mapper

// NodeMapper turns lists of Kubernetes resources into BGSpec Floor 0 node
// values. Implementations must populate data.k8s, data.style, and
// data.meta in line with specs/floor-0-node.json (e.g. position lives
// under style.position, not at the data root).
type NodeMapper interface {
	// TODO: declare per-kind Map* methods (MapDeployments, MapServices,
	// MapIngresses, …) that mirror the Reader's List* surface and return
	// []node.Wrapper from github.com/BumbleGrid/bgbase/node.
}

// EdgeResolver derives BGSpec Floor 0 edges from an already-mapped node
// set. Edges grounded in explicit manifest references (Ingress backends,
// PVC volumeName, CronJob ownerReferences) must set inferred=false;
// label/selector matches must set inferred=true.
type EdgeResolver interface {
	// TODO: ResolveEdges(ctx, []node.Wrapper) ([]edge.Wrapper, error)
}
