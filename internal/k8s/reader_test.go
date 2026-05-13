package k8s

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestReader_unconfiguredReturnsError(test *testing.T) {
	ctx := context.Background()
	reader := NewReader(nil)
	if _, err := reader.ListNamespaces(ctx); err == nil {
		test.Fatal("expected error when client is nil")
	}
	unconfigured := &Reader{}
	if _, err := unconfigured.ListDeployments(ctx, "default"); err == nil {
		test.Fatal("expected error when clientset is nil")
	}
}

func TestReader_listMethods_fakeClientset(test *testing.T) {
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

	cs := fake.NewSimpleClientset(
		ns, pv, ingClass,
		deploy, sts, ds, rs, cj, job,
		svc, cm, sec, pvc, ing, np, hpa,
	)

	reader := NewReaderWithClientset(cs)

	test.Run("ListNamespaces", func(sub *testing.T) {
		items, err := reader.ListNamespaces(ctx)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != nsName {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListPersistentVolumes", func(sub *testing.T) {
		items, err := reader.ListPersistentVolumes(ctx)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "pv1" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListIngressClasses", func(sub *testing.T) {
		items, err := reader.ListIngressClasses(ctx)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "public" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListDeployments", func(sub *testing.T) {
		items, err := reader.ListDeployments(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "web" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListStatefulSets", func(sub *testing.T) {
		items, err := reader.ListStatefulSets(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "db" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListDaemonSets", func(sub *testing.T) {
		items, err := reader.ListDaemonSets(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "agent" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListReplicaSets", func(sub *testing.T) {
		items, err := reader.ListReplicaSets(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "web-rs" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListCronJobs", func(sub *testing.T) {
		items, err := reader.ListCronJobs(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "tick" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListJobs", func(sub *testing.T) {
		items, err := reader.ListJobs(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "once" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListServices", func(sub *testing.T) {
		items, err := reader.ListServices(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "api" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListIngresses", func(sub *testing.T) {
		items, err := reader.ListIngresses(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "edge" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListConfigMaps", func(sub *testing.T) {
		items, err := reader.ListConfigMaps(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "cfg" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListSecrets", func(sub *testing.T) {
		items, err := reader.ListSecrets(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "tok" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListPersistentVolumeClaims", func(sub *testing.T) {
		items, err := reader.ListPersistentVolumeClaims(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "data" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListNetworkPolicies", func(sub *testing.T) {
		items, err := reader.ListNetworkPolicies(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "default-deny" {
			sub.Fatalf("got %#v", items)
		}
	})
	test.Run("ListHorizontalPodAutoscalersV2", func(sub *testing.T) {
		items, err := reader.ListHorizontalPodAutoscalersV2(ctx, nsName)
		if err != nil {
			sub.Fatal(err)
		}
		if len(items) != 1 || items[0].Name != "web" {
			sub.Fatalf("got %#v", items)
		}
	})
}
