// Package k8s — reader.go: typed list operations over a Kubernetes cluster.
//
// Reader is the bgscan equivalent of bgextract's K8sLister: it returns
// raw Kubernetes API objects with no BGSpec translation. Translation is
// the mapper package's job.
package k8s

import (
	"context"
	"fmt"

	"github.com/BumbleGrid/bgscan/internal/mapper"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Compile-time check that Reader implements mapper.K8sLister.
var _ mapper.K8sLister = (*Reader)(nil)

// Reader lists Kubernetes resources used to build a BGSpec Floor 0 graph
// (workloads, services, ingresses, config/secret sources, PVCs, network
// policies, autoscalers, namespaces, persistent volumes, ingress classes).
type Reader struct {
	cs kubernetes.Interface
}

// NewReader returns a Reader bound to the given Client.
func NewReader(c *Client) *Reader {
	if c == nil {
		return &Reader{}
	}
	return &Reader{cs: c.cs}
}

// NewReaderWithClientset returns a Reader backed by cs. It is intended for
// tests that inject client-go's fake.NewSimpleClientset (or any
// kubernetes.Interface implementation).
func NewReaderWithClientset(cs kubernetes.Interface) *Reader {
	if cs == nil {
		return &Reader{}
	}
	return &Reader{cs: cs}
}

func (r *Reader) requireClientset() (kubernetes.Interface, error) {
	if r == nil || r.cs == nil {
		return nil, fmt.Errorf("k8s reader: kubernetes clientset is not configured")
	}
	return r.cs, nil
}

// ListNamespaces implements mapper.K8sLister.
func (r *Reader) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListPersistentVolumes implements mapper.K8sLister.
func (r *Reader) ListPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListIngressClasses implements mapper.K8sLister.
func (r *Reader) ListIngressClasses(ctx context.Context) ([]networkingv1.IngressClass, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListDeployments implements mapper.K8sLister.
func (r *Reader) ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListStatefulSets implements mapper.K8sLister.
func (r *Reader) ListStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListDaemonSets implements mapper.K8sLister.
func (r *Reader) ListDaemonSets(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListReplicaSets implements mapper.K8sLister.
func (r *Reader) ListReplicaSets(ctx context.Context, namespace string) ([]appsv1.ReplicaSet, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListCronJobs implements mapper.K8sLister.
func (r *Reader) ListCronJobs(ctx context.Context, namespace string) ([]batchv1.CronJob, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListJobs implements mapper.K8sLister.
func (r *Reader) ListJobs(ctx context.Context, namespace string) ([]batchv1.Job, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListServices implements mapper.K8sLister.
func (r *Reader) ListServices(ctx context.Context, namespace string) ([]corev1.Service, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListIngresses implements mapper.K8sLister.
func (r *Reader) ListIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListConfigMaps implements mapper.K8sLister.
func (r *Reader) ListConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListSecrets implements mapper.K8sLister.
func (r *Reader) ListSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListPersistentVolumeClaims implements mapper.K8sLister.
func (r *Reader) ListPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListNetworkPolicies implements mapper.K8sLister.
func (r *Reader) ListNetworkPolicies(ctx context.Context, namespace string) ([]networkingv1.NetworkPolicy, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ListHorizontalPodAutoscalersV2 implements mapper.K8sLister.
func (r *Reader) ListHorizontalPodAutoscalersV2(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error) {
	cs, err := r.requireClientset()
	if err != nil {
		return nil, err
	}
	list, err := cs.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
