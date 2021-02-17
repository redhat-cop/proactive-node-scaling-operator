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
	"io/ioutil"
	"math"
	"os"
	"reflect"
	"strings"

	"text/template"

	"github.com/go-logr/logr"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller"
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller/lockedpatch"
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller/lockedresource"
	redhatcopv1alpha1 "github.com/redhat-cop/proactive-node-scaling-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const annotationBase = "proactive-node-scaling-operator.redhat-cop.io"
const watermarkLabel = annotationBase + "/watermark"
const templateFileNameEnv = "TEMPLATE_FILE_NAME"

// NodeScalingWatermarkReconciler reconciles a NodeScalingWatermark object
type NodeScalingWatermarkReconciler struct {
	lockedresourcecontroller.EnforcingReconciler
	Log                         logr.Logger
	watermarkDeploymentTemplate *template.Template
}

type templateData struct {
	NodeScalingWatermark *redhatcopv1alpha1.NodeScalingWatermark
	Replicas             int64
}

// +kubebuilder:rbac:groups=redhatcop.redhat.io,resources=nodescalingwatermarks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=redhatcop.redhat.io,resources=nodescalingwatermarks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=redhatcop.redhat.io,resources=nodescalingwatermarks/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes;pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeScalingWatermark object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *NodeScalingWatermarkReconciler) Reconcile(context context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("nodescalingwatermark", req.NamespacedName)

	// Fetch the EgressIPAM instance
	instance := &redhatcopv1alpha1.NodeScalingWatermark{}
	err := r.GetClient().Get(context, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// your logic here
	// find all selected nodes
	selectedNodes, err := r.getSelectedNodes(context, instance)
	if err != nil {
		log.Error(err, "unable to load selected nodes", "instance", instance.GetName())
		return r.ManageError(context, instance, err)
	}

	log.Info("selected nodes", "count", len(selectedNodes))

	// find all selected pods
	selectedPods, err := r.getSelectedPods(context, selectedNodes)
	if err != nil {
		log.Error(err, "unable to load selected pods", "instance", instance.GetName())
		return r.ManageError(context, instance, err)
	}

	log.Info("selected pods", "count", len(selectedPods))

	// sum pods requests
	totalRequests := r.sumRequests(selectedPods)

	log.Info("selected pods total", "request", totalRequests)

	// calculate ratio with passed pod size and obtain number replicas
	replicas := r.getNeededReplicas(instance, totalRequests)

	log.Info("watermark", "replicas", replicas)

	// process and apply template

	templateData := templateData{
		NodeScalingWatermark: instance,
		Replicas:             replicas,
	}

	obj, err := util.ProcessTemplate(templateData, r.watermarkDeploymentTemplate)

	//objs, err := r.processTemplate(instance, data)
	if err != nil {
		log.Error(err, "unable process watermark pod deployment template from", "instance", instance, "and from services", templateData)
		return r.ManageError(context, instance, err)
	}

	err = r.UpdateLockedResources(context, instance, []lockedresource.LockedResource{
		{
			Unstructured: *obj,
			ExcludedPaths: []string{
				".metadata",
				".status",
			},
		},
	}, []lockedpatch.LockedPatch{})
	if err != nil {
		log.Error(err, "unable to update locked resources")
		return r.ManageError(context, instance, err)
	}

	// for _, obj := range *objs {
	// 	err = r.CreateOrUpdateResource(context, instance, instance.GetNamespace(), &obj)
	// 	if err != nil {
	// 		log.Error(err, "unable to create or update resource", "resource", obj)
	// 		return r.ManageError(context, instance, err)
	// 	}
	// }
	return r.ManageSuccess(context, instance)
}

func (r *NodeScalingWatermarkReconciler) getNeededReplicas(instance *redhatcopv1alpha1.NodeScalingWatermark, totalRequests corev1.ResourceList) int64 {
	replicas := float64(0)
	for measure, podQuantity := range instance.Spec.PausePodSize {
		totalQuantity, ok := totalRequests[measure]
		if !ok {
			continue
		}
		neededReplicas := totalQuantity.AsApproximateFloat64() * float64(100-instance.Spec.WatermarkPercentage) / 100 / podQuantity.AsApproximateFloat64()
		replicas = math.Max(replicas, neededReplicas)
	}
	return int64(replicas)
}

