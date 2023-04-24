package controller

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

const (
	LockspaceProtectionFinalizer         = "virtink.smartx.com/lockspace-protection"
	LockspaceDetectorProtectionFinalizer = "virtink.smartx.com/lockspace-detector-protection"
	LockspaceAttacherProtectionFinalizer = "virtink.smartx.com/lockspace-attacher-protection"

	VirtinkNamespace = "virtink-system"
)

type LockspaceReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	DetectorImageName    string
	AttacherImageName    string
	InitializerImageName string
}

//+kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=lockspaces,verbs=get;list;watch;update
//+kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=lockspaces/status,verbs=get;update
//+kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=locks,verbs=get;list;watch;update;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;update;patch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;delete
//+kubebuilder:rbac:groups="apps",resources=daemonsets,verbs=get;list;watch;create;update;delete

func (r *LockspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var lockspace virtv1alpha1.Lockspace
	if err := r.Get(ctx, req.NamespacedName, &lockspace); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	status := lockspace.Status.DeepCopy()
	if err := r.reconcile(ctx, &lockspace); err != nil {
		reconcileErr := reconcileError{}
		if errors.As(err, &reconcileErr) {
			return reconcileErr.Result, nil
		}
		r.Recorder.Eventf(&lockspace, corev1.EventTypeWarning, "FailedReconcile", "Failed to reconcile Lockspace: %s", err)
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(lockspace.Status, status) {
		if err := r.Status().Update(ctx, &lockspace); err != nil {
			if apierrors.IsConflict(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, fmt.Errorf("update Lockspace status: %s", err)
		}
	}

	return reconcile.Result{}, nil
}

func (r *LockspaceReconciler) reconcile(ctx context.Context, lockspace *virtv1alpha1.Lockspace) error {
	if lockspace.DeletionTimestamp != nil {
		if err := r.cleanupLockspace(ctx, lockspace); err != nil {
			return fmt.Errorf("cleanup Lockspace: %s", err)
		}
	}

	if !controllerutil.ContainsFinalizer(lockspace, LockspaceProtectionFinalizer) {
		controllerutil.AddFinalizer(lockspace, LockspaceProtectionFinalizer)
		return r.Client.Update(ctx, lockspace)
	}

	if !lockspace.Status.Ready {
		if err := r.reconcileNotReadyLockspace(ctx, lockspace); err != nil {
			return fmt.Errorf("reconcile not ready Lockspace: %s", err)
		}
	} else {
		if err := r.reconcileReadyLockspace(ctx, lockspace); err != nil {
			return fmt.Errorf("reconcile ready Lockspace: %s", err)
		}
	}
	return nil
}

func (r *LockspaceReconciler) cleanupLockspace(ctx context.Context, lockspace *virtv1alpha1.Lockspace) error {
	var lockList virtv1alpha1.LockList
	lockSelector := client.MatchingFields{".spec.lockspaceName": lockspace.Name}
	if err := r.Client.List(ctx, &lockList, lockSelector); err != nil {
		return fmt.Errorf("list Lock: %s", err)
	}
	if len(lockList.Items) != 0 {
		for _, lock := range lockList.Items {
			if err := r.Client.Delete(ctx, &lock); err != nil {
				return fmt.Errorf("delete Lock %q: %s", namespacedName(&lock), err)
			}
		}
		return reconcileError{ctrl.Result{RequeueAfter: time.Second}}
	}

	detector, err := r.getLockspaceDetectorOrNil(ctx, lockspace)
	if err != nil {
		return fmt.Errorf("get Lockspace detector: %s", err)
	}
	if detector != nil {
		if controllerutil.ContainsFinalizer(detector, LockspaceDetectorProtectionFinalizer) {
			controllerutil.RemoveFinalizer(detector, LockspaceDetectorProtectionFinalizer)
			if err := r.Client.Update(ctx, detector); err != nil {
				return fmt.Errorf("update Lockspace detector: %s", err)
			}
		}

		if r.Client.Delete(ctx, detector); err != nil {
			return fmt.Errorf("delete Lockspace detector: %s", err)
		}
		return reconcileError{ctrl.Result{Requeue: true}}
	}

	attacher, err := r.getLockspaceAttacherOrNil(ctx, lockspace)
	if err != nil {
		return fmt.Errorf("get Lockspace attacher: %s", err)
	}
	if attacher != nil {
		if controllerutil.ContainsFinalizer(attacher, LockspaceAttacherProtectionFinalizer) {
			controllerutil.RemoveFinalizer(attacher, LockspaceAttacherProtectionFinalizer)
			if err := r.Client.Update(ctx, attacher); err != nil {
				return fmt.Errorf("update Lockspace attacher: %s", err)
			}
		}
	}

	if controllerutil.ContainsFinalizer(lockspace, LockspaceProtectionFinalizer) {
		controllerutil.RemoveFinalizer(lockspace, LockspaceProtectionFinalizer)
		if err := r.Client.Update(ctx, lockspace); err != nil {
			return fmt.Errorf("update Lockspace: %s", err)
		}
	}

	return nil
}

func (r *LockspaceReconciler) getLockspaceDetectorOrNil(ctx context.Context, lockspace *virtv1alpha1.Lockspace) (*appsv1.DaemonSet, error) {
	detectorKey := types.NamespacedName{
		Namespace: VirtinkNamespace,
		Name:      lockspace.GenerateDetectorName(),
	}
	var detector appsv1.DaemonSet
	if err := r.Client.Get(ctx, detectorKey, &detector); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	if !metav1.IsControlledBy(&detector, lockspace) {
		return nil, fmt.Errorf("detector %q is not controlled by Lockspace %q", namespacedName(&detector), namespacedName(lockspace))
	}
	return &detector, nil
}

func (r *LockspaceReconciler) getLockspaceAttacherOrNil(ctx context.Context, lockspace *virtv1alpha1.Lockspace) (*appsv1.DaemonSet, error) {
	attacherKey := types.NamespacedName{
		Namespace: VirtinkNamespace,
		Name:      lockspace.GenerateAttacherName(),
	}
	var attacher appsv1.DaemonSet
	if err := r.Client.Get(ctx, attacherKey, &attacher); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	if !metav1.IsControlledBy(&attacher, lockspace) {
		return nil, fmt.Errorf("attacher %q is not controlled by Lockspace %q", namespacedName(&attacher), namespacedName(lockspace))
	}
	return &attacher, nil
}

func (r *LockspaceReconciler) reconcileNotReadyLockspace(ctx context.Context, lockspace *virtv1alpha1.Lockspace) error {
	pvc, err := r.getLockspacePVCOrNil(ctx, lockspace)
	if err != nil {
		return fmt.Errorf("get Lockspace PVC: %s", err)
	}
	if pvc == nil {
		if err := r.createLockspacePVC(ctx, lockspace); err != nil {
			return fmt.Errorf("create Lockspace PVC: %s", err)
		}
		return nil
	}

	if pvc.Status.Phase == corev1.ClaimBound {
		initializer, err := r.getLockspaceInitializerOrNil(ctx, lockspace)
		if err != nil {
			return fmt.Errorf("get Lockspace initializer: %s", err)
		}
		if initializer == nil {
			if err := r.createLockspaceInitializer(ctx, lockspace, pvc); err != nil {
				return fmt.Errorf("create Lockspace initializer: %s", err)
			}
			return nil
		}

		if initializer.Status.Phase == corev1.PodSucceeded {
			if err := r.Client.Delete(ctx, initializer); err != nil {
				return fmt.Errorf("delete Lockspace initializer: %s", err)
			}
			lockspace.Status.Ready = true
		}
	}

	return nil
}

func (r *LockspaceReconciler) getLockspacePVCOrNil(ctx context.Context, lockspace *virtv1alpha1.Lockspace) (*corev1.PersistentVolumeClaim, error) {
	pvcKey := types.NamespacedName{
		Namespace: VirtinkNamespace,
		Name:      lockspace.GeneratePVCName(),
	}
	var pvc corev1.PersistentVolumeClaim
	if err := r.Client.Get(ctx, pvcKey, &pvc); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	if !metav1.IsControlledBy(&pvc, lockspace) {
		return nil, fmt.Errorf("PVC %q is not controlled by Lockspace %q", namespacedName(&pvc), namespacedName(lockspace))
	}

	return &pvc, nil
}

func (r *LockspaceReconciler) createLockspacePVC(ctx context.Context, lockspace *virtv1alpha1.Lockspace) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: VirtinkNamespace,
			Name:      lockspace.GeneratePVCName(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: func() *string { str := lockspace.Spec.StorageClassName; return &str }(),
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
		},
	}

	if pvc.Spec.Resources.Requests == nil {
		pvc.Spec.Resources.Requests = corev1.ResourceList{}
	}
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse(fmt.Sprintf("%dMi", 3+lockspace.Spec.MaxLocks))

	volumeMode := corev1.PersistentVolumeFilesystem
	if lockspace.Spec.VolumeMode != nil {
		volumeMode = *lockspace.Spec.VolumeMode
	}
	pvc.Spec.VolumeMode = &volumeMode

	if err := controllerutil.SetControllerReference(lockspace, pvc, r.Scheme); err != nil {
		return fmt.Errorf("set PVC controller reference: %s", err)
	}
	return r.Client.Create(ctx, pvc)
}

