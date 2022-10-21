package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	netv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/client-go/tools/record"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

type VMReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	PrerunnerImageName string
}

// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch
// +kubebuilder:rbac:groups=cdi.kubevirt.io,resources=datavolumes,verbs=get;list;watch
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch

func (r *VMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var vm virtv1alpha1.VirtualMachine
	if err := r.Get(ctx, req.NamespacedName, &vm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	status := vm.Status.DeepCopy()
	if err := r.reconcile(ctx, &vm); err != nil {
		r.Recorder.Eventf(&vm, corev1.EventTypeWarning, "FailedReconcile", "Failed to reconcile VM: %s", err)
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(vm.Status, status) {
		if err := r.Status().Update(ctx, &vm); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("update VM status: %s", err)
		}
	}

	if err := r.gcVMPods(ctx, &vm); err != nil {
		return ctrl.Result{}, fmt.Errorf("GC VM Pods: %s", err)
	}

	return ctrl.Result{}, nil
}

func (r *VMReconciler) reconcile(ctx context.Context, vm *virtv1alpha1.VirtualMachine) error {
	if vm.DeletionTimestamp != nil && !vm.DeletionTimestamp.IsZero() {
		return nil
	}

	switch vm.Status.Phase {
	case virtv1alpha1.VirtualMachinePending:
		vm.Status.VMPodName = names.SimpleNameGenerator.GenerateName(fmt.Sprintf("vm-%s-", vm.Name))
		vm.Status.Phase = virtv1alpha1.VirtualMachineScheduling
	case virtv1alpha1.VirtualMachineScheduling, virtv1alpha1.VirtualMachineScheduled:
		var vmPod corev1.Pod
		vmPodKey := types.NamespacedName{
			Name:      vm.Status.VMPodName,
			Namespace: vm.Namespace,
		}
		vmPodNotFound := false
		if err := r.Get(ctx, vmPodKey, &vmPod); err != nil {
			if apierrors.IsNotFound(err) {
				vmPodNotFound = true
			} else {
				return fmt.Errorf("get VM Pod: %s", err)
			}
		}

		if !vmPodNotFound && !metav1.IsControlledBy(&vmPod, vm) {
			vmPodNotFound = true
		}

		if vmPodNotFound {
			if vm.Status.Phase == virtv1alpha1.VirtualMachineScheduling {
				vmPod, err := r.buildVMPod(ctx, vm)
				if err != nil {
					return fmt.Errorf("build VM Pod: %s", err)
				}

				vmPod.Name = vmPodKey.Name
				vmPod.Namespace = vmPodKey.Namespace
				if err := controllerutil.SetControllerReference(vm, vmPod, r.Scheme); err != nil {
					return fmt.Errorf("set VM Pod controller reference: %s", err)
				}
				if err := r.Create(ctx, vmPod); err != nil {
					return fmt.Errorf("create VM Pod: %s", err)
				}
				r.Recorder.Eventf(vm, corev1.EventTypeNormal, "CreatedVMPod", "Created VM Pod %q", vmPod.Name)
			} else {
				vm.Status.Phase = virtv1alpha1.VirtualMachineFailed
			}
		} else {
			switch vmPod.Status.Phase {
			case corev1.PodRunning:
				if vm.Status.Phase == virtv1alpha1.VirtualMachineScheduling {
					vm.Status.VMPodUID = vmPod.UID
					vm.Status.NodeName = vmPod.Spec.NodeName
					vm.Status.Phase = virtv1alpha1.VirtualMachineScheduled
				}
			case corev1.PodSucceeded:
				vm.Status.Phase = virtv1alpha1.VirtualMachineSucceeded
			case corev1.PodFailed:
				vm.Status.Phase = virtv1alpha1.VirtualMachineFailed
			case corev1.PodUnknown:
				vm.Status.Phase = virtv1alpha1.VirtualMachineUnknown
			default:
				// ignored
			}
		}
	case virtv1alpha1.VirtualMachineRunning:
		var vmPod corev1.Pod
		vmPodKey := types.NamespacedName{
			Name:      vm.Status.VMPodName,
			Namespace: vm.Namespace,
		}
		vmPodNotFound := false
		if err := r.Get(ctx, vmPodKey, &vmPod); err != nil {
			if apierrors.IsNotFound(err) {
				vmPodNotFound = true
			} else {
				return fmt.Errorf("get VM Pod: %s", err)
			}
		}

		if !vmPodNotFound && !metav1.IsControlledBy(&vmPod, vm) {
			vmPodNotFound = true
		}
		switch {
		case vmPodNotFound:
			vm.Status.Phase = virtv1alpha1.VirtualMachineFailed
		case vmPod.Status.Phase == corev1.PodSucceeded:
			if vm.Status.Migration == nil {
				vm.Status.Phase = virtv1alpha1.VirtualMachineSucceeded
			}
		case vmPod.Status.Phase == corev1.PodFailed:
			vm.Status.Phase = virtv1alpha1.VirtualMachineFailed
		case vmPod.Status.Phase == corev1.PodUnknown:
			vm.Status.Phase = virtv1alpha1.VirtualMachineUnknown
		}
		if vm.Status.Phase != virtv1alpha1.VirtualMachineRunning {
			return nil
		}

		if err := r.reconcileVMConditions(ctx, vm, &vmPod); err != nil {
			return err
		}

		if vm.Status.Migration != nil {
			switch vm.Status.Migration.Phase {
			case "", virtv1alpha1.VirtualMachineMigrationPending:
				vm.Status.Migration.TargetVMPodName = names.SimpleNameGenerator.GenerateName(fmt.Sprintf("vm-%s-", vm.Name))
				vm.Status.Migration.Phase = virtv1alpha1.VirtualMachineMigrationScheduling
			case virtv1alpha1.VirtualMachineMigrationScheduling:
				var targetVMPod corev1.Pod
				targetVMPodKey := types.NamespacedName{
					Name:      vm.Status.Migration.TargetVMPodName,
					Namespace: vm.Namespace,
				}
				targetVMPodNotFound := false
				if err := r.Get(ctx, targetVMPodKey, &targetVMPod); err != nil {
					if apierrors.IsNotFound(err) {
						vmPodNotFound = true
					} else {
						return fmt.Errorf("get target VM Pod: %s", err)
					}
				}

				if !targetVMPodNotFound && !metav1.IsControlledBy(&targetVMPod, vm) {
					targetVMPodNotFound = true
				}

				if targetVMPodNotFound {
					targetVMPod, err := r.buildTargetVMPod(ctx, vm)
					if err != nil {
						return fmt.Errorf("build target VM Pod: %s", err)
					}

					targetVMPod.Name = targetVMPodKey.Name
					targetVMPod.Namespace = targetVMPodKey.Namespace
					if err := controllerutil.SetControllerReference(vm, targetVMPod, r.Scheme); err != nil {
						return fmt.Errorf("set target VM Pod controller reference: %s", err)
					}
					if err := r.Create(ctx, targetVMPod); err != nil {
						return fmt.Errorf("create target VM Pod: %s", err)
					}
					r.Recorder.Eventf(vm, corev1.EventTypeNormal, "CreatedTargetVMPod", "Created target VM Pod %q", targetVMPod.Name)
				} else {
					switch targetVMPod.Status.Phase {
					case corev1.PodRunning:
						vm.Status.Migration.TargetVMPodUID = targetVMPod.UID
						vm.Status.Migration.TargetNodeName = targetVMPod.Spec.NodeName
						vm.Status.Migration.Phase = virtv1alpha1.VirtualMachineMigrationScheduled
					case corev1.PodFailed, corev1.PodUnknown:
						vm.Status.Migration.TargetVMPodUID = targetVMPod.UID
						vm.Status.Migration.Phase = virtv1alpha1.VirtualMachineMigrationFailed
					}
				}
			}
		}
	case "", virtv1alpha1.VirtualMachineSucceeded, virtv1alpha1.VirtualMachineFailed:
		run := false
		switch vm.Spec.RunPolicy {
		case virtv1alpha1.RunPolicyAlways:
			run = true
		case virtv1alpha1.RunPolicyRerunOnFailure:
			run = vm.Status.Phase == virtv1alpha1.VirtualMachineFailed || vm.Status.Phase == "" || vm.Status.PowerAction == virtv1alpha1.VirtualMachinePowerOn
		case virtv1alpha1.RunPolicyOnce:
			run = vm.Status.Phase == "" || vm.Status.PowerAction == virtv1alpha1.VirtualMachinePowerOn
		case virtv1alpha1.RunPolicyManual:
			run = vm.Status.PowerAction == virtv1alpha1.VirtualMachinePowerOn
		default:
			// ignored
		}

		if run {
			vm.Status.Phase = virtv1alpha1.VirtualMachinePending
		}

		vm.Status = virtv1alpha1.VirtualMachineStatus{
			Phase: vm.Status.Phase,
		}
	default:
		// ignored
	}
	return nil
}

func (r *VMReconciler) buildVMPod(ctx context.Context, vm *virtv1alpha1.VirtualMachine) (*corev1.Pod, error) {
	vmJSON, err := json.Marshal(vm)
	if err != nil {
		return nil, fmt.Errorf("marshal VM: %s", err)
	}

	vmPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      vm.Labels,
			Annotations: vm.Annotations,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			NodeSelector:  vm.Spec.NodeSelector,
			Tolerations:   vm.Spec.Tolerations,
			Affinity:      vm.Spec.Affinity,
			Containers: []corev1.Container{{
				Name:           "cloud-hypervisor",
				Image:          r.PrerunnerImageName,
				Resources:      vm.Spec.Resources,
				LivenessProbe:  vm.Spec.LivenessProbe,
				ReadinessProbe: vm.Spec.ReadinessProbe,
				SecurityContext: &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{"SYS_ADMIN", "NET_ADMIN", "SYS_RESOURCE"},
					},
				},
				Env: []corev1.EnvVar{{
					Name:  "VM_DATA",
					Value: base64.StdEncoding.EncodeToString(vmJSON),
				}},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "virtink",
					MountPath: "/var/run/virtink",
				}},
			}},
			Volumes: []corev1.Volume{{
				Name: "virtink",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}},
		},
	}

	incrementContainerResource(&vmPod.Spec.Containers[0], "devices.virtink.io/kvm")
	incrementContainerResource(&vmPod.Spec.Containers[0], "devices.virtink.io/tun")

	if vmPod.Labels == nil {
		vmPod.Labels = map[string]string{}
	}
	vmPod.Labels["virtink.io/vm.name"] = vm.Name

	if vm.Spec.Instance.Kernel != nil {
		vmPod.Spec.Volumes = append(vmPod.Spec.Volumes, corev1.Volume{
			Name: "virtink-kernel",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})

		volumeMount := corev1.VolumeMount{
			Name:      "virtink-kernel",
			MountPath: "/mnt/virtink-kernel",
		}
		vmPod.Spec.Containers[0].VolumeMounts = append(vmPod.Spec.Containers[0].VolumeMounts, volumeMount)

		vmPod.Spec.InitContainers = append(vmPod.Spec.InitContainers, corev1.Container{
			Name:            "init-kernel",
			Image:           vm.Spec.Instance.Kernel.Image,
			ImagePullPolicy: vm.Spec.Instance.Kernel.ImagePullPolicy,
			Resources:       vm.Spec.Resources,
			Args:            []string{volumeMount.MountPath + "/vmlinux"},
			VolumeMounts:    []corev1.VolumeMount{volumeMount},
		})
	}

	if vm.Spec.Instance.Memory.Hugepages != nil {
		vmPod.Spec.Volumes = append(vmPod.Spec.Volumes, corev1.Volume{
			Name: "hugepages",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: "HugePages",
				},
			},
		})
		volumeMount := corev1.VolumeMount{
			Name:      "hugepages",
			MountPath: "/dev/hugepages",
		}
		vmPod.Spec.Containers[0].VolumeMounts = append(vmPod.Spec.Containers[0].VolumeMounts, volumeMount)
	}

	for _, volume := range vm.Spec.Volumes {
		switch {
		case volume.ContainerDisk != nil:
			vmPod.Spec.Volumes = append(vmPod.Spec.Volumes, corev1.Volume{
				Name: volume.Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})

			volumeMount := corev1.VolumeMount{
				Name:      volume.Name,
				MountPath: "/mnt/" + volume.Name,
			}
			vmPod.Spec.Containers[0].VolumeMounts = append(vmPod.Spec.Containers[0].VolumeMounts, volumeMount)

			vmPod.Spec.InitContainers = append(vmPod.Spec.InitContainers, corev1.Container{
				Name:            "init-volume-" + volume.Name,
				Image:           volume.ContainerDisk.Image,
				ImagePullPolicy: volume.ContainerDisk.ImagePullPolicy,
				Resources:       vm.Spec.Resources,
				Args:            []string{volumeMount.MountPath + "/disk.raw"},
				VolumeMounts:    []corev1.VolumeMount{volumeMount},
			})
		case volume.CloudInit != nil:
			initContainer := corev1.Container{
				Name:      "init-volume-" + volume.Name,
				Image:     vmPod.Spec.Containers[0].Image,
				Resources: vm.Spec.Resources,
				Command:   []string{"virt-init-volume"},
				Args:      []string{"cloud-init"},
			}

			metaData := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("instance-id: %s\nlocal-hostname: %s", vm.UID, vm.Name)))
			initContainer.Args = append(initContainer.Args, metaData)

			var userData string
			switch {
			case volume.CloudInit.UserData != "":
				userData = base64.StdEncoding.EncodeToString([]byte(volume.CloudInit.UserData))
			case volume.CloudInit.UserDataBase64 != "":
				userData = volume.CloudInit.UserDataBase64
			case volume.CloudInit.UserDataSecretName != "":
				vmPod.Spec.Volumes = append(vmPod.Spec.Volumes, corev1.Volume{
					Name: "virtink-cloud-init-user-data",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: volume.CloudInit.UserDataSecretName,
						},
					},
				})
				initContainer.VolumeMounts = append(initContainer.VolumeMounts, corev1.VolumeMount{
					Name:      "virtink-cloud-init-user-data",
					MountPath: "/mnt/virtink-cloud-init-user-data",
				})
				userData = "/mnt/virtink-cloud-init-user-data/value"
			default:
				// ignored
			}
			initContainer.Args = append(initContainer.Args, userData)

			var networkData string
			switch {
			case volume.CloudInit.NetworkData != "":
				networkData = base64.StdEncoding.EncodeToString([]byte(volume.CloudInit.NetworkData))
			case volume.CloudInit.NetworkDataBase64 != "":
				networkData = volume.CloudInit.NetworkDataBase64
			case volume.CloudInit.NetworkDataSecretName != "":
				vmPod.Spec.Volumes = append(vmPod.Spec.Volumes, corev1.Volume{
					Name: "virtink-cloud-init-network-data",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: volume.CloudInit.NetworkDataSecretName,
						},
					},
				})
				vmPod.Spec.Containers[0].VolumeMounts = append(vmPod.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
					Name:      "virtink-cloud-init-network-data",
					MountPath: "/mnt/virtink-cloud-init-network-data",
				})
				networkData = "/mnt/virtink-cloud-init-network-data/value"
			default:
				// ignored
			}
			initContainer.Args = append(initContainer.Args, networkData)

			vmPod.Spec.Volumes = append(vmPod.Spec.Volumes, corev1.Volume{
				Name: volume.Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})

			volumeMount := corev1.VolumeMount{
				Name:      volume.Name,
				MountPath: "/mnt/" + volume.Name,
			}
			vmPod.Spec.Containers[0].VolumeMounts = append(vmPod.Spec.Containers[0].VolumeMounts, volumeMount)
			initContainer.VolumeMounts = append(initContainer.VolumeMounts, volumeMount)
			initContainer.Args = append(initContainer.Args, volumeMount.MountPath+"/cloud-init.iso")
			vmPod.Spec.InitContainers = append(vmPod.Spec.InitContainers, initContainer)
		case volume.ContainerRootfs != nil:
			vmPod.Spec.Volumes = append(vmPod.Spec.Volumes, corev1.Volume{
				Name: volume.Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})

			volumeMount := corev1.VolumeMount{
				Name:      volume.Name,
				MountPath: "/mnt/" + volume.Name,
			}
			vmPod.Spec.Containers[0].VolumeMounts = append(vmPod.Spec.Containers[0].VolumeMounts, volumeMount)

			vmPod.Spec.InitContainers = append(vmPod.Spec.InitContainers, corev1.Container{
				Name:            "init-volume-" + volume.Name,
				Image:           volume.ContainerRootfs.Image,
				ImagePullPolicy: volume.ContainerRootfs.ImagePullPolicy,
				Resources:       vm.Spec.Resources,
				Args:            []string{volumeMount.MountPath + "/rootfs.raw", strconv.FormatInt(volume.ContainerRootfs.Size.Value(), 10)},
				VolumeMounts:    []corev1.VolumeMount{volumeMount},
			})
		case volume.PersistentVolumeClaim != nil, volume.DataVolume != nil:
			var pvcName string
			if volume.PersistentVolumeClaim != nil {
				pvcName = volume.PersistentVolumeClaim.ClaimName
			} else {
				pvcName = volume.DataVolume.VolumeName
			}

			vmPod.Spec.Volumes = append(vmPod.Spec.Volumes, corev1.Volume{
				Name: volume.Name,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			})

			pvcKey := types.NamespacedName{
				Namespace: vm.Namespace,
				Name:      pvcName,
			}
			var pvc corev1.PersistentVolumeClaim
			if err := r.Client.Get(ctx, pvcKey, &pvc); err != nil {
				return nil, fmt.Errorf("get PVC: %s", err)
			}

			if volume.DataVolume != nil {
				var getDataVolumeFunc = func(name, namespace string) (*cdiv1beta1.DataVolume, error) {
					var dv cdiv1beta1.DataVolume
					dvKey := types.NamespacedName{
						Name:      volume.DataVolume.VolumeName,
						Namespace: vm.Namespace,
					}
					if err := r.Client.Get(ctx, dvKey, &dv); err != nil {
						return nil, err
					}
					return &dv, nil
				}
				ready, err := cdiv1beta1.IsPopulated(&pvc, getDataVolumeFunc)
				if err != nil {
					return nil, err
				}
				if !ready {
					return nil, fmt.Errorf("data volume is not ready: %s", volume.DataVolume.VolumeName)
				}
			}

			if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == corev1.PersistentVolumeBlock {
				volumeDevice := corev1.VolumeDevice{
					Name:       volume.Name,
					DevicePath: "/mnt/" + volume.Name,
				}
				vmPod.Spec.Containers[0].VolumeDevices = append(vmPod.Spec.Containers[0].VolumeDevices, volumeDevice)
			} else {
				volumeMount := corev1.VolumeMount{
					Name:      volume.Name,
					MountPath: "/mnt/" + volume.Name,
				}
				vmPod.Spec.Containers[0].VolumeMounts = append(vmPod.Spec.Containers[0].VolumeMounts, volumeMount)
			}
		default:
			// ignored
		}
	}

	var networks []netv1.NetworkSelectionElement
	for i, network := range vm.Spec.Networks {
		var iface *virtv1alpha1.Interface
		for j := range vm.Spec.Instance.Interfaces {
			if vm.Spec.Instance.Interfaces[j].Name == network.Name {
				iface = &vm.Spec.Instance.Interfaces[j]
				break
			}
		}
		if iface == nil {
			return nil, fmt.Errorf("interface not found for network: %s", network.Name)
		}

		if iface.Masquerade != nil {
			vmPod.Spec.InitContainers = append(vmPod.Spec.InitContainers, corev1.Container{
				Name:  "enable-ip-forward",
				Image: r.PrerunnerImageName,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &[]bool{true}[0],
				},
				Command: []string{"sysctl", "-w", "net.ipv4.ip_forward=1"},
			})
		}

		switch {
		case network.Multus != nil:
			networks = append(networks, netv1.NetworkSelectionElement{
				Name:             network.Multus.NetworkName,
				InterfaceRequest: fmt.Sprintf("net%d", i),
				MacRequest:       iface.MAC,
			})

			var nad netv1.NetworkAttachmentDefinition
			nadKey := types.NamespacedName{
				Name:      network.Multus.NetworkName,
				Namespace: vm.Namespace,
			}
			if err := r.Client.Get(ctx, nadKey, &nad); err != nil {
				return nil, fmt.Errorf("get NAD: %s", err)
			}

			resourceName := nad.Annotations["k8s.v1.cni.cncf.io/resourceName"]
			if resourceName != "" {
				incrementContainerResource(&vmPod.Spec.Containers[0], resourceName)
			}
			vmPod.Spec.Containers[0].Env = append(vmPod.Spec.Containers[0].Env, corev1.EnvVar{
				Name: "NETWORK_STATUS",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: fmt.Sprintf("metadata.annotations['%s']", netv1.NetworkStatusAnnot),
					},
				},
			})

			if iface.VhostUser != nil {
				type nadConfig struct {
					Type                      string `json:"type"`
					VhostUserSocketVolumeName string `json:"vhost_user_socket_volume_name,omitempty"`
					VhostUserSocketName       string `json:"vhost_user_socket_name,omitempty"`
				}

				var cfg nadConfig
				if err := json.Unmarshal([]byte(nad.Spec.Config), &cfg); err != nil {
					return nil, fmt.Errorf("unmarshal NAD config: %s", err)
				}

				switch cfg.Type {
				case "kube-ovn":
					if vmPod.Spec.NodeSelector == nil {
						vmPod.Spec.NodeSelector = map[string]string{}
					}
					vmPod.Spec.NodeSelector["ovn.kubernetes.io/ovs_dp_type"] = "userspace"
					vmPod.Annotations["ovn-dpdk.default.ovn.kubernetes.io/mac_address"] = iface.MAC

					vmPod.Spec.Volumes = append(vmPod.Spec.Volumes, corev1.Volume{
						Name: cfg.VhostUserSocketVolumeName,
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					})
					volumeMount := corev1.VolumeMount{
						Name:      cfg.VhostUserSocketVolumeName,
						MountPath: "/var/run/vhost-user",
					}
					vmPod.Spec.Containers[0].VolumeMounts = append(vmPod.Spec.Containers[0].VolumeMounts, volumeMount)

					vmPod.Spec.Containers[0].Env = append(vmPod.Spec.Containers[0].Env, corev1.EnvVar{
						Name:  "VHOST_USER_SOCKET",
						Value: fmt.Sprintf("/var/run/vhost-user/%s", cfg.VhostUserSocketName),
					})
				default:
					return nil, fmt.Errorf("CNI plugin %s is not supported for vhost-uesr", cfg.Type)
				}
			}
		default:
			// ignored
		}
	}

	if len(networks) > 0 {
		networksJSON, err := json.Marshal(networks)
		if err != nil {
			return nil, fmt.Errorf("marshal networks: %s", err)
		}
		vmPod.Annotations["k8s.v1.cni.cncf.io/networks"] = string(networksJSON)
	}

	return &vmPod, nil
}

