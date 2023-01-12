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
	CPU         CPU          `json:"cpu,omitempty"`
	Memory      Memory       `json:"memory,omitempty"`
	Kernel      *Kernel      `json:"kernel,omitempty"`
	Disks       []Disk       `json:"disks,omitempty"`
	FileSystems []FileSystem `json:"fileSystems,omitempty"`
	Interfaces  []Interface  `json:"interfaces,omitempty"`
}

type CPU struct {
	Sockets               uint32 `json:"sockets,omitempty"`
	CoresPerSocket        uint32 `json:"coresPerSocket,omitempty"`
	DedicatedCPUPlacement bool   `json:"dedicatedCPUPlacement,omitempty"`
}

type Memory struct {
	Size      resource.Quantity `json:"size,omitempty"`
	Hugepages *Hugepages        `json:"hugepages,omitempty"`
}

type Hugepages struct {
	// +kubebuilder:default="1Gi"
	// +kubebuilder:validation:Enum="2Mi";"1Gi"
	PageSize string `json:"pageSize,omitempty"`
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

type FileSystem struct {
	Name string `json:"name"`
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
	VhostUser  *InterfaceVhostUser  `json:"vhostUser,omitempty"`
}

type InterfaceBridge struct {
}

type InterfaceMasquerade struct {
	// CIDR for IPv4 network. Default to 10.0.2.0/30 if not specified
	IPv4CIDR string `json:"ipv4CIDR,omitempty"`
	// CIDR for IPv6 network. Default to fd10:0:2::/120 if not specified
	IPv6CIDR string `json:"ipv6CIDR,omitempty"`
}

type InterfaceSRIOV struct {
}

type InterfaceVhostUser struct {
}

type Volume struct {
	Name         string `json:"name"`
	VolumeSource `json:",inline"`
}

func (v *Volume) IsHotpluggable() bool {
	return v.PersistentVolumeClaim != nil && v.PersistentVolumeClaim.Hotpluggable ||
		v.DataVolume != nil && v.DataVolume.Hotpluggable
}

func (v *Volume) PVCName() string {
	switch {
	case v.PersistentVolumeClaim != nil:
		return v.PersistentVolumeClaim.ClaimName
	case v.DataVolume != nil:
		return v.DataVolume.VolumeName
	default:
		return ""
	}
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
	Hotpluggable bool   `json:"hotpluggable,omitempty"`
	ClaimName    string `json:"claimName"`
}

type DataVolumeVolumeSource struct {
	Hotpluggable bool   `json:"hotpluggable,omitempty"`
	VolumeName   string `json:"volumeName"`
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
	Phase        VirtualMachinePhase            `json:"phase,omitempty"`
	VMPodName    string                         `json:"vmPodName,omitempty"`
	VMPodUID     types.UID                      `json:"vmPodUID,omitempty"`
	NodeName     string                         `json:"nodeName,omitempty"`
	PowerAction  VirtualMachinePowerAction      `json:"powerAction,omitempty"`
	Migration    *VirtualMachineStatusMigration `json:"migration,omitempty"`
	Conditions   []metav1.Condition             `json:"conditions,omitempty"`
	VolumeStatus []VolumeStatus                 `json:"volumeStatus,omitempty"`
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

type VirtualMachineStatusMigration struct {
	UID                types.UID                    `json:"uid,omitempty"`
	Phase              VirtualMachineMigrationPhase `json:"phase,omitempty"`
	TargetNodeName     string                       `json:"targetNodeName,omitempty"`
	TargetNodeIP       string                       `json:"targetNodeIP,omitempty"`
	TargetNodePort     int                          `json:"targetNodePort,omitempty"`
	TargetVMPodName    string                       `json:"targetVMPodName,omitempty"`
	TargetVMPodUID     types.UID                    `json:"targetVMPodUID,omitempty"`
	TargetVolumePodUID types.UID                    `json:"targetVolumePodUID,omitempty"`
}

type VirtualMachineConditionType string

const (
	VirtualMachineMigratable VirtualMachineConditionType = "Migratable"
	VirtualMachineReady      VirtualMachineConditionType = "Ready"
)

type VolumeStatus struct {
	Name          string               `json:"name"`
	Phase         VolumePhase          `json:"phase,omitempty"`
	HotplugVolume *HotplugVolumeStatus `json:"hotplugVolume,omitempty"`
}

type VolumePhase string

const (
	VolumePending        VolumePhase = "Pending"
	VolumeAttachedToNode VolumePhase = "AttachedToNode"
	VolumeMountedToPod   VolumePhase = "MountedToPod"
	VolumeReady          VolumePhase = "Ready"
	VolumeDetaching      VolumePhase = "Detaching"
)

type HotplugVolumeStatus struct {
	VolumePodName string    `json:"volumePodName,omitempty"`
	VolumePodUID  types.UID `json:"volumePodUID,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineList is a list of VirtualMachine resources
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []VirtualMachine `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=vmm
// +kubebuilder:printcolumn:name="VM",type=string,JSONPath=`.spec.vmName`
// +kubebuilder:printcolumn:name="Source",type=string,JSONPath=`.status.sourceNodeName`
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.status.targetNodeName`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`

type VirtualMachineMigration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineMigrationSpec   `json:"spec,omitempty"`
	Status VirtualMachineMigrationStatus `json:"status,omitempty"`
}

type VirtualMachineMigrationSpec struct {
	VMName string `json:"vmName"`
}

type VirtualMachineMigrationStatus struct {
	Phase          VirtualMachineMigrationPhase `json:"phase,omitempty"`
	SourceNodeName string                       `json:"sourceNodeName,omitempty"`
	TargetNodeName string                       `json:"targetNodeName,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Scheduling;Scheduled;TargetReady;Running;Sent;Succeeded;Failed

type VirtualMachineMigrationPhase string

const (
	VirtualMachineMigrationPending     VirtualMachineMigrationPhase = "Pending"
	VirtualMachineMigrationScheduling  VirtualMachineMigrationPhase = "Scheduling"
	VirtualMachineMigrationScheduled   VirtualMachineMigrationPhase = "Scheduled"
	VirtualMachineMigrationTargetReady VirtualMachineMigrationPhase = "TargetReady"
	VirtualMachineMigrationRunning     VirtualMachineMigrationPhase = "Running"
	VirtualMachineMigrationSent        VirtualMachineMigrationPhase = "Sent"
	VirtualMachineMigrationSucceeded   VirtualMachineMigrationPhase = "Succeeded"
	VirtualMachineMigrationFailed      VirtualMachineMigrationPhase = "Failed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualMachineMigrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []VirtualMachineMigration `json:"items"`
}