func (r *LockspaceReconciler) getLockspaceInitializerOrNil(ctx context.Context, lockspace *virtv1alpha1.Lockspace) (*corev1.Pod, error) {
	podKey := types.NamespacedName{
		Namespace: VirtinkNamespace,
		Name:      lockspace.GenerateInitializerName(),
	}
	var pod corev1.Pod
	if err := r.Client.Get(ctx, podKey, &pod); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	if !metav1.IsControlledBy(&pod, lockspace) {
		return nil, fmt.Errorf("pod %q is not controlled by Lockspace %q", namespacedName(&pod), namespacedName(lockspace))
	}
	return &pod, nil
}

func (r *LockspaceReconciler) createLockspaceInitializer(ctx context.Context, lockspace *virtv1alpha1.Lockspace, pvc *corev1.PersistentVolumeClaim) error {
	initializerPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lockspace.GenerateInitializerName(),
			Namespace: VirtinkNamespace,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyOnFailure,
			Containers: []corev1.Container{{
				Name:  "lockspace-initializer",
				Image: r.InitializerImageName,
				Env: []corev1.EnvVar{{
					Name:  "LOCKSPACE_NAME",
					Value: lockspace.Name,
				}, {
					Name:  "IO_TIMEOUT_SECONDS",
					Value: fmt.Sprintf("%d", lockspace.Spec.IOTimeoutSeconds),
				}},
			}},
		},
	}
	initializerPod.Spec.Volumes = append(initializerPod.Spec.Volumes, corev1.Volume{
		Name: "lockspace-volume",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvc.Name,
			},
		},
	})
	container := &initializerPod.Spec.Containers[0]
	switch *pvc.Spec.VolumeMode {
	case corev1.PersistentVolumeFilesystem:
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "lockspace-volume",
			MountPath: filepath.Join("/var/lib/sanlock/", lockspace.Name),
		})
	case corev1.PersistentVolumeBlock:
		container.VolumeDevices = append(container.VolumeDevices, corev1.VolumeDevice{
			Name:       "lockspace-volume",
			DevicePath: filepath.Join("/var/lib/sanlock", lockspace.Name, "leases"),
		})
	}

	if err := controllerutil.SetControllerReference(lockspace, initializerPod, r.Scheme); err != nil {
		return fmt.Errorf("set initializer controller reference: %s", err)
	}

	return r.Client.Create(ctx, initializerPod)
}

