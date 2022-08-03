//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CPU) DeepCopyInto(out *CPU) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CPU.
func (in *CPU) DeepCopy() *CPU {
	if in == nil {
		return nil
	}
	out := new(CPU)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudInitVolumeSource) DeepCopyInto(out *CloudInitVolumeSource) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudInitVolumeSource.
func (in *CloudInitVolumeSource) DeepCopy() *CloudInitVolumeSource {
	if in == nil {
		return nil
	}
	out := new(CloudInitVolumeSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerDiskVolumeSource) DeepCopyInto(out *ContainerDiskVolumeSource) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerDiskVolumeSource.
func (in *ContainerDiskVolumeSource) DeepCopy() *ContainerDiskVolumeSource {
	if in == nil {
		return nil
	}
	out := new(ContainerDiskVolumeSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerRootfsVolumeSource) DeepCopyInto(out *ContainerRootfsVolumeSource) {
	*out = *in
	out.Size = in.Size.DeepCopy()
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerRootfsVolumeSource.
func (in *ContainerRootfsVolumeSource) DeepCopy() *ContainerRootfsVolumeSource {
	if in == nil {
		return nil
	}
	out := new(ContainerRootfsVolumeSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataVolumeVolumeSource) DeepCopyInto(out *DataVolumeVolumeSource) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataVolumeVolumeSource.
func (in *DataVolumeVolumeSource) DeepCopy() *DataVolumeVolumeSource {
	if in == nil {
		return nil
	}
	out := new(DataVolumeVolumeSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Disk) DeepCopyInto(out *Disk) {
	*out = *in
	if in.ReadOnly != nil {
		in, out := &in.ReadOnly, &out.ReadOnly
		*out = new(bool)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Disk.
func (in *Disk) DeepCopy() *Disk {
	if in == nil {
		return nil
	}
	out := new(Disk)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Instance) DeepCopyInto(out *Instance) {
	*out = *in
	out.CPU = in.CPU
	in.Memory.DeepCopyInto(&out.Memory)
	if in.Kernel != nil {
		in, out := &in.Kernel, &out.Kernel
		*out = new(Kernel)
		**out = **in
	}
	if in.Disks != nil {
		in, out := &in.Disks, &out.Disks
		*out = make([]Disk, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Interfaces != nil {
		in, out := &in.Interfaces, &out.Interfaces
		*out = make([]Interface, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Instance.
func (in *Instance) DeepCopy() *Instance {
	if in == nil {
		return nil
	}
	out := new(Instance)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Interface) DeepCopyInto(out *Interface) {
	*out = *in
	in.InterfaceBindingMethod.DeepCopyInto(&out.InterfaceBindingMethod)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Interface.
func (in *Interface) DeepCopy() *Interface {
	if in == nil {
		return nil
	}
	out := new(Interface)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InterfaceBindingMethod) DeepCopyInto(out *InterfaceBindingMethod) {
	*out = *in
	if in.Bridge != nil {
		in, out := &in.Bridge, &out.Bridge
		*out = new(InterfaceBridge)
		**out = **in
	}
	if in.SRIOV != nil {
		in, out := &in.SRIOV, &out.SRIOV
		*out = new(InterfaceSRIOV)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InterfaceBindingMethod.
func (in *InterfaceBindingMethod) DeepCopy() *InterfaceBindingMethod {
	if in == nil {
		return nil
	}
	out := new(InterfaceBindingMethod)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InterfaceBridge) DeepCopyInto(out *InterfaceBridge) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InterfaceBridge.
func (in *InterfaceBridge) DeepCopy() *InterfaceBridge {
	if in == nil {
		return nil
	}
	out := new(InterfaceBridge)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InterfaceSRIOV) DeepCopyInto(out *InterfaceSRIOV) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InterfaceSRIOV.
func (in *InterfaceSRIOV) DeepCopy() *InterfaceSRIOV {
	if in == nil {
		return nil
	}
	out := new(InterfaceSRIOV)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Kernel) DeepCopyInto(out *Kernel) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Kernel.
func (in *Kernel) DeepCopy() *Kernel {
	if in == nil {
		return nil
	}
	out := new(Kernel)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Memory) DeepCopyInto(out *Memory) {
	*out = *in
	if in.Size != nil {
		in, out := &in.Size, &out.Size
		x := (*in).DeepCopy()
		*out = &x
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Memory.
func (in *Memory) DeepCopy() *Memory {
	if in == nil {
		return nil
	}
	out := new(Memory)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MultusNetworkSource) DeepCopyInto(out *MultusNetworkSource) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MultusNetworkSource.
func (in *MultusNetworkSource) DeepCopy() *MultusNetworkSource {
	if in == nil {
		return nil
	}
	out := new(MultusNetworkSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Network) DeepCopyInto(out *Network) {
	*out = *in
	in.NetworkSource.DeepCopyInto(&out.NetworkSource)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Network.
func (in *Network) DeepCopy() *Network {
	if in == nil {
		return nil
	}
	out := new(Network)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkSource) DeepCopyInto(out *NetworkSource) {
	*out = *in
	if in.Pod != nil {
		in, out := &in.Pod, &out.Pod
		*out = new(PodNetworkSource)
		**out = **in
	}
	if in.Multus != nil {
		in, out := &in.Multus, &out.Multus
		*out = new(MultusNetworkSource)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkSource.
func (in *NetworkSource) DeepCopy() *NetworkSource {
	if in == nil {
		return nil
	}
	out := new(NetworkSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PersistentVolumeClaimVolumeSource) DeepCopyInto(out *PersistentVolumeClaimVolumeSource) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PersistentVolumeClaimVolumeSource.
func (in *PersistentVolumeClaimVolumeSource) DeepCopy() *PersistentVolumeClaimVolumeSource {
	if in == nil {
		return nil
	}
	out := new(PersistentVolumeClaimVolumeSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodNetworkSource) DeepCopyInto(out *PodNetworkSource) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodNetworkSource.
func (in *PodNetworkSource) DeepCopy() *PodNetworkSource {
	if in == nil {
		return nil
	}
	out := new(PodNetworkSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachine) DeepCopyInto(out *VirtualMachine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachine.
func (in *VirtualMachine) DeepCopy() *VirtualMachine {
	if in == nil {
		return nil
	}
	out := new(VirtualMachine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VirtualMachine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineList) DeepCopyInto(out *VirtualMachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VirtualMachine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineList.
func (in *VirtualMachineList) DeepCopy() *VirtualMachineList {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VirtualMachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineSpec) DeepCopyInto(out *VirtualMachineSpec) {
	*out = *in
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.LivenessProbe != nil {
		in, out := &in.LivenessProbe, &out.LivenessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	if in.ReadinessProbe != nil {
		in, out := &in.ReadinessProbe, &out.ReadinessProbe
		*out = new(v1.Probe)
		(*in).DeepCopyInto(*out)
	}
	in.Instance.DeepCopyInto(&out.Instance)
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Networks != nil {
		in, out := &in.Networks, &out.Networks
		*out = make([]Network, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineSpec.
func (in *VirtualMachineSpec) DeepCopy() *VirtualMachineSpec {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VirtualMachineStatus) DeepCopyInto(out *VirtualMachineStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VirtualMachineStatus.
func (in *VirtualMachineStatus) DeepCopy() *VirtualMachineStatus {
	if in == nil {
		return nil
	}
	out := new(VirtualMachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Volume) DeepCopyInto(out *Volume) {
	*out = *in
	in.VolumeSource.DeepCopyInto(&out.VolumeSource)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Volume.
func (in *Volume) DeepCopy() *Volume {
	if in == nil {
		return nil
	}
	out := new(Volume)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeSource) DeepCopyInto(out *VolumeSource) {
	*out = *in
	if in.ContainerDisk != nil {
		in, out := &in.ContainerDisk, &out.ContainerDisk
		*out = new(ContainerDiskVolumeSource)
		**out = **in
	}
	if in.CloudInit != nil {
		in, out := &in.CloudInit, &out.CloudInit
		*out = new(CloudInitVolumeSource)
		**out = **in
	}
	if in.ContainerRootfs != nil {
		in, out := &in.ContainerRootfs, &out.ContainerRootfs
		*out = new(ContainerRootfsVolumeSource)
		(*in).DeepCopyInto(*out)
	}
	if in.PersistentVolumeClaim != nil {
		in, out := &in.PersistentVolumeClaim, &out.PersistentVolumeClaim
		*out = new(PersistentVolumeClaimVolumeSource)
		**out = **in
	}
	if in.DataVolume != nil {
		in, out := &in.DataVolume, &out.DataVolume
		*out = new(DataVolumeVolumeSource)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeSource.
func (in *VolumeSource) DeepCopy() *VolumeSource {
	if in == nil {
		return nil
	}
	out := new(VolumeSource)
	in.DeepCopyInto(out)
	return out
}
