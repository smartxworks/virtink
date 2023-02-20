package controller

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"reflect"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

// +kubebuilder:webhook:path=/mutate-v1alpha1-virtualmachine,mutating=true,failurePolicy=fail,sideEffects=None,groups=virt.virtink.smartx.com,resources=virtualmachines,verbs=create;update,versions=v1alpha1,name=mutate.virtualmachine.v1alpha1.virt.virtink.smartx.com,admissionReviewVersions={v1,v1beta1}

var memoryOverhead = "256Mi"

type VMMutator struct {
	decoder *admission.Decoder
}

var _ admission.DecoderInjector = &VMMutator{}
var _ admission.Handler = &VMMutator{}

func (h *VMMutator) InjectDecoder(decode *admission.Decoder) error {
	h.decoder = decode
	return nil
}

func (h *VMMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	var vm virtv1alpha1.VirtualMachine
	if err := h.decoder.Decode(req, &vm); err != nil {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("unmarshal VM: %s", err))
	}

	var err error
	switch req.Operation {
	case admissionv1.Create:
		err = MutateVM(ctx, &vm, nil)
	case admissionv1.Update:
		var oldVM virtv1alpha1.VirtualMachine
		if err := h.decoder.DecodeRaw(req.OldObject, &oldVM); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("unmarshal old VM: %s", err))
		}
		err = MutateVM(ctx, &vm, &oldVM)
	default:
		return admission.Allowed("")
	}

	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	vmJSON, err := json.Marshal(vm)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, fmt.Errorf("marshal VM: %s", err))
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, vmJSON)
}