func (r *LockspaceReconciler) reconcileReadyLockspace(ctx context.Context, lockspace *virtv1alpha1.Lockspace) error {
	pvc, err := r.getLockspacePVCOrNil(ctx, lockspace)
	if err != nil {
		return fmt.Errorf("get Lockspace PVC: %s", err)
	}

	attacher, err := r.getLockspaceAttacherOrNil(ctx, lockspace)
	if err != nil {
		return fmt.Errorf("get Lockspace attacher: %s", err)
	}
	if attacher == nil {
		if err := r.createLockspaceAttacher(ctx, lockspace, pvc); err != nil {
			return fmt.Errorf("create Lockspace attacher: %s", err)
		}
	}

	detector, err := r.getLockspaceDetectorOrNil(ctx, lockspace)
	if err != nil {
		return fmt.Errorf("get Lockspace detector: %s", err)
	}
	if detector == nil {
		if err := r.createLockspaceDetector(ctx, lockspace); err != nil {
			return fmt.Errorf("create Lockspace detector: %s", err)
		}
	}
	return nil
}

func (r *LockspaceReconciler) createLockspaceAttacher(ctx context.Context, lockspace *virtv1alpha1.Lockspace, pvc *corev1.PersistentVolumeClaim) error {
	matchLabels := make(map[string]string)
	matchLabels["name"] = lockspace.GenerateAttacherName()

	attacher := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  VirtinkNamespace,
			Name:       lockspace.GenerateAttacherName(),
			Finalizers: []string{LockspaceAttacherProtectionFinalizer},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: matchLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "lockspace-attacher",
					Containers: []corev1.Container{{
						Name:  "lockspace-attacher",
						Image: r.AttacherImageName,
						SecurityContext: &corev1.SecurityContext{
							Privileged: func() *bool { b := true; return &b }(),
						},
						Env: []corev1.EnvVar{{
							Name:  "LOCKSPACE_NAME",
							Value: lockspace.Name,
						}, {
							Name: "NODE_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "spec.nodeName",
								},
							},
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "sanlock-run-dir",
							MountPath: "/var/run/sanlock",
						}, {
							Name:             "sanlock-lib-dir",
							MountPath:        "/var/lib/sanlock",
							MountPropagation: func() *corev1.MountPropagationMode { p := corev1.MountPropagationBidirectional; return &p }(),
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "sanlock-run-dir",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/run/sanlock",
							},
						},
					}, {
						Name: "sanlock-lib-dir",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/lib/sanlock",
							},
						},
					}},
				},
			},
		},
	}

	pod := &attacher.Spec.Template
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: "lockspace-volume",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvc.Name,
			},
		},
	})
	container := &pod.Spec.Containers[0]
	switch *pvc.Spec.VolumeMode {
	case corev1.PersistentVolumeFilesystem:
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "lockspace-volume",
			MountPath: filepath.Join("/var/lib/sanlock", lockspace.Name),
		})
	case corev1.PersistentVolumeBlock:
		container.VolumeDevices = append(container.VolumeDevices, corev1.VolumeDevice{
			Name:       "lockspace-volume",
			DevicePath: filepath.Join("/var/lib/sanlock", lockspace.Name, "leases"),
		})
	}

	if err := controllerutil.SetControllerReference(lockspace, attacher, r.Scheme); err != nil {
		return fmt.Errorf("set attacher controller reference: %s", err)
	}

	return r.Client.Create(ctx, attacher)
}