func (r *NodeScalingWatermarkReconciler) sumRequests(pods []corev1.Pod) corev1.ResourceList {
	totalRequests := corev1.ResourceList{}
	for i := range pods {
		totalRequests = sumResources(totalRequests, getTotalRequests(&pods[i]))
	}
	return totalRequests
}

func (r *NodeScalingWatermarkReconciler) getSelectedPods(context context.Context, nodes []corev1.Node) ([]corev1.Pod, error) {
	selectedPods := []corev1.Pod{}
	for _, node := range nodes {
		podList := &corev1.PodList{}
		err := r.GetClient().List(context, podList, &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("spec.nodeName", node.Name),
		})
		if err != nil {
			r.Log.Error(err, "unable to list pod by field", "spec.nodeName", node.Name)
			return []corev1.Pod{}, err
		}
		selectedPods = append(selectedPods, r.filterWatermarkAndSystemPods(podList.Items)...)
	}
	return selectedPods, nil
}

func (r *NodeScalingWatermarkReconciler) filterWatermarkAndSystemPods(pods []corev1.Pod) []corev1.Pod {
	filteredPods := []corev1.Pod{}
	for i := range pods {
		_, ok := pods[i].Labels[watermarkLabel]
		if strings.HasPrefix(pods[i].Namespace, "kube-") || strings.HasPrefix(pods[i].Namespace, "openshift-") || pods[i].Namespace == "default" || ok {
			continue
		}
		filteredPods = append(filteredPods, pods[i])
	}
	return filteredPods
}

func (r *NodeScalingWatermarkReconciler) getSelectedNodes(context context.Context, instance *redhatcopv1alpha1.NodeScalingWatermark) ([]corev1.Node, error) {
	nodeList := &corev1.NodeList{}
	err := r.GetClient().List(context, nodeList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set(instance.Spec.NodeSelector)),
	})
	if err != nil {
		r.Log.Error(err, "unable to find nodes mathcing", "labels", instance.Spec.NodeSelector)
		return []corev1.Node{}, err
	}
	return nodeList.Items, nil
}

func sumQuantity(left, right resource.Quantity) resource.Quantity {
	result := resource.Quantity{}
	result.Add(left)
	result.Add(right)
	return result
}

func sumResources(left corev1.ResourceList, right corev1.ResourceList) corev1.ResourceList {
	result := left.DeepCopy()
	for measure, value := range right {
		if currentValue, ok := result[measure]; ok {
			result[measure] = sumQuantity(currentValue, value)
		} else {
			result[measure] = value
		}
	}
	return result
}

func getTotalRequests(pod *corev1.Pod) corev1.ResourceList {
	result := corev1.ResourceList{}
	for i := range pod.Spec.Containers {
		result = sumResources(result, pod.Spec.Containers[i].Resources.Requests)
	}
	return result
}

func (r *NodeScalingWatermarkReconciler) initializeTemplate() (*template.Template, error) {
	templateFileName, ok := os.LookupEnv(templateFileNameEnv)
	if !ok {
		templateFileName = "/templates/watermarkDeploymentTemplate.yaml"
	}
	text, err := ioutil.ReadFile(templateFileName)
	if err != nil {
		r.Log.Error(err, "Error reading job template file", "filename", templateFileName)
		return &template.Template{}, err
	}
	watermarkDeploymentTemplate, err := template.New("WatermarkDeployment").Funcs(util.AdvancedTemplateFuncMap(r.GetRestConfig())).Parse(string(text))
	if err != nil {
		r.Log.Error(err, "Error parsing template", "template", string(text))
		return &template.Template{}, err
	}
	return watermarkDeploymentTemplate, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeScalingWatermarkReconciler) SetupWithManager(mgr ctrl.Manager) error {

	watermarkDeploymentTemplate, err := r.initializeTemplate()
	if err != nil {
		r.Log.Error(err, "unable to initialize watermarkDeploymentTemplate")
		return err
	}
	r.watermarkDeploymentTemplate = watermarkDeploymentTemplate

	log2 := r.Log.WithName("IsCreatedOrIsEgressIPsChanged")

	IsRequestChanged := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldPod, ok := e.ObjectOld.(*corev1.Pod)
			if !ok {
				log2.Info("unable to convert event object to Pod,", "event", e)
				return false
			}
			newPod, ok := e.ObjectNew.(*corev1.Pod)
			if !ok {
				log2.Info("unable to convert event object to Pod,", "event", e)
				return false
			}
			oldRequest := getTotalRequests(oldPod)
			newRequest := getTotalRequests(newPod)

			return !reflect.DeepEqual(oldRequest, newRequest)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	err = mgr.GetFieldIndexer().IndexField(context.TODO(), &corev1.Pod{}, "spec.nodeName", func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{pod.Spec.NodeName}
	})

	if err != nil {
		r.Log.Error(err, "unable to create index on `spec.nodeName`")
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&redhatcopv1alpha1.NodeScalingWatermark{}, builder.WithPredicates(util.ResourceGenerationOrFinalizerChangedPredicate{})).
		Watches(&source.Kind{Type: &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
		}}, &enqueForScalingWatermarksSelectingNodeHostingPod{
			r:   r,
			log: r.Log.WithName("enqueForScalingWatermarksSelectingNodeHostingPod"),
		}, builder.WithPredicates(&IsRequestChanged)).
		Complete(r)
}