func MutateVM(ctx context.Context, vm *virtv1alpha1.VirtualMachine, oldVM *virtv1alpha1.VirtualMachine) error {
	if vm.Spec.RunPolicy == "" {
		vm.Spec.RunPolicy = virtv1alpha1.RunPolicyOnce
	}

	if vm.Spec.Instance.CPU.Sockets == 0 {
		vm.Spec.Instance.CPU.Sockets = 1
	}
	if vm.Spec.Instance.CPU.CoresPerSocket == 0 {
		vm.Spec.Instance.CPU.CoresPerSocket = 1
	}

	if vm.Spec.Instance.Memory.Size.IsZero() {
		if !vm.Spec.Resources.Requests.Memory().IsZero() {
			vm.Spec.Instance.Memory.Size = vm.Spec.Resources.Requests.Memory().DeepCopy()
		} else {
			vm.Spec.Instance.Memory.Size = resource.MustParse("1Gi")
		}
	}

	if vm.Spec.Instance.CPU.DedicatedCPUPlacement {
		memSize := resource.MustParse(memoryOverhead)
		if !vm.Spec.Instance.Memory.Size.IsZero() {
			if vm.Spec.Instance.Memory.Hugepages == nil {
				memSize.Add(vm.Spec.Instance.Memory.Size)
			}
		}
		rsList := map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    *resource.NewQuantity(int64(vm.Spec.Instance.CPU.CoresPerSocket*vm.Spec.Instance.CPU.Sockets), resource.DecimalSI),
			corev1.ResourceMemory: memSize,
		}

		if vm.Spec.Resources.Requests == nil {
			vm.Spec.Resources.Requests = rsList
		} else {
			if vm.Spec.Resources.Requests.Cpu().IsZero() {
				vm.Spec.Resources.Requests[corev1.ResourceCPU] = rsList[corev1.ResourceCPU]
			}
			if vm.Spec.Resources.Requests.Memory().IsZero() {
				vm.Spec.Resources.Requests[corev1.ResourceMemory] = rsList[corev1.ResourceMemory]
			}
		}

		if vm.Spec.Resources.Limits == nil {
			vm.Spec.Resources.Limits = rsList
		} else {
			if vm.Spec.Resources.Limits.Cpu().IsZero() {
				vm.Spec.Resources.Limits[corev1.ResourceCPU] = rsList[corev1.ResourceCPU]
			}
			if vm.Spec.Resources.Limits.Memory().IsZero() {
				vm.Spec.Resources.Limits[corev1.ResourceMemory] = rsList[corev1.ResourceMemory]
			}
		}
	}

	if vm.Spec.Instance.Memory.Hugepages != nil {
		hugepagesSize := fmt.Sprintf("hugepages-%s", vm.Spec.Instance.Memory.Hugepages.PageSize)

		if vm.Spec.Resources.Limits == nil {
			vm.Spec.Resources.Limits = corev1.ResourceList{}
		}
		hugepagesLimit, exist := vm.Spec.Resources.Limits[corev1.ResourceName(hugepagesSize)]
		if !exist {
			hugepagesLimit = vm.Spec.Instance.Memory.Size.DeepCopy()
			vm.Spec.Resources.Limits[corev1.ResourceName(hugepagesSize)] = hugepagesLimit
		}
		if vm.Spec.Resources.Requests == nil {
			vm.Spec.Resources.Requests = corev1.ResourceList{}
		}
		if _, exist := vm.Spec.Resources.Requests[corev1.ResourceName(hugepagesSize)]; !exist {
			vm.Spec.Resources.Requests[corev1.ResourceName(hugepagesSize)] = hugepagesLimit.DeepCopy()
		}

		if vm.Spec.Resources.Limits.Cpu().IsZero() && vm.Spec.Resources.Limits.Memory().IsZero() && vm.Spec.Resources.Requests.Cpu().IsZero() && vm.Spec.Resources.Requests.Memory().IsZero() {
			vm.Spec.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(memoryOverhead)
		}
	}

	for i := range vm.Spec.Instance.Interfaces {
		if vm.Spec.Instance.Interfaces[i].MAC == "" {
			var macStr string
			if oldVM != nil {
				for j := range oldVM.Spec.Instance.Interfaces {
					if oldVM.Spec.Instance.Interfaces[j].Name == vm.Spec.Instance.Interfaces[i].Name {
						macStr = oldVM.Spec.Instance.Interfaces[j].MAC
						break
					}
				}
			}
			if macStr == "" {
				mac, err := generateMAC()
				if err != nil {
					return fmt.Errorf("generate MAC: %s", err)
				}
				macStr = mac.String()
			}
			vm.Spec.Instance.Interfaces[i].MAC = macStr
		}

		if vm.Spec.Instance.Interfaces[i].Bridge == nil && vm.Spec.Instance.Interfaces[i].Masquerade == nil && vm.Spec.Instance.Interfaces[i].SRIOV == nil && vm.Spec.Instance.Interfaces[i].VhostUser == nil {
			vm.Spec.Instance.Interfaces[i].InterfaceBindingMethod = virtv1alpha1.InterfaceBindingMethod{
				Bridge: &virtv1alpha1.InterfaceBridge{},
			}
		}

		if vm.Spec.Instance.Interfaces[i].Masquerade != nil {
			if vm.Spec.Instance.Interfaces[i].Masquerade.IPv4CIDR == "" {
				vm.Spec.Instance.Interfaces[i].Masquerade.IPv4CIDR = "10.0.2.0/30"
			}
			if vm.Spec.Instance.Interfaces[i].Masquerade.IPv6CIDR == "" {
				vm.Spec.Instance.Interfaces[i].Masquerade.IPv6CIDR = "fd10:0:2::/120"
			}
		}
	}
	return nil
}

// +kubebuilder:webhook:path=/validate-v1alpha1-virtualmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=virt.virtink.smartx.com,resources=virtualmachines,verbs=create;update,versions=v1alpha1,name=validate.virtualmachine.v1alpha1.virt.virtink.smartx.com,admissionReviewVersions={v1,v1beta1}

type VMValidator struct {
	decoder *admission.Decoder
}

var _ admission.DecoderInjector = &VMValidator{}
var _ admission.Handler = &VMValidator{}

func (h *VMValidator) InjectDecoder(decoder *admission.Decoder) error {
	h.decoder = decoder
	return nil
}

