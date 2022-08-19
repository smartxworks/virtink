package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

func TestValidateVMM(t *testing.T) {
	var scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(virtv1alpha1.AddToScheme(scheme))

	validVMM := &virtv1alpha1.VirtualMachineMigration{
		Spec: virtv1alpha1.VirtualMachineMigrationSpec{
			VMName: "test-vm",
		},
	}

	validVM := &virtv1alpha1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-vm",
		},
		Status: virtv1alpha1.VirtualMachineStatus{
			Conditions: []metav1.Condition{{
				Type:   string(virtv1alpha1.VirtualMachineMigratable),
				Status: metav1.ConditionTrue,
			}},
		},
	}

	tests := []struct {
		vmm           *virtv1alpha1.VirtualMachineMigration
		vm            *virtv1alpha1.VirtualMachine
		invalidDetail string
	}{{
		vmm: validVMM,
		vm:  validVM,
	}, {
		vmm: func() *virtv1alpha1.VirtualMachineMigration {
			vmm := validVMM.DeepCopy()
			vmm.Spec.VMName = ""
			return vmm
		}(),
		vm: validVM,
	}, {
		vmm: validVMM,
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Status.Conditions[0].Status = metav1.ConditionFalse
			vm.Status.Conditions[0].Message = "VM with containerDisk is not migratable"
			return vm
		}(),
		invalidDetail: "VM with containerDisk is not migratable",
	}, {
		vmm: validVMM,
		vm: func() *virtv1alpha1.VirtualMachine {
			vm := validVM.DeepCopy()
			vm.Status.Conditions = []metav1.Condition{}
			return vm
		}(),
		invalidDetail: "VM migratable condition status is unknown",
	}}

	for _, tc := range tests {
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.vm).Build()
		errs := ValidateVMM(context.Background(), c, tc.vmm, nil)
		for _, err := range errs {
			assert.Contains(t, err.Detail, tc.invalidDetail)
		}
	}
}
