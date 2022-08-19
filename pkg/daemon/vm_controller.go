package daemon

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
	"github.com/smartxworks/virtink/pkg/cloudhypervisor"
	"github.com/smartxworks/virtink/pkg/tlsutil"
)

type VMReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	NodeName string
	NodeIP   string
	RelayProvider

	migrationControlBlocks map[types.UID]migrationControlBlock
	mutex                  sync.Mutex
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
	log := ctrl.LoggerFrom(ctx)
	shouldReconcile := (vm.Status.NodeName != "" && vm.Status.NodeName == r.NodeName) ||
		(vm.Status.Migration != nil && vm.Status.Migration.TargetNodeName != "" && vm.Status.Migration.TargetNodeName == r.NodeName)
	if !shouldReconcile {
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
		if vm.Status.Migration == nil {
			if vm.Status.NodeName != r.NodeName {
				return nil
			}
			vmInfo, err := r.getCloudHypervisorClient(vm).VmInfo(ctx)
			if err != nil {
				// TODO: ignore VM not found error
				return fmt.Errorf("get VM info: %s", err)
			}

			if vmInfo.State == "Running" || vmInfo.State == "Paused" {
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
			} else {
				vm.Status.Phase = virtv1alpha1.VirtualMachineSucceeded
			}
		} else {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			migrationControlBlock := r.migrationControlBlocks[vm.UID]

			var daemonCertDirPath = "/var/lib/virtink/daemon/cert"
			switch vm.Status.Migration.Phase {
			case virtv1alpha1.VirtualMachineMigrationScheduled:
				if vm.Status.Migration.TargetNodeName == r.NodeName {
					ctx, cancel := context.WithCancel(context.Background())
					migrationControlBlock.ReceiveMigrationCancelFunc = cancel

					receiveMigrationErrChan := make(chan error, 1)
					go func() {
						if err := r.getMigrationTargetCloudHypervisorClient(vm).VmReceiveMigration(ctx, &cloudhypervisor.ReceiveMigrationData{
							ReceiverUrl: "unix:/var/run/virtink/rx.sock",
						}); err != nil {
							receiveMigrationErrChan <- err
						}
					}()
					migrationControlBlock.ReceiveMigrationErrCh = receiveMigrationErrChan

					receiveMigrationSocketPath := filepath.Join(getMigrationTargetVMSocketDirPath(vm), "rx.sock")
					if err := wait.PollImmediate(time.Second, 10*time.Second, func() (done bool, err error) {
						select {
						case err = <-migrationControlBlock.ReceiveMigrationErrCh:
							log.Error(err, "receive migration")
							r.Recorder.Eventf(vm, corev1.EventTypeWarning, "FailedMigrate", "Failed to receive migration on %s: %s", vm.Status.Migration.TargetNodeName, err)
							vm.Status.Migration.Phase = virtv1alpha1.VirtualMachineMigrationFailed
							return true, nil
						default:
							if _, err := os.Stat(receiveMigrationSocketPath); err != nil {
								if os.IsNotExist(err) {
									return false, nil
								}
								return false, err
							}
							return true, nil
						}
					}); err != nil {
						return err
					}

					clientCACertPool, err := tlsutil.LoadCACert(daemonCertDirPath)
					if err != nil {
						return fmt.Errorf("load CA cert: %s", err)
					}
					tlsConfig := &tls.Config{
						GetCertificate: func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
							return tlsutil.LoadCert(daemonCertDirPath)
						},
						ClientAuth: tls.RequireAndVerifyClientCert,
						ClientCAs:  clientCACertPool,
					}

					port, err := r.RelayTCPToSocket(ctx, "0.0.0.0:0", tlsConfig, receiveMigrationSocketPath)
					if err != nil {
						return fmt.Errorf("start target relay: %s", err)
					}

					vm.Status.Migration.TargetNodePort = port
					vm.Status.Migration.TargetNodeIP = r.NodeIP
					vm.Status.Migration.Phase = virtv1alpha1.VirtualMachineMigrationTargetReady
				}
			case virtv1alpha1.VirtualMachineMigrationTargetReady:
				if vm.Status.NodeName == r.NodeName {
					ctx, cancel := context.WithCancel(context.Background())
					migrationControlBlock.SendMigrationCancelFunc = cancel

					tlsConfig := &tls.Config{
						InsecureSkipVerify: true,
						GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
							return tlsutil.LoadCert(daemonCertDirPath)
						},
					}
					if err := r.RelaySocketToTCP(ctx, filepath.Join(getVMSocketDirPath(vm), "tx.sock"), fmt.Sprintf("%s:%d", vm.Status.Migration.TargetNodeIP, vm.Status.Migration.TargetNodePort), tlsConfig); err != nil {
						return fmt.Errorf("start source relay: %s", err)
					}

					sendMigrationErrChan := make(chan error, 1)
					go func() {
						if err := r.getCloudHypervisorClient(vm).VmSendMigration(ctx, &cloudhypervisor.SendMigrationData{
							DestinationUrl: "unix:/var/run/virtink/tx.sock",
						}); err != nil {
							sendMigrationErrChan <- err
						}
					}()
					migrationControlBlock.SendMigrationErrCh = sendMigrationErrChan
					vm.Status.Migration.Phase = virtv1alpha1.VirtualMachineMigrationRunning
				}
			case virtv1alpha1.VirtualMachineMigrationRunning:
				if vm.Status.NodeName == r.NodeName {
					var vmPod corev1.Pod
					vmPodKey := types.NamespacedName{
						Name:      vm.Status.VMPodName,
						Namespace: vm.Namespace,
					}
					if err := r.Get(ctx, vmPodKey, &vmPod); err != nil {
						return fmt.Errorf("get VM Pod: %s", err)
					}

					if vmPod.Status.Phase == corev1.PodSucceeded {
						vm.Status.Migration.Phase = virtv1alpha1.VirtualMachineMigrationSent
					} else if migrationControlBlock.SendMigrationErrCh == nil {
						vm.Status.Migration.Phase = virtv1alpha1.VirtualMachineMigrationFailed
					} else {
						select {
						case err := <-migrationControlBlock.SendMigrationErrCh:
							if err != nil {
								r.Recorder.Eventf(vm, corev1.EventTypeWarning, "FailedMigrate", "Failed to migrate VM to %s: %s", vm.Status.Migration.TargetNodeName, err)
								vm.Status.Migration.Phase = virtv1alpha1.VirtualMachineMigrationFailed
							}
						default:
							log.Info("VM is sending migration")
							return nil
						}
					}
					if sendDomainCancelFunc := migrationControlBlock.SendMigrationCancelFunc; sendDomainCancelFunc != nil {
						sendDomainCancelFunc()
					}
				}
			case virtv1alpha1.VirtualMachineMigrationSent:
				if vm.Status.Migration.TargetNodeName == r.NodeName {
					vmInfo, err := r.getMigrationTargetCloudHypervisorClient(vm).VmInfo(ctx)
					if err != nil {
						return err
					}
					switch vmInfo.State {
					case "Running":
						vm.Status.Migration.Phase = virtv1alpha1.VirtualMachineMigrationSucceeded
						vm.Status.NodeName = vm.Status.Migration.TargetNodeName
						vm.Status.VMPodName = vm.Status.Migration.TargetVMPodName
						vm.Status.VMPodUID = vm.Status.Migration.TargetVMPodUID
					default:
						log.Info("waiting target VM being Running")
						return nil
					}

					if receiveDomainCancelFunc := migrationControlBlock.ReceiveMigrationCancelFunc; receiveDomainCancelFunc != nil {
						receiveDomainCancelFunc()
					}
				}
			case virtv1alpha1.VirtualMachineMigrationSucceeded, virtv1alpha1.VirtualMachineMigrationFailed:
				delete(r.migrationControlBlocks, vm.UID)
			}

			r.migrationControlBlocks[vm.UID] = migrationControlBlock
			if vm.Status.Migration.Phase == virtv1alpha1.VirtualMachineMigrationFailed ||
				vm.Status.Migration.Phase == virtv1alpha1.VirtualMachineMigrationSucceeded ||
				(vm.Status.Migration.Phase == virtv1alpha1.VirtualMachineMigrationSent && vm.Status.NodeName == r.NodeName) {
				delete(r.migrationControlBlocks, vm.UID)
			}
		}
	}
	return nil
}

