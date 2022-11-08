# Interfaces and Networks

Connecting a VM to a network consists of two parts. First, networks are specified in `spec.networks`. Then, interfaces backed by the networks are added to the VM by specifying them in `spec.instance.interfaces`. Each interface must have a corresponding network with the same name.

An interface defines a virtual network interface of a VM. A network specifies the backend of an interface and declares which logical or physical device it is connected to.

There are multiple ways of configuring an interface as well as a network.

## Networks

Networks are configured in `spec.networks`. Each network should declare its type by defining one of the following fields:

| Type     | Description                                                                                       |
| -------- | ------------------------------------------------------------------------------------------------- |
| `pod`    | Default Kubernetes network                                                                        |
| `multus` | Secondary network provided using [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni) |

### `pod` Network

A `pod` network represents the default pod `eth0` interface configured by cluster network solution that is present in each pod.

```yaml
apiVersion: virt.virtink.smartx.com/v1alpha1
kind: VirtualMachine
spec:
  instance:
    interfaces:
      - name: pod
        bridge: {}
  networks:
    - name: pod
      pod: {}
```

### `multus` Network

It is also possible to connect VMs to secondary networks using [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni). This assumes that Multus CNI is installed across your cluster and a corresponding `NetworkAttachmentDefinition` CRD was created.

