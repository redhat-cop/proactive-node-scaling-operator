package controllers

import (
	"testing"

	redhatcopv1alpha1 "github.com/redhat-cop/proactive-node-scaling-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetNeededReplicas(t *testing.T) {
	r := &NodeScalingWatermarkReconciler{}
	watermark := &redhatcopv1alpha1.NodeScalingWatermark{
		Spec: redhatcopv1alpha1.NodeScalingWatermarkSpec{
			WatermarkPercentage: 20,
			PausePodSize:        corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m"), corev1.ResourceMemory: resource.MustParse("200Mi")},
		},
	}
	for _, tt := range []struct {
		name, cpu, memory string
		want              int64
	}{
		{"empty workload", "0", "0", 0},
		{"CPU determines replicas", "10", "1Gi", 10},
		{"memory determines replicas", "1", "10000Mi", 10},
		{"fractional replica is truncated", "1500m", "0", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(tt.cpu), corev1.ResourceMemory: resource.MustParse(tt.memory)}
			if got := r.getNeededReplicas(watermark, requests); got != tt.want {
				t.Fatalf("replicas = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSumRequests(t *testing.T) {
	container := func(cpu string) corev1.Container {
		return corev1.Container{Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu)}}}
	}
	pods := []corev1.Pod{
		{Spec: corev1.PodSpec{Containers: []corev1.Container{container("250m"), container("500m")}}},
		{Spec: corev1.PodSpec{Containers: []corev1.Container{container("1")}}},
	}
	r := &NodeScalingWatermarkReconciler{}
	total := r.sumRequests(pods)
	cpu := total[corev1.ResourceCPU]
	if cpu.Cmp(resource.MustParse("1750m")) != 0 {
		t.Fatalf("CPU total = %s, want 1750m", cpu.String())
	}
	original := pods[0].Spec.Containers[0].Resources.Requests[corev1.ResourceCPU]
	if original.Cmp(resource.MustParse("250m")) != 0 {
		t.Fatal("input pod was mutated")
	}
}

func TestFilterWatermarkAndSystemPods(t *testing.T) {
	deleted := metav1.Now()
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "user", Namespace: "app"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "system", Namespace: "kube-system"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "openshift", Namespace: "openshift-monitoring"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "watermark", Namespace: "app", Labels: map[string]string{watermarkLabel: ""}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "deleting", Namespace: "app", DeletionTimestamp: &deleted}},
	}
	r := &NodeScalingWatermarkReconciler{}
	got := r.filterWatermarkAndSystemPods(pods)
	if len(got) != 1 || got[0].Name != "user" {
		t.Fatalf("expected only user pod, got %v", got)
	}
}