func (e *enqueForScalingWatermarksSelectingNodeHostingPod) getNode(name string) (*corev1.Node, bool, error) {
	node := &corev1.Node{}
	err := e.r.GetClient().Get(context.TODO(), client.ObjectKey{
		Name: name,
	}, node)
	if err != nil {
		if errors.IsNotFound(err) {
			return &corev1.Node{}, false, nil
		}
		e.log.Error(err, "unable to look up", "node", name)
		return &corev1.Node{}, false, err
	}
	return node, true, nil
}

func (e *enqueForScalingWatermarksSelectingNodeHostingPod) getAllNodeScalingWatermark() ([]redhatcopv1alpha1.NodeScalingWatermark, error) {
	nodeScalingWaetermarkList := &redhatcopv1alpha1.NodeScalingWatermarkList{}
	err := e.r.GetClient().List(context.TODO(), nodeScalingWaetermarkList)
	if err != nil {
		e.log.Error(err, "unable to list NodeScalingWatermark")
		return []redhatcopv1alpha1.NodeScalingWatermark{}, nil
	}
	return nodeScalingWaetermarkList.Items, nil
}

type enqueForScalingWatermarksSelectingNodeHostingPod struct {
	r   *NodeScalingWatermarkReconciler
	log logr.Logger
}

// trigger a egressIPAM reconcile event for those egressIPAM objects that reference this hostsubnet indireclty via the corresponding node.
func (e *enqueForScalingWatermarksSelectingNodeHostingPod) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	pod, ok := evt.Object.(*corev1.Pod)
	if !ok {
		e.log.Info("unable convert event object to pod,", "event", evt)
		return
	}
	e.processReferringNodeScalingWatemarks(pod, q)
}

func (e *enqueForScalingWatermarksSelectingNodeHostingPod) processReferringNodeScalingWatemarks(pod *corev1.Pod, q workqueue.RateLimitingInterface) {

	nodeName := pod.Spec.NodeName
	if nodeName == "" {
		return
	}
	node, found, err := e.getNode(nodeName)
	if err != nil {
		e.log.Error(err, "unable to lookup", "node", nodeName)
		return
	}
	if !found {
		return
	}

	nodeScalingWatermarks, err := e.getAllNodeScalingWatermark()

	if err != nil {
		e.log.Error(err, "unable to list NodeScalingWatermarks")
		return
	}

	for _, nodeScalingWatermark := range nodeScalingWatermarks {
		labelSelector := labels.SelectorFromSet(nodeScalingWatermark.Spec.NodeSelector)
		if labelSelector.Matches(labels.Set(node.Labels)) {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name: nodeScalingWatermark.GetName(),
			}})
		}
	}
}

// Update implements EventHandler
// trigger a router reconcile event for those routes that reference this secret
func (e *enqueForScalingWatermarksSelectingNodeHostingPod) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	pod, ok := evt.ObjectNew.(*corev1.Pod)
	if !ok {
		e.log.Info("unable convert event object to pod,", "event", evt)
		return
	}
	e.processReferringNodeScalingWatemarks(pod, q)
}

// Delete implements EventHandler
func (e *enqueForScalingWatermarksSelectingNodeHostingPod) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	pod, ok := evt.Object.(*corev1.Pod)
	if !ok {
		e.log.Info("unable convert event object to pod,", "event", evt)
		return
	}
	e.processReferringNodeScalingWatemarks(pod, q)
}

// Generic implements EventHandler
func (e *enqueForScalingWatermarksSelectingNodeHostingPod) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	return
}
