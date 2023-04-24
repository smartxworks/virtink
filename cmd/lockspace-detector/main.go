package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
	"github.com/smartxworks/virtink/pkg/sanlock"
)

const (
	LockProtectionFinalizer = "virtink.smartx.com/lock-protection"
)

var (
	scheme = runtime.NewScheme()

	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(virtv1alpha1.AddToScheme(scheme))
}

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=lockspaces,verbs=get;list;watch
//+kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=locks,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=locks/status,verbs=get;update
//+kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines,verbs=get;list;watch
//+kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines/status,verbs=get;update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;update;patch

func main() {
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	lockspace := os.Getenv("LOCKSPACE_NAME")

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:           scheme,
		LeaderElection:   true,
		LeaderElectionID: fmt.Sprintf("%s-detector.virtink.smartx.com", lockspace),
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	ioTimeout, err := strconv.Atoi(os.Getenv("IO_TIMEOUT"))
	if err != nil {
		setupLog.Error(err, "failed to get io_timeout_seconds")
	}
	if err = (&Detector{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Recorder:          mgr.GetEventRecorderFor("lockspace-detector"),
		lockspace:         lockspace,
		ioTimeout:         time.Duration(ioTimeout) * time.Second,
		freeStateDetector: make(map[string]chan struct{}),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Detector")
		os.Exit(1)
	}

	if err = (&LockReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Recorder:  mgr.GetEventRecorderFor("lock-controller"),
		lockspace: lockspace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Lock")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

type Detector struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	lockspace string
	ioTimeout time.Duration

	mutex             sync.Mutex
	freeStateDetector map[string]chan struct{}
}

func (d *Detector) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ls, err := d.getLockspaceOrNil(ctx, d.lockspace)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("get Lockspace: %s", err)
	}
	if ls == nil {
		return ctrl.Result{}, nil
	}

	var node corev1.Node
	if err := d.Get(ctx, req.NamespacedName, &node); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := d.detectNode(ctx, &node); err != nil {
		reconcileErr := reconcileError{}
		if errors.As(err, &reconcileErr) {
			return reconcileErr.Result, nil
		}
		return ctrl.Result{}, fmt.Errorf("reconcile Node: %s", err)
	}
	return ctrl.Result{RequeueAfter: 2 * d.ioTimeout}, nil
}

func (d *Detector) getLockspaceOrNil(ctx context.Context, lockspace string) (*virtv1alpha1.Lockspace, error) {
	lsKey := types.NamespacedName{
		Name: lockspace,
	}
	var ls virtv1alpha1.Lockspace
	if err := d.Client.Get(ctx, lsKey, &ls); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	return &ls, nil
}

func (d *Detector) detectNode(ctx context.Context, node *corev1.Node) error {
	if node == nil || node.DeletionTimestamp != nil {
		d.stopFreeStateDetector(node.Name)
		return nil
	}

	id, exist := node.Annotations["virtink.smartx.com/sanlock-host-id"]
	if !exist {
		return fmt.Errorf("sanlock host %s ID not found", node.Name)
	}
	hostID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return fmt.Errorf("parse uint: %s", err)
	}

	hostStatus, err := sanlock.GetHostStatus(d.lockspace, hostID)
	if err != nil {
		panic(fmt.Errorf("get Sanlock host status: %s", err))
	}

	switch hostStatus {
	case sanlock.HostStatusLive:
		delete(node.Labels, fmt.Sprintf("virtink.smartx.com/dead-lockspace-%s", d.lockspace))
		if err := d.Client.Update(ctx, node); err != nil {
			return fmt.Errorf("update Node labels: %s", err)
		}
		d.stopFreeStateDetector(node.Name)
		ctrl.LoggerFrom(ctx).Info("node is alive", "Node", node.Name)
	case sanlock.HostStatusDead:
		node.Labels[fmt.Sprintf("virtink.smartx.com/dead-lockspace-%s", d.lockspace)] = ""
		if err := d.Client.Update(ctx, node); err != nil {
			return fmt.Errorf("update Node labels: %s", err)
		}
		ctrl.LoggerFrom(ctx).Info("node is dead", "Node", node.Name)

		if err := d.updateVMOnDeadNode(ctx, node.Name); err != nil {
			return fmt.Errorf("update VM on dead Node: %s", err)
		}
	case sanlock.HostStatusFree:
		if _, exist := node.Labels[fmt.Sprintf("virtink.smartx.com/dead-lockspace-%s", d.lockspace)]; !exist {
			go d.startFreeStateDetector(ctx, node)
		}
	case sanlock.HostStatusFail:
		node.Labels[fmt.Sprintf("virtink.smartx.com/dead-lockspace-%s", d.lockspace)] = ""
		if err := d.Client.Update(ctx, node); err != nil {
			return fmt.Errorf("update Node labels: %s", err)
		}
		ctrl.LoggerFrom(ctx).Info("node is failed", "Node", node.Name)

		return reconcileError{ctrl.Result{RequeueAfter: d.ioTimeout}}
	default:
		// ignore
	}

	return nil
}

