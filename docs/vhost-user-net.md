# Vhost-user Network

The vhost-user can achieve better performance than vhost by moving the dataplane from the kernel to userspace in both the guest and host using the library DPDK.

The purpose of this document is to setup vhost-user network through [Kube-OVN](https://github.com/kubeovn/kube-ovn) or [Userspace CNI network plugin](https://github.com/intel/userspace-cni-network-plugin), and test performance of vhost-user, bridge and masquerade interface.

## Setup

### Prerequisites

#### Hugepages

It's required for VMs and DPDK PMD threads to use hugepages on vhost-user network.

To allocate hugepages-1Gi during host booting, the following cmdline could be used:

```bash
GRUB_CMDLINE_LINUX="default_hugepagesz=1G hugepagesz=1G hugepages=60"
```

Usually, the hugepages will be allocated equally from all NUMA nodes, and it can also be modified dynamically after host startup with the following command:

```bash
echo 30 > /sys/devices/system/node/node0/hugepages/hugepages-1048576kB/nr_hugepages
```

#### Dedicated NIC

The NIC is required for communication of VMs on different hosts.

For Intel NICs, the [vfio-pci](https://doc.dpdk.org/guides/linux_gsg/linux_drivers.html#vfio) driver will be used by DPDK PMD threads, the following cmdline should be used to enable `IOMMU`:

```bash
GRUB_CMDLINE_LINUX="iommu=pt intel_iommu=on"
```

For Mellanox NICs, the [bifurcated](https://doc.dpdk.org/guides/linux_gsg/linux_drivers.html#bifurcated-driver) driver will be used, and there is no need to enable `IOMMU`.

#### Dedicated CPUs

It's recommended to reserve some dedicated CPUs on each NUMA node for DPDK PMD threads to run, refer to the following cmdline:

```bash
GRUB_CMDLINE_LINUX="isolcpus=8-11"
```

### Deploy Kube-OVN CNI

The CNI [Kube-OVN](https://kubeovn.github.io/docs/v1.10.x/en/advance/dpdk/) can be used to support both vhost-user and default network for all Pods and VMs in K8S Cluster, but you can also choose [Userspace CNI](#deploy-userspace-cni) instead of Kube-OVN to support only vhost-user network and leave the default network be provided by other K8S network add-ones

#### Configure OVS-DPDK

The default configuration of Kube-OVN is as follows:

```bash
dpdk-init=true
dpdk-socket-mem="1024"
dpdk-hugepage-dir=/dev/hugepages
```

It's recommended to specify CPU and socket memory equally from all NUMA nodes for setting the CPU and hugepages affinity of PMD threads, refer to [ovs-vswitchd.conf.db](https://www.openvswitch.org/support/dist-docs/ovs-vswitchd.conf.db.5.html) for more details.

Configure OVS-DPDK on single NUMA node system with the following command:

```bash
cat > /opt/ovs-config/config.cfg <<EOF
dpdk-init=true
dpdk-lcore-mask=0xf00
pmd-cpu-mask=0xf00
dpdk-socket-mem="1024"
dpdk-socket-limit="1024"
dpdk-hugepage-dir=/dev/hugepages

EOF
```

For multiple NUMA nodes system, E.g configure two NUMA nodes system with the following command:

```bash
cat > /opt/ovs-config/config.cfg <<EOF
dpdk-init=true
dpdk-lcore-mask=0xf00
pmd-cpu-mask=0xf00
dpdk-socket-mem="1024,1024"
dpdk-socket-limit="1024,1024"
dpdk-hugepage-dir=/dev/hugepages

EOF
```

#### Configure Dedicated NIC

As we mentiond before, the PMD threads need [vfio-pci](https://doc.dpdk.org/guides/linux_gsg/linux_drivers.html#vfio) driver for Intel NICs. To make use of `vfio-pci` with the following command:

```bash
modprobe vfio-pci
driverctl set-override 0000:af:00.0 vfio-pci # Change the PCI address.
```

For Mellanox NICs, the PMD threads use [bifurcated driver](https://doc.dpdk.org/guides/linux_gsg/linux_drivers.html#bifurcated-driver) co-exits with the device kernel driver. The [NVIDIA MLNX_OFED driver](https://network.nvidia.com/products/infiniband-drivers/linux/mlnx_ofed/) will be used, you can download it and follow the document [Installing MLNX_OFED](https://docs.nvidia.com/networking/display/MLNXOFEDv551032/Installing+MLNX_OFED) to compelete installation. E.g install `MLNX` driver on Ubuntu22.04 with the following command:

```bash
wget https://www.mellanox.com/downloads/ofed/MLNX_OFED-5.9-0.5.6.0/MLNX_OFED_SRC-debian-5.9-0.5.6.0.tgz
tar xf MLNX_OFED_SRC-debian-5.9-0.5.6.0.tgz
cd MLNX_OFED_SRC-5.9-0.5.6.0/
./install.pl --without-dkms --without-fw-update --force --dpdk
```

After installation, you can run command `/etc/init.d/openibd status` to check InfiniBand device status and command `ibv_devinfo` to get the details of these devices. Refer to [DPDK NVIDIA MLX5 Ethernet Driver](https://doc.dpdk.org/guides/nics/mlx5.html#) for more information about Mellanox NICs.

Finally, create the configuration file `ovs-dpdk-config` in `/opt/ovs-config` directory on node for Kube-OVN:

```bash
cat > /opt/ovs-config/ovs-dpdk-config <<EOF
ENCAP_IP=192.168.0.10/20
DPDK_DEV=0000:af:00.0

EOF
```

> **Note**: It's required the dedicated NICs on K8S nodes can communicate with each other directly through the link layer. You should modify the `ENCAP_IP` for different OVS-DPDK nodes.

#### Configure K8S Cluster

To install K8S cluster, please refer to [Bootstrapping clusters with kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/).

Specify K8S nodes to run OVS-DPDK with the label `ovn.kubernetes.io/ovs_dp_type="userspace"`, Kube-OVN will deploy OVS-DPDK network on these nodes, and delploy OVS network on the nodes without this label:

```bash
kubectl label nodes <node> ovn.kubernetes.io/ovs_dp_type="userspace" # Change the <node> to your node name.
```

It's required to allocate dedicated CPUs and hugepages from a single NUMA node to VM for better performance. For the single NUMA node system, you can simply enable [Kubelet CPU Manager](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/) with the following command:

```bash
cat >> /var/lib/kubelet/config.yaml <<EOF
cpuManagerPolicy: static
reservedSystemCPUs: 0-1,8-11

EOF
```

For the multiple NUMA nodes system, you should use `single-numa-node` policy for [Kubelet Topology Manager](https://kubernetes.io/docs/tasks/administer-cluster/topology-manager/) with the following command:

```bash
cat >> /var/lib/kubelet/config.yaml <<EOF
topologyManagerPolicy: single-numa-node
cpuManagerPolicy: static
reservedSystemCPUs: 0-1,8-11,18-19
memoryManagerPolicy: Static
reservedMemory:
- numaNode: 0
  limits:
    memory: 2148Mi
- numaNode: 1
  limits:
    memory: 2Gi
systemReserved:
  memory: 2Gi
kubeReserved:
  memory: 2Gi

EOF
```

Finally, refer to [Install Kube-OVN](https://kubeovn.github.io/docs/v1.10.x/en/advance/dpdk/#install-kube-ovn) to install Kube-OVN CNI in K8S cluster. Currently, we choose version `v1.10.7` and modify the field `containers[0].resources.requests/limits.hugepages-2Mi: 1Gi` to `hugepages-1Gi: <num>Gi` of DaemonSet `ovs-ovn-dpdk` in the installation script, the _num_ is the number of NUMA nodes on the host, E.g `2Gi` for a two NUMA nodes host.

#### Create Vhost-user Network

The vhost-user network is attached to the Pods as a secondary interface in UDS (Unix Domain Socket) mode, you should install [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni) first.

Now use the following sample to define a vhost-user network named `ovn-dpdk`:

```yaml
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: ovn-dpdk
  namespace: default
spec:
  config: >-
    {
        "cniVersion": "0.3.0",
        "type": "kube-ovn",
        "server_socket": "/run/openvswitch/kube-ovn-daemon.sock",
        "provider": "ovn-dpdk.default.ovn",
        "vhost_user_socket_volume_name": "vhostuser-sockets",
        "vhost_user_socket_name": "sock"
    }
```

#### Create Vhost-user VM

Now use the following sample to create a vhost-user VM named `ubuntu-vhostuser`:

```yaml
kind: VirtualMachine
metadata:
  name: ubuntu-vhostuser
spec:
  resources:
    requests:
      cpu: 1
      memory: 256Mi
      hugepages-1Gi: 1Gi
    limits:
      cpu: 1
      memory: 256Mi
      hugepages-1Gi: 1Gi
  instance:
    cpu:
      sockets: 1
      coresPerSocket: 1
      dedicatedCPUPlacement: true
    memory:
      size: 1Gi
      hugepages:
        pageSize: 1Gi
    disks:
      - name: ubuntu
      - name: cloud-init
    interfaces:
      - name: vhostuser
        vhostUser: {}
  volumes:
    - name: ubuntu
      containerDisk:
        image: smartxworks/virtink-container-disk-ubuntu
    - name: cloud-init
      cloudInit:
        userData: |-
          #cloud-config
          password: password
          chpasswd: { expire: False }
          ssh_pwauth: True
  networks:
    - name: vhostuser
      multus:
        networkName: ovn-dpdk
```

The VM will be schduled only to OVS-DPDK nodes with the label `ovn.kubernetes.io/ovs_dp_type="userspace"`. You have to configure the IP address manually in VM without [DHCP](https://kubeovn.github.io/docs/v1.10.x/en/advance/dhcp/) support. We have a Subnet sample with DHCP support in [Known Issues](#known-issues).

### Deploy Userspace CNI

The Userspace CNI can be used to support only additional vhost-user network with the flexibility to configure the underlay network, but you can also choose [Kube-OVN CNI](#deploy-kube-ovn-cni) instead of Userspace CNI to support both vhost-user and default network.

#### Configure OVS-DPDK

OVS-DPDK can be installed on host from source code, refer to [Open vSwitch with DPDK](https://docs.openvswitch.org/en/latest/intro/install/dpdk/). You can also use the prebuilt package from the distribution repository, E.g on Ubuntu18.04 (or newer), OVS-DPDK can be easily installed with the following command:

```bash
apt-get update
apt-get install openvswitch-switch-dpdk
update-alternatives --set ovs-vswitchd /usr/lib/openvswitch-switch-dpdk/ovs-vswitchd-dpdk
```

You may want to run OVS-DPDK in a container without installing to many packages on host, it's currently recommended to use Kube-OVN container image `kubeovn/kube-ovn:v1.10.7-dpdk`, which includes rich library supports, E.g `mlx5` for Mellanox NICs. To run OVS-DPDK in docker, use the following command:

```bash
modprobe openvswitch
docker run -d --privileged --network host \
    -v /dev/hugepages:/dev/hugepages \
    -v /var/run/openvswitch:/var/run/openvswitch \
    -v /usr/local/var/run/openvswitch:/usr/local/var/run/openvswitch \
    -v /var/log/openvswitch:/var/log/openvswitch \
    --name ovs-dpdk kubeovn/kube-ovn:v1.10.7-dpdk sleep infinity
```

Now configure OVS-DPDK running in container, and it's similar to configure OVS-DPDK running on host. It's recommended to specify CPU and socket memory equally from all NUMA nodes for setting the CPU and hugepages affinity of PMD threads, refer to [ovs-vswitchd.conf.db](https://www.openvswitch.org/support/dist-docs/ovs-vswitchd.conf.db.5.html) for more details.

Configure OVS-DPDK on single NUMA node system with the following command:

```bash
docker exec -it ovs-dpdk bash

export PATH=$PATH:/usr/share/openvswitch/scripts
mkdir -p /usr/local/var/run/openvswitch
ovs-ctl restart --no-ovs-vswitchd --system-id=random
ovs-vsctl --no-wait set Open_vSwitch . other_config:n-revalidator-threads=4
ovs-vsctl --no-wait set Open_vSwitch . other_config:n-handler-threads=10
ovs-vsctl set Open_vSwitch . other_config:dpdk-init=true
ovs-vsctl set Open_vSwitch . other_config:dpdk-lcore-mask=0xf00
ovs-vsctl set Open_vSwitch . other_config:pmd-cpu-mask=0xf00
ovs-vsctl set Open_vSwitch . other_config:dpdk-socket-mem="1024"
ovs-vsctl set Open_vSwitch . other_config:dpdk-socket-limit="1024"
ovs-vsctl set Open_vSwitch . other_config:dpdk-hugepage-dir=/dev/hugepages
ovs-vsctl set Open_vSwitch . other_config:userspace-tso-enable=true
ovs-ctl restart --no-ovsdb-server --system-id=random
```

For multiple NUMA nodes system, E.g configure two NUMA nodes system with the following command:

```bash
docker exec -it ovs-dpdk bash

export PATH=$PATH:/usr/share/openvswitch/scripts
mkdir -p /usr/local/var/run/openvswitch
ovs-ctl restart --no-ovs-vswitchd --system-id=random
ovs-vsctl --no-wait set Open_vSwitch . other_config:n-revalidator-threads=4
ovs-vsctl --no-wait set Open_vSwitch . other_config:n-handler-threads=10
ovs-vsctl set Open_vSwitch . other_config:dpdk-init=true
ovs-vsctl set Open_vSwitch . other_config:dpdk-lcore-mask=0xf00
ovs-vsctl set Open_vSwitch . other_config:pmd-cpu-mask=0xf00
ovs-vsctl set Open_vSwitch . other_config:dpdk-socket-mem="1024,124"
ovs-vsctl set Open_vSwitch . other_config:dpdk-socket-limit="1024,1024"
ovs-vsctl set Open_vSwitch . other_config:dpdk-hugepage-dir=/dev/hugepages
ovs-vsctl set Open_vSwitch . other_config:userspace-tso-enable=true
ovs-ctl restart --no-ovsdb-server --system-id=random
```

Finally, create a `netdev` type OVS bridge named `br-int`:

```bash
ovs-vsctl --may-exist add-br br-int \
    -- set Bridge br-int datapath_type=netdev \
    -- br-set-external-id br-int bridge-id br-int \
    -- set bridge br-int fail-mode=standalone
```

#### Configure Dedicated NIC

As we mentiond before, the PMD threads need [vfio-pci](https://doc.dpdk.org/guides/linux_gsg/linux_drivers.html#vfio) driver for Intel NICs. To make use of `vfio-pci` with the following command:

```bash
modprobe vfio-pci
driverctl set-override 0000:af:00.0 vfio-pci # Change the PCI address.
```

For Mellanox NICs, the PMD threads use the [bifurcated driver](https://doc.dpdk.org/guides/linux_gsg/linux_drivers.html#bifurcated-driver) co-exits with the device kernel driver. The [NVIDIA MLNX_OFED driver](https://network.nvidia.com/products/infiniband-drivers/linux/mlnx_ofed/) will be used, you can download it and follow the document [Installing MLNX_OFED](https://docs.nvidia.com/networking/display/MLNXOFEDv551032/Installing+MLNX_OFED) to compelete installation. E.g install `MLNX` driver on Ubuntu22.04 with the following command:

```bash
wget https://www.mellanox.com/downloads/ofed/MLNX_OFED-5.9-0.5.6.0/MLNX_OFED_SRC-debian-5.9-0.5.6.0.tgz
tar xf MLNX_OFED_SRC-debian-5.9-0.5.6.0.tgz
cd MLNX_OFED_SRC-5.9-0.5.6.0/
./install.pl --without-dkms --without-fw-update --force --dpdk
```

After installation, you can run command `/etc/init.d/openibd status` to check InfiniBand device status and command `ibv_devinfo` to get the details of these devices. Refer to [DPDK NVIDIA MLX5 Ethernet Driver](https://doc.dpdk.org/guides/nics/mlx5.html#) for more information about Mellanox NICs.

Finally, use the PCI address to add the dedicated NIC to OVS bridge `br-int`:

```bash
ovs-vsctl --timeout 10 add-port br-int dpdk0 \
    -- set Interface dpdk0 type=dpdk options:dpdk-devargs=0000:58:00.0
```

#### Configure K8S Cluster

To install K8S cluster, please refer to [Bootstrapping clusters with kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/).

It's required to allocate dedicated CPUs and hugepages from a single NUMA node to VMs for better performance. For the single NUMA node system, you can simply enable [Kubelet CPU Manager](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/) with the following command:

```bash
cat >> /var/lib/kubelet/config.yaml <<EOF
cpuManagerPolicy: static
reservedSystemCPUs: 0-1,8-11

EOF
```

For the multiple NUMA nodes system, you should use `single-numa-node` policy for [Kubelet Topology Manager](https://kubernetes.io/docs/tasks/administer-cluster/topology-manager/) with the following command:

```bash
cat >> /var/lib/kubelet/config.yaml <<EOF
topologyManagerPolicy: single-numa-node
cpuManagerPolicy: static
reservedSystemCPUs: 0-1,8-11,18-19
memoryManagerPolicy: Static
reservedMemory:
- numaNode: 0
  limits:
    memory: 2148Mi
- numaNode: 1
  limits:
    memory: 2Gi
systemReserved:
  memory: 2Gi
kubeReserved:
  memory: 2Gi

EOF
```

Finally, choose a network add-ones to provide default network for cluster, E.g [Calico](https://github.com/projectcalico/calico).

#### Create Vhost-user Network

The vhost-user network is attached to the Pods as a secondary interface in UDS (Unix Domain Socket) mode, you should install [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni) first.

The Userspace binary `userspace` needs to be compiled and copied to `/opt/cni/bin` directory, refer to [Userspace CNI](https://github.com/intel/userspace-cni-network-plugin) for more details. It's recommended to use version `v1.3` (or newer), you can use command `make install && make` to compile the binary from the source code.

Now use the following sample to define a vhost-user network named `ovs-dpdk`:

```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: ovs-dpdk
spec:
  config: '{
        "cniVersion": "0.3.1",
        "type": "userspace",
        "name": "nad-ovs-dpdk",
        "kubeconfig": "/etc/cni/net.d/multus.d/multus.kubeconfig",
        "logFile": "/var/log/nad-ovs-dpdk.log",
        "logLevel": "info",
        "host": {
                "engine": "ovs-dpdk",
                "iftype": "vhostuser",
                "netType": "bridge",
                "vhost": {
                        "mode": "client"
                },
                "bridge": {
                        "bridgeName": "br-int"
                }
        },
        "container": {
                "engine": "ovs-dpdk",
                "iftype": "vhostuser",
                "netType": "interface",
                "vhost": {
                        "mode": "server"
                }
        }
    }'
```

> **Note**: The container should be configured with `server` mode in vhost-user network.

#### Create Vhost-user VM

Now use the following sample to create a vhost-user VM named `ubuntu-vhostuser`:

```yaml
kind: VirtualMachine
metadata:
  name: ubuntu-vhostuser
spec:
  resources:
    requests:
      cpu: 1
      memory: 256Mi
      hugepages-1Gi: 1Gi
    limits:
      cpu: 1
      memory: 256Mi
      hugepages-1Gi: 1Gi
  instance:
    cpu:
      sockets: 1
      coresPerSocket: 1
      dedicatedCPUPlacement: true
    memory:
      size: 1Gi
      hugepages:
        pageSize: 1Gi
    disks:
      - name: ubuntu
      - name: cloud-init
    interfaces:
      - name: pod
      - name: vhostuser
        vhostUser: {}
  volumes:
    - name: ubuntu
      containerDisk:
        image: smartxworks/virtink-container-disk-ubuntu
    - name: cloud-init
      cloudInit:
        userData: |-
          #cloud-config
          password: password
          chpasswd: { expire: False }
          ssh_pwauth: True
  networks:
    - name: pod
      pod: {}
    - name: vhostuser
      multus:
        networkName: ovs-dpdk
```

You can label the K8S nodes which can run OVS-DPDK, and use NodeSelector to schedule VMs to these nodes. You have to configure the IP address manually in VM.

## Known Issues

### Kube-OVN CNI

- The [DHCP](https://kubeovn.github.io/docs/v1.10.x/en/advance/dhcp/) is disable in default Subnet for vhost-user VMs, you may configure the IP address in VM. To create a Subnet with DHCP support, refer to the following yaml:

  ```yaml
  apiVersion: kubeovn.io/v1
  kind: Subnet
  metadata:
    name: sn-dhcp
  spec:
    cidrBlock: "172.18.0.0/16"
    default: false
    namespaces:
      - default
    disableGatewayCheck: true
    disableInterConnection: false
    excludeIps:
      - 172.18.0.1
    gateway: 172.18.0.1
    gatewayType: distributed
    natOutgoing: true
    protocol: IPv4
    provider: ovn
    vpc: ovn-cluster
    enableDHCP: true
    dhcpV4Options: "server_id=172.18.0.100,server_mac=00:00:00:2E:2F:B8,lease_time=3600,router=172.18.0.1,netmask=255.255.0.0,dns_server=223.5.5.5"
  ```

- Kube-OVN supports vhost-user over its overlay network, but the current OVS userspace TSO implementation supports flat and VLAN networks only (i.e. no support for TSO over tunneled connection [VxLAN, GRE, IPinIP, etc.]), and there seems to be no plan to [support userspace TSO over tunnel](https://github.com/openvswitch/ovs-issues/issues/269), you may experience poor TCP bandwidth without TSO support. We are discuss supporting vhost-user over underlay network in Kube-OVN, refer to issue [Support OVS-DPDK on underlay network](https://github.com/kubeovn/kube-ovn/issues/2056) for more details. To enable userspace TSO, you can currently deploy OVS-DPDK network on a single K8S node with Kube-OVN CNI, leave other nodes be deployed with OVS network. Or you can choose Userspace CNI with the flexibility to configure the underlay network.

- By Kube-OVN CNI, a VM can get one vhost-user interface at most, but you can assign more than one vhost-user interface to a VM by using Userspace CNI.

- Currently, there is only `x86` platform support for Kube-OVN OVS-DPDK container images on [Docker Hub](https://hub.docker.com/layers/kubeovn/kube-ovn/v1.10.7-dpdk/images/sha256-05bea69cebb9973354e23736d2e3a03ace24325140fec4e6e28e4b62d7305f1a?context=explore), you may build the container images for other platforms by yourself.

- Currently, Kube-OVN uses `ip_tables` module with command `iptable-legacy`, you may switch back to `ip_tables` from `nf_tables` if necessary.

- The SR-IOV VFs can not be used as dedicated NICs, otherwise the unicast frame will be dropped if the MAC address is not destined to PF or VFs.

- VMs may fail to start with error `insufficient hugepages-1Gi`, it's because the `Burstable` QoS Pod `ovs-ovn-dpdk` will use 1Gi hugepages-1Gi on each NUMA node, but these hugepages are not recorded by Kubelet Memory Manager, the VM will fail to start if it is scheduled to use these hugepages. We want to reserve some hugepages for OVS-DPDK, but it's currently not supported by K8S, refer to [Add hugepages supports for SystemReserved and KubeReserved in KubeletConfiguration](https://discuss.kubernetes.io/t/can-we-add-hugepages-supports-for-systemreserved-and-kubereserved-in-kubeletconfiguration/22049) for more details. As a workaround, you may create a `Guaranteed` QoS pod with 1Gi hugepages-1Gi resource requirment to _occupy_ the hugepages used by container `ovs-ovn-dpdk` in Kubelet Memory Manager state file, to make sure the VMs will not be scheduled to use these hugepages.

### Userspace CNI

- There is no DHCP support for vhost-user interface, you may configure the IP address in VM manually. You can assign vhost-user as the secondary interface of VM, and manage the VM over default bridge (or masquerade) interface.

- The SR-IOV VFs can not be used as dedicated NICs, otherwise the unicast frame will be dropped if the MAC address is not destined to PF or VFs.

- VMs may fail to start with error `insufficient hugepages-1Gi`, it's because the OVS-DPDK will use 1Gi hugepages-1Gi on each NUMA node for PMD threads, but these hugepages are not recorded by Kubelet Memory Manager, the VM will fail to start if it is scheduled to use these hugepages. We want to reserve some hugepages for OVS-DPDK, but it's currently not supported by K8S, refer to [Add hugepages supports for SystemReserved and KubeReserved in KubeletConfiguration](https://discuss.kubernetes.io/t/can-we-add-hugepages-supports-for-systemreserved-and-kubereserved-in-kubeletconfiguration/22049) for more details. As a workaround, you may create a `Guaranteed` QoS pod with 1Gi hugepages-1Gi resource requirment to _occupy_ the hugepages used by OVS-DPDK in Kubelet Memory Manager state file, to make sure the VMs will not be scheduled to use these hugepages.

## Performance

There is a [performance test report](./vhost-user-net-test.md) of vhost-user, bridge and masquerade interface. We choose Userspace CNI to configure underlay network with TSO support for better vhost-user TCP bandwidth.