func (h *VMValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	var vm virtv1alpha1.VirtualMachine
	if err := h.decoder.Decode(req, &vm); err != nil {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("unmarshal VM: %s", err))
	}

	var errs field.ErrorList
	switch req.Operation {
	case admissionv1.Create:
		errs = ValidateVM(ctx, &vm, nil)
	case admissionv1.Update:
		var oldVM virtv1alpha1.VirtualMachine
		if err := h.decoder.DecodeRaw(req.OldObject, &oldVM); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("unmarshal old VM: %s", err))
		}
		errs = ValidateVM(ctx, &vm, &oldVM)
	default:
		return admission.Allowed("")
	}

	if len(errs) > 0 {
		return webhook.Denied(errs.ToAggregate().Error())
	}
	return admission.Allowed("")
}

func ValidateVM(ctx context.Context, vm *virtv1alpha1.VirtualMachine, oldVM *virtv1alpha1.VirtualMachine) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, ValidateVMSpec(ctx, &vm.Spec, field.NewPath("spec"))...)
	if oldVM != nil {
		errs = append(errs, ValidateVMUpdate(ctx, vm, oldVM)...)
	}
	return errs
}

func ValidateVMSpec(ctx context.Context, spec *virtv1alpha1.VirtualMachineSpec, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if spec == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if spec.Instance.CPU.DedicatedCPUPlacement {
		cpuRequestField := fieldPath.Child("resources.requests").Child(string(corev1.ResourceCPU))
		if spec.Resources.Requests.Cpu().IsZero() {
			errs = append(errs, field.Required(cpuRequestField, ""))
		} else if spec.Resources.Requests.Cpu().Value() != int64(spec.Instance.CPU.Sockets*spec.Instance.CPU.CoresPerSocket) {
			errs = append(errs, field.Invalid(cpuRequestField, spec.Resources.Requests.Cpu().String(), "must equal to number of vCPUs"))
		}

		cpuLimitField := fieldPath.Child("resources.limits").Child(string(corev1.ResourceCPU))
		if spec.Resources.Limits.Cpu().IsZero() {
			errs = append(errs, field.Required(cpuLimitField, ""))
		} else if !spec.Resources.Limits.Cpu().Equal(*spec.Resources.Requests.Cpu()) {
			errs = append(errs, field.Invalid(cpuLimitField, spec.Resources.Limits.Cpu().String(), "must equal to CPU request"))
		}

		memoryRequestField := fieldPath.Child("resources.requests").Child(string(corev1.ResourceMemory))
		memRequired := resource.MustParse(memoryOverhead)
		if spec.Instance.Memory.Hugepages == nil {
			memRequired.Add(spec.Instance.Memory.Size)
		}
		if spec.Resources.Requests.Memory().IsZero() {
			errs = append(errs, field.Required(memoryRequestField, ""))
		} else if spec.Resources.Requests.Memory().Cmp(memRequired) < 0 {
			errs = append(errs, field.Invalid(memoryRequestField, spec.Resources.Requests.Memory().String(), fmt.Sprintf("must not be less than %s", memRequired.String())))
		}

		memoryLimitField := fieldPath.Child("resources.limits").Child(string(corev1.ResourceMemory))
		if spec.Resources.Limits.Memory().IsZero() {
			errs = append(errs, field.Required(memoryLimitField, ""))
		} else if !spec.Resources.Limits.Memory().Equal(*spec.Resources.Requests.Memory()) {
			errs = append(errs, field.Invalid(memoryLimitField, spec.Resources.Limits.Memory().String(), "must equal to memory request"))
		}
	}

	if spec.Instance.Memory.Hugepages != nil {
		resourcesField := fieldPath.Child("resources")
		if spec.Resources.Limits.Cpu().IsZero() && spec.Resources.Limits.Memory().IsZero() && spec.Resources.Requests.Cpu().IsZero() && spec.Resources.Requests.Memory().IsZero() {
			errs = append(errs, field.Forbidden(resourcesField, "hugepages require cpu or memory"))
		}

		hugepagesSize := fmt.Sprintf("hugepages-%s", spec.Instance.Memory.Hugepages.PageSize)
		hugepagesRequestField := resourcesField.Child("requests").Child(hugepagesSize)
		hugepagesRequest, exist := spec.Resources.Requests[corev1.ResourceName(hugepagesSize)]
		if !exist {
			errs = append(errs, field.Required(hugepagesRequestField, ""))
		} else if !hugepagesRequest.Equal(spec.Instance.Memory.Size) {
			errs = append(errs, field.Invalid(hugepagesRequestField, hugepagesRequest.String(), "must equal to instance memory size"))
		}

		hugepagesLimitField := resourcesField.Child("limits").Child(hugepagesSize)
		hugepagesLimit, exist := spec.Resources.Limits[corev1.ResourceName(hugepagesSize)]
		if !exist {
			errs = append(errs, field.Required(hugepagesLimitField, ""))
		} else if !hugepagesLimit.Equal(hugepagesRequest) {
			errs = append(errs, field.Invalid(hugepagesLimitField, hugepagesLimit.String(), "must equal to hugepages request"))
		}
	}

	errs = append(errs, ValidateInstance(ctx, &spec.Instance, fieldPath.Child("instance"))...)

	volumeNames := map[string]struct{}{}
	for i, volume := range spec.Volumes {
		fieldPath := fieldPath.Child("volumes").Index(i)
		if _, ok := volumeNames[volume.Name]; ok {
			errs = append(errs, field.Duplicate(fieldPath.Child("name"), volume.Name))
		}
		volumeNames[volume.Name] = struct{}{}
		errs = append(errs, ValidateVolume(ctx, &volume, fieldPath)...)
	}

	networkNames := map[string]struct{}{}
	for i, network := range spec.Networks {
		fieldPath := fieldPath.Child("networks").Index(i)
		if _, ok := networkNames[network.Name]; ok {
			errs = append(errs, field.Duplicate(fieldPath.Child("name"), network.Name))
		}
		networkNames[network.Name] = struct{}{}
		errs = append(errs, ValidateNetwork(ctx, &network, fieldPath)...)
	}

	return errs
}

