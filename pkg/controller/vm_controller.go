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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/storage/names"
	"k8s.io/client-go/tools/record"
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

	// TODO: GC
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
	case virtv1alpha1.VirtualMachineScheduling, virtv1alpha1.VirtualMachineScheduled, virtv1alpha1.VirtualMachineRunning:
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
					Privileged: func() *bool { v := true; return &v }(),
				},
				Args: []string{base64.StdEncoding.EncodeToString(vmJSON)},
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
			Args:            []string{volumeMount.MountPath + "/vmlinux"},
			VolumeMounts:    []corev1.VolumeMount{volumeMount},
		})
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
				Args:            []string{volumeMount.MountPath + "/disk.raw"},
				VolumeMounts:    []corev1.VolumeMount{volumeMount},
			})
		case volume.CloudInit != nil:
			initContainer := corev1.Container{
				Name:    "init-volume-" + volume.Name,
				Image:   vmPod.Spec.Containers[0].Image,
				Command: []string{"virt-init-volume"},
				Args:    []string{"cloud-init"},
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
				Args:            []string{volumeMount.MountPath + "/rootfs.raw", strconv.FormatInt(volume.ContainerRootfs.Size.Value(), 10)},
				VolumeMounts:    []corev1.VolumeMount{volumeMount},
			})
		default:
			// ignored
		}
	}

	var networks []netv1.NetworkSelectionElement
	for i, network := range vm.Spec.Networks {
		switch {
		case network.Multus != nil:
			networks = append(networks, netv1.NetworkSelectionElement{
				Name:             network.Multus.NetworkName,
				InterfaceRequest: fmt.Sprintf("net%d", i),
			})
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

func (r *VMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&virtv1alpha1.VirtualMachine{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
