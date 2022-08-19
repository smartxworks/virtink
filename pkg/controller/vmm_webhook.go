package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/r3labs/diff/v2"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

// +kubebuilder:webhook:path=/validate-v1alpha1-virtualmachinemigration,mutating=false,failurePolicy=fail,sideEffects=None,groups=virt.virtink.smartx.com,resources=virtualmachinemigrations,verbs=create;update,versions=v1alpha1,name=validate.virtualmachinemigration.v1alpha1.virt.virtink.smartx.com,admissionReviewVersions={v1,v1beta1}

type VMMValidator struct {
	client.Client
	decoder *admission.Decoder
}

var _ admission.DecoderInjector = &VMMValidator{}
var _ admission.Handler = &VMMValidator{}

func (h *VMMValidator) InjectDecoder(decoder *admission.Decoder) error {
	h.decoder = decoder
	return nil
}

// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines,verbs=get;list
// +kubebuilder:rbac:groups=virt.virtink.smartx.com,resources=virtualmachines/status,verbs=get

func (h *VMMValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	var vmm virtv1alpha1.VirtualMachineMigration
	if err := h.decoder.Decode(req, &vmm); err != nil {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("unmarshal VMM: %s", err))
	}

	var errs field.ErrorList
	switch req.Operation {
	case admissionv1.Create:
		errs = ValidateVMM(ctx, h.Client, &vmm, nil)
	case admissionv1.Update:
		var oldVMM virtv1alpha1.VirtualMachineMigration
		if err := h.decoder.DecodeRaw(req.OldObject, &oldVMM); err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("unmarshal old VMM: %s", err))
		}
		errs = ValidateVMM(ctx, h.Client, &vmm, &oldVMM)

		changes, err := diff.Diff(oldVMM.Spec, vmm.Spec, diff.SliceOrdering(true))
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("diff VMM: %s", err))
		}

		if len(changes) != 0 {
			errs = append(errs, field.Forbidden(field.NewPath("spec"), "VMM spec may not be updated"))
		}
	default:
		return admission.Allowed("")
	}

	if len(errs) > 0 {
		return webhook.Denied(errs.ToAggregate().Error())
	}
	return admission.Allowed("")
}

func ValidateVMM(ctx context.Context, c client.Client, vmm *virtv1alpha1.VirtualMachineMigration, oldVMM *virtv1alpha1.VirtualMachineMigration) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, ValidateVMMSpec(ctx, c, vmm.Namespace, &vmm.Spec, field.NewPath("spec"))...)
	return errs
}

func ValidateVMMSpec(ctx context.Context, c client.Client, namespace string, spec *virtv1alpha1.VirtualMachineMigrationSpec, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if spec == nil {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}
	errs = append(errs, ValidateVMName(ctx, c, namespace, spec.VMName, fieldPath.Child("vmName"))...)
	return errs
}

func ValidateVMName(ctx context.Context, c client.Client, namespace string, vmName string, fieldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if vmName == "" {
		errs = append(errs, field.Required(fieldPath, ""))
		return errs
	}

	vmKey := client.ObjectKey{Namespace: namespace, Name: vmName}
	var vm virtv1alpha1.VirtualMachine
	if err := c.Get(ctx, vmKey, &vm); err != nil {
		if apierrors.IsNotFound(err) {
			errs = append(errs, field.NotFound(fieldPath, vmName))
		} else {
			errs = append(errs, field.InternalError(fieldPath, err))
		}
	}

	migratableCondition := meta.FindStatusCondition(vm.Status.Conditions, string(virtv1alpha1.VirtualMachineMigratable))
	if migratableCondition == nil {
		errs = append(errs, field.Forbidden(fieldPath, "VM migratable condition status is unknown"))
		return errs
	}

	if migratableCondition.Status != metav1.ConditionTrue {
		errs = append(errs, field.Forbidden(fieldPath, migratableCondition.Message))
	}

	return errs
}
