package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

func TestValidateVM(t *testing.T) {
	validVM := &virtv1alpha1.VirtualMachine{
		Spec: virtv1alpha1.VirtualMachineSpec{
			Instance: virtv1alpha1.Instance{
				CPU: virtv1alpha1.CPU{
					Sockets:        1,
					CoresPerSocket: 1,
				},
				Memory: virtv1alpha1.Memory{
					Size: func() *resource.Quantity { q := resource.MustParse("1Gi"); return &q }(),
				},
				Disks: []virtv1alpha1.Disk{{
					Name: "vol-1",
				}},
				Interfaces: []virtv1alpha1.Interface{{
					Name: "net-1",
					InterfaceBindingMethod: virtv1alpha1.InterfaceBindingMethod{
						Bridge: &virtv1alpha1.InterfaceBridge{},
					},
					MAC: "c6:1c:ba:0a:45:88",
				}},
			},
			Volumes: []virtv1alpha1.Volume{{
				Name: "vol-1",
				VolumeSource: virtv1alpha1.VolumeSource{
					ContainerDisk: &virtv1alpha1.ContainerDiskVolumeSource{
						Image: "container-disk",
					},
				},
			}},
			Networks: []virtv1alpha1.Network{{
				Name: "net-1",
				NetworkSource: virtv1alpha1.NetworkSource{
					Pod: &virtv1alpha1.PodNetworkSource{},
				},
			}},
		},
	}

	tests := []struct {
		vm            *virtv1alpha1.VirtualMachine
		invalidFields []string
	}{{
		vm: validVM,
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.CPU.DedicatedCPUPlacement = true
			vm.Spec.Resources = corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			}
			return vm
		}(),
		invalidFields: []string{"spec.resources.requests.cpu", "spec.resources.limits.cpu"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.CPU.DedicatedCPUPlacement = true
			vm.Spec.Resources = corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			}
			return vm
		}(),
		invalidFields: []string{"spec.resources.limits.cpu"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.CPU.DedicatedCPUPlacement = true
			vm.Spec.Resources = corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			}
			return vm
		}(),
		invalidFields: []string{"spec.resources.requests.cpu"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.CPU.DedicatedCPUPlacement = true
			vm.Spec.Resources = corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("0"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			}
			return vm
		}(),
		invalidFields: []string{"spec.resources.requests.memory", "spec.resources.limits.memory"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.CPU.DedicatedCPUPlacement = true
			vm.Spec.Resources = corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("0"),
				},
			}
			return vm
		}(),
		invalidFields: []string{"spec.resources.limits.memory"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.CPU.DedicatedCPUPlacement = true
			vm.Spec.Resources = corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("512Mi"),
				},
			}
			return vm
		}(),
		invalidFields: []string{"spec.resources.requests.memory"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.CPU.Sockets = 0
			return vm
		}(),
		invalidFields: []string{"spec.instance.cpu.sockets"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.CPU.CoresPerSocket = 0
			return vm
		}(),
		invalidFields: []string{"spec.instance.cpu.coresPerSocket"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Memory.Size = nil
			return vm
		}(),
		invalidFields: []string{"spec.instance.memory.size"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Memory.Size = func() *resource.Quantity { q := resource.MustParse("0"); return &q }()
			return vm
		}(),
		invalidFields: []string{"spec.instance.memory.size"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Memory.Size = func() *resource.Quantity { q := resource.MustParse("-1Gi"); return &q }()
			return vm
		}(),
		invalidFields: []string{"spec.instance.memory.size"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Disks[0].Name = ""
			return vm
		}(),
		invalidFields: []string{"spec.instance.disks[0].name"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].Name = ""
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0].name"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].MAC = ""
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0].mac"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].MAC = "01:1c:ba:0a:45:8"
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0].mac"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].Bridge = nil
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0]"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].InterfaceBindingMethod.Masquerade = &virtv1alpha1.InterfaceMasquerade{}
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0].masquerade"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].InterfaceBindingMethod.Bridge = nil
			vm.Spec.Instance.Interfaces[0].InterfaceBindingMethod.Masquerade = &virtv1alpha1.InterfaceMasquerade{
				CIDR: "",
			}
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0].masquerade.cidr"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].InterfaceBindingMethod.Bridge = nil
			vm.Spec.Instance.Interfaces[0].InterfaceBindingMethod.Masquerade = &virtv1alpha1.InterfaceMasquerade{
				CIDR: "10.0.2.0/31",
			}
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0].masquerade.cidr"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].InterfaceBindingMethod.SRIOV = &virtv1alpha1.InterfaceSRIOV{}
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0].sriov"},
	}, {

		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Volumes[0].Name = ""
			return vm
		}(),
		invalidFields: []string{"spec.volumes[0].name"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Volumes[0].VolumeSource.ContainerDisk = nil
			return vm
		}(),
		invalidFields: []string{"spec.volumes[0]"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Volumes[0].VolumeSource.CloudInit = &virtv1alpha1.CloudInitVolumeSource{}
			return vm
		}(),
		invalidFields: []string{"spec.volumes[0].cloudInit"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Networks[0].Name = ""
			return vm
		}(),
		invalidFields: []string{"spec.networks[0].name"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Networks[0].NetworkSource.Pod = nil
			return vm
		}(),
		invalidFields: []string{"spec.networks[0]"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Networks[0].NetworkSource.Multus = &virtv1alpha1.MultusNetworkSource{}
			return vm
		}(),
		invalidFields: []string{"spec.networks[0].multus"},
	}}

	for _, tc := range tests {
		errs := ValidateVM(context.Background(), tc.vm, nil)
		for _, err := range errs {
			assert.Contains(t, tc.invalidFields, err.Field)
		}
	}
}