func (r *VMReconciler) buildTargetVMPod(ctx context.Context, vm *virtv1alpha1.VirtualMachine) (*corev1.Pod, error) {
	pod, err := r.buildVMPod(ctx, vm)
	if err != nil {
		return nil, err
	}
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "RECEIVE_MIGRATION",
		Value: "true",
	})

	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}
	affinity := pod.Spec.Affinity

	if affinity.PodAntiAffinity == nil {
		affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	}
	podAntiAffinity := affinity.PodAntiAffinity
	if podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = []corev1.PodAffinityTerm{}
	}
	podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"virtink.io/vm.name": vm.Name,
			},
		},
		TopologyKey: "kubernetes.io/hostname",
	})
	return pod, nil
}

func (r *VMReconciler) reconcileVMConditions(ctx context.Context, vm *virtv1alpha1.VirtualMachine, vmPod *corev1.Pod) error {
	for _, condition := range vmPod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			readyCondition := metav1.Condition{
				Type:    string(virtv1alpha1.VirtualMachineReady),
				Status:  metav1.ConditionStatus(condition.Status),
				Reason:  condition.Reason,
				Message: condition.Message,
			}
			if readyCondition.Reason == "" {
				readyCondition.Reason = string(readyCondition.Status)
			}
			meta.SetStatusCondition(&vm.Status.Conditions, readyCondition)
		}
	}

	if meta.FindStatusCondition(vm.Status.Conditions, string(virtv1alpha1.VirtualMachineMigratable)) == nil {
		migratableCondition, err := r.calculateMigratableCondition(ctx, vm)
		if err != nil {
			return fmt.Errorf("calculate VM migratable condition: %s", err)
		}
		meta.SetStatusCondition(&vm.Status.Conditions, *migratableCondition)
	}
	return nil
}

