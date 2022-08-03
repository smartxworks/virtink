package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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
			vm.Spec.Instance.Interfaces[0].Bridge = nil
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0]"},
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