func ValidateInstance(ctx context.Context, instance *virtv1alpha1.Instance, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if instance == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	errs = append(errs, ValidateCPU(ctx, &instance.CPU, fieldPath.Child("cpu"))...)
	errs = append(errs, ValidateMemory(ctx, &instance.Memory, fieldPath.Child("memory"))...)

	if instance.Kernel != nil {
		errs = append(errs, ValidateKernel(ctx, instance.Kernel, fieldPath.Child("kernel"))...)
	}

	diskNames := map[string]struct{}{}
	for i, disk := range instance.Disks {
		fieldPath := fieldPath.Child("disks").Index(i)
		if _, ok := diskNames[disk.Name]; ok {
			errs = append(errs, field.Duplicate(fieldPath.Child("name"), disk.Name))
		}
		diskNames[disk.Name] = struct{}{}
		errs = append(errs, ValidateDisk(ctx, &disk, fieldPath)...)
	}

	for i, fs := range instance.FileSystems {
		fieldPath := fieldPath.Child("fileSystems").Index(i)
		if _, ok := diskNames[fs.Name]; ok {
			errs = append(errs, field.Duplicate(fieldPath.Child("name"), fs.Name))
		}
		diskNames[fs.Name] = struct{}{}
		errs = append(errs, ValidateFileSystem(ctx, &fs, fieldPath)...)
	}

	ifaceNames := map[string]struct{}{}
	for i, iface := range instance.Interfaces {
		fieldPath := fieldPath.Child("interfaces").Index(i)
		if _, ok := ifaceNames[iface.Name]; ok {
			errs = append(errs, field.Duplicate(fieldPath.Child("name"), iface.Name))
		}
		ifaceNames[iface.Name] = struct{}{}
		if iface.InterfaceBindingMethod.VhostUser != nil {
			if !instance.CPU.DedicatedCPUPlacement || instance.Memory.Hugepages == nil {
				errs = append(errs, field.Forbidden(fieldPath.Child("vhostUser"), "may not use vhost-user interface without dedicated CPU placement and hugepages"))
			}
		}
		errs = append(errs, ValidateInterface(ctx, &iface, fieldPath)...)
	}

	return errs
}

func ValidateCPU(ctx context.Context, cpu *virtv1alpha1.CPU, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if cpu == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if cpu.Sockets == 0 {
		errs = append(errs, field.Required(fieldPath.Child("sockets"), ""))
	}
	if cpu.CoresPerSocket <= 0 {
		errs = append(errs, field.Required(fieldPath.Child("coresPerSocket"), ""))
	}
	return errs
}