func (r *VMReconciler) calculateMigratableCondition(ctx context.Context, vm *virtv1alpha1.VirtualMachine) (*metav1.Condition, error) {
	if vm.Spec.Instance.CPU.DedicatedCPUPlacement {
		return &metav1.Condition{
			Type:    string(virtv1alpha1.VirtualMachineMigratable),
			Status:  metav1.ConditionFalse,
			Reason:  "CPUNotMigratable",
			Message: "migration is disabled when VM has enabled dedicated CPU placement",
		}, nil
	}

	for _, network := range vm.Spec.Networks {
		for _, iface := range vm.Spec.Instance.Interfaces {
			if iface.Name != network.Name {
				continue
			}
			if network.Pod != nil && iface.Bridge != nil {
				return &metav1.Condition{
					Type:    string(virtv1alpha1.VirtualMachineMigratable),
					Status:  metav1.ConditionFalse,
					Reason:  "InterfaceNotMigratable",
					Message: "migration is disabled when VM has a bridged interface to the pod network",
				}, nil
			}
			if iface.SRIOV != nil {
				return &metav1.Condition{
					Type:    string(virtv1alpha1.VirtualMachineMigratable),
					Status:  metav1.ConditionFalse,
					Reason:  "InterfaceNotMigratable",
					Message: "migration is disabled when VM has a SR-IOV interface",
				}, nil
			}
			if iface.VhostUser != nil {
				return &metav1.Condition{
					Type:    string(virtv1alpha1.VirtualMachineMigratable),
					Status:  metav1.ConditionFalse,
					Reason:  "InterfaceNotMigratable",
					Message: "migration is disable when VM has a vhost-user interface",
				}, nil
			}
		}
	}

	for _, volume := range vm.Spec.Volumes {
		if volume.ContainerRootfs != nil {
			return &metav1.Condition{
				Type:    string(virtv1alpha1.VirtualMachineMigratable),
				Status:  metav1.ConditionFalse,
				Reason:  "VolumeNotMigratable",
				Message: "migration is disabled when VM has a containerRootfs volume",
			}, nil
		}
		if volume.ContainerDisk != nil {
			return &metav1.Condition{
				Type:    string(virtv1alpha1.VirtualMachineMigratable),
				Status:  metav1.ConditionFalse,
				Reason:  "VolumeNotMigratable",
				Message: "migration is disabled when VM has a containerDisk volume",
			}, nil
		}
	}

	return &metav1.Condition{
		Type:   string(virtv1alpha1.VirtualMachineMigratable),
		Status: metav1.ConditionTrue,
		Reason: "Migratable",
	}, nil
}

