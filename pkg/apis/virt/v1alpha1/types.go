package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=vm
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Node",type=string,JSONPath=`.status.nodeName`

// VirtualMachine is a specification for a VirtualMachine resource
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec"`
	Status VirtualMachineStatus `json:"status,omitempty"`
}

// VirtualMachineSpec is the spec for a VirtualMachine resource
type VirtualMachineSpec struct {
	NodeSelector   map[string]string           `json:"nodeSelector,omitempty"`
	Affinity       *corev1.Affinity            `json:"affinity,omitempty"`
	Tolerations    []corev1.Toleration         `json:"tolerations,omitempty"`
	Resources      corev1.ResourceRequirements `json:"resources,omitempty"`
	LivenessProbe  *corev1.Probe               `json:"livenessProbe,omitempty"`
	ReadinessProbe *corev1.Probe               `json:"readinessProbe,omitempty"`

	RunPolicy RunPolicy `json:"runPolicy,omitempty"`

	Instance Instance  `json:"instance"`
	Volumes  []Volume  `json:"volumes,omitempty"`
	Networks []Network `json:"networks,omitempty"`
}

// +kubebuilder:validation:Enum=Always;RerunOnFailure;Once;Manual;Halted

type RunPolicy string

const (
	RunPolicyAlways         RunPolicy = "Always"
	RunPolicyRerunOnFailure RunPolicy = "RerunOnFailure"
	RunPolicyOnce           RunPolicy = "Once"
	RunPolicyManual         RunPolicy = "Manual"
	RunPolicyHalted         RunPolicy = "Halted"
)

type Instance struct {
	CPU        CPU         `json:"cpu,omitempty"`
	Memory     Memory      `json:"memory"`
	Kernel     *Kernel     `json:"kernel,omitempty"`
	Disks      []Disk      `json:"disks,omitempty"`
	Interfaces []Interface `json:"interfaces,omitempty"`
}

type CPU struct {
	Sockets               uint32 `json:"sockets,omitempty"`
	CoresPerSocket        uint32 `json:"coresPerSocket,omitempty"`
	DedicatedCPUPlacement bool   `json:"dedicatedCPUPlacement,omitempty"`
}

type Memory struct {
	Size *resource.Quantity `json:"size,omitempty"`
}

type Kernel struct {
	Image           string            `json:"image"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	Cmdline         string            `json:"cmdline"`
}

type Disk struct {
	Name     string `json:"name"`
	ReadOnly *bool  `json:"readOnly,omitempty"`
}

type Interface struct {
	Name                   string `json:"name"`
	MAC                    string `json:"mac,omitempty"`
	InterfaceBindingMethod `json:",inline"`
}

type InterfaceBindingMethod struct {
	Bridge     *InterfaceBridge     `json:"bridge,omitempty"`
	Masquerade *InterfaceMasquerade `json:"masquerade,omitempty"`
	SRIOV      *InterfaceSRIOV      `json:"sriov,omitempty"`
}

type InterfaceBridge struct {
}

type InterfaceMasquerade struct {
	CIDR string `json:"cidr,omitempty"`
}

type InterfaceSRIOV struct {
}

type Volume struct {
	Name         string `json:"name"`
	VolumeSource `json:",inline"`
}

type VolumeSource struct {
	ContainerDisk         *ContainerDiskVolumeSource         `json:"containerDisk,omitempty"`
	CloudInit             *CloudInitVolumeSource             `json:"cloudInit,omitempty"`
	ContainerRootfs       *ContainerRootfsVolumeSource       `json:"containerRootfs,omitempty"`
	PersistentVolumeClaim *PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	DataVolume            *DataVolumeVolumeSource            `json:"dataVolume,omitempty"`
}

type ContainerDiskVolumeSource struct {
	Image           string            `json:"image"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

type CloudInitVolumeSource struct {
	UserData              string `json:"userData,omitempty"`
	UserDataBase64        string `json:"userDataBase64,omitempty"`
	UserDataSecretName    string `json:"userDataSecretName,omitempty"`
	NetworkData           string `json:"networkData,omitempty"`
	NetworkDataBase64     string `json:"networkDataBase64,omitempty"`
	NetworkDataSecretName string `json:"networkDataSecretName,omitempty"`
}

type ContainerRootfsVolumeSource struct {
	Image           string            `json:"image"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	Size            resource.Quantity `json:"size"`
}

type PersistentVolumeClaimVolumeSource struct {
	ClaimName string `json:"claimName"`
}

type DataVolumeVolumeSource struct {
	VolumeName string `json:"volumeName"`
}

type Network struct {
	Name          string `json:"name"`
	NetworkSource `json:",inline"`
}

type NetworkSource struct {
	Pod    *PodNetworkSource    `json:"pod,omitempty"`
	Multus *MultusNetworkSource `json:"multus,omitempty"`
}

type PodNetworkSource struct {
}

type MultusNetworkSource struct {
	NetworkName string `json:"networkName"`
}

// VirtualMachineStatus is the status for a VirtualMachine resource
type VirtualMachineStatus struct {
	Phase       VirtualMachinePhase       `json:"phase,omitempty"`
	VMPodName   string                    `json:"vmPodName,omitempty"`
	VMPodUID    types.UID                 `json:"vmPodUID,omitempty"`
	NodeName    string                    `json:"nodeName,omitempty"`
	PowerAction VirtualMachinePowerAction `json:"powerAction,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Scheduling;Scheduled;Running;Succeeded;Failed;Unknown

type VirtualMachinePhase string

const (
	VirtualMachinePending    VirtualMachinePhase = "Pending"
	VirtualMachineScheduling VirtualMachinePhase = "Scheduling"
	VirtualMachineScheduled  VirtualMachinePhase = "Scheduled"
	VirtualMachineRunning    VirtualMachinePhase = "Running"
	VirtualMachineSucceeded  VirtualMachinePhase = "Succeeded"
	VirtualMachineFailed     VirtualMachinePhase = "Failed"
	VirtualMachineUnknown    VirtualMachinePhase = "Unknown"
)

// +kubebuilder:validation:Enum=PowerOn;PowerOff;Shutdown;Reset;Reboot;Pause;Resume

type VirtualMachinePowerAction string

const (
	VirtualMachinePowerOn  VirtualMachinePowerAction = "PowerOn"
	VirtualMachinePowerOff VirtualMachinePowerAction = "PowerOff"
	VirtualMachineShutdown VirtualMachinePowerAction = "Shutdown"
	VirtualMachineReset    VirtualMachinePowerAction = "Reset"
	VirtualMachineReboot   VirtualMachinePowerAction = "Reboot"
	VirtualMachinePause    VirtualMachinePowerAction = "Pause"
	VirtualMachineResume   VirtualMachinePowerAction = "Resume"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineList is a list of VirtualMachine resources
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []VirtualMachine `json:"items"`
}