func (r *LockspaceReconciler) createLockspaceDetector(ctx context.Context, lockspace *virtv1alpha1.Lockspace) error {
	matchLabels := make(map[string]string)
	matchLabels["name"] = lockspace.GenerateDetectorName()

	detector := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  VirtinkNamespace,
			Name:       lockspace.GenerateDetectorName(),
			Finalizers: []string{LockspaceDetectorProtectionFinalizer},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: matchLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "lockspace-detector",
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{{
									MatchExpressions: []corev1.NodeSelectorRequirement{{
										Key:      fmt.Sprintf("virtink.smartx.com/dead-lockspace-%s", lockspace.Name),
										Operator: corev1.NodeSelectorOpDoesNotExist,
									}},
								}},
							},
						},
					},
					Containers: []corev1.Container{{
						Name:  "lockspace-detector",
						Image: r.DetectorImageName,
						Env: []corev1.EnvVar{{
							Name:  "LOCKSPACE_NAME",
							Value: lockspace.Name,
						}, {
							Name:  "IO_TIMEOUT",
							Value: strconv.Itoa(int(lockspace.Spec.IOTimeoutSeconds)),
						}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "sanlock-run-dir",
							MountPath: "/var/run/sanlock",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "sanlock-run-dir",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/run/sanlock",
							},
						},
					}},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(lockspace, detector, r.Scheme); err != nil {
		return fmt.Errorf("set detector controller reference: %s", err)
	}

	return r.Client.Create(ctx, detector)
}

func (r *LockspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &virtv1alpha1.Lock{}, ".spec.lockspaceName", func(obj client.Object) []string {
		lock := obj.(*virtv1alpha1.Lock)
		return []string{lock.Spec.LockspaceName}
	}); err != nil {
		return fmt.Errorf("index Lock by lockspaceName: %s", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&virtv1alpha1.Lockspace{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

func namespacedName(obj metav1.ObjectMetaAccessor) types.NamespacedName {
	meta := obj.GetObjectMeta()
	return types.NamespacedName{
		Namespace: meta.GetNamespace(),
		Name:      meta.GetName(),
	}
}
