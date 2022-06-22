package daemon

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
	"github.com/smartxworks/virtink/pkg/cloudhypervisor"
)

type VMReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	NodeName string
}

// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch

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
	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

func (r *VMReconciler) reconcile(ctx context.Context, vm *virtv1alpha1.VirtualMachine) error {
	if vm.Status.NodeName == "" || vm.Status.NodeName != r.NodeName {
		return nil
	}

	if vm.DeletionTimestamp != nil && !vm.DeletionTimestamp.IsZero() {
		return nil
	}

	switch vm.Status.Phase {
	case virtv1alpha1.VirtualMachineScheduled:
		vmInfo, err := r.getCloudHypervisorClient(vm).VmInfo(ctx)
		if err != nil {
			// TODO: ignore VM not found error
			return fmt.Errorf("get VM info: %s", err)
		}

		if vmInfo.State == "Running" || vmInfo.State == "Paused" {
			vm.Status.Phase = virtv1alpha1.VirtualMachineRunning
		}
	case virtv1alpha1.VirtualMachineRunning:
		if vm.Spec.RunPolicy == virtv1alpha1.RunPolicyHalted {
			// TODO: shutdown with graceful timeout
			if err := r.getCloudHypervisorClient(vm).VmShutdown(ctx); err != nil {
				r.Recorder.Eventf(vm, corev1.EventTypeWarning, "FailedPowerOff", "Failed to powered off VM")
				return fmt.Errorf("power off VM: %s", err)
			}
		} else {
			switch vm.Status.PowerAction {
			case virtv1alpha1.VirtualMachinePowerOff:
				if err := r.getCloudHypervisorClient(vm).VmShutdown(ctx); err != nil {
					r.Recorder.Eventf(vm, corev1.EventTypeWarning, "FailedPowerOff", "Failed to powered off VM")
				} else {
					r.Recorder.Eventf(vm, corev1.EventTypeNormal, "PoweredOff", "Powered off VM")
				}
			case virtv1alpha1.VirtualMachineShutdown:
				if err := r.getCloudHypervisorClient(vm).VmPowerButton(ctx); err != nil {
					r.Recorder.Eventf(vm, corev1.EventTypeWarning, "FailedShutdown", "Failed to shutdown VM")
				} else {
					r.Recorder.Eventf(vm, corev1.EventTypeNormal, "Shutdown", "Shutdown VM")
				}
			case virtv1alpha1.VirtualMachineReset:
				if err := r.getCloudHypervisorClient(vm).VmReboot(ctx); err != nil {
					r.Recorder.Eventf(vm, corev1.EventTypeWarning, "FailedReset", "Failed to reset VM")
				} else {
					r.Recorder.Eventf(vm, corev1.EventTypeNormal, "Reset", "Reset VM")
				}
			case virtv1alpha1.VirtualMachineReboot:
				// TODO: reboot
				if err := r.getCloudHypervisorClient(vm).VmReboot(ctx); err != nil {
					r.Recorder.Eventf(vm, corev1.EventTypeWarning, "FailedReboot", "Failed to reboot VM")
				} else {
					r.Recorder.Eventf(vm, corev1.EventTypeNormal, "Rebooted", "Rebooted VM")
				}
			case virtv1alpha1.VirtualMachinePause:
				if err := r.getCloudHypervisorClient(vm).VmPause(ctx); err != nil {
					r.Recorder.Eventf(vm, corev1.EventTypeWarning, "FailedPause", "Failed to pause VM")
				} else {
					r.Recorder.Eventf(vm, corev1.EventTypeNormal, "Paused", "Paused VM")
				}
			case virtv1alpha1.VirtualMachineResume:
				if err := r.getCloudHypervisorClient(vm).VmResume(ctx); err != nil {
					r.Recorder.Eventf(vm, corev1.EventTypeWarning, "FailedResume", "Failed to resume VM")
				} else {
					r.Recorder.Eventf(vm, corev1.EventTypeNormal, "Resumed", "Resumed VM")
				}
			default:
				// ignored
			}

			vm.Status.PowerAction = ""
		}
	default:
		// ignored
	}
	return nil
}

func (r *VMReconciler) getCloudHypervisorClient(vm *virtv1alpha1.VirtualMachine) *cloudhypervisor.Client {
	return cloudhypervisor.NewClient(fmt.Sprintf("/var/lib/kubelet/pods/%s/volumes/kubernetes.io~empty-dir/virtink/ch.sock", string(vm.Status.VMPodUID)))
}

func (r *VMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&virtv1alpha1.VirtualMachine{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