func (d *Detector) updateVMOnDeadNode(ctx context.Context, node string) error {
	var vmList virtv1alpha1.VirtualMachineList
	vmSelector := client.MatchingFields{".status.nodeName": node}
	if err := d.Client.List(ctx, &vmList, vmSelector); err != nil {
		return fmt.Errorf("list VM: %s", err)
	}
	for _, vm := range vmList.Items {
		if vm.Status.Phase != virtv1alpha1.VirtualMachineScheduling && vm.Status.Phase != virtv1alpha1.VirtualMachineScheduled && vm.Status.Phase != virtv1alpha1.VirtualMachineRunning {
			continue
		}
		for _, l := range vm.Spec.Locks {
			var lock virtv1alpha1.Lock
			lockKey := types.NamespacedName{
				Namespace: vm.Namespace,
				Name:      l,
			}
			if err := d.Client.Get(ctx, lockKey, &lock); err != nil {
				return fmt.Errorf("get Lock: %s", err)
			}
			if lock.Spec.LockspaceName == d.lockspace {
				vm.Status.Phase = virtv1alpha1.VirtualMachineFailed
				if vm.Spec.EnableHA {
					vm.Status = virtv1alpha1.VirtualMachineStatus{
						Phase: virtv1alpha1.VirtualMachinePending,
					}
				}
				if err := d.Client.Status().Update(ctx, &vm); err != nil {
					return fmt.Errorf("update VM status: %s", err)
				}
				break
			}
		}
	}

	return nil
}

func (d *Detector) startFreeStateDetector(ctx context.Context, node *corev1.Node) {
	if _, exist := d.freeStateDetector[node.Name]; !exist {
		d.mutex.Lock()
		stop := make(chan struct{})
		d.freeStateDetector[node.Name] = stop
		d.mutex.Unlock()

		timeout := time.NewTicker(8*d.ioTimeout + sanlock.WatchdogFireTimeoutDefaultSeconds*time.Second)
		defer timeout.Stop()

		select {
		case <-stop:
			// no-operation
		case <-timeout.C:
			nodeKey := types.NamespacedName{
				Name: node.Name,
			}
			if err := d.Client.Get(ctx, nodeKey, node); err != nil {
				ctrl.LoggerFrom(ctx).Error(err, "failed to get Node in free state detector", "Node", node.Name)
			} else {
				node.Labels[fmt.Sprintf("virtink.smartx.com/dead-lockspace-%s", d.lockspace)] = ""
				if err := d.Client.Update(ctx, node); err != nil {
					ctrl.LoggerFrom(ctx).Error(err, "failed to update Node in free state detector", "Node", node.Name)
				}
				ctrl.LoggerFrom(ctx).Info("node is dead, detected by free state detector", "Node", node.Name)

				if err := d.updateVMOnDeadNode(ctx, node.Name); err != nil {
					ctrl.LoggerFrom(ctx).Error(err, "failed to update VM on dead Node", "Node", node.Name)
				}
			}
		}

		d.mutex.Lock()
		delete(d.freeStateDetector, node.Name)
		d.mutex.Unlock()
	}
}

func (d *Detector) stopFreeStateDetector(node string) {
	if ch, exist := d.freeStateDetector[node]; exist {
		close(ch)
	}
}

func (d *Detector) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &virtv1alpha1.VirtualMachine{}, ".status.nodeName", func(obj client.Object) []string {
		vm := obj.(*virtv1alpha1.VirtualMachine)
		return []string{vm.Status.NodeName}
	}); err != nil {
		return fmt.Errorf("index VM by Node name: %s", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(d)
}

type LockReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	lockspace string
}

