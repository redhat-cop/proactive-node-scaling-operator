//go:build integration
// +build integration

/*
Copyright 2021 Red Hat Community of Practice.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	redhatcopv1alpha1 "github.com/redhat-cop/proactive-node-scaling-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// TestAPIs verifies the checked-in CRD can be installed and defaults are applied.
func TestAPIs(t *testing.T) {
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	t.Cleanup(func() {
		if err := testEnv.Stop(); err != nil {
			t.Errorf("stop envtest: %v", err)
		}
	})
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := redhatcopv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	apiClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	watermark := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "redhatcop.redhat.io/v1alpha1",
		"kind":       "NodeScalingWatermark",
		"metadata":   map[string]interface{}{"name": "smoke-test", "namespace": "default"},
		"spec":       map[string]interface{}{"watermarkPercentage": int64(20)},
	}}
	if err := apiClient.Create(ctx, watermark); err != nil {
		t.Fatal(err)
	}
	stored := &redhatcopv1alpha1.NodeScalingWatermark{}
	if err := apiClient.Get(ctx, client.ObjectKeyFromObject(watermark), stored); err != nil {
		t.Fatal(err)
	}
	if stored.Spec.PriorityClassName != "proactive-node-autoscaling-pods" || len(stored.Spec.PausePodSize) != 2 {
		t.Fatalf("CRD defaults missing: %+v", stored.Spec)
	}
	// Exercise the upgraded template/status helpers through a real reconcile.
	if err := apiClient.Create(ctx, &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "worker", Labels: stored.Spec.NodeSelector}}); err != nil {
		t.Fatal(err)
	}
	if err := apiClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "workload"}}); err != nil {
		t.Fatal(err)
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "workload", Namespace: "workload"},
		Spec:       corev1.PodSpec{NodeName: "worker", Containers: []corev1.Container{{Name: "app", Image: "registry.k8s.io/pause:3.10.2", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1")}}}}},
	}
	if err := apiClient.Create(ctx, pod); err != nil {
		t.Fatal(err)
	}
	t.Setenv(templateFileNameEnv, filepath.Join("..", "config", "templates", "watermarkDeploymentTemplate.yaml"))
	reconciler := &NodeScalingWatermarkReconciler{
		EnforcingReconciler: lockedresourcecontroller.NewEnforcingReconciler(apiClient, scheme, cfg, apiClient, nil, true, false),
		Log:                 logr.Discard(),
	}
	reconciler.watermarkDeploymentTemplate, err = reconciler.initializeTemplate()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: stored.Name, Namespace: stored.Namespace}}); err != nil {
		t.Fatal(err)
	}
	deployment := &appsv1.Deployment{}
	if err := apiClient.Get(ctx, client.ObjectKeyFromObject(stored), deployment); err != nil {
		t.Fatal(err)
	}
	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != 1 {
		t.Fatalf("unexpected replicas: %v", deployment.Spec.Replicas)
	}
	if deployment.Spec.Template.Spec.PriorityClassName != stored.Spec.PriorityClassName {
		t.Fatal("template did not preserve priority class")
	}
	if err := apiClient.Get(ctx, client.ObjectKeyFromObject(stored), stored); err != nil {
		t.Fatal(err)
	}
	if len(stored.Status.Conditions) == 0 {
		t.Fatal("reconcile did not update status")
	}

	invalid := stored.DeepCopy()
	invalid.Name = "invalid-watermark"
	invalid.ResourceVersion = ""
	invalid.Spec.WatermarkPercentage = 101
	if err := apiClient.Create(ctx, invalid); err == nil {
		t.Fatal("expected percentage above 100 to be rejected")
	}
}