func (r *VMReconciler) gcVMPods(ctx context.Context, vm *virtv1alpha1.VirtualMachine) error {
	var vmPodList corev1.PodList
	if err := r.List(ctx, &vmPodList, client.MatchingFields{"vmUID": string(vm.UID)}); err != nil {
		return fmt.Errorf("list VM Pods: %s", err)
	}

	for _, vmPod := range vmPodList.Items {
		if vmPod.DeletionTimestamp != nil && !vmPod.DeletionTimestamp.IsZero() {
			continue
		}

		if vmPod.Name == vm.Status.VMPodName || vm.Status.Migration != nil && vmPod.Name == vm.Status.Migration.TargetVMPodName {
			continue
		}

		if err := r.Delete(ctx, &vmPod); client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("delete VM Pod: %s", err)
		}
		r.Recorder.Eventf(vm, corev1.EventTypeNormal, "DeletedVMPod", fmt.Sprintf("Deleted VM Pod %q", vmPod.Name))
	}
	return nil
}

func incrementContainerResource(container *corev1.Container, resourceName string) {
	if container.Resources.Requests == nil {
		container.Resources.Requests = corev1.ResourceList{}
	}
	request := container.Resources.Requests[corev1.ResourceName(resourceName)]
	request = resource.MustParse(strconv.FormatInt(request.Value()+1, 10))
	container.Resources.Requests[corev1.ResourceName(resourceName)] = request

	if container.Resources.Limits == nil {
		container.Resources.Limits = corev1.ResourceList{}
	}
	limit := container.Resources.Limits[corev1.ResourceName(resourceName)]
	limit = resource.MustParse(strconv.FormatInt(limit.Value()+1, 10))
	container.Resources.Limits[corev1.ResourceName(resourceName)] = limit
}

func (r *VMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, "vmUID", func(obj client.Object) []string {
		pod := obj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)
		if owner != nil && owner.APIVersion == virtv1alpha1.SchemeGroupVersion.String() && owner.Kind == "VirtualMachine" {
			return []string{string(owner.UID)}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("index Pods by VM UID: %s", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&virtv1alpha1.VirtualMachine{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
