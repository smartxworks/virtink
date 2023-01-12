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
					Size: resource.MustParse("1Gi"),
				},
				Disks: []virtv1alpha1.Disk{{
					Name: "vol-1",
				}, {
					Name: "vol-3",
				}},
				FileSystems: []virtv1alpha1.FileSystem{{
					Name: "vol-2",
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
			}, {
				Name: "vol-2",
				VolumeSource: virtv1alpha1.VolumeSource{
					PersistentVolumeClaim: &virtv1alpha1.PersistentVolumeClaimVolumeSource{
						ClaimName: "vol-2",
					},
				},
			}, {
				Name: "vol-3",
				VolumeSource: virtv1alpha1.VolumeSource{
					PersistentVolumeClaim: &virtv1alpha1.PersistentVolumeClaimVolumeSource{
						ClaimName:    "vol-3",
						Hotpluggable: true,
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
		oldVM         *virtv1alpha1.VirtualMachine
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
					corev1.ResourceMemory: resource.MustParse("1280Mi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1280Mi"),
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
					corev1.ResourceMemory: resource.MustParse("1280Mi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("0"),
					corev1.ResourceMemory: resource.MustParse("1280Mi"),
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
					corev1.ResourceMemory: resource.MustParse("1280Mi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("1280Mi"),
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
					corev1.ResourceMemory: resource.MustParse("1280Mi"),
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
					corev1.ResourceMemory: resource.MustParse("1280Mi"),
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
			vm.Spec.Instance.Memory.Hugepages = &virtv1alpha1.Hugepages{
				PageSize: "1Gi",
			}
			return vm
		}(),
		invalidFields: []string{"spec.resources", "spec.resources.requests.hugepages-1Gi", "spec.resources.limits.hugepages-1Gi"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Memory.Hugepages = &virtv1alpha1.Hugepages{
				PageSize: "1Gi",
			}
			vm.Spec.Resources = corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceMemory: resource.MustParse(memoryOverhead),
					"hugepages-1Gi":       resource.MustParse("1025Mi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					"hugepages-1Gi": resource.MustParse("1025Mi"),
				},
			}
			return vm
		}(),
		invalidFields: []string{"spec.resources.requests.hugepages-1Gi"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Memory.Hugepages = &virtv1alpha1.Hugepages{
				PageSize: "2Mi",
			}
			vm.Spec.Resources = corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceMemory: resource.MustParse(memoryOverhead),
					"hugepages-2Mi":       resource.MustParse("1024Mi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					"hugepages-2Mi": resource.MustParse("1025Mi"),
				},
			}
			return vm
		}(),
		invalidFields: []string{"spec.resources.limits.hugepages-2Mi"},
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
			vm.Spec.Instance.Memory.Size = resource.MustParse("0")
			return vm
		}(),
		invalidFields: []string{"spec.instance.memory.size"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Memory.Size = resource.MustParse("-1Gi")
			return vm
		}(),
		invalidFields: []string{"spec.instance.memory.size"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Memory.Size = resource.MustParse("1025Mi")
			vm.Spec.Instance.Memory.Hugepages = &virtv1alpha1.Hugepages{
				PageSize: "1Gi",
			}
			vm.Spec.Resources = corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceMemory: resource.MustParse(memoryOverhead),
					"hugepages-1Gi":       resource.MustParse("1025Mi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					"hugepages-1Gi": resource.MustParse("1025Mi"),
				},
			}
			return vm
		}(),
		invalidFields: []string{"spec.instance.memory.size"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Memory.Size = resource.MustParse("511Mi")
			vm.Spec.Instance.Memory.Hugepages = &virtv1alpha1.Hugepages{
				PageSize: "2Mi",
			}
			vm.Spec.Resources = corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceMemory: resource.MustParse(memoryOverhead),
					"hugepages-2Mi":       resource.MustParse("511Mi"),
				},
				Limits: map[corev1.ResourceName]resource.Quantity{
					"hugepages-2Mi": resource.MustParse("511Mi"),
				},
			}
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
			vm.Spec.Instance.FileSystems[0].Name = ""
			return vm
		}(),
		invalidFields: []string{"spec.instance.fileSystems[0].name"},
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
				IPv4CIDR: "",
				IPv6CIDR: "",
			}
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0].masquerade.ipv4CIDR", "spec.instance.interfaces[0].masquerade.ipv6CIDR"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].InterfaceBindingMethod.Bridge = nil
			vm.Spec.Instance.Interfaces[0].InterfaceBindingMethod.Masquerade = &virtv1alpha1.InterfaceMasquerade{
				IPv4CIDR: "10.0.2.0/31",
				IPv6CIDR: "fd10:0:2::/127",
			}
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0].masquerade.ipv4CIDR", "spec.instance.interfaces[0].masquerade.ipv6CIDR"},
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
			vm.Spec.Instance.Interfaces[0].InterfaceBindingMethod.VhostUser = &virtv1alpha1.InterfaceVhostUser{}
			return vm
		}(),
		invalidFields: []string{"spec.instance.interfaces[0].vhostUser"},
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
			vm.Spec.Volumes[0].ContainerDisk.Image = "updated-container-disk-image"
			vm.Spec.Volumes[2].PersistentVolumeClaim.ClaimName = "updated-pvc-name"
			return vm
		}(),
		oldVM:         validVM.DeepCopy(),
		invalidFields: []string{"spec.volumes[vol-1]", "spec.volumes[vol-3]"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Volumes = append(vm.Spec.Volumes, virtv1alpha1.Volume{
				Name: "hotplug-pvc-1",
				VolumeSource: virtv1alpha1.VolumeSource{
					PersistentVolumeClaim: &virtv1alpha1.PersistentVolumeClaimVolumeSource{
						ClaimName:    "hotplug-pvc-1",
						Hotpluggable: true,
					},
				},
			}, virtv1alpha1.Volume{
				Name: "hotplug-pvc-2",
				VolumeSource: virtv1alpha1.VolumeSource{
					PersistentVolumeClaim: &virtv1alpha1.PersistentVolumeClaimVolumeSource{
						ClaimName:    "hotplug-pvc-2",
						Hotpluggable: false,
					},
				},
			}, virtv1alpha1.Volume{
				Name: "hotplug-dv-1",
				VolumeSource: virtv1alpha1.VolumeSource{
					DataVolume: &virtv1alpha1.DataVolumeVolumeSource{
						VolumeName:   "hotplug-dv-1",
						Hotpluggable: true,
					},
				},
			}, virtv1alpha1.Volume{
				Name: "hotplug-dv-2",
				VolumeSource: virtv1alpha1.VolumeSource{
					PersistentVolumeClaim: &virtv1alpha1.PersistentVolumeClaimVolumeSource{
						ClaimName:    "hotplug-dv-2",
						Hotpluggable: false,
					},
				},
			})
			return vm
		}(),
		oldVM:         validVM.DeepCopy(),
		invalidFields: []string{"spec.volumes[hotplug-pvc-2]", "spec.volumes[hotplug-dv-2]"},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Spec.Volumes = []virtv1alpha1.Volume{vm.Spec.Volumes[1]}
			vm.Spec.Instance.Disks = []virtv1alpha1.Disk{}
			return vm
		}(),
		oldVM:         validVM.DeepCopy(),
		invalidFields: []string{"spec.volumes[vol-1]"},
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
		errs := ValidateVM(context.Background(), tc.vm, tc.oldVM)
		for _, err := range errs {
			assert.Contains(t, tc.invalidFields, err.Field, err.Detail)
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
		oldVM  *virtv1alpha1.VirtualMachine
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
			vm := oldVM.DeepCopy()
			vm.Spec.Resources.Requests = nil
			return vm
		}(),
		assert: func(vm *virtv1alpha1.VirtualMachine) {
			assert.Equal(t, "1Gi", vm.Spec.Instance.Memory.Size.String())
		},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := oldVM.DeepCopy()
			vm.Spec.Instance.CPU.DedicatedCPUPlacement = true
			vm.Spec.Resources.Requests = nil
			vm.Spec.Instance.Memory.Size = resource.MustParse("1Gi")
			return vm
		}(),
		assert: func(vm *virtv1alpha1.VirtualMachine) {
			assert.Equal(t, "1", vm.Spec.Resources.Requests.Cpu().String())
			assert.Equal(t, "1", vm.Spec.Resources.Limits.Cpu().String())
			assert.Equal(t, "1280Mi", vm.Spec.Resources.Requests.Memory().String())
			assert.Equal(t, "1280Mi", vm.Spec.Resources.Limits.Memory().String())
		},
	}, {
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := oldVM.DeepCopy()
			vm.Spec.Resources.Requests = nil
			vm.Spec.Instance.Memory.Hugepages = &virtv1alpha1.Hugepages{
				PageSize: "1Gi",
			}
			return vm
		}(),
		assert: func(vm *virtv1alpha1.VirtualMachine) {
			assert.True(t, vm.Spec.Resources.Limits["hugepages-1Gi"].Equal(resource.MustParse("1Gi")))
			assert.True(t, vm.Spec.Resources.Requests["hugepages-1Gi"].Equal(resource.MustParse("1Gi")))
			assert.True(t, vm.Spec.Resources.Requests[corev1.ResourceMemory].Equal(resource.MustParse(memoryOverhead)))
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
		oldVM: func() *virtv1alpha1.VirtualMachine {
			vm := oldVM.DeepCopy()
			vm.Spec.Instance.Interfaces[0].MAC = "52:54:00:dd:0d:5b"
			return vm
		}(),
		assert: func(vm *virtv1alpha1.VirtualMachine) {
			assert.Equal(t, "52:54:00:dd:0d:5b", vm.Spec.Instance.Interfaces[0].MAC)
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
			assert.Equal(t, vm.Spec.Instance.Interfaces[0].Masquerade.IPv4CIDR, "10.0.2.0/30")
			assert.Equal(t, vm.Spec.Instance.Interfaces[0].Masquerade.IPv6CIDR, "fd10:0:2::/120")
		},
	}}
	for _, tc := range tests {
		err := MutateVM(context.Background(), tc.vm, tc.oldVM)
		assert.Nil(t, err)
		tc.assert(tc.vm)
	}
}
