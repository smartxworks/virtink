// Code generated by cloud-hypervisor-client-gen. DO NOT EDIT.

package cloudhypervisor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(socketPath string) *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
				DisableKeepAlives: true,
			},
		},
	}
}

// Add a new device to the VM
func (c *Client) VmAddDevice(ctx context.Context, arg *DeviceConfig) (*PciDeviceInfo, error) {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return nil, fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.add-device", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	var ret *PciDeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decode response: %s", err)
	}

	return ret, nil
}

// Add a new disk to the VM
func (c *Client) VmAddDisk(ctx context.Context, arg *DiskConfig) (*PciDeviceInfo, error) {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return nil, fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.add-disk", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	var ret *PciDeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decode response: %s", err)
	}

	return ret, nil
}

// Add a new virtio-fs device to the VM
func (c *Client) VmAddFs(ctx context.Context, arg *FsConfig) (*PciDeviceInfo, error) {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return nil, fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.add-fs", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	var ret *PciDeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decode response: %s", err)
	}

	return ret, nil
}

// Add a new network device to the VM
func (c *Client) VmAddNet(ctx context.Context, arg *NetConfig) (*PciDeviceInfo, error) {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return nil, fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.add-net", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	var ret *PciDeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decode response: %s", err)
	}

	return ret, nil
}

// Add a new pmem device to the VM
func (c *Client) VmAddPmem(ctx context.Context, arg *PmemConfig) (*PciDeviceInfo, error) {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return nil, fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.add-pmem", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	var ret *PciDeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decode response: %s", err)
	}

	return ret, nil
}

// Add a new vDPA device to the VM
func (c *Client) VmAddVdpa(ctx context.Context, arg *VdpaConfig) (*PciDeviceInfo, error) {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return nil, fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.add-vdpa", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	var ret *PciDeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decode response: %s", err)
	}

	return ret, nil
}

// Add a new vsock device to the VM
func (c *Client) VmAddVsock(ctx context.Context, arg *VsockConfig) (*PciDeviceInfo, error) {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return nil, fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.add-vsock", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	var ret *PciDeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decode response: %s", err)
	}

	return ret, nil
}

