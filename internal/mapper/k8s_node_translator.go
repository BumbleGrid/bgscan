package mapper

import (
	"context"
	"fmt"

	"github.com/BumbleGrid/bgbase/node"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

func k8sRESTID(clusterNodeID, restPath string) string {
	return fmt.Sprintf("%s/k8s/%s", clusterNodeID, restPath)
}

func nodeFromK8s(
	tctx K8sTranslateContext,
	restPath, label string,
	bgKind node.BgKind,
	parent *string,
) node.Data {
	metaCopy := tctx.Meta
	return node.Data{
		ID:            k8sRESTID(tctx.ClusterNodeID, restPath),
		Label:         label,
		Floor:         tctx.Floor,
		BgKind:        bgKind,
		Parent:        parent,
		InfraProvider: node.InfraProviderKubernetes,
		Meta:          &metaCopy,
	}
}

func (*NodeTranslator) TranslateNamespaces(ctx context.Context, tctx K8sTranslateContext, items []corev1.Namespace) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.ClusterNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s", item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindNamespace, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslatePersistentVolumes(ctx context.Context, tctx K8sTranslateContext, items []corev1.PersistentVolume) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.ClusterNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("persistentvolumes/%s", item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindStorage, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateIngressClasses(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.IngressClass) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.ClusterNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("ingressclasses/%s", item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindGateway, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateDeployments(ctx context.Context, tctx K8sTranslateContext, items []appsv1.Deployment) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/deployments/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindWorkload, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateStatefulSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.StatefulSet) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/statefulsets/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindWorkload, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateDaemonSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.DaemonSet) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/daemonsets/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindWorkload, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateReplicaSets(ctx context.Context, tctx K8sTranslateContext, items []appsv1.ReplicaSet) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/replicasets/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindWorkload, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateCronJobs(ctx context.Context, tctx K8sTranslateContext, items []batchv1.CronJob) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/cronjobs/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindJobRunner, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateJobs(ctx context.Context, tctx K8sTranslateContext, items []batchv1.Job) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/jobs/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindJobRunner, &parent))
	}
	return out, nil
}

func serviceBgKind(svc *corev1.Service) node.BgKind {
	switch svc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		return node.BgKindLoadBalancer
	case corev1.ServiceTypeExternalName:
		return node.BgKindExternalService
	default:
		return node.BgKindServiceDiscovery
	}
}

func (*NodeTranslator) TranslateServices(ctx context.Context, tctx K8sTranslateContext, items []corev1.Service) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/services/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, serviceBgKind(&item), &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateIngresses(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.Ingress) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/ingresses/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindGateway, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateConfigMaps(ctx context.Context, tctx K8sTranslateContext, items []corev1.ConfigMap) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/configmaps/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindConfigSource, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateSecrets(ctx context.Context, tctx K8sTranslateContext, items []corev1.Secret) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/secrets/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindSecretSource, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslatePersistentVolumeClaims(ctx context.Context, tctx K8sTranslateContext, items []corev1.PersistentVolumeClaim) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/persistentvolumeclaims/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindStorage, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateNetworkPolicies(ctx context.Context, tctx K8sTranslateContext, items []networkingv1.NetworkPolicy) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/networkpolicies/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindNetworkPolicy, &parent))
	}
	return out, nil
}

func (*NodeTranslator) TranslateHorizontalPodAutoscalersV2(ctx context.Context, tctx K8sTranslateContext, items []autoscalingv2.HorizontalPodAutoscaler) ([]node.Data, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]node.Data, 0, len(items))
	parent := tctx.NamespaceParentNodeID
	for idx := range items {
		item := items[idx]
		restPath := fmt.Sprintf("namespaces/%s/horizontalpodautoscalers/%s", item.Namespace, item.Name)
		out = append(out, nodeFromK8s(tctx, restPath, item.Name, node.BgKindWorkload, &parent))
	}
	return out, nil
}
