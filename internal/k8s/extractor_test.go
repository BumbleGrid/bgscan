package k8s

import (
	"context"
	"testing"

	"github.com/BumbleGrid/bgbase/node"
	"github.com/BumbleGrid/bgscan/internal/mapper"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestExtractor_fakeClientset(t *testing.T) {
	ctx := context.Background()
	nsName := "demo"

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	pv := &corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "pv1"}}
	ingClass := &networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: "public"}}
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: nsName},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: nsName},
	}

	cs := fake.NewSimpleClientset(ns, pv, ingClass, deploy, svc)
	reader := NewReaderWithClientset(cs)
	trans := mapper.NewNodeTranslator()
	res := mapper.NewEdgeResolver()
	tctx := mapper.K8sTranslateContext{
		Floor:         0,
		Meta:          node.Meta{ExtractorVersion: "test"},
		ClusterNodeID: "cluster/main",
	}

	content, err := Floor0Extractor(ctx, reader, trans, res, tctx)
	if err != nil {
		t.Fatal(err)
	}
	if content.Floor != 0 {
		t.Fatalf("Floor = %d", content.Floor)
	}
	if len(content.Nodes) < 5 {
		t.Fatalf("expected at least 5 nodes, got %d", len(content.Nodes))
	}
}