// Boot the previously created VM instance.
func (c *Client) VmBoot(ctx context.Context) error {

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.boot", nil)
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Takes a VM coredump.
func (c *Client) VmCoredump(ctx context.Context, arg *VmCoredumpData) error {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.coredump", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Get counters from the VM
func (c *Client) VmCounters(ctx context.Context) (*VmCounters, error) {

	req, err := http.NewRequest("GET", "http://localhost/api/v1/vm.counters", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	var ret *VmCounters
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decode response: %s", err)
	}

	return ret, nil
}

// Create the cloud-hypervisor Virtual Machine (VM) instance. The instance is not booted, only created.
func (c *Client) VmCreate(ctx context.Context, arg *VmConfig) error {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.create", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Delete the cloud-hypervisor Virtual Machine (VM) instance.
func (c *Client) VmDelete(ctx context.Context) error {

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.delete", nil)
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Returns general information about the cloud-hypervisor Virtual Machine (VM) instance.
func (c *Client) VmInfo(ctx context.Context) (*VmInfo, error) {

	req, err := http.NewRequest("GET", "http://localhost/api/v1/vm.info", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	var ret *VmInfo
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decode response: %s", err)
	}

	return ret, nil
}

// Pause a previously booted VM instance.
func (c *Client) VmPause(ctx context.Context) error {

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.pause", nil)
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Trigger a power button in the VM
func (c *Client) VmPowerButton(ctx context.Context) error {

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.power-button", nil)
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Reboot the VM instance.
func (c *Client) VmReboot(ctx context.Context) error {

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.reboot", nil)
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Receive a VM migration from URL
func (c *Client) VmReceiveMigration(ctx context.Context, arg *ReceiveMigrationData) error {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.receive-migration", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Remove a device from the VM
func (c *Client) VmRemoveDevice(ctx context.Context, arg *VmRemoveDevice) error {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.remove-device", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Resize the VM
func (c *Client) VmResize(ctx context.Context, arg *VmResize) error {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.resize", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Resize a memory zone
func (c *Client) VmResizeZone(ctx context.Context, arg *VmResizeZone) error {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.resize-zone", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Restore a VM from a snapshot.
func (c *Client) VmRestore(ctx context.Context, arg *RestoreConfig) error {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.restore", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Resume a previously paused VM instance.
func (c *Client) VmResume(ctx context.Context) error {

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.resume", nil)
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Send a VM migration to URL
func (c *Client) VmSendMigration(ctx context.Context, arg *SendMigrationData) error {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.send-migration", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Shut the VM instance down.
func (c *Client) VmShutdown(ctx context.Context) error {

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.shutdown", nil)
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Returns a VM snapshot.
func (c *Client) VmSnapshot(ctx context.Context, arg *VmSnapshotConfig) error {
	reqBody, err := json.Marshal(arg)
	if err != nil {
		return fmt.Errorf("encode request: %s", err)
	}

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vm.snapshot", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

// Ping the VMM to check for API server availability
func (c *Client) VmmPing(ctx context.Context) (*VmmPingResponse, error) {

	req, err := http.NewRequest("GET", "http://localhost/api/v1/vmm.ping", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	var ret *VmmPingResponse
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("decode response: %s", err)
	}

	return ret, nil
}

// Shuts the cloud-hypervisor VMM.
func (c *Client) VmmShutdown(ctx context.Context) error {

	req, err := http.NewRequest("PUT", "http://localhost/api/v1/vmm.shutdown", nil)
	if err != nil {
		return fmt.Errorf("build request: %s", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(body))
	}

	return nil
}

type BalloonConfig struct {
	DeflateOnOom      bool  `json:"deflate_on_oom,omitempty"`
	FreePageReporting bool  `json:"free_page_reporting,omitempty"`
	Size              int64 `json:"size"`
}

type ConsoleConfig struct {
	File  string `json:"file,omitempty"`
	Iommu bool   `json:"iommu,omitempty"`
	Mode  string `json:"mode"`
}

type CpuAffinity struct {
	HostCpus []int `json:"host_cpus,omitempty"`
	Vcpu     int   `json:"vcpu,omitempty"`
}

type CpuFeatures struct {
	Amx bool `json:"amx,omitempty"`
}

type CpuTopology struct {
	CoresPerDie    int `json:"cores_per_die,omitempty"`
	DiesPerPackage int `json:"dies_per_package,omitempty"`
	Packages       int `json:"packages,omitempty"`
	ThreadsPerCore int `json:"threads_per_core,omitempty"`
}

type CpusConfig struct {
	Affinity    []*CpuAffinity `json:"affinity,omitempty"`
	BootVcpus   int            `json:"boot_vcpus"`
	Features    *CpuFeatures   `json:"features,omitempty"`
	KvmHyperv   bool           `json:"kvm_hyperv,omitempty"`
	MaxPhysBits int            `json:"max_phys_bits,omitempty"`
	MaxVcpus    int            `json:"max_vcpus"`
	Topology    *CpuTopology   `json:"topology,omitempty"`
}

type DeviceConfig struct {
	Id         string `json:"id,omitempty"`
	Iommu      bool   `json:"iommu,omitempty"`
	Path       string `json:"path"`
	PciSegment int16  `json:"pci_segment,omitempty"`
}

type DeviceNode struct {
	Children  []string                 `json:"children,omitempty"`
	Id        string                   `json:"id,omitempty"`
	PciBdf    string                   `json:"pci_bdf,omitempty"`
	Resources []map[string]interface{} `json:"resources,omitempty"`
}

type DiskConfig struct {
	Direct            bool               `json:"direct,omitempty"`
	Id                string             `json:"id,omitempty"`
	Iommu             bool               `json:"iommu,omitempty"`
	NumQueues         int                `json:"num_queues,omitempty"`
	Path              string             `json:"path"`
	PciSegment        int16              `json:"pci_segment,omitempty"`
	QueueSize         int                `json:"queue_size,omitempty"`
	RateLimiterConfig *RateLimiterConfig `json:"rate_limiter_config,omitempty"`
	Readonly          bool               `json:"readonly,omitempty"`
	VhostSocket       string             `json:"vhost_socket,omitempty"`
	VhostUser         bool               `json:"vhost_user,omitempty"`
}

type FsConfig struct {
	Id         string `json:"id,omitempty"`
	NumQueues  int    `json:"num_queues"`
	PciSegment int16  `json:"pci_segment,omitempty"`
	QueueSize  int    `json:"queue_size"`
	Socket     string `json:"socket"`
	Tag        string `json:"tag"`
}

type MemoryConfig struct {
	HotplugMethod  string              `json:"hotplug_method,omitempty"`
	HotplugSize    int64               `json:"hotplug_size,omitempty"`
	HotpluggedSize int64               `json:"hotplugged_size,omitempty"`
	HugepageSize   int64               `json:"hugepage_size,omitempty"`
	Hugepages      bool                `json:"hugepages,omitempty"`
	Mergeable      bool                `json:"mergeable,omitempty"`
	Prefault       bool                `json:"prefault,omitempty"`
	Shared         bool                `json:"shared,omitempty"`
	Size           int64               `json:"size"`
	Thp            bool                `json:"thp,omitempty"`
	Zones          []*MemoryZoneConfig `json:"zones,omitempty"`
}

type MemoryZoneConfig struct {
	File           string `json:"file,omitempty"`
	HostNumaNode   int    `json:"host_numa_node,omitempty"`
	HotplugSize    int64  `json:"hotplug_size,omitempty"`
	HotpluggedSize int64  `json:"hotplugged_size,omitempty"`
	HugepageSize   int64  `json:"hugepage_size,omitempty"`
	Hugepages      bool   `json:"hugepages,omitempty"`
	Id             string `json:"id"`
	Mergeable      bool   `json:"mergeable,omitempty"`
	Prefault       bool   `json:"prefault,omitempty"`
	Shared         bool   `json:"shared,omitempty"`
	Size           int64  `json:"size"`
}

type NetConfig struct {
	HostMac           string             `json:"host_mac,omitempty"`
	Id                string             `json:"id,omitempty"`
	Iommu             bool               `json:"iommu,omitempty"`
	Ip                string             `json:"ip,omitempty"`
	Mac               string             `json:"mac,omitempty"`
	Mask              string             `json:"mask,omitempty"`
	Mtu               int                `json:"mtu,omitempty"`
	NumQueues         int                `json:"num_queues,omitempty"`
	PciSegment        int16              `json:"pci_segment,omitempty"`
	QueueSize         int                `json:"queue_size,omitempty"`
	RateLimiterConfig *RateLimiterConfig `json:"rate_limiter_config,omitempty"`
	Tap               string             `json:"tap,omitempty"`
	VhostMode         string             `json:"vhost_mode,omitempty"`
	VhostSocket       string             `json:"vhost_socket,omitempty"`
	VhostUser         bool               `json:"vhost_user,omitempty"`
}

type NumaConfig struct {
	Cpus           []int           `json:"cpus,omitempty"`
	Distances      []*NumaDistance `json:"distances,omitempty"`
	GuestNumaId    int             `json:"guest_numa_id"`
	MemoryZones    []string        `json:"memory_zones,omitempty"`
	SgxEpcSections []string        `json:"sgx_epc_sections,omitempty"`
}

type NumaDistance struct {
	Destination int `json:"destination"`
	Distance    int `json:"distance"`
}

// Payloads to boot in guest
type PayloadConfig struct {
	Cmdline   string `json:"cmdline,omitempty"`
	Firmware  string `json:"firmware,omitempty"`
	Initramfs string `json:"initramfs,omitempty"`
	Kernel    string `json:"kernel,omitempty"`
}

// Information about a PCI device
type PciDeviceInfo struct {
	Bdf string `json:"bdf"`
	Id  string `json:"id"`
}

type PlatformConfig struct {
	IommuSegments  []int16  `json:"iommu_segments,omitempty"`
	NumPciSegments int16    `json:"num_pci_segments,omitempty"`
	OemStrings     []string `json:"oem_strings,omitempty"`
	SerialNumber   string   `json:"serial_number,omitempty"`
	Tdx            bool     `json:"tdx,omitempty"`
	Uuid           string   `json:"uuid,omitempty"`
}

type PmemConfig struct {
	DiscardWrites bool   `json:"discard_writes,omitempty"`
	File          string `json:"file"`
	Id            string `json:"id,omitempty"`
	Iommu         bool   `json:"iommu,omitempty"`
	PciSegment    int16  `json:"pci_segment,omitempty"`
	Size          int64  `json:"size,omitempty"`
}

// Defines an IO rate limiter with independent bytes/s and ops/s limits. Limits are defined by configuring each of the _bandwidth_ and _ops_ token buckets.
type RateLimiterConfig struct {
	Bandwidth *TokenBucket `json:"bandwidth,omitempty"`
	Ops       *TokenBucket `json:"ops,omitempty"`
}

type ReceiveMigrationData struct {
	ReceiverUrl string `json:"receiver_url"`
}

type RestoreConfig struct {
	Prefault  bool   `json:"prefault,omitempty"`
	SourceUrl string `json:"source_url"`
}

type RngConfig struct {
	Iommu bool   `json:"iommu,omitempty"`
	Src   string `json:"src"`
}

type SendMigrationData struct {
	DestinationUrl string `json:"destination_url"`
	Local          bool   `json:"local,omitempty"`
}

type SgxEpcConfig struct {
	Id       string `json:"id"`
	Prefault bool   `json:"prefault,omitempty"`
	Size     int64  `json:"size"`
}

// Defines a token bucket with a maximum capacity (_size_), an initial burst size (_one_time_burst_) and an interval for refilling purposes (_refill_time_). The refill-rate is derived from _size_ and _refill_time_, and it is the constant rate at which the tokens replenish. The refill process only starts happening after the initial burst budget is consumed. Consumption from the token bucket is unbounded in speed which allows for bursts bound in size by the amount of tokens available. Once the token bucket is empty, consumption speed is bound by the refill-rate.
type TokenBucket struct {
	OneTimeBurst int64 `json:"one_time_burst,omitempty"`
	RefillTime   int64 `json:"refill_time"`
	Size         int64 `json:"size"`
}

type TpmConfig struct {
	Socket string `json:"socket"`
}

type VdpaConfig struct {
	Id         string `json:"id,omitempty"`
	Iommu      bool   `json:"iommu,omitempty"`
	NumQueues  int    `json:"num_queues"`
	Path       string `json:"path"`
	PciSegment int16  `json:"pci_segment,omitempty"`
}

// Virtual machine configuration
type VmConfig struct {
	Balloon  *BalloonConfig  `json:"balloon,omitempty"`
	Console  *ConsoleConfig  `json:"console,omitempty"`
	Cpus     *CpusConfig     `json:"cpus,omitempty"`
	Devices  []*DeviceConfig `json:"devices,omitempty"`
	Disks    []*DiskConfig   `json:"disks,omitempty"`
	Fs       []*FsConfig     `json:"fs,omitempty"`
	Iommu    bool            `json:"iommu,omitempty"`
	Memory   *MemoryConfig   `json:"memory,omitempty"`
	Net      []*NetConfig    `json:"net,omitempty"`
	Numa     []*NumaConfig   `json:"numa,omitempty"`
	Payload  *PayloadConfig  `json:"payload"`
	Platform *PlatformConfig `json:"platform,omitempty"`
	Pmem     []*PmemConfig   `json:"pmem,omitempty"`
	Rng      *RngConfig      `json:"rng,omitempty"`
	Serial   *ConsoleConfig  `json:"serial,omitempty"`
	SgxEpc   []*SgxEpcConfig `json:"sgx_epc,omitempty"`
	Tpm      *TpmConfig      `json:"tpm,omitempty"`
	Vdpa     []*VdpaConfig   `json:"vdpa,omitempty"`
	Vsock    *VsockConfig    `json:"vsock,omitempty"`
	Watchdog bool            `json:"watchdog,omitempty"`
}

type VmCoredumpData struct {
	DestinationUrl string `json:"destination_url,omitempty"`
}

type VmCounters struct {
}

// Virtual Machine information
type VmInfo struct {
	Config           *VmConfig              `json:"config"`
	DeviceTree       map[string]*DeviceNode `json:"device_tree,omitempty"`
	MemoryActualSize int64                  `json:"memory_actual_size,omitempty"`
	State            string                 `json:"state"`
}

type VmRemoveDevice struct {
	Id string `json:"id,omitempty"`
}

type VmResize struct {
	DesiredBalloon int64 `json:"desired_balloon,omitempty"`
	DesiredRam     int64 `json:"desired_ram,omitempty"`
	DesiredVcpus   int   `json:"desired_vcpus,omitempty"`
}

type VmResizeZone struct {
	DesiredRam int64  `json:"desired_ram,omitempty"`
	Id         string `json:"id,omitempty"`
}

type VmSnapshotConfig struct {
	DestinationUrl string `json:"destination_url,omitempty"`
}

// Virtual Machine Monitor information
type VmmPingResponse struct {
	Version string `json:"version"`
}

type VsockConfig struct {
	Cid        int64  `json:"cid"`
	Id         string `json:"id,omitempty"`
	Iommu      bool   `json:"iommu,omitempty"`
	PciSegment int16  `json:"pci_segment,omitempty"`
	Socket     string `json:"socket"`
}
