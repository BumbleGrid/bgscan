package k8s

import (
	"context"
	"testing"

	"github.com/BumbleGrid/bgscan/internal/mapper"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestReader_K8sListerInterface(t *testing.T) {
	var _ mapper.K8sLister = (*Reader)(nil)
}

func TestReader_unconfiguredReturnsError(t *testing.T) {
	ctx := context.Background()
	r := NewReader(nil)
	if _, err := r.ListNamespaces(ctx); err == nil {
		t.Fatal("expected error when client is nil")
	}
	r2 := &Reader{}
	if _, err := r2.ListDeployments(ctx, "default"); err == nil {
		t.Fatal("expected error when clientset is nil")
	}
}

func TestReader_listMethods_fakeClientset(t *testing.T) {
	ctx := context.Background()
	nsName := "demo"

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	pv := &corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "pv1"}}
	ingClass := &networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: "public"}}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: nsName},
	}
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: nsName},
	}
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "agent", Namespace: nsName},
	}
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "web-rs", Namespace: nsName},
	}

	cj := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: "tick", Namespace: nsName},
	}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: "once", Namespace: nsName},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: nsName},
	}
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: nsName},
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cfg", Namespace: nsName},
	}
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "tok", Namespace: nsName},
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: nsName},
	}
	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "default-deny", Namespace: nsName},
	}
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: nsName},
	}

	// client-go's fake.NewSimpleClientset wires CoreV1, AppsV1, BatchV1, NetworkingV1,
	// AutoscalingV2, etc., so list calls exercise the same code paths as production.
	cs := fake.NewSimpleClientset(
		ns, pv, ingClass,
		deploy, sts, ds, rs, cj, job,
		svc, cm, sec, pvc, ing, np, hpa,
	)

	r := NewReaderWithClientset(cs)

	t.Run("ListNamespaces", func(t *testing.T) {
		items, err := r.ListNamespaces(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != nsName {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListPersistentVolumes", func(t *testing.T) {
		items, err := r.ListPersistentVolumes(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "pv1" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListIngressClasses", func(t *testing.T) {
		items, err := r.ListIngressClasses(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "public" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListDeployments", func(t *testing.T) {
		items, err := r.ListDeployments(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "web" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListStatefulSets", func(t *testing.T) {
		items, err := r.ListStatefulSets(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "db" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListDaemonSets", func(t *testing.T) {
		items, err := r.ListDaemonSets(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "agent" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListReplicaSets", func(t *testing.T) {
		items, err := r.ListReplicaSets(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "web-rs" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListCronJobs", func(t *testing.T) {
		items, err := r.ListCronJobs(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "tick" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListJobs", func(t *testing.T) {
		items, err := r.ListJobs(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "once" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListServices", func(t *testing.T) {
		items, err := r.ListServices(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "api" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListIngresses", func(t *testing.T) {
		items, err := r.ListIngresses(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "edge" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListConfigMaps", func(t *testing.T) {
		items, err := r.ListConfigMaps(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "cfg" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListSecrets", func(t *testing.T) {
		items, err := r.ListSecrets(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "tok" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListPersistentVolumeClaims", func(t *testing.T) {
		items, err := r.ListPersistentVolumeClaims(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "data" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListNetworkPolicies", func(t *testing.T) {
		items, err := r.ListNetworkPolicies(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "default-deny" {
			t.Fatalf("got %#v", items)
		}
	})
	t.Run("ListHorizontalPodAutoscalersV2", func(t *testing.T) {
		items, err := r.ListHorizontalPodAutoscalersV2(ctx, nsName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "web" {
			t.Fatalf("got %#v", items)
		}
	})
}
