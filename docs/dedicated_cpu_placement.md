# Dedicated CPU Placement

Some workloads require stronger guarantees in terms of latency and/or performance in order to operate acceptably. Virtink, relying on the Kubernetes CPU manager, is able to pin guest's vCPUs to the host's pCPUs.

## Kubernetes CPU Manager

Kubernetes CPU manager is a mechanism that affects the scheduling of workloads. The default `none` policy explicitly enables the existing default CPU affinity scheme, providing no affinity beyond what the OS scheduler does automatically. To enable VMs with dedicated CPU resources, the policy of Kubernetes CPU manager should be set to `static`, which allows containers in `Guaranteed` pods with integer CPU requests access to exclusive CPUs on the node. For setting CPU manager policy refer to the [Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/cpu-management-policies/).

## Requesting Dedicated CPU Resources

Setting `spec.instance.cpu.dedicatedCPUPlacement` to `true` in a VM spec will indicate the desire to allocate dedicated CPU resource to the VM.

Expressing the desired amount of VM's vCPUs must be done by setting both the guest topology in `spec.instance.cpu` (`sockets` and `coresPerSocket`) and corresponding number of vCPUs (counted as `sockets * coresPerSocket`) in `spec.resources.[requests/limits].cpu`.

Example:

```yaml
apiVersion: virt.virtink.smartx.com/v1alpha1
kind: VirtualMachine
spec:
  instance:
    cpu:
      sockets: 2
      coresPerSocket: 1
      dedicatedCPUPlacement: true
    memory:
      size: 2Gi
  resources:
    requests:
      cpu: 2
      memory: 2.2Gi
    limits:
      cpu: 2
      memory: 2.2Gi
```
