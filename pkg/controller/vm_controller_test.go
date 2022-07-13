package controller

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

var _ = Describe("VM controller", func() {
	Context("for a Pending VM", func() {
		var vmKey types.NamespacedName

		BeforeEach(func() {
			By("creating a new VM")
			vmKey = types.NamespacedName{
				Name:      uuid.New().String(),
				Namespace: "default",
			}
			vm := virtv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vmKey.Name,
					Namespace: vmKey.Namespace,
				},
				Spec: virtv1alpha1.VirtualMachineSpec{
					RunPolicy: virtv1alpha1.RunPolicyManual,
				},
			}
			Expect(k8sClient.Create(ctx, &vm)).To(Succeed())

			By("updating VM phase to Pending")
			Eventually(func() bool {
				var vm virtv1alpha1.VirtualMachine
				Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
				vm.Status.Phase = virtv1alpha1.VirtualMachinePending
				return k8sClient.Status().Update(ctx, &vm) == nil
			}).Should(BeTrue())
		})

		It("should generate VM pod name", func() {
			Eventually(func() bool {
				var vm virtv1alpha1.VirtualMachine
				Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
				return vm.Status.VMPodName != ""
			}).Should(BeTrue())
		})

		It("should update VM phase to Scheduling", func() {
			Eventually(func() bool {
				var vm virtv1alpha1.VirtualMachine
				Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
				return vm.Status.Phase == virtv1alpha1.VirtualMachineScheduling
			}).Should(BeTrue())
		})

		It("should create a VM pod", func() {
			var vm virtv1alpha1.VirtualMachine
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
				return vm.Status.VMPodName != ""
			}).Should(BeTrue())

			vmPodKey := types.NamespacedName{Name: vm.Status.VMPodName, Namespace: vmKey.Namespace}
			Eventually(func() bool {
				var vmPod corev1.Pod
				return k8sClient.Get(ctx, vmPodKey, &vmPod) == nil
			}).Should(BeTrue())
		})
	})

	Context("for a Scheduling VM", func() {
		var vmKey types.NamespacedName
		var vmPodKey types.NamespacedName
		var nodeName string

		BeforeEach(func() {
			By("creating a new VM")
			vmKey = types.NamespacedName{
				Name:      uuid.New().String(),
				Namespace: "default",
			}
			vm := virtv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vmKey.Name,
					Namespace: vmKey.Namespace,
				},
				Spec: virtv1alpha1.VirtualMachineSpec{
					RunPolicy: virtv1alpha1.RunPolicyManual,
				},
			}
			Expect(k8sClient.Create(ctx, &vm)).To(Succeed())

			By("updating VM phase to Scheduling")
			vmPodKey = types.NamespacedName{
				Name:      uuid.New().String(),
				Namespace: vmKey.Namespace,
			}
			Eventually(func() bool {
				var vm virtv1alpha1.VirtualMachine
				Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
				vm.Status.Phase = virtv1alpha1.VirtualMachineScheduling
				vm.Status.VMPodName = vmPodKey.Name
				return k8sClient.Status().Update(ctx, &vm) == nil
			}).Should(BeTrue())

			By("checking the VM pod has been created")
			Eventually(func() bool {
				var vmPod corev1.Pod
				return k8sClient.Get(ctx, vmPodKey, &vmPod) == nil
			}).Should(BeTrue())

			By("binding the VM pod")
			nodeName = uuid.New().String()
			Expect(bindPod(ctx, vmPodKey, nodeName)).To(Succeed())
		})

		Context("when the VM pod is Running", func() {
			BeforeEach(func() {
				By("updating VM pod phase to Running")
				Eventually(func() bool {
					var vmPod corev1.Pod
					Expect(k8sClient.Get(ctx, vmPodKey, &vmPod)).To(Succeed())
					vmPod.Status.Phase = corev1.PodRunning
					return k8sClient.Status().Update(ctx, &vmPod) == nil
				}).Should(BeTrue())
			})

			It("should update VM phase to Scheduled", func() {
				Eventually(func() bool {
					var vm virtv1alpha1.VirtualMachine
					Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
					return vm.Status.Phase == virtv1alpha1.VirtualMachineScheduled
				}).Should(BeTrue())
			})

			It("should set VM nodeName", func() {
				Eventually(func() bool {
					var vm virtv1alpha1.VirtualMachine
					Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
					return vm.Status.NodeName == nodeName
				}).Should(BeTrue())
			})
		})

		Context("when the VM pod is Failed", func() {
			BeforeEach(func() {
				By("updating VM pod phase to Failed")
				Eventually(func() bool {
					var vmPod corev1.Pod
					Expect(k8sClient.Get(ctx, vmPodKey, &vmPod)).To(Succeed())
					vmPod.Status.Phase = corev1.PodFailed
					return k8sClient.Status().Update(ctx, &vmPod) == nil
				}).Should(BeTrue())
			})

			It("should update VM phase to Failed", func() {
				Eventually(func() bool {
					var vm virtv1alpha1.VirtualMachine
					Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
					return vm.Status.Phase == virtv1alpha1.VirtualMachineFailed
				}).Should(BeTrue())
			})
		})

		Context("when the VM pod is Succeeded", func() {
			BeforeEach(func() {
				By("updating VM pod phase to Succeeded")
				Eventually(func() bool {
					var vmPod corev1.Pod
					Expect(k8sClient.Get(ctx, vmPodKey, &vmPod)).To(Succeed())
					vmPod.Status.Phase = corev1.PodSucceeded
					return k8sClient.Status().Update(ctx, &vmPod) == nil
				}).Should(BeTrue())
			})

			It("should update VM phase to Succeeded", func() {
				Eventually(func() bool {
					var vm virtv1alpha1.VirtualMachine
					Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
					return vm.Status.Phase == virtv1alpha1.VirtualMachineSucceeded
				}).Should(BeTrue())
			})
		})
	})

	Context("for a Running VM", func() {
		var vmKey types.NamespacedName
		var vmPodKey types.NamespacedName

		BeforeEach(func() {
			By("creating a new VM")
			vmKey = types.NamespacedName{
				Name:      uuid.New().String(),
				Namespace: "default",
			}
			vm := virtv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vmKey.Name,
					Namespace: vmKey.Namespace,
				},
				Spec: virtv1alpha1.VirtualMachineSpec{
					RunPolicy: virtv1alpha1.RunPolicyManual,
				},
			}
			Expect(k8sClient.Create(ctx, &vm)).To(Succeed())

			By("updating VM phase to Scheduling")
			vmPodKey = types.NamespacedName{
				Name:      uuid.New().String(),
				Namespace: vmKey.Namespace,
			}
			Eventually(func() bool {
				var vm virtv1alpha1.VirtualMachine
				Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
				vm.Status.Phase = virtv1alpha1.VirtualMachineScheduling
				vm.Status.VMPodName = vmPodKey.Name
				return k8sClient.Status().Update(ctx, &vm) == nil
			}).Should(BeTrue())

			By("checking the VM pod has been created")
			Eventually(func() bool {
				var vmPod corev1.Pod
				return k8sClient.Get(ctx, vmPodKey, &vmPod) == nil
			}).Should(BeTrue())

			By("binding the VM pod")
			nodeName := uuid.New().String()
			Expect(bindPod(ctx, vmPodKey, nodeName)).To(Succeed())

			By("updating VM pod phase to Running")
			Eventually(func() bool {
				var vmPod corev1.Pod
				Expect(k8sClient.Get(ctx, vmPodKey, &vmPod)).To(Succeed())
				vmPod.Status.Phase = corev1.PodRunning
				return k8sClient.Status().Update(ctx, &vmPod) == nil
			}).Should(BeTrue())

			By("updating VM phase to Running")
			Eventually(func() bool {
				var vm virtv1alpha1.VirtualMachine
				Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
				vm.Status.Phase = virtv1alpha1.VirtualMachineRunning
				vm.Status.VMPodName = vmPodKey.Name
				vm.Status.NodeName = nodeName
				return k8sClient.Status().Update(ctx, &vm) == nil
			}).Should(BeTrue())
		})

		Context("when the VM pod is Failed", func() {
			BeforeEach(func() {
				By("updating VM pod phase to Failed")
				Eventually(func() bool {
					var vmPod corev1.Pod
					Expect(k8sClient.Get(ctx, vmPodKey, &vmPod)).To(Succeed())
					vmPod.Status.Phase = corev1.PodFailed
					return k8sClient.Status().Update(ctx, &vmPod) == nil
				}).Should(BeTrue())
			})

			It("should update VM phase to Failed", func() {
				Eventually(func() bool {
					var vm virtv1alpha1.VirtualMachine
					Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
					return vm.Status.Phase == virtv1alpha1.VirtualMachineFailed
				}).Should(BeTrue())
			})
		})

		Context("when the VM pod is Succeeded", func() {
			BeforeEach(func() {
				By("updating VM pod phase to Succeeded")
				Eventually(func() bool {
					var vmPod corev1.Pod
					Expect(k8sClient.Get(ctx, vmPodKey, &vmPod)).To(Succeed())
					vmPod.Status.Phase = corev1.PodSucceeded
					return k8sClient.Status().Update(ctx, &vmPod) == nil
				}).Should(BeTrue())
			})

			It("should update VM phase to Succeeded", func() {
				Eventually(func() bool {
					var vm virtv1alpha1.VirtualMachine
					Expect(k8sClient.Get(ctx, vmKey, &vm)).To(Succeed())
					return vm.Status.Phase == virtv1alpha1.VirtualMachineSucceeded
				}).Should(BeTrue())
			})
		})
	})
})

func bindPod(ctx context.Context, podKey types.NamespacedName, nodeName string) error {
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	binding := &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podKey.Name,
			Namespace: podKey.Namespace,
		},
		Target: corev1.ObjectReference{
			Kind: "Node",
			Name: nodeName,
		},
	}

	return kubeClient.CoreV1().RESTClient().
		Post().
		Namespace(podKey.Namespace).
		Resource("pods").
		Name(podKey.Name).
		SubResource("binding").
		Body(binding).
		Do(ctx).
		Error()
}
