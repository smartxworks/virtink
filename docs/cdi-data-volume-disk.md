# CDI DataVolume disk

DataVolume is a CRD provided by the [Containerized Data Importer (CDI)](https://github.com/kubevirt/containerized-data-importer) project. The DataVolume is an abstraction on top of the standard Kubernetes PVC and can be used to automate creation and population of a PVC with data. Virtink integrates with DataVolume to dynamically create PVC and import VM image into PVC to make it as a disk of VM.

---
*Warning:* Only UEFI format images are supported by Virtink.
---

## Requirements

- A CSI driver deployed (for testing you can use [csi-driver-host-path](https://github.com/kubernetes-csi/csi-driver-host-path))
- [CDI deployed](https://github.com/kubevirt/containerized-data-importer#deploy-it)

## Examples

Create a VM with DataVolume disk.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: virt.virtink.smartx.com/v1alpha1
kind: VirtualMachine
metadata:
  name: ubuntu-datavolume
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
      dataVolume:
        volumeName: ubuntu
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
---
apiVersion: cdi.kubevirt.io/v1beta1
kind: DataVolume
metadata:
  name: ubuntu
spec:
  source:
      http:
        url: https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img
  pvc:
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 8Gi
EOF
```

CDI will create a PVC for the DataVolume and import the  https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img into the PVC, then Virtink use this image as a disk to start VM.
