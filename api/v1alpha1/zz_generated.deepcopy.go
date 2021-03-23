// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeScalingWatermark) DeepCopyInto(out *NodeScalingWatermark) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeScalingWatermark.
func (in *NodeScalingWatermark) DeepCopy() *NodeScalingWatermark {
	if in == nil {
		return nil
	}
	out := new(NodeScalingWatermark)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NodeScalingWatermark) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeScalingWatermarkList) DeepCopyInto(out *NodeScalingWatermarkList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NodeScalingWatermark, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeScalingWatermarkList.
func (in *NodeScalingWatermarkList) DeepCopy() *NodeScalingWatermarkList {
	if in == nil {
		return nil
	}
	out := new(NodeScalingWatermarkList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NodeScalingWatermarkList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeScalingWatermarkSpec) DeepCopyInto(out *NodeScalingWatermarkSpec) {
	*out = *in
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PausePodSize != nil {
		in, out := &in.PausePodSize, &out.PausePodSize
		*out = make(v1.ResourceList, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeScalingWatermarkSpec.
func (in *NodeScalingWatermarkSpec) DeepCopy() *NodeScalingWatermarkSpec {
	if in == nil {
		return nil
	}
	out := new(NodeScalingWatermarkSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeScalingWatermarkStatus) DeepCopyInto(out *NodeScalingWatermarkStatus) {
	*out = *in
	in.EnforcingReconcileStatus.DeepCopyInto(&out.EnforcingReconcileStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeScalingWatermarkStatus.
func (in *NodeScalingWatermarkStatus) DeepCopy() *NodeScalingWatermarkStatus {
	if in == nil {
		return nil
	}
	out := new(NodeScalingWatermarkStatus)
	in.DeepCopyInto(out)
	return out
}