The following example defines a network which uses the [Open vSwitch CNI plugin](https://github.com/k8snetworkplumbingwg/ovs-cni), which will connect the VM to Open vSwitch's bridge `br1` on the host. Other CNI plugins such as [bridge](https://www.cni.dev/plugins/current/main/bridge/) might be used as well. For their installation and usage refer to the respective project documentation.

First the `NetworkAttachmentDefinition` needs to be created.

```yaml
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: ovs-br1
spec:
  config: |
    {
      "cniVersion": "0.3.1",
      "type": "ovs",
      "bridge": "br1"
    }
```

With following definition, the VM will be connected to the secondary Open vSwitch network.

```yaml
apiVersion: virt.virtink.smartx.com/v1alpha1
kind: VirtualMachine
spec:
  instance:
    interfaces:
      - name: ovs
        bridge: {}
  networks:
    - name: ovs
      multus:
        networkName: ovs-br1
```

> **Note**: The [macvlan](https://www.cni.dev/plugins/current/main/macvlan/) CNI plugin cannot work with bridge interface, since the unicast frame to VM will be dropped without `passthru` mode.

## VM Network Interfaces

VM network interfaces are configured in `spec.instance.interfaces`. They describe properties of virtual interfaces as "seen" inside guest instances. The same network may be connected to a VM in multiple different ways, each with their own connectivity guarantees and characteristics.

Each interface should declare its type by defining one of the following fields:

| Type         | Description                                     |
| ------------ | ----------------------------------------------- |
| `bridge`     | Connect using a linux bridge                    |
| `masquerade` | Connect using iptables rules to NAT the traffic |
| `sriov`      | Passthrough a SR-IOV PCI device via VFIO        |

Each interface may also have additional configuration fields that modify properties "seen" inside guest instances, as listed below:

| Name  | Format                                     | Default value | Description                                 |
| ----- | ------------------------------------------ | ------------- | ------------------------------------------- |
| `mac` | `ff:ff:ff:ff:ff:ff` or `FF-FF-FF-FF-FF-FF` |               | MAC address as seen inside the guest system |

### `bridge` Mode

In `bridge` mode, VMs are connected to the network through a Linux bridge. The pod network IPv4 address is delegated to the VM via DHCPv4. The VM should be configured to use DHCP to acquire IPv4 addresses.

```yaml
apiVersion: virt.virtink.smartx.com/v1alpha1
kind: VirtualMachine
spec:
  instance:
    interfaces:
      - name: pod
        bridge: {}
  networks:
    - name: pod
      pod: {}
```

At this time, `bridge` mode doesn't support additional configuration fields.

> **Note**: due to IPv4 address delegation, in `bridge` mode the pod doesn't have an IP address configured, which may introduce issues with third-party solutions that may rely on it. For example, Istio may not work in this mode.

### `masquerade` Mode

In `masquerade` mode, Virtink allocates internal IP addresses to VMs and hides them behind NAT. All the traffic exiting VMs is "NAT'ed" using pod IP addresses. A guest operating system should be configured to use DHCP to acquire IPv4 addresses. Currently all ports are forwarded into the VM.

```yaml
apiVersion: virt.virtink.smartx.com/v1alpha1
kind: VirtualMachine
spec:
  instance:
    interfaces:
      - name: pod
        masquerade: {}
  networks:
    - name: pod
      pod: {}
```

> **Note**: `masquerade` is only allowed to connect to the pod network.

> **Note**: The network default CIDR is `10.0.2.0/30`, and can be configured using the `cidr` field.

### `sriov` Mode

In `sriov` mode, VMs are directly exposed to an SR-IOV PCI device, usually allocated by [SR-IOV Network Device Plugin](https://github.com/k8snetworkplumbingwg/sriov-network-device-plugin). The device is passed through into the guest operating system as a host device, using the [VFIO](https://www.kernel.org/doc/html/latest/driver-api/vfio.html#:~:text=The%20VFIO%20driver%20is%20an,non%2Dprivileged%2C%20userspace%20drivers.) userspace interface, to maintain high networking performance.

#### How to Expose SR-IOV VFs to Virtink

First you should have [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni), [SR-IOV CNI](https://github.com/k8snetworkplumbingwg/sriov-cni) and [SR-IOV Network Device Plugin](https://github.com/k8snetworkplumbingwg/sriov-network-device-plugin) installed across the cluster. For their installation and usage refer to the respective project documentation.

Then you should create some VFs on the host's SR-IOV capable device. Please consult the vendor of your device for how to do so.

To expose SR-IOV VFs to Virtink, each VF's driver should be changed to `vfio-pci`. Below is an example of how to do so:

```bash
export VF_ADDR=0000:58:01.2 # change it to your VF's PCI address
modprobe vfio_pci
export DRIVER=$(lspci -s $VF_ADDR -k | grep driver | awk '{print $5}')
echo $VF_ADDR > /sys/bus/pci/drivers/$DRIVER/unbind
export VENDOR_ID=$(lspci -s $VF_ADDR -Dn | awk '{split($3,a,":"); print a[1]}')
export DEVICE_ID=$(lspci -s $VF_ADDR -Dn | awk '{split($3,a,":"); print a[2]}')
echo $VENDOR_ID $DEVICE_ID > /sys/bus/pci/drivers/vfio-pci/new_id
```

Now create a config for SR-IOV device plugin to capture this VF:

```bash
echo $VENDOR_ID # make sure it's the vendor ID of your VF
echo $DEVICE_ID # make sure it's the device ID of your VF
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: sriovdp-config
  namespace: kube-system
data:
  config.json: |
    {
      "resourceList": [{
        "resourceName": "mellanox_SRIOV_25G",
        "selectors": {
          "vendors": ["$VENDOR_ID"],
          "devices": ["$DEVICE_ID"],
          "drivers": ["vfio-pci"]
        }
      }]
    }
EOF
```

This will expose the VF as a node resource named `intel.com/mellanox_SRIOV_25G`. You can check if the VF was successfully exposed with following command:

```bash
kubectl get nodes -o=jsonpath-as-json="{.items[*]['status.capacity']}" | grep mellanox_SRIOV_25G
```

Finally, create a `NetworkAttachmentDefinition` for the VF:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: mellanox-sriov-25g
  annotations:
    k8s.v1.cni.cncf.io/resourceName: intel.com/mellanox_SRIOV_25G
spec:
  config: |
    {
      "cniVersion": "0.3.1",
      "type": "sriov"
    }
EOF
```

This will make the VF a `multus` network for Virtink to use.

#### Start an SR-IOV VM

To create a VM that will attach to the aforementioned network, refer to the following VM spec:

```yaml
apiVersion: virt.virtink.smartx.com/v1alpha1
kind: VirtualMachine
spec:
  instance:
    interfaces:
      - name: sriov
        sriov: {}
  networks:
    - name: sriov
      multus:
        networkName: mellanox-sriov-25g
```
