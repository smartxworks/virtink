package controller

import (
	"context"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

type VMMReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachinemigrations,verbs=get;list;watch
// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachinemigrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines,verbs=get;list;watch
// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;update;patch

func (r *VMMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var vmm virtv1alpha1.VirtualMachineMigration
	if err := r.Get(ctx, req.NamespacedName, &vmm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	status := vmm.Status.DeepCopy()
	if err := r.reconcile(ctx, &vmm); err != nil {
		r.Recorder.Eventf(&vmm, corev1.EventTypeWarning, "FailedReconcile", "Failed to reconcile VMM: %s", err)
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(vmm.Status, status) {
		if err := r.Status().Update(ctx, &vmm); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("update VMM status: %s", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *VMMReconciler) reconcile(ctx context.Context, vmm *virtv1alpha1.VirtualMachineMigration) error {
	if vmm.DeletionTimestamp != nil && !vmm.DeletionTimestamp.IsZero() {
		return nil
	}
	var vm virtv1alpha1.VirtualMachine
	vmKey := client.ObjectKey{
		Name:      vmm.Spec.VMName,
		Namespace: vmm.Namespace,
	}
	vmNotFound := false
	if err := r.Client.Get(ctx, vmKey, &vm); err != nil {
		if apierrors.IsNotFound(err) {
			vmNotFound = true
		} else {
			return fmt.Errorf("get vm: %s", err)
		}
	}

	if vmm.Status.Phase == virtv1alpha1.VirtualMachineMigrationSucceeded ||
		vmm.Status.Phase == virtv1alpha1.VirtualMachineMigrationFailed {
		if vmNotFound || !vm.DeletionTimestamp.IsZero() || vm.Status.Migration == nil || vm.Status.Migration.UID != vmm.UID {
			return nil
		}

		vm.Status.Migration = nil
		if err := r.Client.Status().Update(ctx, &vm); err != nil {
			return fmt.Errorf("reset vm migration status: %s", err)
		}
		return nil
	}

	if vmNotFound || !vm.DeletionTimestamp.IsZero() || (vm.Status.Migration != nil && vm.Status.Migration.UID != vmm.UID) {
		vmm.Status.Phase = virtv1alpha1.VirtualMachineMigrationFailed
		return nil
	}

	if vm.Status.Migration == nil {
		vm.Status.Migration = &virtv1alpha1.VirtualMachineStatusMigration{
			UID: vmm.UID,
		}
		if err := r.Client.Status().Update(ctx, &vm); err != nil {
			return fmt.Errorf("set VM migration status: %s", err)
		}
		vmm.Status.SourceNodeName = vm.Status.NodeName
		return nil
	}

	vmm.Status.Phase = vm.Status.Migration.Phase
	if vm.Status.Migration.TargetNodeName != "" {
		vmm.Status.TargetNodeName = vm.Status.Migration.TargetNodeName
	}

	return nil
}

func (r *VMMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &virtv1alpha1.VirtualMachineMigration{}, ".metadata.uid", func(obj client.Object) []string {
		vmm := obj.(*virtv1alpha1.VirtualMachineMigration)
		return []string{string(vmm.UID)}
	}); err != nil {
		return fmt.Errorf("index VMM by UID: %s", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&virtv1alpha1.VirtualMachineMigration{}).
		Watches(&source.Kind{Type: &virtv1alpha1.VirtualMachine{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
				vm := obj.(*virtv1alpha1.VirtualMachine)
				if vm.Status.Migration == nil || vm.Status.Migration.UID == "" {
					return nil
				}

				var vmmList virtv1alpha1.VirtualMachineMigrationList
				if err := r.Client.List(context.Background(), &vmmList, client.MatchingFields{".metadata.uid": string(vm.Status.Migration.UID)}); err != nil {
					return nil
				}

				var requests []reconcile.Request
				for _, vmm := range vmmList.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: vmm.Namespace,
							Name:      vmm.Name,
						},
					})
				}
				return requests
			})).
		Complete(r)
}