func TestMutateVM(t *testing.T) {
	oldVM := &virtv1alpha1.VirtualMachine{
		Spec: virtv1alpha1.VirtualMachineSpec{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("1Gi"),
				},
			},
			Instance: virtv1alpha1.Instance{
				Interfaces: []virtv1alpha1.Interface{{
					Name: "pod",
				}},
			},
		},
	}

	tests := []struct {
		vm     *virtv1alpha1.VirtualMachine
		assert func(vm *virtv1alpha1.VirtualMachine)
	}{{
		vm: func() *virtv1alpha1.VirtualMachine {
			return oldVM.DeepCopy()
		}(),
		assert: func(vm *virtv1alpha1.VirtualMachine) {
			assert.Equal(t, uint32(1), vm.Spec.Instance.CPU.Sockets)
		},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			return oldVM.DeepCopy()
		}(),
		assert: func(vm *virtv1alpha1.VirtualMachine) {
			assert.Equal(t, uint32(1), vm.Spec.Instance.CPU.CoresPerSocket)
		},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			return oldVM.DeepCopy()
		}(),
		assert: func(vm *virtv1alpha1.VirtualMachine) {
			assert.Equal(t, "1Gi", vm.Spec.Instance.Memory.Size.String())
		},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			return oldVM.DeepCopy()
		}(),
		assert: func(vm *virtv1alpha1.VirtualMachine) {
			assert.NotEmpty(t, vm.Spec.Instance.Interfaces[0].MAC)
		},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			return oldVM.DeepCopy()
		}(),
		assert: func(vm *virtv1alpha1.VirtualMachine) {
			assert.NotNil(t, vm.Spec.Instance.Interfaces[0].Bridge)
		},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := oldVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].InterfaceBindingMethod = virtv1alpha1.InterfaceBindingMethod{
				Masquerade: &virtv1alpha1.InterfaceMasquerade{},
			}
			return vm
		}(),
		assert: func(vm *virtv1alpha1.VirtualMachine) {
			assert.Equal(t, vm.Spec.Instance.Interfaces[0].Masquerade.CIDR, "10.0.2.0/30")
		},
	}}
	for _, tc := range tests {
		err := MutateVM(context.Background(), tc.vm, nil)
		assert.Nil(t, err)
		tc.assert(tc.vm)
	}
}
