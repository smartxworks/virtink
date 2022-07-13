# Virtink: Lightweight Virtualization Add-on for Kubernetes

[![build](https://github.com/smartxworks/virtink/actions/workflows/build.yml/badge.svg)](https://github.com/smartxworks/virtink/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/smartxworks/virtink)](https://goreportcard.com/report/github.com/smartxworks/virtink)
[![codecov](https://codecov.io/gh/smartxworks/virtink/branch/main/graph/badge.svg?token=6GXYM2BFLT)](https://codecov.io/gh/smartxworks/virtink)

Virtink is a [Kubernetes](https://github.com/kubernetes/kubernetes) add-on for running [Cloud Hypervisor](https://github.com/cloud-hypervisor/cloud-hypervisor) virtual machines. By using Cloud Hypervisor as the underlying hypervisor, Virtink enables a lightweight and secure way to run fully virtualized workloads in a canonical Kubernetes cluster.

Compared to [KubeVirt](https://github.com/kubevirt/kubevirt), Virtink:

- does not use libvirt or QEMU. By leveraging Cloud Hypervisor, VMs has lower memory (≈30MB) footprints, higher performance and smaller attack surface.
- does not require a long-running per-Pod launcher process, which further reduces runtime memory overhead (≈80MB).
- is an especially good fit for running fully isolated Kubernetes clusters in an existing Kubernetes cluster. See our [Cluster API provider](https://github.com/smartxworks/cluster-api-provider-virtink) and the [knest](https://github.com/smartxworks/knest) tool for more details.

Virtink consists of 3 components:

- `virt-controller` is the cluster-wide controller, responsible for creating Pods to run Cloud Hypervisor VMs.
- `virt-daemon` is the per-Node daemon, responsible for further controlling Cloud Hypervisor VMs on Node bases.
- `virt-prerunner` is the per-Pod pre-runner, responsible for preparing VM networks and building Cloud Hypervisor VM configuration.

**NOTE**: Virtink is still a work in progress, its API may change without prior notice.

## Installation

### Requirements

A few requirements need to be met before you can begin:

- Kubernetes cluster 1.16 ~ 1.24
- Kubernetes apiserver must have `--allow-privileged=true` in order to run Virtink's privileged DaemonSet. It's usually set by default.

#### Container Runtime Support

Virtink currently supports the following container runtimes:

- Docker
- containerd

Other container runtimes, which do not use virtualization features, should work too. However, they are not tested officially.

#### Hardware Virtualization Support

Hardware with virtualization support is required. You should check if `/dev/kvm` exists on each Kubernetes nodes.

### Install Virtink

Install all Virtink components:

```bash
kubectl apply -f https://github.com/smartxworks/virtink/releases/download/v0.8.0/virtink.yaml
```

Once you have deployed Virtink, you can [create your virtual machines](#create-a-vm).

## Getting Started

### Create a VM

Apply the following manifest to Kubernetes. Note it uses a container disk and as such doesn’t persist data.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: virt.virtink.smartx.com/v1alpha1
kind: VirtualMachine
metadata:
  name: ubuntu-container-disk
spec:
  instance:
    memory:
      size: 1Gi
    disks:
      - name: ubuntu
      - name: cloud-init
    interfaces:
      - name: pod
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
EOF
```

Like starting pods, it will take some time to pull the image and start running the VM. You can wait for the VM become running as follows:

```bash
kubectl wait vm ubuntu-container-disk --for jsonpath='{.status.phase}'=Running
```

### Access the VM (via SSH)

The easiest way to access the VM is via a SSH client inside the cluster. You can access the VM created above as follows:

```bash
export VM_NAME=ubuntu-container-disk
export VM_POD_NAME=$(kubectl get vm $VM_NAME -o jsonpath='{.status.vmPodName}')
export VM_IP=$(kubectl get pod $VM_POD_NAME -o jsonpath='{.status.podIP}')
kubectl run $VM_NAME-ssh --rm --image=alpine --restart=Never -it -- /bin/sh -c "apk add openssh-client && ssh ubuntu@$VM_IP"
```

Enter `password` when you are prompted to enter password, which is set by the cloud-init data in the VM manifest.

### Manage the VM

Virtink supports various VM power actions. For example, you can ACPI shutdown the VM created above as follows:

```bash
export VM_NAME=ubuntu-container-disk
export POWER_ACTION=Shutdown
kubectl patch vm $VM_NAME --subresource=status --type=merge -p "{\"status\":{\"powerAction\":\"$POWER_ACTION\"}}"
```

You can also `PowerOff`, `Reset`, `Reboot` or `Pause` a running VM, or `Resume` a paused one. To start a powered-off VM, you can `PowerOn` it.

## License

This project is distributed under the [Apache License, Version 2.0](LICENSE).