func ValidateMemory(ctx context.Context, memory *virtv1alpha1.Memory, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if memory == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	memSize := memory.Size.Value()
	if memSize <= 0 {
		errs = append(errs, field.Invalid(fieldPath.Child("size"), memSize, "must be greater than 0"))
	}
	if memory.Hugepages != nil {
		q := resource.MustParse(memory.Hugepages.PageSize)
		hugepagesSize := q.Value()
		if memSize%hugepagesSize != 0 {
			errs = append(errs, field.Invalid(fieldPath.Child("size"), memSize, fmt.Sprintf("%d is not positive integer multiple of %s", memSize, memory.Hugepages.PageSize)))
		}
	}

	return errs
}

func ValidateKernel(ctx context.Context, kernel *virtv1alpha1.Kernel, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if kernel == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if kernel.Image == "" {
		errs = append(errs, field.Required(fieldPath.Child("image"), ""))
	}
	if kernel.Cmdline == "" {
		errs = append(errs, field.Required(fieldPath.Child("cmdline"), ""))
	}
	return errs
}

func ValidateDisk(ctx context.Context, disk *virtv1alpha1.Disk, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if disk == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if disk.Name == "" {
		errs = append(errs, field.Required(fieldPath.Child("name"), ""))
	}
	return errs
}

func ValidateFileSystem(ctx context.Context, fs *virtv1alpha1.FileSystem, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if fs == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if fs.Name == "" {
		errs = append(errs, field.Required(fieldPath.Child("name"), ""))
	}
	return errs
}

func ValidateInterface(ctx context.Context, iface *virtv1alpha1.Interface, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if iface == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if iface.Name == "" {
		errs = append(errs, field.Required(fieldPath.Child("name"), ""))
	}
	errs = append(errs, ValidateMAC(iface.MAC, fieldPath.Child("mac"))...)
	errs = append(errs, ValidateInterfaceBindingMethod(ctx, &iface.InterfaceBindingMethod, fieldPath)...)
	return errs
}

func ValidateMAC(mac string, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if mac == "" {
		errs = append(errs, field.Required(fieldPath, ""))
	}
	if _, err := net.ParseMAC(mac); err != nil {
		errs = append(errs, field.Invalid(fieldPath, mac, err.Error()))
	}
	return errs
}

func ValidateInterfaceBindingMethod(ctx context.Context, bindingMethod *virtv1alpha1.InterfaceBindingMethod, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if bindingMethod == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	cnt := 0
	if bindingMethod.Bridge != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("bridge"), "may not specify more than 1 binding method"))
		}
	}
	if bindingMethod.Masquerade != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("masquerade"), "may not specify more than 1 binding method"))
		} else {
			errs = append(errs, ValidateCIDR(bindingMethod.Masquerade.IPv4CIDR, 4, fieldPath.Child("masquerade").Child("ipv4CIDR"))...)
			errs = append(errs, ValidateCIDR(bindingMethod.Masquerade.IPv6CIDR, 4, fieldPath.Child("masquerade").Child("ipv6CIDR"))...)
		}
	}
	if bindingMethod.SRIOV != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("sriov"), "may not specify more than 1 binding method"))
		}
	}
	if bindingMethod.VhostUser != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("vhostUser"), "may not specify more than 1 binding method"))
		}
	}

	if cnt == 0 {
		errs = append(errs, field.Required(fieldPath, "at least 1 binding method is required"))
	}
	return errs
}

func ValidateCIDR(cidr string, capacity int, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if cidr == "" {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		errs = append(errs, field.Invalid(fieldPath, cidr, err.Error()))
	}
	if subnet == nil {
		errs = append(errs, field.Invalid(fieldPath, cidr, "must specify subnet"))
	} else {
		if ones, bits := subnet.Mask.Size(); (1 << (bits - ones)) < capacity {
			errs = append(errs, field.Invalid(fieldPath, cidr, fmt.Sprintf("must contain at least %d IPs", capacity)))
		}
	}
	return errs
}