func (r *VMReconciler) getCloudHypervisorClient(vm *virtv1alpha1.VirtualMachine) *cloudhypervisor.Client {
	return cloudhypervisor.NewClient(filepath.Join(getVMSocketDirPath(vm), "ch.sock"))
}

func getVMSocketDirPath(vm *virtv1alpha1.VirtualMachine) string {
	return filepath.Join("var/lib/kubelet/pods", string(vm.Status.VMPodUID), "volumes/kubernetes.io~empty-dir/virtink/")
}

func (r *VMReconciler) getMigrationTargetCloudHypervisorClient(vm *virtv1alpha1.VirtualMachine) *cloudhypervisor.Client {
	return cloudhypervisor.NewClient(filepath.Join(getMigrationTargetVMSocketDirPath(vm), "ch.sock"))
}
func getMigrationTargetVMSocketDirPath(vm *virtv1alpha1.VirtualMachine) string {
	return filepath.Join("/var/lib/kubelet/pods", string(vm.Status.Migration.TargetVMPodUID), "volumes/kubernetes.io~empty-dir/virtink/")
}

func (r *VMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.migrationControlBlocks = map[types.UID]migrationControlBlock{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&virtv1alpha1.VirtualMachine{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

//go:generate mockgen -destination=mock/relay_provider.go -package=mock . RelayProvider

type RelayProvider interface {
	RelaySocketToTCP(ctx context.Context, socketPath string, tcpAddr string, tlsConfig *tls.Config) error
	RelayTCPToSocket(ctx context.Context, tcpAddr string, tlsConfig *tls.Config, socketPath string) (int, error)
}

type migrationControlBlock struct {
	SendMigrationErrCh         <-chan error
	SendMigrationCancelFunc    context.CancelFunc
	ReceiveMigrationErrCh      <-chan error
	ReceiveMigrationCancelFunc context.CancelFunc
}