func (r *LockReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var lock virtv1alpha1.Lock
	if err := r.Get(ctx, req.NamespacedName, &lock); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if lock.Spec.LockspaceName != r.lockspace {
		return ctrl.Result{}, nil
	}

	status := lock.Status.DeepCopy()
	if err := r.reconcile(ctx, &lock); err != nil {
		reconcileErr := reconcileError{}
		if errors.As(err, &reconcileErr) {
			return reconcileErr.Result, nil
		}
		r.Recorder.Eventf(&lock, corev1.EventTypeWarning, "FailedReconcile", "Failed to reconcile Lock: %s", err)
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(lock.Status, status) {
		if err := r.Status().Update(ctx, &lock); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("update Lock status: %s", err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *LockReconciler) reconcile(ctx context.Context, lock *virtv1alpha1.Lock) error {
	if lock.DeletionTimestamp != nil {
		if err := r.cleanupLock(ctx, lock); err != nil {
			return fmt.Errorf("cleanup Lock: %s", err)
		}
		return nil
	}

	if !controllerutil.ContainsFinalizer(lock, LockProtectionFinalizer) {
		controllerutil.AddFinalizer(lock, LockProtectionFinalizer)
		return r.Client.Update(ctx, lock)
	}

	if !lock.Status.Ready {
		if err := r.reconcileNotReadyLock(ctx, lock); err != nil {
			return fmt.Errorf("reconcile not ready Lock: %s", err)
		}
		lock.Status.Ready = true
	}

	return nil
}

func (r *LockReconciler) cleanupLock(ctx context.Context, lock *virtv1alpha1.Lock) error {
	var vmList virtv1alpha1.VirtualMachineList
	vmSelector := client.MatchingFields{".spec.locks": lock.Name}
	if err := r.Client.List(ctx, &vmList, vmSelector); err != nil {
		return fmt.Errorf("list VM: %s", err)
	}
	isVMPodFound := false
	for _, vm := range vmList.Items {
		pod, err := r.getVMPodOrNil(ctx, &vm)
		if err != nil {
			return fmt.Errorf("get VM Pod: %s", err)
		}
		if pod != nil {
			isVMPodFound = true
			// TODO: shutdown the VMM
			if err := r.Client.Delete(ctx, pod); err != nil {
				return fmt.Errorf("delete VM Pod: %s", err)
			}
		}
	}
	if isVMPodFound {
		return reconcileError{ctrl.Result{RequeueAfter: time.Second}}
	}

	leaseFilePath := filepath.Join("/var/lib/sanlock", lock.Spec.LockspaceName, "leases")
	_, err := sanlock.SearchResource(lock.Spec.LockspaceName, leaseFilePath, lock.Name)
	if err != nil {
		if err == sanlock.ENOENT {
			// no-operation
		} else {
			return fmt.Errorf("search Sanlock resource: %s", err)
		}
	} else {
		if err := sanlock.DeleteResource(lock.Spec.LockspaceName, leaseFilePath, lock.Name); err != nil {
			return fmt.Errorf("delete Sanlock resource: %s", err)
		}
	}

	if controllerutil.ContainsFinalizer(lock, LockProtectionFinalizer) {
		controllerutil.RemoveFinalizer(lock, LockProtectionFinalizer)
		if err := r.Client.Update(ctx, lock); err != nil {
			return fmt.Errorf("update Lock finalizer: %s", err)
		}
	}

	return nil
}

func (r *LockReconciler) getVMPodOrNil(ctx context.Context, vm *virtv1alpha1.VirtualMachine) (*corev1.Pod, error) {
	var pod corev1.Pod
	podKey := types.NamespacedName{
		Namespace: vm.Namespace,
		Name:      vm.Status.VMPodName,
	}
	if err := r.Client.Get(ctx, podKey, &pod); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	if !metav1.IsControlledBy(&pod, vm) {
		return nil, fmt.Errorf("pod %q is not controlled by VM %q", namespacedName(&pod), namespacedName(vm))
	}
	return &pod, nil
}

func (r *LockReconciler) reconcileNotReadyLock(ctx context.Context, lock *virtv1alpha1.Lock) error {
	leaseFilePath := filepath.Join("/var/lib/sanlock", lock.Spec.LockspaceName, "leases")
	offset, err := sanlock.SearchResource(lock.Spec.LockspaceName, leaseFilePath, lock.Name)
	if err != nil {
		if err == sanlock.ENOENT {
			offset, err = sanlock.CreateResource(lock.Spec.LockspaceName, leaseFilePath, lock.Name)
			if err != nil {
				return fmt.Errorf("create Sanlock resource: %s", err)
			}
		} else {
			return fmt.Errorf("search Sanlock resource: %s", err)
		}
	}
	lock.Status.Offset = offset
	return nil
}

func (r *LockReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &virtv1alpha1.VirtualMachine{}, ".spec.locks", func(obj client.Object) []string {
		vm := obj.(*virtv1alpha1.VirtualMachine)
		return vm.Spec.Locks
	}); err != nil {
		return fmt.Errorf("index VM by Lock: %s", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&virtv1alpha1.Lock{}).
		Complete(r)
}

func namespacedName(obj metav1.ObjectMetaAccessor) types.NamespacedName {
	meta := obj.GetObjectMeta()
	return types.NamespacedName{
		Namespace: meta.GetNamespace(),
		Name:      meta.GetName(),
	}
}

type reconcileError struct {
	ctrl.Result
}

func (rerr reconcileError) Error() string {
	return fmt.Sprintf("requeue: %v, requeueAfter: %s", rerr.Requeue, rerr.RequeueAfter)
}