func ValidateVolume(ctx context.Context, volume *virtv1alpha1.Volume, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if volume == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if volume.Name == "" {
		errs = append(errs, field.Required(fieldPath.Child("name"), ""))
	}
	errs = append(errs, ValidateVolumeSource(ctx, &volume.VolumeSource, fieldPath)...)
	return errs
}

func ValidateVolumeSource(ctx context.Context, source *virtv1alpha1.VolumeSource, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if source == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	cnt := 0
	if source.ContainerDisk != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("containerDisk"), "may not specify more than 1 volume source"))
		} else {
			errs = append(errs, ValidateContainerDiskVolumeSource(ctx, source.ContainerDisk, fieldPath.Child("containerDisk"))...)
		}
	}
	if source.CloudInit != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("cloudInit"), "may not specify more than 1 volume source"))
		} else {
			errs = append(errs, ValidateCloudInitVolumeSource(ctx, source.CloudInit, fieldPath.Child("cloudInit"))...)
		}
	}
	if source.ContainerRootfs != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("containerRootfs"), "may not specify more than 1 volume source"))
		} else {
			errs = append(errs, ValidateContainerRootfsVolumeSource(ctx, source.ContainerRootfs, fieldPath.Child("containerRootfs"))...)
		}
	}
	if source.PersistentVolumeClaim != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("persistentVolumeClaim"), "may not specify more than 1 volume source"))
		} else {
			errs = append(errs, ValidatePersistentVolumeClaimSource(ctx, source.PersistentVolumeClaim, fieldPath.Child("persistentVolumeClaim"))...)
		}
	}
	if source.DataVolume != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("dataVolume"), "may not specify more than 1 volume source"))
		} else {
			errs = append(errs, ValidateDataVolumeSource(ctx, source.DataVolume, fieldPath.Child("dataVolume"))...)
		}
	}
	if cnt == 0 {
		errs = append(errs, field.Required(fieldPath, "at least 1 volume source is required"))
	}
	return errs
}

func ValidateContainerDiskVolumeSource(ctx context.Context, source *virtv1alpha1.ContainerDiskVolumeSource, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if source == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if source.Image == "" {
		errs = append(errs, field.Required(fieldPath.Child("image"), ""))
	}
	return errs
}

func ValidateCloudInitVolumeSource(ctx context.Context, source *virtv1alpha1.CloudInitVolumeSource, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if source == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	userDataCnt := 0
	if source.UserData != "" {
		userDataCnt++
		if userDataCnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("userData"), "may not specify more than 1 user data"))
		}
	}
	if source.UserDataBase64 != "" {
		userDataCnt++
		if userDataCnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("userDataBase64"), "may not specify more than 1 user data"))
		}
	}
	if source.UserDataSecretName != "" {
		userDataCnt++
		if userDataCnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("userDataSecretName"), "may not specify more than 1 user data"))
		}
	}

	networkDataCnt := 0
	if source.NetworkData != "" {
		networkDataCnt++
		if networkDataCnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("networkData"), "may not specify more than 1 network data"))
		}
	}
	if source.NetworkDataBase64 != "" {
		networkDataCnt++
		if networkDataCnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("networkDataBase64"), "may not specify more than 1 network data"))
		}
	}
	if source.NetworkDataSecretName != "" {
		networkDataCnt++
		if networkDataCnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("networkDataSecretName"), "may not specify more than 1 network data"))
		}
	}
	return errs
}

func ValidateContainerRootfsVolumeSource(ctx context.Context, source *virtv1alpha1.ContainerRootfsVolumeSource, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if source == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if source.Image == "" {
		errs = append(errs, field.Required(fieldPath.Child("image"), ""))
	}
	if source.Size.Value() <= 0 {
		errs = append(errs, field.Invalid(fieldPath.Child("size"), source.Size.Value(), "must be greater than 0"))
	}
	return errs
}

func ValidatePersistentVolumeClaimSource(ctx context.Context, source *virtv1alpha1.PersistentVolumeClaimVolumeSource, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if source == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if source.ClaimName == "" {
		errs = append(errs, field.Required(fieldPath.Child("claimName"), ""))
	}
	return errs
}

