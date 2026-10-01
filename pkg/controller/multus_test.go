package controller

import (
	"context"
	"encoding/json"
	"testing"

	netv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

func TestBuildVMPodMultusNetworkNamespace(t *testing.T) {
	var scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(virtv1alpha1.AddToScheme(scheme))
	utilruntime.Must(netv1.AddToScheme(scheme))

	nads := []*netv1.NetworkAttachmentDefinition{{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "vm-ns",
			Name:      "local-net",
		},
	}, {
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "net-ns",
			Name:      "shared-net",
			Annotations: map[string]string{
				"k8s.v1.cni.cncf.io/resourceName": "example.com/shared-net",
			},
		},
	}}

	tests := []struct {
		networkName  string
		expectedNAD  netv1.NetworkSelectionElement
		expectedRes  string
		expectFailed bool
	}{{
		networkName: "local-net",
		expectedNAD: netv1.NetworkSelectionElement{Namespace: "vm-ns", Name: "local-net"},
	}, {
		networkName: "net-ns/shared-net",
		expectedNAD: netv1.NetworkSelectionElement{Namespace: "net-ns", Name: "shared-net"},
		expectedRes: "example.com/shared-net",
	}, {
		// The NAD is in another namespace than the VM's
		networkName:  "shared-net",
		expectFailed: true,
	}}

	for _, tc := range tests {
		vm := &virtv1alpha1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "vm-ns",
				Name:      "vm",
			},
			Spec: virtv1alpha1.VirtualMachineSpec{
				Instance: virtv1alpha1.Instance{
					CPU: virtv1alpha1.CPU{
						Sockets:        1,
						CoresPerSocket: 1,
					},
					Memory: virtv1alpha1.Memory{
						Size: resource.MustParse("1Gi"),
					},
					Interfaces: []virtv1alpha1.Interface{{
						Name: "net",
						InterfaceBindingMethod: virtv1alpha1.InterfaceBindingMethod{
							Bridge: &virtv1alpha1.InterfaceBridge{},
						},
						MAC: "c6:1c:ba:0a:45:88",
					}},
				},
				Networks: []virtv1alpha1.Network{{
					Name: "net",
					NetworkSource: virtv1alpha1.NetworkSource{
						Multus: &virtv1alpha1.MultusNetworkSource{
							NetworkName: tc.networkName,
						},
					},
				}},
			},
		}

		builder := fake.NewClientBuilder().WithScheme(scheme)
		for _, nad := range nads {
			builder = builder.WithObjects(nad.DeepCopy())
		}
		r := &VMReconciler{Client: builder.Build(), Scheme: scheme}

		vmPod, err := r.buildVMPod(context.Background(), vm)
		if tc.expectFailed {
			assert.Error(t, err, tc.networkName)
			continue
		}
		require.NoError(t, err, tc.networkName)

		var networks []netv1.NetworkSelectionElement
		require.NoError(t, json.Unmarshal([]byte(vmPod.Annotations["k8s.v1.cni.cncf.io/networks"]), &networks))
		require.Len(t, networks, 1, tc.networkName)
		assert.Equal(t, tc.expectedNAD.Namespace, networks[0].Namespace, tc.networkName)
		assert.Equal(t, tc.expectedNAD.Name, networks[0].Name, tc.networkName)

		if tc.expectedRes != "" {
			request := vmPod.Spec.Containers[0].Resources.Requests[corev1.ResourceName(tc.expectedRes)]
			assert.Equal(t, int64(1), request.Value(), tc.networkName)
		}
	}
}

func TestValidateMultusNetworkSource(t *testing.T) {
	tests := []struct {
		networkName string
		valid       bool
	}{
		{networkName: "ovs-br1", valid: true},
		{networkName: "net-ns/ovs-br1", valid: true},
		{networkName: "", valid: false},
		{networkName: "/ovs-br1", valid: false},
		{networkName: "net-ns/", valid: false},
		{networkName: "net-ns/ovs-br1/extra", valid: false},
		{networkName: "Net_NS/ovs-br1", valid: false},
		{networkName: "OVS_br1", valid: false},
	}

	for _, tc := range tests {
		errs := ValidateMultusNetworkSource(context.Background(), &virtv1alpha1.MultusNetworkSource{
			NetworkName: tc.networkName,
		}, field.NewPath("multus"))
		if tc.valid {
			assert.Empty(t, errs, tc.networkName)
		} else {
			assert.NotEmpty(t, errs, tc.networkName)
			for _, err := range errs {
				assert.Equal(t, "multus.networkName", err.Field, tc.networkName)
			}
		}
	}
}
