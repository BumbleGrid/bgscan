// Package mapper translates raw Kubernetes API objects into BGSpec Floor 0
// node and edge values defined by github.com/BumbleGrid/bgbase
// (node.Data, node.Meta, edge.Data, …). The interfaces declared here let
// callers (cmd, future subcommands, tests) depend on behavior rather than
// concrete types.
//
// Source of truth for emitted shapes: bgspec/floor-0-node.json and
// bgspec/floor-0-edge.json. Renderer hints (color, shape, position) belong
// under data.style; operational metadata under data.meta.
package mapper

import (
	"context"

	"github.com/BumbleGrid/bgbase/edge"
	"github.com/BumbleGrid/bgbase/node"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// K8sTranslateContext carries floor, extractor meta, and graph parent ids
// needed to populate Parent, Meta, and namespaced K8s fields when turning
// API objects into node.Data values.
type K8sTranslateContext struct {
	// Floor is the BGSpec floor index assigned to every emitted node
	// (Floor 0 for raw Kubernetes resources).
	Floor int

	// Meta is the per-run node.Meta block (extractedAt, extractorVersion,
	// optional team/repo/etc.) stamped onto every emitted node.
	Meta node.Meta

	// ClusterNodeID is the Floor 0 id of the synthetic cluster root
	// (parent of Namespace nodes and other cluster-scoped children such
	// as IngressClass).
	ClusterNodeID string

	// NamespaceName is the Kubernetes namespace for a namespaced batch
	// (e.g. workloads listed under one namespace). Leave empty when
	// translating cluster-scoped lists only.
	NamespaceName string

	// NamespaceParentNodeID is the Floor 0 id of the Namespace node that
	// should parent namespaced resources in this batch. Leave empty for
	// cluster-scoped translators.
	NamespaceParentNodeID string
}

// K8sLister reads raw Kubernetes resources from the cluster and returns
// API objects only, with no BGSpec translation.
type K8sLister interface {
	ListNamespaces(ctx context.Context) ([]corev1.Namespace, error)
	ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error)
	ListIngressClasses(ctx context.Context) ([]networkingv1.IngressClass, error)

	ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error)
	ListStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error)
	ListDaemonSets(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error)
	ListReplicaSets(ctx context.Context, namespace string) ([]appsv1.ReplicaSet, error)

	ListCronJobs(ctx context.Context, namespace string) ([]batchv1.CronJob, error)
	ListJobs(ctx context.Context, namespace string) ([]batchv1.Job, error)

	ListServices(ctx context.Context, namespace string) ([]corev1.Service, error)
	ListIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error)

	ListConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error)
	ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error)
	ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error)

	ListNetworkPolicies(ctx context.Context, namespace string) ([]networkingv1.NetworkPolicy, error)

	ListHorizontalPodAutoscalersV2(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error)
}

// K8sNodeTranslator turns raw Kubernetes slices into BGSpec Floor 0
// node.Data values. Implementations must populate data.k8s and data.meta
// in line with specs/floor-0-node.json (e.g. position lives under
// style.position, not at the data root). Leave data.style at its zero
// value for a later layout pass.
type K8sNodeTranslator interface {
	TranslateNamespaces(ctx context.Context, tctx K8sTranslateContext, items []corev1.Namespace) ([]node.Data, error)
	TranslatePersistentVolumes(ctx context.Context, tctx K8sTranslateContext, items []corev1.PersistentVolume) ([]node.Data, error)
	TranslateIngressClasses(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.IngressClass) ([]node.Data, error)

	TranslateDeployments(ctx context.Context, tctx K8sTranslateContext, items []appsv1.Deployment) ([]node.Data, error)
	TranslateStatefulSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.StatefulSet) ([]node.Data, error)
	TranslateDaemonSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.DaemonSet) ([]node.Data, error)
	TranslateReplicaSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.ReplicaSet) ([]node.Data, error)

	TranslateCronJobs(ctx context.Context, tctx K8sTranslateContext, items []batchv1.CronJob) ([]node.Data, error)
	TranslateJobs(ctx context.Context, tctx K8sTranslateContext, items []batchv1.Job) ([]node.Data, error)

	TranslateServices(ctx context.Context, tctx K8sTranslateContext, items []corev1.Service) ([]node.Data, error)
	TranslateIngresses(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.Ingress) ([]node.Data, error)

	TranslateConfigMaps(ctx context.Context, tctx K8sTranslateContext, items []corev1.ConfigMap) ([]node.Data, error)
	TranslateSecrets(ctx context.Context, tctx K8sTranslateContext, items []corev1.Secret) ([]node.Data, error)
	TranslatePersistentVolumeClaims(ctx context.Context, tctx K8sTranslateContext, items []corev1.PersistentVolumeClaim) ([]node.Data, error)

	TranslateNetworkPolicies(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.NetworkPolicy) ([]node.Data, error)

	TranslateHorizontalPodAutoscalersV2(ctx context.Context, tctx K8sTranslateContext, items []autoscalingv2.HorizontalPodAutoscaler) ([]node.Data, error)
}

// K8sEdgeResolver builds the complete Floor 0 edge list after translation
// has produced every node, using that node slice alone.
type K8sEdgeResolver interface {
	// ResolveEdges emits edge.Data values from the assembled nodes; leave
	// Style at its zero value for a later styling pass. Set Inferred to
	// true for edges derived from label selector matching, and to false
	// for edges grounded in explicit manifest references such as an
	// Ingress rule backend or a PVC volumeName.
	ResolveEdges(ctx context.Context, nodes []node.Data) ([]edge.Data, error)
}