func ValidateDataVolumeSource(ctx context.Context, source *virtv1alpha1.DataVolumeVolumeSource, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if source == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if source.VolumeName == "" {
		errs = append(errs, field.Required(fieldPath.Child("volumeName"), ""))
	}
	return errs
}

func ValidateNetwork(ctx context.Context, network *virtv1alpha1.Network, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if network == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if network.Name == "" {
		errs = append(errs, field.Required(fieldPath.Child("name"), ""))
	}
	errs = append(errs, ValidateNetworkSource(ctx, &network.NetworkSource, fieldPath)...)
	return errs
}

func ValidateNetworkSource(ctx context.Context, source *virtv1alpha1.NetworkSource, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if source == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	cnt := 0
	if source.Pod != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("pod"), "may not specify more than 1 network source"))
		} else {
			errs = append(errs, ValidatePodNetworkSource(ctx, source.Pod, fieldPath.Child("pod"))...)
		}
	}
	if source.Multus != nil {
		cnt++
		if cnt > 1 {
			errs = append(errs, field.Forbidden(fieldPath.Child("multus"), "may not specify more than 1 network source"))
		} else {
			errs = append(errs, ValidateMultusNetworkSource(ctx, source.Multus, fieldPath.Child("multus"))...)
		}
	}
	if cnt == 0 {
		errs = append(errs, field.Required(fieldPath, "at least 1 network source is required"))
	}
	return errs
}

func ValidatePodNetworkSource(ctx context.Context, source *virtv1alpha1.PodNetworkSource, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if source == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}
	return errs
}

func ValidateMultusNetworkSource(ctx context.Context, source *virtv1alpha1.MultusNetworkSource, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if source == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	if source.NetworkName == "" {
		errs = append(errs, field.Required(fieldPath.Child("networkName"), ""))
	}
	return errs
}

func ValidateVMUpdate(ctx context.Context, vm *virtv1alpha1.VirtualMachine, oldVM *virtv1alpha1.VirtualMachine) field.ErrorList {
	var errs field.ErrorList
	tmpOldVM := oldVM.DeepCopy()
	tmpOldVM.Spec.RunPolicy = vm.Spec.RunPolicy
	tmpOldVM.Spec.Volumes = vm.Spec.Volumes
	tmpOldVM.Spec.Instance.Disks = vm.Spec.Instance.Disks
	if !reflect.DeepEqual(tmpOldVM.Spec, vm.Spec) {
		errs = append(errs, field.Forbidden(field.NewPath("spec"), "VM spec may not be updated except runPolicy, volumes, instance.disks"))
	}

	for _, oldVolume := range oldVM.Spec.Volumes {
		var newVolume *virtv1alpha1.Volume
		for _, volume := range vm.Spec.Volumes {
			if volume.Name == oldVolume.Name {
				newVolume = &volume
				break
			}
		}
		if newVolume != nil {
			if !reflect.DeepEqual(oldVolume, *newVolume) {
				errs = append(errs, field.Forbidden(field.NewPath("spec").Child("volumes").Key(oldVolume.Name), "VM volume may not be updated"))
			}
		} else {
			if !oldVolume.IsHotpluggable() {
				errs = append(errs, field.Forbidden(field.NewPath("spec").Child("volumes").Key(oldVolume.Name), "Unhotpluggable volume may not be hotunplugged"))
			}
		}
	}

	for _, newVolume := range vm.Spec.Volumes {
		var oldVolume *virtv1alpha1.Volume
		for _, volume := range oldVM.Spec.Volumes {
			if volume.Name == newVolume.Name {
				oldVolume = &volume
				break
			}
		}
		if oldVolume == nil {
			if !newVolume.IsHotpluggable() {
				errs = append(errs, field.Forbidden(field.NewPath("spec").Child("volumes").Key(newVolume.Name), "Unhotpluggable volume may not be hotplugged"))
			}
		}
	}
	return errs
}

func generateMAC() (net.HardwareAddr, error) {
	prefix := []byte{0x52, 0x54, 0x00}
	suffix := make([]byte, 3)
	if _, err := rand.Read(suffix); err != nil {
		return nil, fmt.Errorf("rand: %s", err)
	}
	return net.HardwareAddr(append(prefix, suffix...)), nil
}
