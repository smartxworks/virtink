# Kubrid: Modern and Light Virtualization Add-on for Kubernetes

[![build](https://github.com/smartxworks/kubrid/actions/workflows/build.yml/badge.svg)](https://github.com/smartxworks/kubrid/actions/workflows/build.yml)

Kubrid is a [Kubernetes](https://github.com/kubernetes/kubernetes) add-on for running [Cloud Hypervisor](https://github.com/cloud-hypervisor/cloud-hypervisor) virtual machines. By using Cloud Hypervisor as the underlying hypervisor, Kubrid enables a light and secure way to run fully virtualized workloads in a canonical Kubernetes cluster.

Compared to [KubeVirt](https://github.com/kubevirt/kubevirt), Kubrid:

- does not use libvirt or QEMU. By leveraging Cloud Hypervisor, VMs has lower memory footprints, higher performance and smaller attack surface.
- does not require a long-running per-Pod launcher process, which further reduces runtime overheads.
- is an especially good fit for running fully isolated Kubernetes clusters in an existing Kubernetes cluster. See our [Cluster API provider](https://github.com/smartxworks/cluster-api-provider-kubrid) and the [skink](https://github.com/smartxworks/skink) tool for more details.

Kubrid consists of 3 components:

- `kubrid-controller` is the cluster-wide controller, responsible for creating Pods to run Cloud Hypervisor VMs.
- `kubrid-daemon` is the per-Node daemon, responsible for further controlling Cloud Hypervisor VMs on Node bases.
- `kubrid-prerunner` is the per-Pod pre-runner, responsible for preparing VM networks and building Cloud Hypervisor VM configuration.

**NOTE**: Kubrid is still a work in progress, its API may change without prior notice.

## Prerequisites

Kubrid relies on [cert-manager](https://cert-manager.io/) 1.0 or above for SSL certificate management. You can install cert-manager as follows:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.8.0/cert-manager.yaml
```

## Installation

Kubrid can be installed as follows:

```bash
kubectl apply -f https://github.com/smartxworks/kubrid/releases/download/v0.1.0/kubrid.yaml
```

## License

This project is distributed under the [Mozilla Public License Version 2.0](LICENSE).
