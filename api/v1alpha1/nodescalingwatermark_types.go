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

package v1alpha1

import (
	"github.com/redhat-cop/operator-utils/pkg/util/apis"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeScalingWatermarkSpec defines the desired state of NodeScalingWatermark
type NodeScalingWatermarkSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// WatermarkPercentage: percentage of aggregated capacity of the selectd nodes after which the cluster should start scaling
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:Minimum=1
	WatermarkPercentage int `json:"watermarkPercentage"`

	// NodeSelector for the nodes for which the watermark will be calculated. These nodes should be controlled by an autoscaler.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:={node-role.kubernetes.io/worker:""}
	NodeSelector map[string]string `json:"nodeSelector"`

	// Tolerations is the tolerations  that the pause pod should have.

	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// PausePodSize is size of the pause pods used to mark the watermark, smaller pods will distributed better but consume slightly more resources. Tuning may be required to find the optimal size.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:={memory: "200Mi",cpu: "200m"}
	PausePodSize corev1.ResourceList `json:"pausePodSize"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="k8s.gcr.io/pause"
	PausePodImage string `json:"pausePodImage"`

	// PriorityClassName is the priorityClassName assigned to the pause pods, if not set it will be default to low-priority
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=low-priority
	PriorityClassName string `json:"priorityClassName"`
}

// NodeScalingWatermarkStatus defines the observed state of NodeScalingWatermark
type NodeScalingWatermarkStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions this is the general status of the main reconciler
	// +kubebuilder:validation:Optional
	apis.EnforcingReconcileStatus `json:",inline,omitempty"`
}

func (m *NodeScalingWatermark) GetEnforcingReconcileStatus() apis.EnforcingReconcileStatus {
	return m.Status.EnforcingReconcileStatus
}

func (m *NodeScalingWatermark) SetEnforcingReconcileStatus(reconcileStatus apis.EnforcingReconcileStatus) {
	m.Status.EnforcingReconcileStatus = reconcileStatus
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NodeScalingWatermark is the Schema for the nodescalingwatermarks API
type NodeScalingWatermark struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeScalingWatermarkSpec   `json:"spec,omitempty"`
	Status NodeScalingWatermarkStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NodeScalingWatermarkList contains a list of NodeScalingWatermark
type NodeScalingWatermarkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeScalingWatermark `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeScalingWatermark{}, &NodeScalingWatermarkList{})
}
